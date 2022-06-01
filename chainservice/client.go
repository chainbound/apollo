package chainservice

import (
	"context"
	"math/big"
	"sync"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethclient"
)

type MetricsClient struct {
	mu     sync.Mutex
	client *ethclient.Client
	// rateLimiter            ratelimit.Limiter
	contractCallRequests   uint64
	headerByNumberRequests uint64
	subscribeRequests      uint64
	filterRequests         uint64

	headerCache map[int64]*types.Header
}

func NewRateLimitedClient(client *ethclient.Client) *MetricsClient {
	return &MetricsClient{
		client:      client,
		headerCache: make(map[int64]*types.Header),
	}
}

func (c *MetricsClient) CallContract(ctx context.Context, msg ethereum.CallMsg, blockNumber *big.Int) ([]byte, error) {
	c.contractCallRequests++
	// c.rateLimiter.Take()

	return c.client.CallContract(ctx, msg, blockNumber)
}

func (c *MetricsClient) HeaderByNumber(ctx context.Context, number *big.Int) (*types.Header, error) {
	if header, ok := c.headerCache[number.Int64()]; ok {
		return header, nil
	}

	c.headerByNumberRequests++
	// c.rateLimiter.Take()

	header, err := c.client.HeaderByNumber(ctx, number)
	if err != nil {
		return nil, err
	}

	c.mu.Lock()
	defer c.mu.Unlock()
	c.headerCache[number.Int64()] = header

	return header, nil
}

func (c *MetricsClient) SubscribeFilterLogs(ctx context.Context, query ethereum.FilterQuery, ch chan<- types.Log) (ethereum.Subscription, error) {
	c.subscribeRequests++

	return c.client.SubscribeFilterLogs(ctx, query, ch)
}

func (c *MetricsClient) FilterLogs(ctx context.Context, query ethereum.FilterQuery) ([]types.Log, error) {
	c.filterRequests++

	return c.client.FilterLogs(ctx, query)
}
