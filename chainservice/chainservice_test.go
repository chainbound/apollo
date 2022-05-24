package chainservice

import (
	"context"
	"fmt"
	"log"
	"math/big"
	"testing"
	"time"

	"github.com/chainbound/apollo/dsl"
	"github.com/chainbound/apollo/types"
)

const (
	rpcUrl = "wss://arb-mainnet.g.alchemy.com/v2/5_JWUuiS1cewWFpLzRxdjgZM0yLA4Uqp"
)

func newChainService() *ChainService {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*5)
	defer cancel()

	c, err := NewChainService(time.Second*20, 5).Connect(ctx, rpcUrl)
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
	schema, err := dsl.NewSchema("../test")
	if err != nil {
		t.Fatal(err)
	}

	service := newChainService()

	blocks := make(chan *big.Int)

	res := make(chan types.CallResult)
	service.RunMethodCaller(schema.Queries[0], true, blocks, res)

	// Latest block, then close
	blocks <- nil
	close(blocks)

	for res := range res {
		fmt.Println(res)
	}
}

// func TestFilterEvents(t *testing.T) {
// 	schema, err := generate.ParseV2("./test")
// 	if err != nil {
// 		t.Fatal(err)
// 	}

// 	service := newChainService()
// 	res := make(chan CallResult)
// 	maxWorkers := 32

// 	service.FilterEvents(schema, big.NewInt(10000000), big.NewInt(10000500), res, maxWorkers)

// 	count := 0
// 	for r := range res {
// 		if r.Err != nil {
// 			fmt.Println(r.Err)
// 			continue
// 		}

// 		fmt.Println(r)
// 		count++
// 		fmt.Println(count)
// 	}
// }

func TestListenForEvents(t *testing.T) {
	schema, err := dsl.NewSchema("../test")
	if err != nil {
		t.Fatal(err)
	}

	service := newChainService()
	res := make(chan types.CallResult)

	service.ListenForEvents(schema.Queries[0], res)

	for r := range res {
		if r.Err != nil {
			fmt.Println(r.Err)
			continue
		}

		fmt.Println(r)
	}

}
