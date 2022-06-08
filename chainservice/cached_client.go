package chainservice

import (
	"context"
	"fmt"
	"math/big"
	"sync"

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

func (c *CachedClient) CallContract(ctx context.Context, msg ethereum.CallMsg, blockNumber *big.Int) ([]byte, error) {
	if data, ok := c.requestCache.Get(*msg.To); ok {
		return data.([]byte), nil
	}

	c.contractCallRequests++

	data, err := c.client.CallContract(ctx, msg, blockNumber)
	if err != nil {
		return nil, err
	}

	// Cache the request
	c.requestCache.Add(*msg.To, data)

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
	var wg sync.WaitGroup

	parts := int64(30)
	from := fromBlock.Int64()
	to := toBlock.Int64()
	chunk := (to - from) / parts

	errChan := make(chan error)
	done := make(chan bool)

	var logs []types.Log

	for i := from; i < to; i += chunk {
		// Don't go over the end
		if i+chunk > to {
			chunk = to - i
		}

		fmt.Println("from", i, "to", i+chunk)

		wg.Add(1)
		go func(i int64, chunk int64) {
			defer wg.Done()
			res, err := c.FilterLogs(ctx, ethereum.FilterQuery{
				Topics:    topics,
				FromBlock: big.NewInt(i),
				ToBlock:   big.NewInt(i + chunk),
			})

			if err != nil {
				errChan <- err
			}

			fmt.Println(len(res))

			logs = append(logs, res...)
		}(i, chunk)
	}

	go func() {
		wg.Wait()
		done <- true
	}()

	select {
	case err := <-errChan:
		return nil, err
	case <-done:
		fmt.Println("DONE")
		return logs, nil
	}
}
