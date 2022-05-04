package generate

import (
	"errors"
	"fmt"
	"os"

	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"gopkg.in/yaml.v2"
)

var (
	ErrMethodsAndEvents = errors.New("can't parse methods AND events for one contract")
	ErrTooManyEvents    = errors.New("can't parse more than 1 event per contract")
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

func (s SchemaV2) Validate() error {
	for _, cs := range s.Contracts {
		if err := cs.Validate(); err != nil {
			return err
		}
	}

	return nil
}

type ContractSchemaV2 struct {
	Address_ common.Address `yaml:"address"`
	Name_    string         `yaml:"name"`
	AbiPath  string         `yaml:"abi"`
	Methods_ []MethodV2     `yaml:"methods"`
	Events_  []EventV2      `yaml:"events"`
	Abi      abi.ABI        `yaml:"-"`
}

func (cs ContractSchemaV2) Name() string {
	return cs.Name_
}

func (cs ContractSchemaV2) Address() common.Address {
	return cs.Address_
}

func (cs ContractSchemaV2) Methods() []MethodV2 {
	return cs.Methods_
}

func (cs ContractSchemaV2) Events() []EventV2 {
	return cs.Events_
}

func (cs ContractSchemaV2) Validate() error {
	if len(cs.Methods()) > 0 && len(cs.Events()) > 0 {
		return fmt.Errorf("%s: %w", cs.Name(), ErrMethodsAndEvents)
	}

	if len(cs.Events()) > 1 {
		return fmt.Errorf("%s: %w", cs.Name(), ErrTooManyEvents)
	}

	return nil
}

type MethodV2 struct {
	Name_    string            `yaml:"name"`
	Inputs_  map[string]string `yaml:"inputs"` // Args can be empty
	Outputs_ []string          `yaml:"outputs"`
}

func (m MethodV2) Name() string {
	return m.Name_
}

func (m MethodV2) Inputs() map[string]string {
	return m.Inputs_
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
