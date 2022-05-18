package generate

import (
	"testing"

	"github.com/XMonetae-DeFi/apollo/dsl"
)

func TestGenerateCsvHeaders(t *testing.T) {
	schema, err := dsl.NewSchema("../test")
	if err != nil {
		t.Fatal(err)
	}

	_ = schema

	// for _, s := range schema.Contracts {
	// 	c := GenerateCsvHeader(*s)

	// 	fmt.Println(c)
	// }

}
