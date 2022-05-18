package dsl

import (
	"fmt"
	"testing"
)

func TestNewSchema(t *testing.T) {
	s, err := NewSchema("..")
	if err != nil {
		t.Fatal(err)
	}

	fmt.Println(s)
}
