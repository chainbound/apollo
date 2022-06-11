package chainservice

import (
	"context"
	"fmt"
	"math/big"
	"time"

	"github.com/chainbound/apollo/log"
	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethclient"
	lru "github.com/hashicorp/golang-lru"
	"github.com/rs/zerolog"
)

type CachedClient struct {
	client                 *ethclient.Client
	contractCallRequests   uint64
	headerByNumberRequests uint64
	subscribeRequests      uint64
	filterRequests         uint64

	logger zerolog.Logger

	// We use a cache here for caching requests.
	// This could save us a lot of calls for requests
	// like erc20.decimals(), erc20.name(), or other
	// immutable values.
	cache       *lru.Cache
	headerCache *lru.Cache
	cacheHits   int64
}

func NewCachedClient(client *ethclient.Client) *CachedClient {
	cache, _ := lru.New(8192)
	hc, _ := lru.New(8192)
	return &CachedClient{
		client:      client,
		cache:       cache,
		headerCache: hc,
		logger:      log.NewLogger("smart_client"),
	}
}

// genCallKey will generate a unique key per contract call. If the call
// returns an immutable value, like ERC20 metadata, the key will be the
// contract address. Otherwise it will be a combination of contract
// address, calldata and blocknumber.
func genCallKey(msg ethereum.CallMsg, blockNumber *big.Int) string {
	hex := common.Bytes2Hex(msg.Data)
	if hex == "313ce567" {
		return msg.To.String() + hex
	}

	if hex == "95d89b41" {
		return msg.To.String() + hex
	}

	return msg.To.String() + string(msg.Data) + blockNumber.String()
}

func (c *CachedClient) CallContract(ctx context.Context, msg ethereum.CallMsg, blockNumber *big.Int) ([]byte, error) {
	key := genCallKey(msg, blockNumber)
	if data, ok := c.cache.Get(key); ok {
		c.logger.Trace().Str("to", msg.To.String()).Str("data", string(data.([]byte))).Msg("cache hit")
		c.cacheHits++
		return data.([]byte), nil
	}

	c.contractCallRequests++

	data, err := c.client.CallContract(ctx, msg, blockNumber)
	if err != nil {
		return nil, err
	}

	// Cache the request
	c.cache.Add(key, data)

	return data, nil
}

func (c *CachedClient) HeaderByNumber(ctx context.Context, number *big.Int) (*types.Header, error) {
	if number != nil {
		if header, ok := c.headerCache.Get(number.Int64()); ok {
			c.cacheHits++
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

// SmartFilterLogs splits up the range in equally large parts. It will concurrently try to get all the logs for the parts,
// but if one fails because the response was too large, it will split them up in smaller parts and do the same thing.
// NOTE: this is now  done serially, because when doing it concurrently on an Erigon archive node for a lot of events,
// my (big) machine almost crashed. A reasonable improvement we can make here is to use a small amount of concurrency,
// e.g. 2 - 4 concurrent `eth_getLogs` requests. If this fails (context timeouts), we can both increase the block range (parts)
// and decrease the number of concurrent requests.
func (c *CachedClient) SmartFilterLogs(ctx context.Context, topics [][]common.Hash, fromBlock, toBlock *big.Int) ([]types.Log, error) {
	var logs []types.Log

	parts := int64(50)
	from := fromBlock.Int64()
	to := toBlock.Int64()

	retry := func(parts int64) ([]types.Log, error) {
		chunk := (to - from) / parts
		for i := from; i < to; i += chunk {
			ctx, cancel := context.WithTimeout(context.Background(), time.Second*30)
			defer cancel()
			// Don't go over the end
			if i+chunk > to {
				chunk = to - i
			}

			res, err := c.FilterLogs(ctx, ethereum.FilterQuery{
				Topics:    topics,
				FromBlock: big.NewInt(i),
				ToBlock:   big.NewInt(i + chunk),
			})

			c.logger.Debug().Int("n_logs", len(res)).Msg("got logs")

			if err != nil {
				return nil, err
			}

			logs = append(logs, res...)
		}

		return logs, nil
	}

	// Keep looping until we get a result. That's the only time this loop will break,
	// otherwise it will just keep retrying by increasing the parts.
	for {
		c.logger.Debug().Int64("parts", parts).Msg("smart filter logs")
		logs, err := retry(parts)
		if err != nil {
			fmt.Println(err)
			parts *= 2

			c.logger.Debug().Msg("failed, retrying")

			continue
		}

		return logs, nil
	}
}
