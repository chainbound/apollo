package dsl

import (
	"io/ioutil"

	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/gohcl"
	"github.com/hashicorp/hcl/v2/hclsyntax"
	"github.com/zclconf/go-cty/cty"
	"github.com/zclconf/go-cty/cty/function"
	"github.com/zclconf/go-cty/cty/function/stdlib"
)

type DynamicSchema struct {
	Chain     string      `hcl:"chain"`
	Contracts []*Contract `hcl:"contract,block"`

	EvalContext hcl.EvalContext
}

type Contract struct {
	Name    string `hcl:"name,label"`
	Address string `hcl:"address,label"`
	AbiPath string `hcl:"abi"`

	Methods []*Method `hcl:"method,block"`
	Events  []*Event  `hcl:"event,block"`
}

type Method struct {
	Name    string            `hcl:"name,label"`
	Inputs  map[string]string `hcl:"inputs,optional"`
	Outputs []string          `hcl:"outputs"`
	Saves   Save              `hcl:"save,block"`
}

type Event struct {
	Name    string   `hcl:"name,label"`
	Outputs []string `hcl:"outputs"`
	Saves   Save     `hcl:"save,block"`
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
			"parse_decimals": ParseDecimals,
		},
		Variables: map[string]cty.Value{},
	})
}

func NewSchema(schemaPath string) (*DynamicSchema, error) {
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
		EvalContext: ctx,
	}

	diags = gohcl.DecodeBody(file.Body, &ctx, s)
	if diags.HasErrors() {
		return nil, diags.Errs()[0]
	}

	return s, nil
}

// EvaluateSaveBlock updates the evaluation context and
// evaluates the save block. The results will be returned as a map.
func EvaluateSaveBlock(ctxVars map[string]cty.Value) {

}
