package generate

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"gopkg.in/yaml.v2"
)

type Chain string

const (
	Ethereum  Chain = "ethereum"
	Arbitrum  Chain = "arbitrum"
	Optimisim Chain = "optimism"
	BSC       Chain = "bsc"
	Polygon   Chain = "polygon"
	Fantom    Chain = "fantom"
)

type SchemaV1 struct {
	Chains Chains
}

type SchemaV2 struct {
	Chain     Chain               `yaml:"chain"`
	Contracts []*ContractSchemaV2 `yaml:"contracts"`
}

type ContractSchemaV2 struct {
	Address  common.Address `yaml:"address"`
	Name_    string         `yaml:"name"`
	AbiPath  string         `yaml:"abi"`
	Methods_ []MethodV2     `yaml:"methods"`
	Abi      abi.ABI        `yaml:"-"`
}

func (cs ContractSchemaV2) Name() string {
	return cs.Name_
}

func (cs ContractSchemaV2) Methods() []MethodV2 {
	return cs.Methods_
}

type MethodV2 struct {
	Name_    string            `yaml:"name"`
	Args_    map[string]string `yaml:"args,omitempty"` // Args can be empty
	Outputs_ []string          `yaml:"outputs"`
}

func (m MethodV2) Name() string {
	return m.Name_
}

func (m MethodV2) Args() map[string]string {
	return m.Args_
}

func (m MethodV2) Outputs() []string {
	return m.Outputs_
}

// func (s SchemaV1) ContractMethods(contract common.Address) []string {
// 	return s.Chains.ContractMethods(contract)
// }

func (s SchemaV1) ContractSchemas() []ContractSchemaV1 {
	return s.Chains.ContractSchemas()
}

// Map chainName => contractSchemas
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

func ParseV2(path string) (*SchemaV2, error) {
	var schema SchemaV2

	file, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("ParseV2: reading file: %w", err)
	}

	if err = yaml.Unmarshal(file, &schema); err != nil {
		return nil, fmt.Errorf("ParseV2: parsing yaml: %w", err)
	}

	for _, contract := range schema.Contracts {
		// TODO: path should be steady (use set config path)
		f, err := os.Open(contract.AbiPath)
		if err != nil {
			return nil, fmt.Errorf("ParseV2: reading ABI file: %w", err)
		}

		abi, err := abi.JSON(f)
		if err != nil {
			return nil, fmt.Errorf("ParseV2: parsing ABI")
		}

		contract.Abi = abi
	}

	return &schema, nil
}
