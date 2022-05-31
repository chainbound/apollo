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
	"github.com/zclconf/go-cty/cty"
	"go.uber.org/ratelimit"
)

type ChainService struct {
	logger zerolog.Logger

	rlClients   map[apolloTypes.Chain]*RateLimitedClient
	blockDaters map[apolloTypes.Chain]BlockDater

	rpcs map[apolloTypes.Chain]string

	defaultTimeout    time.Duration
	requestsPerSecond int
}

func NewChainService(defaultTimeout time.Duration, requestsPerSecond int, rpcs map[apolloTypes.Chain]string) *ChainService {
	return &ChainService{
		defaultTimeout:    defaultTimeout,
		requestsPerSecond: requestsPerSecond,
		rpcs:              rpcs,
		rlClients:         make(map[apolloTypes.Chain]*RateLimitedClient),
		blockDaters:       make(map[apolloTypes.Chain]BlockDater),
		logger:            log.NewLogger("chainservice"),
	}
}

func (c *ChainService) Connect(ctx context.Context, chain apolloTypes.Chain) (*ChainService, error) {
	client, err := ethclient.DialContext(ctx, c.rpcs[chain])
	if err != nil {
		return nil, fmt.Errorf("Connect: %w", err)
	}

	c.logger.Debug().Str("rpc", c.rpcs[chain]).Msg("connected to rpc")

	c.rlClients[chain] = NewRateLimitedClient(client, ratelimit.New(c.requestsPerSecond))
	c.blockDaters[chain] = NewBlockDater(client)
	return c, nil
}

type EvaluationResult struct {
	Name string
	Err  error
	Res  map[string]cty.Value
}

func (c *ChainService) Start(ctx context.Context, schema *dsl.DynamicSchema, opts apolloTypes.ApolloOpts, out chan<- apolloTypes.CallResult) error {
	blocks := make(chan *big.Int)

	c.logger.Info().Msgf("running with %d queries", len(schema.Queries))
	for _, query := range schema.Queries {
		if _, err := c.Connect(ctx, query.Chain); err != nil {
			return err
		}

		startBlock := schema.StartBlock
		endBlock := schema.EndBlock
		interval := schema.Interval

		// If we're running in realtime mode we don't need all this
		if !opts.Realtime {
			if schema.StartBlock == 0 && schema.StartTime != 0 {
				startBlock, err = c.BlockByTimestamp(ctx, query.Chain, schema.StartTime)
				if err != nil {
					return err
				}
			}

			if schema.EndBlock == 0 && schema.EndTime != 0 {
				endBlock, err = c.BlockByTimestamp(ctx, query.Chain, schema.EndTime)
				if err != nil {
					return err
				}
			}

			if schema.Interval == 0 && schema.TimeInterval != 0 {
				interval, err = c.SecondsToBlockInterval(ctx, query.Chain, schema.TimeInterval)
				if err != nil {
					return err
				}
			}
		}

		go func(query *dsl.Query) {
			switch {
			// CONTRACT METHODS
			case query.HasContractMethods():
				c.RunMethodCaller(query, opts.Realtime, blocks, out)

				// Start main program loop
				if opts.Realtime {
					go func() {
						for {
							blocks <- nil
							time.Sleep(time.Duration(schema.Interval) * time.Second)
						}
					}()
				} else {
					go func() {
						for i := startBlock; i < endBlock; i += interval {
							blocks <- big.NewInt(i)
						}

						close(blocks)
					}()
				}

			// GLOBAL EVENTS
			case query.HasGlobalEvents():
				if opts.Realtime {
					c.ListenForGlobalEvents(query, out)
				} else {
					c.FilterGlobalEvents(query, big.NewInt(startBlock), big.NewInt(endBlock), out)
				}

			// CONTRACT EVENTS
			case query.HasContractEvents():
				if opts.Realtime {
					c.ListenForEvents(query, out)
				} else {
					c.FilterEvents(query, big.NewInt(startBlock), big.NewInt(endBlock), out)
				}
			}
		}(query)
	}

	return nil
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
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	rawInt, err := c.rlClients[chain].client.BalanceAt(ctx, address, block)
	if err != nil {
		return 0, err
	}

	raw := new(big.Float).SetInt(rawInt)

	decimals := new(big.Int).Exp(big.NewInt(10), big.NewInt(18), nil)

	parsed, _ := raw.Quo(raw, new(big.Float).SetInt(decimals)).Float64()

	return parsed, nil
}

func (c ChainService) TokenBalance(chain apolloTypes.Chain, address, tokenAddress common.Address, block *big.Int) (float64, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	client := c.rlClients[chain].client

	tokenCaller, err := erc20.NewErc20Caller(tokenAddress, client)
	if err != nil {
		return 0, fmt.Errorf("creating erc20 caller: %w", err)
	}

	opts := &bind.CallOpts{Context: ctx, BlockNumber: block}
	rawDecimals, err := tokenCaller.Decimals(opts)
	// rawInt, err := .BalanceAt(ctx, address, block)
	if err != nil {
		return 0, fmt.Errorf("reading erc20 decimals: %w", err)
	}

	rawInt, err := tokenCaller.BalanceOf(opts, address)
	if err != nil {
		return 0, fmt.Errorf("reading erc20 balanceOf: %w", err)
	}

	raw := new(big.Float).SetInt(rawInt)

	decimals := new(big.Int).Exp(big.NewInt(10), big.NewInt(int64(rawDecimals)), nil)

	parsed, _ := raw.Quo(raw, new(big.Float).SetInt(decimals)).Float64()

	return parsed, nil
}

func (c ChainService) DumpMetrics() {
	for chain, client := range c.rlClients {
		c.logger.Info().Str("chain", string(chain)).Msgf("contract_calls: %d requests", client.contractCallRequests)
		c.logger.Info().Str("chain", string(chain)).Msgf("header_by_number: %d requests", client.headerByNumberRequests)
		c.logger.Info().Str("chain", string(chain)).Msgf("subscribe_logs: %d requests", client.subscribeRequests)
		c.logger.Info().Str("chain", string(chain)).Msgf("filter_logs: %d requests", client.filterRequests)
	}
}
