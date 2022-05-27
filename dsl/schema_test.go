package dsl

import (
	"encoding/json"
	"fmt"
	"testing"
)

func TestNewSchema(t *testing.T) {
	s, err := NewSchema("../test")
	if err != nil {
		t.Fatal(err)
	}

	for k, v := range s.Variables {
		fmt.Println(k, ":", v.GoString())
	}

	// s.Queries = nil

	sjson, err := json.MarshalIndent(s.Queries, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	fmt.Printf("%s\n", string(sjson))
}
