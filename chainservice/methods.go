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

	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
)

// RunMethodCaller starts a listener on the `blocks` channel, and on every incoming block it will execute all methods concurrently
// on the given blockNumber.
func (c *ChainService) RunMethodCaller(query *dsl.Query, realtime bool, blocks <-chan *big.Int, out chan<- apolloTypes.CallResult) {
	res := make(chan apolloTypes.CallResult)
	var wg sync.WaitGroup
	var wg1 sync.WaitGroup

	wg1.Add(1)
	go func() {
		defer wg1.Done()
		// For every incoming blockNumber, loop over contract methods and start a goroutine for each method.
		// This way, every eth_call will happen concurrently.
		for blockNumber := range blocks {
			c.logger.Trace().Str("block", blockNumber.String()).Msg("new block")
			wg.Add(1)
			go func(blockNumber *big.Int) {
				defer wg.Done()
				for _, contract := range query.Contracts {
					var wg2 sync.WaitGroup
					var results []*apolloTypes.CallResult
					for _, method := range contract.Methods {
						wg2.Add(1)
						go func(contract *dsl.Contract, method *dsl.Method) {
							defer wg2.Done()
							result, err := c.CallMethod(query.Chain, contract.Address(), contract.Abi, method, blockNumber)
							if err != nil {
								res <- apolloTypes.CallResult{
									Err: err,
								}
								return
							}

							results = append(results, result)
						}(contract, method)
					}

					wg2.Wait()

					if len(results) > 0 {
						res <- *aggregateCallResults(results...)
					}
				}
			}(blockNumber)
		}

		// Wait for all the goroutines to finish
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

			r.QueryName = query.Name

			out <- r
		}
	}()

	wg1.Wait()
}

// CallMethod executes all the methods on the contract, and aggregates their results into a CallResult
func (c ChainService) CallMethod(chain apolloTypes.Chain, address common.Address, abi abi.ABI, method *dsl.Method, blockNumber *big.Int) (*apolloTypes.CallResult, error) {
	start := time.Now()
	inputs := make(map[string]any)
	outputs := make(map[string]any)
	rlClient := c.clients[chain]

	c.rateLimiter.Take()

	// If there are no methods on the contract, return
	ctx, cancel := context.WithTimeout(context.Background(), c.defaultTimeout)
	defer cancel()

	msg, err := generate.BuildCallMsg(address, method, abi)
	if err != nil {
		return nil, fmt.Errorf("building call message: %w", err)
	}
	c.logger.Trace().Str("to", msg.To.String()).Str("input", common.Bytes2Hex(msg.Data)).Str("method", method.Name()).Msg("built call message")

	raw, err := rlClient.CallContract(ctx, msg, blockNumber)
	if err != nil {
		return nil, fmt.Errorf("calling contract method: %w", err)
	}
	c.logger.Trace().Str("to", msg.To.String()).Str("method", method.Name()).Str("block_number", blockNumber.String()).Str("time", time.Since(start).String()).Msg("called method")

	// We only want the correct value here (specified in the schema)
	results, err := abi.Unpack(method.Name(), raw)
	if err != nil {
		return nil, fmt.Errorf("unpacking abi for %s: %w", method.Name(), err)
	}

	for _, o := range method.Outputs {
		result := matchABIValue(o, abi.Methods[method.Name()].Outputs, results)
		outputs[o] = result
	}

	for k, v := range method.Inputs() {
		inputs[k] = v
	}

	actualBlockNumber := uint64(0)
	block, err := rlClient.HeaderByNumber(ctx, blockNumber)
	if err != nil {
		return nil, fmt.Errorf("getting block number %w", err)
	}

	if blockNumber == nil {
		actualBlockNumber = block.Number.Uint64()
	} else {
		actualBlockNumber = blockNumber.Uint64()
	}

	return &apolloTypes.CallResult{
		Type:            apolloTypes.Method,
		BlockNumber:     actualBlockNumber,
		BlockHash:       block.Hash(),
		Timestamp:       block.Time,
		Chain:           chain,
		Identifier:      address.String(),
		ContractAddress: address,
		Inputs:          inputs,
		Outputs:         outputs,
	}, nil
}
