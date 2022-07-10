package chainservice

import (
	"context"
	"fmt"
	"math/big"
	"time"

	"github.com/chainbound/apollo/bindings/erc20"
	"github.com/chainbound/apollo/dsl"
	"github.com/chainbound/apollo/log"
	apolloTypes "github.com/chainbound/apollo/types"

	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/rs/zerolog"
	"go.uber.org/ratelimit"
)

type ChainService struct {
	logger zerolog.Logger

	// clients is a map that keeps a CachedClient per chain
	clients map[apolloTypes.Chain]*CachedClient
	// blockDaters is a map that keeps a BlockDater per chain
	blockDaters map[apolloTypes.Chain]BlockDater

	// actionsPerSecond defines how many atomic actions can be executed per second.
	// It acts as an internal rate-limiter and can be set with the --rate-limit option.
	actionsPerSecond int
	// rateLimiter is the actual rate limiter which uses actionsPerSecond. Every time an atomic that contains
	// network requests is started, we call rateLimiter.Take(), which will block if the bucket is full.
	rateLimiter ratelimit.Limiter

	// rpcs is a map from a chain to an api url.
	rpcs map[apolloTypes.Chain]string

	// defaultTimeout is the default timeout after which any network request
	// that the chainservice makes will time out.
	defaultTimeout time.Duration

	// startTime is populated when chainservice.Start is called. Used by metrics.
	startTime time.Time

	// processingTime is populated when the schema is completed.
	processingTime time.Duration

	logParts int
}

func NewChainService(defaultTimeout time.Duration, actionsPerSecond, logParts int, rpcs map[apolloTypes.Chain]string) *ChainService {
	return &ChainService{
		defaultTimeout:   defaultTimeout,
		actionsPerSecond: actionsPerSecond,
		rpcs:             rpcs,
		clients:          make(map[apolloTypes.Chain]*CachedClient),
		blockDaters:      make(map[apolloTypes.Chain]BlockDater),
		rateLimiter:      ratelimit.New(actionsPerSecond),
		logger:           log.NewLogger("chainservice"),
		logParts:         logParts,
	}
}

// Connect will create a CachedClient and a BlockDater for the given chain
// and store them in the maps.
func (c *ChainService) Connect(ctx context.Context, chain apolloTypes.Chain) (*ChainService, error) {
	client, err := ethclient.DialContext(ctx, c.rpcs[chain])
	if err != nil {
		return nil, fmt.Errorf("Connect: %w", err)
	}

	c.logger.Debug().Str("rpc", c.rpcs[chain]).Msg("connected to rpc")

	c.clients[chain] = NewCachedClient(client, c.logParts)
	c.blockDaters[chain] = NewBlockDater(client)
	return c, nil
}

// Start handles the schema. For every query
// * it calls Connect to connect the client and block dater
// * it determines the start and end blocks (using BlockDater), and run that
// query concurrently.
func (c *ChainService) Start(ctx context.Context, schema *dsl.DynamicSchema, opts apolloTypes.ApolloOpts, out chan<- apolloTypes.CallResult) error {
	queryChannels := make(map[string]chan apolloTypes.CallResult, len(schema.QuerySchemas))
	c.startTime = time.Now()

	c.logger.Info().Msgf("running with %d queries", len(schema.QuerySchemas))
	for i, query := range schema.QuerySchemas {
		// Change query name into something that is guaranteed to be unique
		// If we already have a client for this chain, don't create a new one
		if _, ok := c.clients[query.Chain]; !ok {
			if _, err := c.Connect(ctx, query.Chain); err != nil {
				return err
			}
		}

		query.StartBlock = schema.StartBlock
		query.EndBlock = schema.EndBlock
		query.BlockInterval = schema.BlockInterval

		// If we're running in realtime mode we don't need all this
		if !opts.Realtime {
			// Fill in start, end and interval blocks per query, since these can differ
			if schema.StartBlock == 0 && schema.StartTime != 0 {
				query.StartBlock, err = c.BlockByTimestamp(ctx, query.Chain, schema.StartTime)
				if err != nil {
					return err
				}
			}

			if schema.EndBlock == 0 && schema.EndTime != 0 {
				query.EndBlock, err = c.BlockByTimestamp(ctx, query.Chain, schema.EndTime)
				if err != nil {
					return err
				}
			}

			if schema.BlockInterval == 0 && schema.TimeInterval != 0 {
				query.BlockInterval, err = c.SecondsToBlockInterval(ctx, query.Chain, schema.TimeInterval)
				if err != nil {
					return err
				}
			}
		}

		queryKey := fmt.Sprintf("%d-%s", i, query.Name)
		ch := c.handleQuery(query, opts)
		// Problem, can't just use query.Name here since these are not always unique,
		// in a loop for example. We need the loop variable to
		queryChannels[queryKey] = ch
	}

	// Keep looping until all query channels are expired
	for len(queryChannels) > 0 {
		for query, ch := range queryChannels {
			select {
			case msg, ok := <-ch:
				// If there are no more messages, remove this query channel and proceed
				if !ok {
					c.logger.Debug().Str("query", query).Msg("query completed")
					delete(queryChannels, query)
					continue
				}

				// msg.QueryName = strings.Split(msg.QueryName, "-")[1]
				// fmt.Printf("%+v\n", msg)
				out <- msg
			default:
			}
		}
	}

	c.processingTime = time.Since(c.startTime)
	close(out)

	return nil
}

