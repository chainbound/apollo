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

	sjson, err := json.MarshalIndent(s, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	fmt.Printf("%s\n", string(sjson))
}
