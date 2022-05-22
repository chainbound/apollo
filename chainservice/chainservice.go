package chainservice

import (
	"context"
	"fmt"
	"math/big"
	"sync"
	"time"

	"github.com/XMonetae-DeFi/apollo/dsl"
	"github.com/XMonetae-DeFi/apollo/generate"
	"github.com/XMonetae-DeFi/apollo/log"
	atypes "github.com/XMonetae-DeFi/apollo/types"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
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

// RunMethodCaller starts a listener on the `blocks` channel, and on every incoming block it will execute all methods concurrently
// on the given blockNumber.
func (c *ChainService) RunMethodCaller(schema *dsl.DynamicSchema, realtime bool, blocks <-chan *big.Int, out chan<- atypes.CallResult, maxWorkers int) {
	res := make(chan atypes.CallResult)
	var wg sync.WaitGroup

	go func() {
		// For every incoming blockNumber, loop over contract methods and start a goroutine for each method.
		// This way, every eth_call will happen concurrently.
		for blockNumber := range blocks {
			wg.Add(1)
			go func(blockNumber *big.Int) {
				defer wg.Done()

				for _, contract := range schema.Contracts {
					c.CallMethods(schema.Chain, contract, blockNumber, res)
				}
			}(blockNumber)
		}

		wg.Wait()

		// When all of our methods have executed AND the blocks channel was closed on the other side,
		// close the out channel
		close(res)
	}()

	// If we're in realtime mode, add the current timestamp.
	// Most blockchains have very rough Block.Timestamp updates,
	// which are not realtime at all.
	go func() {
		for r := range res {
			if realtime {
				r.Timestamp = uint64(time.Now().UnixMilli() / 1000)
			}

			out <- r
		}
		close(out)
	}()
}

// CallMethods executes all the methods on the contract, and aggregates their results into a CallResult
func (c ChainService) CallMethods(chain atypes.Chain, contract *dsl.Contract, blockNumber *big.Int, out chan<- atypes.CallResult) {
	inputs := make(map[string]any)
	outputs := make(map[string]any)

	// If there are no methods on the contract, return
	if len(contract.Methods) == 0 {
		return
	}

	c.logger.Debug().Str("contract", contract.Name).Msg("calling contract methods")

	ctx, cancel := context.WithTimeout(context.Background(), c.defaultTimeout)
	defer cancel()

	for _, method := range contract.Methods {
		c.rateLimiter.Take()
		msg, err := generate.BuildCallMsg(contract.Address(), method, contract.Abi)
		if err != nil {
			out <- atypes.CallResult{
				Err: fmt.Errorf("building call message: %w", err),
			}
			return
		}
		c.logger.Trace().Str("to", msg.To.String()).Str("input", common.Bytes2Hex(msg.Data)).Msg("built call message")

		raw, err := c.client.CallContract(ctx, msg, blockNumber)
		if err != nil {
			out <- atypes.CallResult{
				Err: fmt.Errorf("calling contract method: %w", err),
			}
			return
		}
		c.logger.Trace().Str("to", msg.To.String()).Str("method", method.Name()).Msg("called method")

		// We only want the correct value here (specified in the schema)
		results, err := contract.Abi.Unpack(method.Name(), raw)
		if err != nil {
			out <- atypes.CallResult{
				Err: fmt.Errorf("unpacking abi for %s: %w", method.Name(), err),
			}
			return
		}

		for _, o := range method.Outputs {
			result := matchABIValue(o, contract.Abi.Methods[method.Name()].Outputs, results)
			outputs[o] = result
		}

		for k, v := range method.Inputs() {
			inputs[k] = v
		}
	}

	actualBlockNumber := uint64(0)
	block, err := c.client.HeaderByNumber(ctx, blockNumber)
	if err != nil {
		out <- atypes.CallResult{
			Err: err,
		}
		return
	}

	if blockNumber == nil {
		actualBlockNumber = block.Number.Uint64()
	} else {
		actualBlockNumber = blockNumber.Uint64()
	}

	out <- atypes.CallResult{
		Type:            atypes.Method,
		BlockNumber:     actualBlockNumber,
		Timestamp:       block.Time,
		Chain:           chain,
		ContractName:    contract.Name,
		ContractAddress: contract.Address(),
		Inputs:          inputs,
		Outputs:         outputs,
	}
}

func (c ChainService) FilterEvents(schema *dsl.DynamicSchema, fromBlock, toBlock *big.Int, out chan<- atypes.CallResult, maxWorkers int) {
	res := make(chan atypes.CallResult)
	var wg sync.WaitGroup

	if toBlock.Cmp(big.NewInt(0)) == 0 {
		toBlock = nil
	}

	go func() {
		for _, cs := range schema.Contracts {
			for _, event := range cs.Events {
				c.logger.Debug().Str("contract", cs.Name).Str("event", event.Name()).Msg("filtering contract events")
				// Get first topic in Bytes (to filter events)
				topic, err := generate.GetTopic(event.Name(), cs.Abi)
				if err != nil {
					res <- atypes.CallResult{
						Err: fmt.Errorf("generating topic id: %w", err),
					}
					return
				}

				indexedEvents := make(map[string]int)
				abiEvent := cs.Abi.Events[event.Name()]

				// Collect the indexes for the events that are "indexed" (they appear in the "topics" of the log)
				for i, arg := range abiEvent.Inputs {
					if arg.Indexed {
						for _, o := range event.Outputs() {
							if arg.Name == o {
								// First index is always the main topic
								indexedEvents[arg.Name] = i + 1
							}
						}
					}
				}

				// NOTE: eth_getLogs allows for unlimited returned logs as long as the block range is <= 2000,
				// but at a block range of 2000, we're going to need a lot of requests. For now we can try to run
				// it with this hardcoded value, but we might need to read it from a config / implement a retry pattern.
				blockRange := int64(4096)
				blockDiff := toBlock.Int64() - fromBlock.Int64()
				if blockDiff < blockRange {
					blockRange = blockDiff
				}

				for i := fromBlock.Int64(); i < toBlock.Int64(); i += blockRange {
					ctx, cancel := context.WithTimeout(context.Background(), c.defaultTimeout)
					defer cancel()

					start, end := big.NewInt(i), big.NewInt(i+blockRange-1)
					logs, err := c.client.FilterLogs(ctx, ethereum.FilterQuery{
						FromBlock: start,
						ToBlock:   end,
						Addresses: []common.Address{cs.Address()},
						Topics: [][]common.Hash{
							{topic},
						},
					})

					if err != nil {
						res <- atypes.CallResult{
							Err: fmt.Errorf("getting logs from node: %w", err),
						}
						return
					}

					c.logger.Trace().Str("start_block", start.String()).Str("end_block", end.String()).Int("n_logs", len(logs)).Msg("filtered logs")

					for _, log := range logs {
						wg.Add(1)
						go func(log types.Log) {
							defer wg.Done()
							result, err := c.HandleLog(log, schema.Chain, cs, event, indexedEvents)
							if err != nil {
								res <- atypes.CallResult{
									Err: fmt.Errorf("handling log: %w", err),
								}
								return
							}

							res <- *result
						}(log)
					}
				}
			}
		}

		wg.Wait()

		close(res)
	}()

	// If we called more than one method, we want to aggregate the results
	go func() {
		for r := range res {
			out <- r
		}
	}()
}

func (c ChainService) ListenForEvents(schema *dsl.DynamicSchema, out chan<- atypes.CallResult, maxWorkers int) {
	res := make(chan atypes.CallResult)
	logChan := make(chan types.Log)
	var wg sync.WaitGroup

	go func() {
		for _, cs := range schema.Contracts {
			for _, event := range cs.Events {
				// Get first topic in Bytes (to filter events)
				topic, err := generate.GetTopic(event.Name_, cs.Abi)
				if err != nil {
					res <- atypes.CallResult{
						Err: fmt.Errorf("generating topic id: %w", err),
					}
					return
				}

				indexedEvents := make(map[string]int)
				abiEvent := cs.Abi.Events[event.Name_]

				// Collect the indexes for the events that are "indexed" (they appear in the "topics" of the log)
				for i, arg := range abiEvent.Inputs {
					if arg.Indexed {
						for _, o := range event.Outputs_ {
							if arg.Name == o {
								// First index is always the main topic
								indexedEvents[arg.Name] = i + 1
							}
						}
					}
				}

				ctx, cancel := context.WithTimeout(context.Background(), c.defaultTimeout)
				defer cancel()

				sub, err := c.client.SubscribeFilterLogs(ctx, ethereum.FilterQuery{
					Addresses: []common.Address{cs.Address()},
					Topics: [][]common.Hash{
						{topic},
					},
				}, logChan)
				if err != nil {
					res <- atypes.CallResult{
						Err: fmt.Errorf("subscribing to logs: %w", err),
					}
					return
				}

				c.logger.Debug().Str("contract", cs.Name).Str("event", event.Name()).Msg("subscribed to events")

				defer sub.Unsubscribe()

				for log := range logChan {
					wg.Add(1)
					go func(log types.Log) {
						defer wg.Done()
						result, err := c.HandleLog(log, schema.Chain, cs, event, indexedEvents)
						if err != nil {
							res <- atypes.CallResult{
								Err: fmt.Errorf("handling log: %w", err),
							}
							return
						}

						res <- *result
					}(log)
				}
			}
		}
	}()

	// If we're in realtime mode, add the current timestamp.
	// Most blockchains have very rough Block.Timestamp updates,
	// which are not realtime at all.
	go func() {
		for r := range res {
			r.Timestamp = uint64(time.Now().UnixMilli() / 1000)

			out <- r
		}

		close(out)
	}()
}

func (c ChainService) HandleLog(log types.Log, chain atypes.Chain, cs *dsl.Contract, event *dsl.Event, indexedEvents map[string]int) (*atypes.CallResult, error) {
	// Check and wait for rate limiter if necessary
	c.rateLimiter.Take()
	ctx, cancel := context.WithTimeout(context.Background(), c.defaultTimeout)
	defer cancel()

	outputs := make(map[string]any)
	for _, event := range event.Outputs_ {
		if idx, ok := indexedEvents[event]; ok {
			outputs[event] = fmt.Sprint(common.BytesToAddress(log.Topics[idx][:]))
		}
	}

	tmp := make(map[string]any)
	if len(outputs) < len(event.Outputs_) {
		err := cs.Abi.UnpackIntoMap(tmp, event.Name_, log.Data)
		if err != nil {
			return nil, fmt.Errorf("unpacking log.Data: %w", err)
		}
	}

	for k, v := range tmp {
		outputs[k] = v
	}

	h, err := c.client.HeaderByNumber(ctx, big.NewInt(int64(log.BlockNumber)))
	if err != nil {
		if err != nil {
			return nil, fmt.Errorf("getting block header: %w", err)
		}
	}

	return &atypes.CallResult{
		Type:            atypes.Event,
		Chain:           chain,
		ContractName:    cs.Name,
		ContractAddress: cs.Address(),
		BlockNumber:     log.BlockNumber,
		TxHash:          log.TxHash,
		Timestamp:       h.Time,
		Outputs:         outputs,
	}, nil
}

func matchABIValue(outputName string, outputs abi.Arguments, results []any) any {
	if len(results) == 1 {
		return results[0]
	}

	for i, o := range outputs {
		if o.Name == outputName {
			return results[i]
		}
	}

	return nil
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
