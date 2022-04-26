package client

import (
	"context"
	"fmt"
	"math/big"

	"github.com/XMonetae-DeFi/apollo/db"
	"github.com/XMonetae-DeFi/apollo/generate"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/ethclient"
)

type ChainService struct {
	client *ethclient.Client
	// TODO: do we need this?
	db *db.DB
}

func NewChainService(db *db.DB) *ChainService {
	return &ChainService{
		db: db,
	}
}

func (c *ChainService) Connect(ctx context.Context, rpcUrl string) (*ChainService, error) {
	if !c.db.IsConnected() {
		// Connect in place
		if _, err := c.db.Connect(); err != nil {
			return nil, fmt.Errorf("Connect: %w", err)
		}
	}

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
	Err          error
	ContractName string
	MethodName   string
	Inputs       map[string]string
	Outputs      map[string]any
}

// ExecContractCalls calls all the methods for the given ContractSchema concurrently. All results will be sent on the resChan,
// and they can be ordered by contract with CallResult.ContractName. If there is an error, it will be in CallResult.Err,
// and the rest of the properties will be unset.
// TODO: fix concurrency model here. The channel will stay open even when all calls have finished
func (c *ChainService) ExecContractCalls(ctx context.Context, schema *generate.SchemaV2, blocks <-chan *big.Int) chan CallResult {
	out := make(chan CallResult)
	go func() {
		// For every incoming blockNumber, we re-run everything
		for blockNumber := range blocks {
			for _, contract := range schema.Contracts {
				for _, method := range contract.Methods() {
					fmt.Println("Calling method", method.Name())
					msg, err := generate.BuildCallMsg(contract.Address, method, contract.Abi)
					if err != nil {
						out <- CallResult{
							Err: err,
						}
					}

					raw, err := c.client.CallContract(ctx, msg, blockNumber)
					if err != nil {
						out <- CallResult{
							Err: err,
						}
					}

					// We only want the correct value here (specified in the schema)
					results, err := contract.Abi.Unpack(method.Name(), raw)
					if err != nil {
						out <- CallResult{
							Err: err,
						}
					}

					outputs := make(map[string]any)
					for _, o := range method.Outputs() {
						result := matchABIValue(o, contract.Abi.Methods[method.Name()].Outputs, results)
						outputs[o] = result
					}

					out <- CallResult{
						ContractName: contract.Name(),
						MethodName:   method.Name(),
						Inputs:       method.Args(),
						Outputs:      outputs,
					}
				}
			}
		}

		close(out)
	}()

	return out
}

func matchABIValue(outputName string, outputs abi.Arguments, results []any) any {
	if len(results) == 1 {
		return results[0]
	}

	fmt.Println("Matching output", outputName)
	for i, o := range outputs {
		if o.Name == outputName {
			return results[i]
		}
	}

	return nil
}
