package chainservice

import (
	"context"
	"fmt"
	"log"
	"math/big"
	"testing"
	"time"

	"github.com/XMonetae-DeFi/apollo/generate"
)

const (
	rpcUrl = "wss://arb-mainnet.g.alchemy.com/v2/5_JWUuiS1cewWFpLzRxdjgZM0yLA4Uqp"
)

func newChainService() *ChainService {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*5)
	defer cancel()

	c, err := NewChainService().Connect(ctx, rpcUrl)
	if err != nil {
		log.Fatal(err)
	}

	return c
}

func TestConnect(t *testing.T) {
	c := newChainService()

	if !c.IsConnected() {
		t.Fatal("not connected")
	}
}

func TestExecCallContracts(t *testing.T) {
	schema, err := generate.ParseV2("../schema.v2.yml")
	if err != nil {
		t.Fatal(err)
	}

	service := newChainService()

	blocks := make(chan *big.Int)

	res := make(chan CallResult)
	service.RunMethodCaller(context.Background(), schema, true, blocks, res, 10)

	// Latest block, then close
	blocks <- nil
	close(blocks)

	for res := range res {
		fmt.Printf("Result from [%s]\n", res.ContractName)
		for k, v := range res.Inputs {
			fmt.Println(k, ":", v)
		}

		for k, v := range res.Outputs {
			fmt.Println(k, ":", v)
		}
	}
}
