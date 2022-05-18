package generate

import (
	"testing"
)

func TestGenerateCsvHeaders(t *testing.T) {
	schema, err := ParseV2("../test")
	if err != nil {
		t.Fatal(err)
	}

	_ = schema

	// for _, s := range schema.Contracts {
	// 	c := GenerateCsvHeader(*s)

	// 	fmt.Println(c)
	// }

}
