package generate

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/ethereum/go-ethereum/common"
)

// TODO: interface
type SchemaV1 struct {
	Chains Chains
}

// func (s SchemaV1) ContractMethods(contract common.Address) []string {
// 	return s.Chains.ContractMethods(contract)
// }

func (s SchemaV1) ContractSchemas() []ContractSchemaV1 {
	return s.Chains.ContractSchemas()
}

type Chains map[string]ContractSchemasV1

func (c Chains) ContractSchemas() []ContractSchemaV1 {
	var s []ContractSchemaV1
	for _, v := range c {
		for _, schema := range v {
			s = append(s, schema)
		}
	}

	return s
}

func (s *SchemaV1) UnmarshalJSON(data []byte) error {
	if err := json.Unmarshal(data, &s.Chains); err != nil {
		return err
	}

	return nil
}

type ContractSchemasV1 map[common.Address]ContractSchemaV1

type ContractSchemaV1 struct {
	AbiPath string `json:"abi"`
	Name    string `json:"name"`
	// Apart from Abi, we have no idea what these values will look like
	// so we define a map of methods to interfaces
	Methods map[string]MethodV1 `json:"methods"`
}

func (c ContractSchemaV1) ContractMethods() (methods []string) {
	for k := range c.Methods {
		methods = append(methods, k)
	}

	return
}

type MethodV1 struct {
	// Arguments can be anything
	Arguments map[string]interface{}
	// Outputs is the only known field in this struct
	Outputs []string `json:"outputs"`
}

func (m *MethodV1) UnmarshalJSON(data []byte) error {
	// Unmarshal everything into arguments
	if err := json.Unmarshal(data, &m.Arguments); err != nil {
		return err
	}

	var outputs []string
	// Find our known value (outputs)
	for _, arg := range m.Arguments["outputs"].([]interface{}) {
		outputs = append(outputs, arg.(string))
	}

	m.Outputs = outputs
	delete(m.Arguments, "outputs")

	return nil
}

func ParseV1(path string) (*SchemaV1, error) {
	var schema SchemaV1

	file, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("ParseV1: reading schema: %w", err)
	}

	if err = json.Unmarshal(file, &schema); err != nil {
		return nil, fmt.Errorf("ParseV1: parsing schema: %w", err)
	}

	return &schema, nil
}
