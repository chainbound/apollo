package generate

import (
	"fmt"
	"strings"

	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/zclconf/go-cty/cty"
)

type Column struct {
	Name  string
	Type  string
	Final bool // utility flag for setting types, finalized when type is known
}

func GenerateCreateDDL(tableName string, cols map[string]cty.Value) (string, error) {
	columns, err := GenerateColumns(cols)
	if err != nil {
		return "", err
	}

	ddl := fmt.Sprintf("DROP TABLE IF EXISTS %s;\n", tableName)

	ddl += fmt.Sprintf("CREATE TABLE %s (\n\tid SERIAL PRIMARY KEY,\n", tableName)
	for _, col := range columns {
		ddl += fmt.Sprintf("\t%s %s,\n", col.Name, col.Type)
	}

	ddl = strings.TrimSuffix(ddl, ",\n")
	ddl += "\n);"

	return ddl, nil
}

func GenerateInsertSQL(tableName string, toInsert map[string]string) string {
	columns := "("
	values := "("

	for col, val := range toInsert {
		columns += col + ","
		values += fmt.Sprintf("'%s',", val)
	}

	columns = strings.TrimSuffix(columns, ",") + ")"
	values = strings.TrimSuffix(values, ",") + ")"

	return fmt.Sprintf("INSERT INTO %s %s VALUES %s;", tableName, columns, values)
}

// AddColumnTypesFromABI cross-references the name ("event" or "method") with the ABI,
// to fill in which types the columns need to be. These types get converted to SQL types
// eventually.
func AddColumnTypesFromABI(name string, abi abi.ABI, columns []*Column) []*Column {
	for _, col := range columns {
		// METHODS
		method := abi.Methods[name]
		for _, i := range method.Inputs {
			if i.Name == col.Name {
				col.Type = ABIToSQLType(ABIType(i.Type.String()))
				col.Final = true
			}
		}

		for _, o := range method.Outputs {
			if o.Name == col.Name {
				col.Type = ABIToSQLType(ABIType(o.Type.String()))
				col.Final = true
			}
		}

		// Some ABI outputs have no name (when it's the only return value)
		// so this is what we check here. Any col that's not finalized is
		// the one that we need to link to the ABI return value
		if len(method.Outputs) == 1 && !col.Final {
			col.Type = ABIToSQLType(ABIType(method.Outputs[0].Type.String()))
			col.Final = true
		}

		// EVENTS
		event := abi.Events[name]
		for _, o := range event.Inputs {
			if o.Name == col.Name {
				col.Type = ABIToSQLType(ABIType(o.Type.String()))
				col.Final = true
			}
		}
	}

	return columns
}

func GenerateColumns(cols map[string]cty.Value) ([]Column, error) {
	columns := make([]Column, len(cols))

	i := 0
	for k, v := range cols {
		columns[i] = Column{
			Name:  k,
			Type:  CtyToSQLType(v.Type()),
			Final: true,
		}
		i++
	}

	return columns, nil
}
