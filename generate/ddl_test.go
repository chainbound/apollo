package generate

import (
	"fmt"
	"os"
	"testing"

	"github.com/ethereum/go-ethereum/accounts/abi"
)

func TestGenerateColumns(t *testing.T) {
	schema, err := ParseV2("../schema.v2.yml")
	if err != nil {
		t.Fatal(err)
	}

	var cols []*Column
	for _, s := range schema.Contracts {
		c, err := GenerateColumns(s)
		if err != nil {
			t.Fatal(err)
		}

		cols = append(cols, c...)
	}

	fmt.Println(cols)
}

func TestGenerateDDL(t *testing.T) {
	schema, err := ParseV2("../schema.v2.yml")
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

	for _, s := range schema.Contracts {
		ddl, err := GenerateDDL(abi, s)
		if err != nil {
			t.Fatal(err)
		}

		fmt.Println(ddl)
	}
}
