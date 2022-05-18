package dsl

import (
	"fmt"
	"testing"
)

func TestNewSchema(t *testing.T) {
	s, err := NewSchema("../schema.v3.hcl")
	if err != nil {
		t.Fatal(err)
	}

	fmt.Println(s)
}
