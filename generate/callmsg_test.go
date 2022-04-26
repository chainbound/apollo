package generate

import (
	"fmt"
	"os"
	"testing"

	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
)

func TestBuildInput(t *testing.T) {
	schema, err := ParseV2("../schema.v2.yml")
	if err != nil {
		t.Fatal(err)
	}

	method := schema.Contracts[0].Methods()[0]
	file, err := os.Open("../erc20.abi.json")
	if err != nil {
		t.Fatal(err)
	}

	abi, err := abi.JSON(file)
	if err != nil {
		t.Fatal(err)
	}

	input, err := BuildCallInput(method, abi)
	if err != nil {
		t.Fatal(err)
	}

	fmt.Println("Input:", common.Bytes2Hex(input))
}
