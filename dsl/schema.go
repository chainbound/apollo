package dsl

import (
	"fmt"
	"io/ioutil"
	"os"
	"path"

	acommon "github.com/XMonetae-DeFi/apollo/common"

	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"

	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/gohcl"
	"github.com/hashicorp/hcl/v2/hclsyntax"
	"github.com/zclconf/go-cty/cty"
	"github.com/zclconf/go-cty/cty/function"
	"github.com/zclconf/go-cty/cty/function/stdlib"
	"github.com/zclconf/go-cty/cty/gocty"
)

type DynamicSchema struct {
	Chain     acommon.Chain `hcl:"chain"`
	Contracts []*Contract   `hcl:"contract,block"`

	EvalContext *hcl.EvalContext
}

type Contract struct {
	Name     string `hcl:"name,label"`
	Address_ string `hcl:"address,label"`
	AbiPath  string `hcl:"abi"`

	Methods []*Method `hcl:"method,block"`
	Events  []*Event  `hcl:"event,block"`
	Saves   Save      `hcl:"save,block"`

	Abi abi.ABI
}

func (c Contract) Address() common.Address {
	return common.HexToAddress(c.Address_)
}

type Method struct {
	Name_   string            `hcl:"name,label"`
	Inputs_ map[string]string `hcl:"inputs,optional"`
	Outputs []string          `hcl:"outputs"`
}

func (m Method) Name() string {
	return m.Name_
}

func (m Method) Inputs() map[string]string {
	return m.Inputs_
}

type Event struct {
	Name_    string   `hcl:"name,label"`
	Outputs_ []string `hcl:"outputs"`
}

func (e Event) Name() string {
	return e.Name_
}

func (e Event) Outputs() []string {
	return e.Outputs_
}

type Save struct {
	// These should be decoded in a later step with different evaluation contexts,
	// because they should provide access to things like inputs, outputs,
	// block numbers, tx hashes etc.
	Options hcl.Body `hcl:",remain"`
}

// InitialContext returns the initial context at the start of evaluation.
// It has nothing but the most basic functions and variables.
func InitialContext() hcl.EvalContext {
	return hcl.EvalContext(hcl.EvalContext{
		Functions: map[string]function.Function{
			"upper":          stdlib.UpperFunc,
			"lower":          stdlib.LowerFunc,
			"abs":            stdlib.AbsoluteFunc,
			"parse_decimals": ParseDecimals,
			"eq":             stdlib.EqualFunc,
		},
		Variables: map[string]cty.Value{},
	})
}

func NewSchema(confDir string) (*DynamicSchema, error) {
	schemaPath := path.Join(confDir, "schema.hcl")
	f, err := ioutil.ReadFile(schemaPath)
	if err != nil {
		return nil, err
	}

	file, diags := hclsyntax.ParseConfig(f, schemaPath, hcl.InitialPos)
	if diags.HasErrors() {
		return nil, diags.Errs()[0]
	}

	ctx := InitialContext()
	s := &DynamicSchema{
		EvalContext: &ctx,
	}

	diags = gohcl.DecodeBody(file.Body, &ctx, s)
	if diags.HasErrors() {
		return nil, diags.Errs()[0]
	}

	for _, contract := range s.Contracts {
		f, err := os.Open(path.Join(confDir, contract.AbiPath))
		if err != nil {
			return nil, fmt.Errorf("ParseV2: reading ABI file: %w", err)
		}

		abi, err := abi.JSON(f)
		if err != nil {
			return nil, fmt.Errorf("ParseV2: parsing ABI")
		}

		contract.Abi = abi
	}

	return s, nil
}

// EvaluateSaveBlock updates the evaluation context and
// evaluates the save block. The results will be returned as a map.
func (s *DynamicSchema) EvaluateSaveBlock(contractName string, vars map[string]cty.Value) (map[string]cty.Value, error) {
	s.EvalContext.Variables = vars
	saves := make(map[string]cty.Value)

	fmt.Println(vars)

	for _, c := range s.Contracts {
		if c.Name == contractName {
			mv := make(map[string]cty.Value)
			diags := gohcl.DecodeBody(c.Saves.Options, s.EvalContext, &mv)
			if diags.HasErrors() {
				return nil, diags.Errs()[0]
			}

			for k, v := range mv {
				saves[k] = v
			}
		}
	}

	return saves, nil
}

func GenerateVarMap(cr acommon.CallResult) map[string]cty.Value {
	m := make(map[string]cty.Value)

	m["contract_address"], _ = gocty.ToCtyValue(cr.ContractAddress.String(), cty.String)

	m["blocknumber"], _ = gocty.ToCtyValue(cr.BlockNumber, cty.Number)
	m["timestamp"], _ = gocty.ToCtyValue(cr.Timestamp, cty.Number)

	for k, v := range cr.Inputs {
		switch v.(type) {
		case string:
			m[k], _ = gocty.ToCtyValue(v, cty.String)
		default:
			m[k], _ = gocty.ToCtyValue(v, cty.Number)
		}
	}

	for k, v := range cr.Outputs {
		switch v.(type) {
		case string:
			m[k], _ = gocty.ToCtyValue(v, cty.String)
		default:
			m[k], _ = gocty.ToCtyValue(v, cty.Number)
		}
	}

	if cr.Type == acommon.Event {
		m["tx_hash"], _ = gocty.ToCtyValue(cr.TxHash.String(), cty.String)
	}

	return m
}
