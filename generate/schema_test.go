package generate

import (
	"fmt"
	"testing"
)

func TestParseV1(t *testing.T) {
	schema, err := ParseV1("../schema.v1.json")
	if err != nil {
		t.Fatal(err)
	}

	for k, v := range schema.Chains {
		fmt.Println("chain:", k)
		for a, val := range v {
			fmt.Println(a, val)
		}
	}
}

func TestParseV2(t *testing.T) {
	schema, err := ParseV2("../schema.v2.yml")
	if err != nil {
		t.Fatal(err)
	}

	fmt.Printf("%+v\n", schema)
}
