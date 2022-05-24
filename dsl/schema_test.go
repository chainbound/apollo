package dsl

import (
	"fmt"
	"testing"
)

func TestNewSchema(t *testing.T) {
	s, err := NewSchema("../test")
	if err != nil {
		t.Fatal(err)
	}

	for k, v := range s.Variables {
		fmt.Println(k, ":", v)
	}
	// sjson, err := json.MarshalIndent(s.Variables, "", "  ")
	// if err != nil {
	// 	t.Fatal(err)
	// }
	// fmt.Printf("%s\n", string(sjson))
}