// handleQuery is a non-blocking function that handles an individual query. It returns a channel
// on which the results are sent.
func (c ChainService) handleQuery(query *dsl.QuerySchema, opts apolloTypes.ApolloOpts) chan apolloTypes.CallResult {
	blocks := make(chan *big.Int)
	out := make(chan apolloTypes.CallResult)
	c.logger.Debug().Str("query", query.Name).Msg("starting query")

	switch {
	// CONTRACT METHODS
	case query.HasContractMethods():
		go c.RunMethodCaller(query, opts.Realtime, blocks, out)

		// Start main program loop
		if opts.Realtime {
			go func() {
				for {
					blocks <- nil
					time.Sleep(time.Duration(query.BlockInterval) * time.Second)
				}
			}()
		} else {
			c.logger.Debug().Str("query", query.Name).Msg("running in historical mode")
			go func() {
				for i := query.StartBlock; i < query.EndBlock; i += query.BlockInterval {
					blocks <- big.NewInt(i)
				}
				close(blocks)
			}()
		}

	// GLOBAL EVENTS
	case query.HasGlobalEvents():
		c.logger.Debug().Msg("global events")
		if opts.Realtime {
			go func() {
				c.ListenForGlobalEvents(query, out)
			}()
		} else {
			go func() {
				c.FilterGlobalEvents(query, big.NewInt(query.StartBlock), big.NewInt(query.EndBlock), out)
			}()
		}

	// CONTRACT EVENTS
	case query.HasContractEvents():
		c.logger.Debug().Msg("contract events")
		if opts.Realtime {
			go func() {
				c.ListenForEvents(query, out)
			}()
		} else {
			go func() {
				c.FilterEvents(query, big.NewInt(query.StartBlock), big.NewInt(query.EndBlock), out)
			}()
		}
	}

	return out
}

func (c ChainService) BlockByTimestamp(ctx context.Context, chain apolloTypes.Chain, timestamp int64) (int64, error) {
	blockDater := c.blockDaters[chain]
	c.logger.Info().Int64("timestamp", timestamp).Msg("finding block number")
	n, err := blockDater.BlockNumberByTimestamp(ctx, timestamp)
	if err != nil {
		return 0, err
	}

	c.logger.Info().Int64("timestamp", timestamp).Int64("block_number", n).Msg("blocknumber found")
	return n, nil
}

func (c ChainService) SecondsToBlockInterval(ctx context.Context, chain apolloTypes.Chain, seconds int64) (int64, error) {
	blockDater := c.blockDaters[chain]
	c.logger.Info().Int64("seconds", seconds).Msg("converting seconds to block interval")
	n, err := blockDater.SecondsToBlockInterval(ctx, seconds)
	if err != nil {
		return 0, err
	}

	c.logger.Info().Int64("seconds", seconds).Int64("blocks", n).Msg("set block interval")
	return n, nil
}

func (c ChainService) Balance(chain apolloTypes.Chain, address common.Address, block *big.Int) (float64, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	rawInt, err := c.clients[chain].client.BalanceAt(ctx, address, block)
	if err != nil {
		return 0, err
	}

	raw := new(big.Float).SetInt(rawInt)

	decimals := new(big.Int).Exp(big.NewInt(10), big.NewInt(18), nil)

	parsed, _ := raw.Quo(raw, new(big.Float).SetInt(decimals)).Float64()

	return parsed, nil
}

func (c ChainService) TokenBalance(chain apolloTypes.Chain, address, tokenAddress common.Address, block *big.Int) (float64, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	client := c.clients[chain].client

	tokenCaller, err := erc20.NewErc20Caller(tokenAddress, client)
	if err != nil {
		return 0, fmt.Errorf("creating erc20 caller: %w", err)
	}

	opts := &bind.CallOpts{Context: ctx, BlockNumber: block}
	rawDecimals, err := tokenCaller.Decimals(opts)
	// rawInt, err := .BalanceAt(ctx, address, block)
	if err != nil {
		return 0, fmt.Errorf("reading erc20 decimals (block: %d): %w", block, err)
	}

	rawInt, err := tokenCaller.BalanceOf(opts, address)
	if err != nil {
		return 0, fmt.Errorf("reading erc20 balanceOf (block: %d): %w", block, err)
	}

	raw := new(big.Float).SetInt(rawInt)

	decimals := new(big.Int).Exp(big.NewInt(10), big.NewInt(int64(rawDecimals)), nil)

	parsed, _ := raw.Quo(raw, new(big.Float).SetInt(decimals)).Float64()

	return parsed, nil
}

func (c ChainService) DumpMetrics() {
	for chain, client := range c.clients {
		c.logger.Info().Str("chain", string(chain)).Msgf("contract_calls: %d requests", client.contractCallRequests)
		c.logger.Info().Str("chain", string(chain)).Msgf("header_by_number: %d requests", client.headerByNumberRequests)
		c.logger.Info().Str("chain", string(chain)).Msgf("subscribe_logs: %d requests", client.subscribeRequests)
		c.logger.Info().Str("chain", string(chain)).Msgf("filter_logs: %d requests", client.filterRequests)
		c.logger.Info().Str("chain", string(chain)).Msgf("cache_hits: %d requests", client.cacheHits)

		if c.processingTime == 0 {
			c.logger.Info().Str("chain", string(chain)).Msgf("processing_time: %s", time.Since(c.startTime))
		} else {
			c.logger.Info().Str("chain", string(chain)).Msgf("processing_time: %s", c.processingTime)
		}
	}
}
