package main

import (
	"context"
	"fmt"
	"log"
	"math/big"
	"time"

	"github.com/XMonetae-DeFi/apollo/chainservice"
	"github.com/XMonetae-DeFi/apollo/generate"
)

const (
	rpcUrl = "wss://arb-mainnet.g.alchemy.com/v2/5_JWUuiS1cewWFpLzRxdjgZM0yLA4Uqp"
)

func main() {
	schema, err := generate.ParseV2("schema.v2.yml")
	if err != nil {
		log.Fatal(err)
	}

	// db, err := db.NewDB(db.DbSettings{
	// 	Host:     "172.17.0.2",
	// 	User:     "chainreader",
	// 	Password: "chainreader",
	// 	Name:     "postgres",
	// }).Connect()

	if err != nil {
		log.Fatal(err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), time.Second*5)
	defer cancel()

	service, err := chainservice.NewChainService().Connect(ctx, rpcUrl)
	if err != nil {
		log.Fatal(err)
	}

	// for _, s := range schema.Contracts {
	// 	ddl, err := generate.GenerateDDL(*s)
	// 	if err != nil {
	// 		log.Fatal(err)
	// 	}

	// 	err = db.CreateTable(ctx, ddl)
	// 	if err != nil {
	// 		log.Fatal(err)
	// 	}
	// }

	blocks := make(chan *big.Int)

	res := service.RunMethodCaller(context.Background(), schema, blocks)

	// Start main program loop
	go func() {
		for {
			blocks <- nil
			time.Sleep(5 * time.Second)
		}
	}()

	for res := range res {
		fmt.Printf("Result from %s [%s]\n", res.MethodName, res.ContractName)
		fmt.Printf("%+v\n", res.Outputs)
	}

}
