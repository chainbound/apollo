package generate

import (
	"testing"
)

func TestParseV1(t *testing.T) {
	_, err := ParseV1("../schema.v1.json")
	if err != nil {
		t.Fatal(err)
	}

	// for k, v := range schema.Chains {
	// 	fmt.Println(k)
	// 	fmt.Printf("%+v\n", v)
	// }
}
