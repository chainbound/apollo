package generate

import (
	"fmt"
	"testing"
)

func TestGenerateColumns(t *testing.T) {
	schema, err := ParseV2("../test")
	if err != nil {
		t.Fatal(err)
	}

	var cols []*Column
	for _, s := range schema.Contracts {
		c, err := GenerateColumns(*s)
		if err != nil {
			t.Fatal(err)
		}

		cols = append(cols, c...)
	}

	for _, col := range cols {
		fmt.Println(col)
	}
}

func TestGenerateDDL(t *testing.T) {
	schema, err := ParseV2("../test")
	if err != nil {
		t.Fatal(err)
	}

	for _, s := range schema.Contracts {
		ddl, err := GenerateCreateDDL(*s)
		if err != nil {
			t.Fatal(err)
		}

		fmt.Println(ddl)
	}
}

func TestGenerateInsertSQL(t *testing.T) {
	m := map[string]string{
		"timestamp":   "1650246095",
		"blocknumber": "10000279",
		"chain":       "arbitrum",
		"contract":    "0x905dfCD5649217c42684f23958568e533C711Aa3",
		"amount0In":   "0",
		"amount1In":   "2000000",
		"amount0Out":  "666273506300276",
		"amount1Out":  "0",
	}

	ddl := GenerateInsertSQL("eth_usdc_swaps", m)

	fmt.Println(ddl)
}
