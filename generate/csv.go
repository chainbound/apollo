package generate

import "github.com/zclconf/go-cty/cty"

func GenerateCsvHeader(cols map[string]cty.Value) []string {
	columns := make([]string, len(cols))

	// The only dynamic table columns are the arguments and the return values
	i := 0

	for k := range cols {
		columns[i] = k
		i++
	}

	return columns
}
