package generate

import (
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

type SchemaV2 struct {
	Chain     Chain               `yaml:"chain"`
	Contracts []*ContractSchemaV2 `yaml:"contracts"`
}

type ContractSchemaV2 struct {
	Address  common.Address `yaml:"address"`
	Name_    string         `yaml:"name"`
	AbiPath  string         `yaml:"abi"`
	Methods_ []MethodV2     `yaml:"methods"`
	Events_  []EventV2      `yaml:"events"`
	Abi      abi.ABI        `yaml:"-"`
}

func (cs ContractSchemaV2) Name() string {
	return cs.Name_
}

func (cs ContractSchemaV2) Methods() []MethodV2 {
	return cs.Methods_
}

func (cs ContractSchemaV2) Events() []EventV2 {
	return cs.Events_
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

type EventV2 struct {
	Name_    string   `yaml:"name"`
	Outputs_ []string `yaml:"outputs"`
}

func (e EventV2) Name() string {
	return e.Name_
}

func (e EventV2) Outputs() []string {
	return e.Outputs_
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
