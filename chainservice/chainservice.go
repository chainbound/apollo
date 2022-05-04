package chainservice

import (
	"context"
	"fmt"
	"math/big"
	"sync"
	"time"

	"github.com/XMonetae-DeFi/apollo/generate"
	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethclient"
)

type ChainService struct {
	client *ethclient.Client
}

func NewChainService() *ChainService {
	return &ChainService{}
}

func (c *ChainService) Connect(ctx context.Context, rpcUrl string) (*ChainService, error) {
	client, err := ethclient.DialContext(ctx, rpcUrl)
	if err != nil {
		return nil, fmt.Errorf("Connect: %w", err)
	}

	c.client = client
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

type ResultType int

const (
	Event ResultType = iota
	Method
)

type CallResult struct {
	Err             error
	Chain           generate.Chain
	Type            ResultType
	ContractName    string
	ContractAddress common.Address
	BlockNumber     uint64
	Timestamp       uint64
	Inputs          map[string]string
	Outputs         map[string]any
}

// RunMethodCaller starts a listener on the `blocks` channel, and on every incoming block it will execute all methods concurrently
// on the given blockNumber.
func (c *ChainService) RunMethodCaller(ctx context.Context, schema *generate.SchemaV2, realtime bool, blocks <-chan *big.Int, out chan<- CallResult, maxWorkers int) {
	res := make(chan CallResult)
	var wg sync.WaitGroup

	// TODO: worker pool for blocks (every 32 or something)
	nworkers := 1
	go func() {
		// For every incoming blockNumber, loop over contract methods and start a goroutine for each method.
		// This way, every eth_call will happen concurrently.
		for blockNumber := range blocks {
			wg.Add(1)
			go func(blockNumber *big.Int) {
				defer wg.Done()
				nworkers++

				for _, contract := range schema.Contracts {
					c.CallMethods(ctx, schema.Chain, contract, blockNumber, res)
				}
			}(blockNumber)

			if nworkers%maxWorkers == 0 {
				wg.Wait()
			}
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
func (c ChainService) CallMethods(ctx context.Context, chain generate.Chain, contract *generate.ContractSchemaV2, blockNumber *big.Int, out chan<- CallResult) {
	inputs := make(map[string]string)
	outputs := make(map[string]any)

	// If there are no methods on the contract, return
	if len(contract.Methods()) == 0 {
		return
	}

	for _, method := range contract.Methods() {
		msg, err := generate.BuildCallMsg(contract.Address(), method, contract.Abi)
		if err != nil {
			out <- CallResult{
				Err: err,
			}
			return
		}

		raw, err := c.client.CallContract(ctx, msg, blockNumber)
		if err != nil {
			out <- CallResult{
				Err: err,
			}
			return
		}

		// We only want the correct value here (specified in the schema)
		results, err := contract.Abi.Unpack(method.Name(), raw)
		if err != nil {
			out <- CallResult{
				Err: err,
			}
			return
		}

		for _, o := range method.Outputs() {
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
		out <- CallResult{
			Err: err,
		}
		return
	}

	if blockNumber == nil {
		actualBlockNumber = block.Number.Uint64()
	} else {
		actualBlockNumber = blockNumber.Uint64()
	}

	out <- CallResult{
		Type:            Method,
		BlockNumber:     actualBlockNumber,
		Timestamp:       block.Time,
		Chain:           chain,
		ContractName:    contract.Name(),
		ContractAddress: contract.Address(),
		Inputs:          inputs,
		Outputs:         outputs,
	}
}

func (c ChainService) FilterEvents(ctx context.Context, schema *generate.SchemaV2, fromBlock, toBlock *big.Int, out chan<- CallResult, maxWorkers int) {
	res := make(chan CallResult)
	var wg sync.WaitGroup

	nworkers := 1
	go func() {
		for _, cs := range schema.Contracts {
			for _, event := range cs.Events() {
				// Get first topic in Bytes (to filter events)
				topic, err := generate.GetTopic(event.Name(), cs.Abi)
				if err != nil {
					res <- CallResult{
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

				logs, err := c.client.FilterLogs(ctx, ethereum.FilterQuery{
					FromBlock: fromBlock,
					ToBlock:   toBlock,
					Addresses: []common.Address{cs.Address()},
					Topics: [][]common.Hash{
						{topic},
					},
				})

				if err != nil {
					res <- CallResult{
						Err: fmt.Errorf("getting logs from node: %w", err),
					}
					return
				}

				for _, log := range logs {
					outputs := make(map[string]any)
					for _, event := range event.Outputs() {
						if idx, ok := indexedEvents[event]; ok {
							outputs[event] = common.BytesToAddress(log.Topics[idx][:])
						}
					}

					if len(outputs) < len(event.Outputs()) {
						err := cs.Abi.UnpackIntoMap(outputs, event.Name(), log.Data)
						if err != nil {
							res <- CallResult{
								Err: fmt.Errorf("unpacking log.Data: %w", err),
							}
							return
						}
					}

					go func(log types.Log) {
						defer wg.Done()
						h, err := c.client.HeaderByNumber(ctx, big.NewInt(int64(log.BlockNumber)))
						if err != nil {
							if err != nil {
								res <- CallResult{
									Err: fmt.Errorf("getting block header: %w", err),
								}
								return
							}
						}

						res <- CallResult{
							Type:            Event,
							Chain:           schema.Chain,
							ContractName:    cs.Name(),
							ContractAddress: cs.Address(),
							BlockNumber:     log.BlockNumber,
							Timestamp:       h.Time,
							Outputs:         outputs,
						}
					}(log)
					wg.Add(1)

					if nworkers%maxWorkers == 0 {
						wg.Wait()
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

		// close(out)
	}()
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
