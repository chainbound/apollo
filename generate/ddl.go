package generate

import (
	"fmt"
	"strings"

	"github.com/ethereum/go-ethereum/accounts/abi"
)

type Column struct {
	Name  string
	Type  string
	Final bool // utility flag for setting types
}

type Type string

// Type Enum
const (
	Uint256 Type = "uint256"
	String  Type = "string"
	Address Type = "address"
)

var (
	types = map[Type]string{
		Uint256: "BIGINT",
		String:  "VARCHAR(55)", // strings are used for things like chain names
		Address: "VARCHAR(40)", // addresses should be stored as strings without 0x prefix
	}
)

func ConvertType(solType Type) string {
	return types[solType]
}

func GenerateDDL(abi abi.ABI, schema ContractSchemaV1) (string, error) {
	columns, err := GenerateColumns(schema)
	if err != nil {
		return "", err
	}

	for k := range schema.Methods {
		columns = AddColumnTypesFromABI(k, abi, columns)
	}

	ddl := fmt.Sprintf("CREATE TABLE IF NOT EXISTS %s (\n", schema.Name)
	for _, col := range columns {
		if col.Name == "timestamp" {
			ddl += fmt.Sprintf("\t%s %s PRIMARY KEY,\n", col.Name, col.Type)
		} else {
			ddl += fmt.Sprintf("\t%s %s,\n", col.Name, col.Type)
		}
	}

	ddl = strings.TrimSuffix(ddl, ",\n")
	ddl += "\n);"

	return ddl, nil
}

func AddColumnTypesFromABI(methodName string, abi abi.ABI, columns []*Column) []*Column {
	for _, col := range columns {
		method := abi.Methods[methodName]
		for _, i := range method.Inputs {
			if i.Name == col.Name {
				col.Type = ConvertType(Type(i.Type.String()))
			}
			col.Final = true
		}

		for _, o := range method.Outputs {
			if o.Name == col.Name {
				col.Type = ConvertType(Type(o.Type.String()))
				col.Final = true
			}
		}

		if len(method.Outputs) == 1 && !col.Final {
			col.Type = ConvertType(Type(method.Outputs[0].Type.String()))
			col.Final = true
		}
	}

	return columns
}

func GenerateColumns(schema ContractSchemaV1) ([]*Column, error) {
	columns := []*Column{
		{
			Name:  "timestamp",
			Type:  ConvertType(Uint256),
			Final: true,
		},
		{
			Name:  "chain",
			Type:  ConvertType(String),
			Final: true,
		},
		{
			Name:  "contract",
			Type:  ConvertType(Address),
			Final: true,
		},
	}

	// The only dynamic table columns are the arguments and the return values
	for _, call := range schema.Methods {
		for arg := range call.Arguments {
			columns = append(columns, &Column{
				Name: arg,
				// Type will be read from ABI
			})
		}

		for _, output := range call.Outputs {
			columns = append(columns, &Column{
				Name: output,
				// Type will be read from ABI
			})
		}
	}

	return columns, nil
}
