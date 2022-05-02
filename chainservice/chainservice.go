package chainservice

import (
	"context"
	"fmt"
	"math/big"
	"sync"
	"time"

	"github.com/XMonetae-DeFi/apollo/generate"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
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

type CallResult struct {
	Err             error
	Chain           generate.Chain
	ContractName    string
	ContractAddress common.Address
	BlockNumber     uint64
	Timestamp       uint64
	MethodName      string
	Inputs          map[string]string
	Outputs         map[string]any
}

// RunMethodCaller starts a listener on the `blocks` channel, and on every incoming block it will execute all methods concurrently
// on the given blockNumber.
func (c *ChainService) RunMethodCaller(ctx context.Context, schema *generate.SchemaV2, blocks <-chan *big.Int) <-chan CallResult {
	out := make(chan CallResult)
	var wg sync.WaitGroup

	go func() {
		// For every incoming blockNumber, loop over contract methods and start a goroutine for each method.
		// This way, every eth_call will happen concurrently.
		for blockNumber := range blocks {
			for _, contract := range schema.Contracts {
				for _, method := range contract.Methods() {
					go func(chain generate.Chain, contract *generate.ContractSchemaV2, method generate.MethodV2, blockNumber *big.Int) {
						c.CallMethod(ctx, chain, contract, method, blockNumber, out)
						wg.Done()
					}(schema.Chain, contract, method, blockNumber)
					wg.Add(1)
				}
			}

		}

		wg.Wait()
		// When all of our methods have executed AND the blocks channel was closed on the other side,
		// close the out channel
		close(out)
	}()

	return out
}

// CallMethod executes a contract method
func (c ChainService) CallMethod(ctx context.Context, chain generate.Chain, contract *generate.ContractSchemaV2, method generate.MethodV2, blockNumber *big.Int, out chan<- CallResult) {
	msg, err := generate.BuildCallMsg(contract.Address, method, contract.Abi)
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

	actualBlock, err := c.client.BlockNumber(ctx)
	if err != nil {
		out <- CallResult{
			Err: err,
		}
		return
	}

	timestamp := uint64(time.Now().UnixMilli() / 1000)
	// We only want the correct value here (specified in the schema)
	results, err := contract.Abi.Unpack(method.Name(), raw)
	if err != nil {
		out <- CallResult{
			Err: err,
		}
		return
	}

	outputs := make(map[string]any)
	for _, o := range method.Outputs() {
		result := matchABIValue(o, contract.Abi.Methods[method.Name()].Outputs, results)
		outputs[o] = result
	}

	out <- CallResult{
		BlockNumber:     actualBlock,
		Timestamp:       timestamp,
		Chain:           chain,
		ContractName:    contract.Name(),
		ContractAddress: contract.Address,
		MethodName:      method.Name(),
		Inputs:          method.Args(),
		Outputs:         outputs,
	}
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
