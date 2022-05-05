package generate

import (
	"fmt"
	"testing"
)

func TestGenerateCsvHeaders(t *testing.T) {
	schema, err := ParseV2("../test")
	if err != nil {
		t.Fatal(err)
	}

	for _, s := range schema.Contracts {
		c := GenerateCsvHeader(*s)

		fmt.Println(c)
	}

}
