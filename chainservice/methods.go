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
	var wg sync.WaitGroup
	c.logger.Debug().Msg("contract methods")

	// For every incoming blockNumber, loop over contract methods and starts a goroutine for each method.
	// This way, every eth_call will happen concurrently.
	for blockNumber := range blocks {
		wg.Add(1)
		c.logger.Trace().Str("block", blockNumber.String()).Msg("new block")
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
							out <- apolloTypes.CallResult{
								Err: err,
							}
							return
						}

						results = append(results, result)
					}(contract, method)
				}

				wg2.Wait()

				if len(results) > 0 {
					callResult := *aggregateCallResults(results...)
					// If we're in realtime mode, add the current timestamp.
					// Most blockchains have very rough Block.Timestamp updates,
					// which are not realtime at all.
					if realtime {
						callResult.Timestamp = uint64(time.Now().UnixMilli() / 1000)
					}

					callResult.QueryName = query.Name
					out <- callResult
				}
			}
		}(blockNumber)
	}

	wg.Wait()
	close(out)
}

// CallMethod executes all the methods on the contract, and aggregates their results into a CallResult
func (c ChainService) CallMethod(chain apolloTypes.Chain, address common.Address, abi abi.ABI, method *dsl.Method, blockNumber *big.Int) (*apolloTypes.CallResult, error) {
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
	c.logger.Trace().Str("to", msg.To.String()).Str("method", method.Name()).Str("block_number", blockNumber.String()).Msg("called method")

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
	c.logger.Trace().Msg("got block header")

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
