package chainservice

import (
	"context"
	"math/big"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethclient"
	"go.uber.org/ratelimit"
)

type RateLimitedClient struct {
	client                 *ethclient.Client
	rateLimiter            ratelimit.Limiter
	contractCallRequests   uint64
	headerByNumberRequests uint64
	subscribeRequests      uint64
	filterRequests         uint64
}

func NewRateLimitedClient(client *ethclient.Client, rl ratelimit.Limiter) *RateLimitedClient {
	return &RateLimitedClient{
		client:      client,
		rateLimiter: rl,
	}
}

func (c *RateLimitedClient) CallContract(ctx context.Context, msg ethereum.CallMsg, blockNumber *big.Int) ([]byte, error) {
	c.contractCallRequests++
	c.rateLimiter.Take()

	return c.client.CallContract(ctx, msg, blockNumber)
}

func (c *RateLimitedClient) HeaderByNumber(ctx context.Context, number *big.Int) (*types.Header, error) {
	c.headerByNumberRequests++
	c.rateLimiter.Take()

	return c.client.HeaderByNumber(ctx, number)
}

func (c *RateLimitedClient) SubscribeFilterLogs(ctx context.Context, query ethereum.FilterQuery, ch chan<- types.Log) (ethereum.Subscription, error) {
	c.subscribeRequests++
	c.rateLimiter.Take()

	return c.client.SubscribeFilterLogs(ctx, query, ch)
}

func (c *RateLimitedClient) FilterLogs(ctx context.Context, query ethereum.FilterQuery) ([]types.Log, error) {
	c.filterRequests++
	c.rateLimiter.Take()

	return c.client.FilterLogs(ctx, query)
}
