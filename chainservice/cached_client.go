package chainservice

import (
	"context"
	"fmt"
	"math/big"
	"time"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethclient"
	lru "github.com/hashicorp/golang-lru"
)

type CachedClient struct {
	client                 *ethclient.Client
	contractCallRequests   uint64
	headerByNumberRequests uint64
	subscribeRequests      uint64
	filterRequests         uint64

	// We use a requestCache here for caching requests.
	// This could save us a lot of calls for requests
	// like erc20.decimals(), erc20.name(), or other
	// immutable values.
	requestCache *lru.Cache
}

func NewCachedClient(client *ethclient.Client) *CachedClient {
	hc, _ := lru.New(8192)
	return &CachedClient{
		client:       client,
		requestCache: hc,
	}
}

// PROBLEM: this key as a cache will only work when it's the same blocknumber AND to address
// We want it to cache common static requests as well, like erc20 decimals, symbol, and name
func genCallKey(msg ethereum.CallMsg, blockNumber *big.Int) string {
	if common.Bytes2Hex(msg.Data) == "313ce567" {
		return msg.To.String()
	}

	return msg.To.String() + string(msg.Data) + blockNumber.String()
}

func (c *CachedClient) CallContract(ctx context.Context, msg ethereum.CallMsg, blockNumber *big.Int) ([]byte, error) {
	key := genCallKey(msg, blockNumber)
	if data, ok := c.requestCache.Get(key); ok {
		return data.([]byte), nil
	}

	c.contractCallRequests++

	data, err := c.client.CallContract(ctx, msg, blockNumber)
	if err != nil {
		return nil, err
	}

	// Cache the request
	c.requestCache.Add(key, data)

	return data, nil
}

func (c *CachedClient) HeaderByNumber(ctx context.Context, number *big.Int) (*types.Header, error) {
	if number != nil {
		if header, ok := c.requestCache.Get(number.Int64()); ok {
			return header.(*types.Header), nil
		}
	}

	c.headerByNumberRequests++

	header, err := c.client.HeaderByNumber(ctx, number)
	if err != nil {
		return nil, err
	}

	c.requestCache.Add(header.Number.Int64(), header)

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

// SmartFilterLogs splits up the range in equally large parts. It will concurrently try to get all the logs for the parts,
// but if one fails because the response was too large, it will split them up in smaller parts and do the same thing.
func (c *CachedClient) SmartFilterLogs(ctx context.Context, topics [][]common.Hash, fromBlock, toBlock *big.Int) ([]types.Log, error) {
	var logs []types.Log

	parts := int64(100)
	from := fromBlock.Int64()
	to := toBlock.Int64()

	retry := func(parts int64) ([]types.Log, error) {
		fmt.Println("retrying with parts", parts)
		chunk := (to - from) / parts
		for i := from; i < to; i += chunk {
			ctx, cancel := context.WithTimeout(context.Background(), time.Second*30)
			defer cancel()
			// Don't go over the end
			if i+chunk > to {
				chunk = to - i
			}

			fmt.Println("from", i, "to", i+chunk)

			// time.Sleep(time.Second * 5)
			// wg.Add(1)
			// go func(i int64, chunk int64) {
			// 	defer wg.Done()
			res, err := c.FilterLogs(ctx, ethereum.FilterQuery{
				Topics:    topics,
				FromBlock: big.NewInt(i),
				ToBlock:   big.NewInt(i + chunk),
			})

			if err != nil {
				return nil, err
			}

			fmt.Println(len(res))

			logs = append(logs, res...)
		}

		return logs, nil
	}

	// Keep looping until we get a result. That's the only time this loop will break,
	// otherwise it will just keep retrying by increasing the parts.
	for {
		logs, err := retry(parts)
		if err != nil {
			fmt.Println(err)
			parts *= 2

			continue
		}

		return logs, nil
	}
}
