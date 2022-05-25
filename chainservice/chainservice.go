package chainservice

import (
	"context"
	"fmt"
	"math/big"
	"time"

	"github.com/chainbound/apollo/dsl"
	"github.com/chainbound/apollo/log"
	apolloTypes "github.com/chainbound/apollo/types"

	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/rs/zerolog"
	"github.com/zclconf/go-cty/cty"
	"go.uber.org/ratelimit"
)

type ChainService struct {
	rlClient   *RateLimitedClient
	blockDater BlockDater
	logger     zerolog.Logger

	defaultTimeout    time.Duration
	requestsPerSecond int
}

func NewChainService(defaultTimeout time.Duration, requestsPerSecond int) *ChainService {
	return &ChainService{
		defaultTimeout:    defaultTimeout,
		requestsPerSecond: requestsPerSecond,
		logger:            log.NewLogger("chainservice"),
	}
}

func (c *ChainService) Connect(ctx context.Context, rpcUrl string) (*ChainService, error) {
	client, err := ethclient.DialContext(ctx, rpcUrl)
	if err != nil {
		return nil, fmt.Errorf("Connect: %w", err)
	}

	c.logger.Debug().Str("rpc", rpcUrl).Msg("connected to rpc")

	c.rlClient = NewRateLimitedClient(client, ratelimit.New(c.requestsPerSecond))
	c.blockDater = NewBlockDater(client)
	return c, nil
}

func (c ChainService) IsConnected() bool {
	if c.rlClient.client == nil {
		return false
	} else {
		_, err := c.rlClient.client.NetworkID(context.Background())
		return err == nil
	}
}

type EvaluationResult struct {
	Name string
	Err  error
	Res  map[string]cty.Value
}

func (c *ChainService) Start(schema *dsl.DynamicSchema, opts apolloTypes.ApolloOpts, out chan<- apolloTypes.CallResult) {
	blocks := make(chan *big.Int)

	for _, query := range schema.Queries {
		switch {

		// CONTRACT METHODS
		case query.HasContractMethods():
			c.RunMethodCaller(query, opts.Realtime, blocks, out)

			// Start main program loop
			if opts.Realtime {
				go func() {
					for {
						blocks <- nil
						time.Sleep(time.Duration(opts.Interval) * time.Second)
					}
				}()
			} else {
				go func() {
					for i := opts.StartBlock; i < opts.EndBlock; i += opts.Interval {
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
				c.FilterGlobalEvents(query, big.NewInt(opts.StartBlock), big.NewInt(opts.EndBlock), out)
			}

		// CONTRACT EVENTS
		case query.HasContractEvents():
			if opts.Realtime {
				c.ListenForEvents(query, out)
			} else {
				c.FilterEvents(query, big.NewInt(opts.StartBlock), big.NewInt(opts.EndBlock), out)
			}
		}
	}
}

func (c ChainService) BlockByTimestamp(ctx context.Context, timestamp int64) (int64, error) {
	c.logger.Info().Int64("timestamp", timestamp).Msg("finding block number")
	n, err := c.blockDater.BlockNumberByTimestamp(ctx, timestamp)
	if err != nil {
		return 0, err
	}

	c.logger.Info().Int64("timestamp", timestamp).Int64("block_number", n).Msg("blocknumber found")
	return n, nil
}

func (c ChainService) SecondsToBlockInterval(ctx context.Context, seconds int64) (int64, error) {
	c.logger.Info().Int64("seconds", seconds).Msg("converting seconds to block interval")
	n, err := c.blockDater.SecondsToBlockInterval(ctx, seconds)
	if err != nil {
		return 0, err
	}

	c.logger.Info().Int64("seconds", seconds).Int64("blocks", n).Msg("set block interval")
	return n, nil
}

func (c ChainService) DumpMetrics() {
	c.logger.Info().Msgf("contract_calls: %d requests", c.rlClient.contractCallRequests)
	c.logger.Info().Msgf("header_by_number: %d requests", c.rlClient.headerByNumberRequests)
	c.logger.Info().Msgf("subscribe_logs: %d requests", c.rlClient.subscribeRequests)
	c.logger.Info().Msgf("filter_logs: %d requests", c.rlClient.filterRequests)
}
