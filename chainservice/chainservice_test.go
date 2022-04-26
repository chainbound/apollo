package client

import (
	"context"
	"fmt"
	"log"
	"testing"
	"time"

	"github.com/XMonetae-DeFi/apollo/db"
	"github.com/XMonetae-DeFi/apollo/generate"
)

const (
	rpcUrl = "wss://arb-mainnet.g.alchemy.com/v2/5_JWUuiS1cewWFpLzRxdjgZM0yLA4Uqp"
)

func newClient() *ChainService {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*5)
	defer cancel()
	db, err := db.NewDB(db.DbSettings{
		Host:     "172.17.0.2",
		User:     "chainreader",
		Password: "chainreader",
		Name:     "postgres",
	}).Connect()

	if err != nil {
		log.Fatal(err)
	}

	c, err := NewChainService(db).Connect(ctx, rpcUrl)
	if err != nil {
		log.Fatal(err)
	}

	return c
}

func TestConnect(t *testing.T) {
	c := newClient()

	if !c.IsConnected() {
		t.Fatal("not connected")
	}
}

func TestExecCallContracts(t *testing.T) {
	schema, err := generate.ParseV2("../schema.v2.yml")
	if err != nil {
		t.Fatal(err)
	}

	service := newClient()

	chanMap := make(chan CallResult)

	for _, s := range schema.Contracts {
		service.ExecContractCalls(context.Background(), s, chanMap, nil)
	}

	for res := range chanMap {
		fmt.Println(res)
	}
}
