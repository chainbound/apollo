package chainservice

import (
	"context"
	"fmt"
	"math/big"
	"sync"
	"time"

	"github.com/chainbound/apollo/dsl"
	"github.com/chainbound/apollo/generate"
	apolloTypes "github.com/chainbound/apollo/types"
	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
)

func (c ChainService) FilterEvents(schema *dsl.DynamicSchema, fromBlock, toBlock *big.Int, out chan<- apolloTypes.CallResult) {
	res := make(chan apolloTypes.CallResult)
	var wg sync.WaitGroup

	if toBlock.Cmp(big.NewInt(0)) == 0 {
		toBlock = nil
	}

	go func() {
		c.FilterGlobalEvents(schema, fromBlock, toBlock, res)
		for _, cs := range schema.Contracts {
			for _, event := range cs.Events {
				c.logger.Debug().Str("contract", cs.Name).
					Str("event", event.Name()).Str("from_block", fromBlock.String()).
					Str("to_block", toBlock.String()).Msg("filtering contract events")

				// Get first topic in Bytes (to filter events)
				topic, err := generate.GetTopic(event.Name(), cs.Abi)
				if err != nil {
					res <- apolloTypes.CallResult{
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
						res <- apolloTypes.CallResult{
							Err: fmt.Errorf("getting logs from node: %w", err),
						}
						return
					}

					c.logger.Trace().Str("start_block", start.String()).Str("end_block", end.String()).Int("n_logs", len(logs)).Msg("filtered logs")

					for _, log := range logs {
						wg.Add(1)
						go func(log types.Log) {
							defer wg.Done()
							result, err := c.HandleLog(log, schema.Chain, cs.Name, cs.Abi, event, indexedEvents)
							if err != nil {
								res <- apolloTypes.CallResult{
									Err: fmt.Errorf("handling log: %w", err),
								}
								return
							}

							results := []*apolloTypes.CallResult{result}
							for _, method := range event.Methods {
								c.logger.Trace().Int64("block_offset", method.BlockOffset).Msg("calling method at event")
								callResult, err := c.CallMethod(schema.Chain, cs.Name, cs.Address(), cs.Abi, method, big.NewInt(int64(log.BlockNumber)+method.BlockOffset))
								if err != nil {
									res <- apolloTypes.CallResult{
										Err: fmt.Errorf("calling method on event: %w", err),
									}
									return
								}

								results = append(results, callResult)
							}

							res <- *aggregateCallResults(results...)
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

func (c ChainService) FilterGlobalEvents(schema *dsl.DynamicSchema, fromBlock, toBlock *big.Int, res chan<- apolloTypes.CallResult) {
	var wg sync.WaitGroup

	for _, event := range schema.Events {
		c.logger.Debug().
			Str("event", event.Name()).Str("from_block", fromBlock.String()).
			Str("to_block", toBlock.String()).Msg("filtering global events")

		// Get first topic in Bytes (to filter events)
		topic, err := generate.GetTopic(event.Name(), event.Abi)
		if err != nil {
			res <- apolloTypes.CallResult{
				Err: fmt.Errorf("generating topic id: %w", err),
			}
			return
		}

		indexedEvents := make(map[string]int)
		abiEvent := event.Abi.Events[event.Name()]

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
				Topics: [][]common.Hash{
					{topic},
				},
			})

			if err != nil {
				res <- apolloTypes.CallResult{
					Err: fmt.Errorf("getting logs from node: %w", err),
				}
				return
			}

			c.logger.Trace().Str("start_block", start.String()).Str("end_block", end.String()).Int("n_logs", len(logs)).Msg("filtered logs")

			for _, log := range logs {
				wg.Add(1)
				go func(log types.Log) {
					defer wg.Done()
					result, err := c.HandleLog(log, schema.Chain, event.OutputName(), event.Abi, event, indexedEvents)
					if err != nil {
						res <- apolloTypes.CallResult{
							Err: fmt.Errorf("handling log: %w", err),
						}
						return
					}

					results := []*apolloTypes.CallResult{result}
					for _, method := range event.Methods {
						c.logger.Trace().Int64("block_offset", method.BlockOffset).Msg("calling method at event")
						callResult, err := c.CallMethod(schema.Chain, event.OutputName(), log.Address, event.Abi, method, big.NewInt(int64(log.BlockNumber)+method.BlockOffset))
						if err != nil {
							res <- apolloTypes.CallResult{
								Err: fmt.Errorf("calling method on event: %w", err),
							}
							return
						}

						results = append(results, callResult)
					}

					finalResult := aggregateCallResults(results...)
					finalResult.Type = apolloTypes.GlobalEvent
					res <- *finalResult
				}(log)
			}
		}
	}

	wg.Wait()
}

func (c ChainService) ListenForEvents(schema *dsl.DynamicSchema, out chan<- apolloTypes.CallResult) {
	res := make(chan apolloTypes.CallResult)
	logChan := make(chan types.Log)

	go func() {
		for _, cs := range schema.Contracts {
			for _, event := range cs.Events {
				// Get first topic in Bytes (to filter events)
				topic, err := generate.GetTopic(event.Name_, cs.Abi)
				if err != nil {
					res <- apolloTypes.CallResult{
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
					res <- apolloTypes.CallResult{
						Err: fmt.Errorf("subscribing to logs: %w", err),
					}
					return
				}

				c.logger.Debug().Str("contract", cs.Name).Str("event", event.Name()).Msg("subscribed to events")

				defer sub.Unsubscribe()

				for log := range logChan {
					go func(log types.Log) {
						result, err := c.HandleLog(log, schema.Chain, cs.Name, cs.Abi, event, indexedEvents)
						if err != nil {
							res <- apolloTypes.CallResult{
								Err: fmt.Errorf("handling log: %w", err),
							}
							return
						}

						results := []*apolloTypes.CallResult{result}
						for _, method := range event.Methods {
							c.logger.Trace().Int64("block_offset", method.BlockOffset).Msg("calling method at event")
							callResult, err := c.CallMethod(schema.Chain, cs.Name, cs.Address(), cs.Abi, method, big.NewInt(int64(log.BlockNumber)+method.BlockOffset))
							if err != nil {
								res <- apolloTypes.CallResult{
									Err: fmt.Errorf("calling method on event: %w", err),
								}
								return
							}

							results = append(results, callResult)
						}

						res <- *aggregateCallResults(results...)
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

func (c ChainService) ListenForGlobalEvents(schema *dsl.DynamicSchema, res chan<- apolloTypes.CallResult) {
	logChan := make(chan types.Log)

	for _, event := range schema.Events {
		// Get first topic in Bytes (to filter events)
		topic, err := generate.GetTopic(event.Name_, event.Abi)
		if err != nil {
			res <- apolloTypes.CallResult{
				Err: fmt.Errorf("generating topic id: %w", err),
			}
			return
		}

		indexedEvents := make(map[string]int)
		abiEvent := event.Abi.Events[event.Name_]

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
			Topics: [][]common.Hash{
				{topic},
			},
		}, logChan)
		if err != nil {
			res <- apolloTypes.CallResult{
				Err: fmt.Errorf("subscribing to logs: %w", err),
			}
			return
		}

		c.logger.Debug().Str("event", event.Name()).Msg("subscribed to global events")

		defer sub.Unsubscribe()

		for log := range logChan {
			go func(log types.Log) {
				result, err := c.HandleLog(log, schema.Chain, event.OutputName(), event.Abi, event, indexedEvents)
				if err != nil {
					res <- apolloTypes.CallResult{
						Err: fmt.Errorf("handling log: %w", err),
					}
					return
				}

				results := []*apolloTypes.CallResult{result}
				for _, method := range event.Methods {
					c.logger.Trace().Int64("block_offset", method.BlockOffset).Msg("calling method at event")
					callResult, err := c.CallMethod(schema.Chain, event.OutputName(), log.Address, event.Abi, method, big.NewInt(int64(log.BlockNumber)+method.BlockOffset))
					if err != nil {
						res <- apolloTypes.CallResult{
							Err: fmt.Errorf("calling method on event: %w", err),
						}
						return
					}

					results = append(results, callResult)
				}

				res <- *aggregateCallResults(results...)
			}(log)
		}
	}
}

func (c ChainService) HandleLog(log types.Log, chain apolloTypes.Chain, contractName string, abi abi.ABI, event *dsl.Event, indexedEvents map[string]int) (*apolloTypes.CallResult, error) {
	// Check and wait for rate limiter if necessary
	c.logger.Trace().Str("event", event.Name_).Msg("handling log")
	c.rateLimiter.Take()
	ctx, cancel := context.WithTimeout(context.Background(), c.defaultTimeout)
	defer cancel()

	outputs := make(map[string]any)
	for _, event := range event.Outputs_ {
		if idx, ok := indexedEvents[event]; ok {
			outputs[event] = fmt.Sprint(common.BytesToAddress(log.Topics[idx][:]))
		}
	}

	fmt.Println(outputs)
	// fmt.Println(abi)
	tmp := make(map[string]any)
	if len(outputs) < len(event.Outputs_) {
		err := abi.UnpackIntoMap(tmp, event.Name_, log.Data)
		if err != nil {
			return nil, fmt.Errorf("unpacking log.Data: %w", err)
		}
	}

	fmt.Println(tmp)

	for k, v := range tmp {
		outputs[k] = v
	}

	h, err := c.client.HeaderByNumber(ctx, big.NewInt(int64(log.BlockNumber)))
	if err != nil {
		if err != nil {
			return nil, fmt.Errorf("getting block header: %w", err)
		}
	}

	return &apolloTypes.CallResult{
		Type:            apolloTypes.Event,
		Chain:           chain,
		ContractName:    contractName,
		ContractAddress: log.Address,
		BlockNumber:     log.BlockNumber,
		TxHash:          log.TxHash,
		Timestamp:       h.Time,
		Outputs:         outputs,
	}, nil
}