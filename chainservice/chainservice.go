package chainservice

import (
	"context"
	"fmt"
	"time"

	"github.com/chainbound/apollo/log"

	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/rs/zerolog"
	"github.com/zclconf/go-cty/cty"
	"go.uber.org/ratelimit"
)

type ChainService struct {
	client     *ethclient.Client
	blockDater BlockDater
	logger     zerolog.Logger

	defaultTimeout time.Duration
	rateLimiter    ratelimit.Limiter
}

func NewChainService(defaultTimeout time.Duration, requestsPerSecond int) *ChainService {
	return &ChainService{
		defaultTimeout: defaultTimeout,
		rateLimiter:    ratelimit.New(requestsPerSecond),
		logger:         log.NewLogger("chainservice"),
	}
}

func (c *ChainService) Connect(ctx context.Context, rpcUrl string) (*ChainService, error) {
	client, err := ethclient.DialContext(ctx, rpcUrl)
	if err != nil {
		return nil, fmt.Errorf("Connect: %w", err)
	}

	c.logger.Debug().Str("rpc", rpcUrl).Msg("connected to rpc")

	c.client = client
	c.blockDater = NewBlockDater(client)
	return c, nil
}

func (c ChainService) IsConnected() bool {
	if c.client == nil {
		return false
	} else {
		_, err := c.client.NetworkID(context.Background())
		return err == nil
	}
}

type EvaluationResult struct {
	Name string
	Err  error
	Res  map[string]cty.Value
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
