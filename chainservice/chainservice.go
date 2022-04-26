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

// ExecContractCalls calls all the methods for the given ContractSchema concurrently.
func (c *ChainService) ExecContractCalls(ctx context.Context, schema *generate.ContractSchemaV2, resChan chan<- CallResult, blockNumber *big.Int) {
	for _, method := range schema.Methods() {
		go func(method generate.MethodV2) {
			fmt.Println("Calling method", method.Name())
			msg, err := generate.BuildCallMsg(schema.Address, method, schema.Abi)
			if err != nil {
				resChan <- CallResult{
					Err: err,
				}
			}

			raw, err := c.client.CallContract(ctx, msg, blockNumber)
			if err != nil {
				resChan <- CallResult{
					Err: err,
				}
			}

			// We only want the correct value here (specified in the schema)
			results, err := schema.Abi.Unpack(method.Name(), raw)
			if err != nil {
				resChan <- CallResult{
					Err: err,
				}
			}

			fmt.Println("Results: ", results)

			outputs := make(map[string]any)
			for _, o := range method.Outputs() {
				result := matchABIValue(o, schema.Abi.Methods[method.Name()].Outputs, results)
				outputs[o] = result
			}

			resChan <- CallResult{
				ContractName: schema.Name(),
				MethodName:   method.Name(),
				Inputs:       method.Args(),
				Outputs:      outputs,
			}
		}(method)
	}
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
