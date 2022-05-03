package generate

import (
	"fmt"
	"testing"
)

func TestGenerateColumns(t *testing.T) {
	schema, err := ParseV2("../schema.v2.yml")
	if err != nil {
		t.Fatal(err)
	}

	var cols []*Column
	for _, s := range schema.Contracts {
		c, err := GenerateColumns(*s)
		if err != nil {
			t.Fatal(err)
		}

		cols = append(cols, c...)
	}

	for _, col := range cols {
		fmt.Println(col)
	}
}

func TestGenerateDDL(t *testing.T) {
	schema, err := ParseV2("../schema.v2.yml")
	if err != nil {
		t.Fatal(err)
	}

	for _, s := range schema.Contracts {
		ddl, err := GenerateDDL(*s)
		if err != nil {
			t.Fatal(err)
		}

		fmt.Println(ddl)
	}
}
