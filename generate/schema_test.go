package generate

import (
	"fmt"
	"testing"
)

func TestParseV2(t *testing.T) {
	schema, err := ParseV2("../test")
	if err != nil {
		t.Fatal(err)
	}

	fmt.Printf("%+v\n", schema)
}
