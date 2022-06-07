package chainservice

import (
	"context"
	"math/big"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethclient"
	lru "github.com/hashicorp/golang-lru"
)

type CachedClient struct {
	client *ethclient.Client
	// rateLimiter            ratelimit.Limiter
	contractCallRequests   uint64
	headerByNumberRequests uint64
	subscribeRequests      uint64
	filterRequests         uint64

	// lruCaches are thread safe
	headerCache *lru.Cache

	decimalCache *lru.Cache
}

func NewCachedClient(client *ethclient.Client) *CachedClient {
	hc, _ := lru.New(4096)
	dc, _ := lru.New(4096)
	return &CachedClient{
		client:       client,
		headerCache:  hc,
		decimalCache: dc,
	}
}

func (c *CachedClient) CallContract(ctx context.Context, msg ethereum.CallMsg, blockNumber *big.Int) ([]byte, error) {
	if common.Bytes2Hex(msg.Data) == "313ce567" {
		if data, ok := c.decimalCache.Get(*msg.To); ok {
			return data.([]byte), nil
		}
	}

	c.contractCallRequests++

	data, err := c.client.CallContract(ctx, msg, blockNumber)
	if err != nil {
		return nil, err
	}

	// If decimals is called, we want to cache it
	if common.Bytes2Hex(msg.Data) == "313ce567" {
		c.decimalCache.Add(*msg.To, data)
	}

	return data, nil
}

func (c *CachedClient) HeaderByNumber(ctx context.Context, number *big.Int) (*types.Header, error) {
	if number != nil {
		if header, ok := c.headerCache.Get(number.Int64()); ok {
			return header.(*types.Header), nil
		}
	}

	c.headerByNumberRequests++

	header, err := c.client.HeaderByNumber(ctx, number)
	if err != nil {
		return nil, err
	}

	c.headerCache.Add(header.Number.Int64(), header)

	return header, nil
}

func (c *CachedClient) SubscribeFilterLogs(ctx context.Context, query ethereum.FilterQuery, ch chan<- types.Log) (ethereum.Subscription, error) {
	c.subscribeRequests++

	return c.client.SubscribeFilterLogs(ctx, query, ch)
}

func (c *CachedClient) FilterLogs(ctx context.Context, query ethereum.FilterQuery) ([]types.Log, error) {
	c.filterRequests++

	return c.client.FilterLogs(ctx, query)
}
