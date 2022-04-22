package db

import (
	"context"
	"fmt"
	"log"
	"os"
	"testing"
	"time"

	"github.com/XMonetae-DeFi/chainreader/generate"
	"github.com/ethereum/go-ethereum/accounts/abi"
)

func newDB() *DB {
	db, err := NewDB(DbSettings{
		Host:     "172.17.0.2",
		User:     "chainreader",
		Password: "chainreader",
		Name:     "postgres",
	}).Connect()

	if err != nil {
		log.Fatal(err)
	}

	return db
}

func TestConnect(t *testing.T) {
	db := newDB()
	if !db.isConnected() {
		t.Fatal("not connected")
	}
}

func TestCreateTable(t *testing.T) {
	db := newDB()
	schema, err := generate.ParseV1("../schema.v1.json")
	if err != nil {
		t.Fatal(err)
	}

	file, err := os.Open("../erc20.abi.json")
	if err != nil {
		t.Fatal(err)
	}

	abi, err := abi.JSON(file)
	if err != nil {
		t.Fatal(err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), time.Second*5)
	defer cancel()

	for _, s := range schema.ContractSchemas() {
		ddl, err := generate.GenerateDDL(abi, s)
		if err != nil {
			t.Fatal(err)
		}

		fmt.Println(ddl)
		err = db.CreateTable(ctx, ddl)
		if err != nil {
			t.Fatal(err)
		}
	}

}
