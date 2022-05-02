package generate

import (
	"fmt"
	"strings"

	"github.com/ethereum/go-ethereum/accounts/abi"
)

type Column struct {
	Name  string
	Type  string
	Final bool // utility flag for setting types, finalized when type is known
}

func GenerateDDL(schema ContractSchemaV2) (string, error) {
	columns, err := GenerateColumns(schema)
	if err != nil {
		return "", err
	}

	for _, m := range schema.Methods() {
		columns = AddColumnTypesFromABI(m.Name(), schema.Abi, columns)
	}

	ddl := fmt.Sprintf("CREATE TABLE IF NOT EXISTS %s (\n", schema.Name())
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
	}

	return columns
}

func GenerateColumns(cs ContractSchemaV2) ([]*Column, error) {
	columns := []*Column{
		{
			Name:  "timestamp",
			Type:  ABIToSQLType(Uint256),
			Final: true,
		},
		{
			Name:  "blocknumber",
			Type:  ABIToSQLType(Uint256),
			Final: true,
		},
		{
			Name:  "chain",
			Type:  ABIToSQLType(String),
			Final: true,
		},
		{
			Name:  "contract",
			Type:  ABIToSQLType(Address),
			Final: true,
		},
	}

	// The only dynamic table columns are the arguments and the return values
	for _, call := range cs.Methods() {
		for arg := range call.Args() {
			columns = append(columns, &Column{
				Name: arg,
				// Type will be read from ABI
			})
		}

		for _, output := range call.Outputs() {
			columns = append(columns, &Column{
				Name: output,
				// Type will be read from ABI
			})
		}
	}

	return columns, nil
}
