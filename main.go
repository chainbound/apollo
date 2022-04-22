package main

import (
	"fmt"
	"log"

	"github.com/XMonetae-DeFi/chainreader/generate"
)

func main() {
	schema, err := generate.ParseV1("schema.v1.json")
	if err != nil {
		log.Fatal(err)
	}

	for k, v := range schema.Chains {
		fmt.Println(k)
		fmt.Printf("%+v\n", v)
	}
}
