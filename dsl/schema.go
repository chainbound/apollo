package dsl

import (
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"path"
	"time"

	"github.com/chainbound/apollo/types"

	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/gohcl"
	"github.com/hashicorp/hcl/v2/hcldec"
	"github.com/hashicorp/hcl/v2/hclsyntax"
	"github.com/zclconf/go-cty/cty"
	"github.com/zclconf/go-cty/cty/gocty"
)

var (
	ErrNoIntervalRealtime                 = errors.New("no interval defined for realtime method calls")
	ErrNoIntervalHistorical               = errors.New("no interval defined for historical method calls")
	ErrIntervalDefinedForHistoricalEvents = errors.New("interval defined for historical events")
)

type DynamicSchema struct {
	StartTime    int64                `hcl:"start_time,optional"`
	EndTime      int64                `hcl:"end_time,optional"`
	TimeInterval int64                `hcl:"time_interval,optional"`
	StartBlock   int64                `hcl:"start_block,optional"`
	EndBlock     int64                `hcl:"end_block,optional"`
	Interval     int64                `hcl:"interval,optional"`
	Variables    map[string]cty.Value `hcl:"variables,optional"`
	// Contract schema's
	Queries []*Query `hcl:"query,block"`

	EvalContext *hcl.EvalContext
}

// EvalVariables loads the variables into the top-level evaluation context.
func (s *DynamicSchema) EvalVariables() {
	for k, v := range s.Variables {
		s.EvalContext.Variables[k] = v

		for _, query := range s.Queries {
			query.EvalContext.Variables[k] = v
		}
	}
}

type Query struct {
	Name      string      `hcl:"name,label"`
	Chain     types.Chain `hcl:"chain"`
	Contracts []*Contract `hcl:"contract,block"`
	// Global events
	Events  []*Event `hcl:"event,block"`
	Filters hcl.Body `hcl:"filter,remain"`
	Saves   Save     `hcl:"save,block"`

	EvalContext *hcl.EvalContext
}

// type Filter struct {
// 	Options hcl.Body `hcl:",remain"`
// }

// EvalTransforms evaluates the transformation block per contract / top-level method.
// The identifier is the OutputName of the method or the name of the contract in other
// cases.
func (q *Query) EvalTransforms(tp types.ResultType, identifier string) error {
	if tp == types.GlobalEvent {
		for _, event := range q.Events {
			if event.Transforms == nil {
				return nil
			}

			if event.OutputName() == identifier {
				mv := make(map[string]cty.Value)
				diags := gohcl.DecodeBody(event.Transforms.Options, q.EvalContext, &mv)
				if diags.HasErrors() {
					return diags.Errs()[0]
				}

				for k, v := range mv {
					q.EvalContext.Variables[k] = v
				}
			}
		}
	} else {
		for _, c := range q.Contracts {
			if c.Transforms == nil {
				return nil
			}

			if c.Address().String() == identifier {
				mv := make(map[string]cty.Value)
				diags := gohcl.DecodeBody(c.Transforms.Options, q.EvalContext, &mv)
				if diags.HasErrors() {
					return diags.Errs()[0]
				}

				for k, v := range mv {
					q.EvalContext.Variables[k] = v
				}
			}
		}
	}

	return nil
}

func (s *DynamicSchema) EvalFilter(queryName string) (bool, error) {
	filterspec := hcldec.AttrSpec{
		Name: "filter",
		Type: cty.List(cty.Bool),
	}

	var filters []bool
	for _, q := range s.Queries {
		if q.Name == queryName {
			if q.Filters == nil {
				return true, nil
			}

			v, diags := hcldec.Decode(q.Filters, &filterspec, q.EvalContext)
			if diags.HasErrors() {
				return false, diags.Errs()[0]
			}

			err := gocty.FromCtyValue(v, &filters)
			if err != nil {
				return false, err
			}
		}
	}

	// Check if all outputs evaluate to true, otherwise return false
	for _, result := range filters {
		if !result {
			return false, nil
		}
	}

	return true, nil
}

// EvalSave updates the evaluation context, evaluates the transform blocks and then
// evaluates the save block. The results will be returned as a map.
func (s *DynamicSchema) EvalSave(tp types.ResultType, queryName string, identifier string, vars map[string]cty.Value) (map[string]cty.Value, error) {
	outputs := make(map[string]cty.Value)
	for _, q := range s.Queries {
		if q.Name == queryName {
			for k, v := range vars {
				q.EvalContext.Variables[k] = v
			}

			if err := q.EvalTransforms(tp, identifier); err != nil {
				return nil, err
			}

			diags := gohcl.DecodeBody(q.Saves.Options, q.EvalContext, &outputs)
			if diags.HasErrors() {
				return nil, diags.Errs()[0]
			}
		}
	}

	ok, err := s.EvalFilter(queryName)
	if err != nil {
		return nil, err
	}

	if !ok {
		return nil, nil
	}

	return outputs, nil
}

func (s DynamicSchema) Validate(opts types.ApolloOpts) error {
	hasMethods := false
	hasEvents := false
	for _, q := range s.Queries {
		for _, c := range q.Contracts {
			hasMethods = len(c.Methods) > 0
			hasEvents = len(c.Events) > 0
		}
	}

	if hasMethods {
		if opts.Realtime {
			if s.Interval == 0 && s.TimeInterval == 0 {
				return ErrNoIntervalRealtime
			}
		}

		if (s.StartBlock != 0 && s.EndBlock != 0) || (s.StartTime != 0 && s.EndTime != 0) {
			if s.Interval == 0 && s.TimeInterval == 0 {
				return ErrNoIntervalHistorical
			}
		}
	}

	if hasEvents {
		if !opts.Realtime {
			if s.Interval != 0 {
				return ErrIntervalDefinedForHistoricalEvents
			}

			if s.TimeInterval != 0 {
				return ErrIntervalDefinedForHistoricalEvents
			}
		}
	}

	return nil
}

func (q Query) HasGlobalEvents() bool {
	return len(q.Events) > 0
}

func (q Query) HasContractEvents() (hasContractEvents bool) {
	for _, c := range q.Contracts {
		if len(c.Events) > 0 {
			hasContractEvents = true
		}
	}

	return
}

func (q Query) HasContractMethods() (hasContractMethods bool) {
	for _, c := range q.Contracts {
		if len(c.Methods) > 0 {
			hasContractMethods = true
		}
	}

	return
}

type Contract struct {
	// TODO: remove
	// Name     string `hcl:"name,label"`
	Address_ string `hcl:"address,label"`
	AbiPath  string `hcl:"abi"`

	Methods []*Method `hcl:"method,block"`
	Events  []*Event  `hcl:"event,block"`

	Transforms *Transform `hcl:"transform,block"`

	// The ABI will get injected when decoding the schema
	Abi abi.ABI
}

func (c Contract) Address() common.Address {
	return common.HexToAddress(c.Address_)
}

type Method struct {
	BlockOffset int64             `hcl:"block_offset,optional"`
	Name_       string            `hcl:"name,label"`
	Inputs_     map[string]string `hcl:"inputs,optional"`
	Outputs     []string          `hcl:"outputs"`
}

func (m Method) Name() string {
	return m.Name_
}

func (m Method) Inputs() map[string]string {
	return m.Inputs_
}

type Event struct {
	Name_    string    `hcl:"name,label"`
	AbiPath  string    `hcl:"abi,optional"`
	Outputs_ []string  `hcl:"outputs"`
	Methods  []*Method `hcl:"method,block"`

	Transforms *Transform `hcl:"transform,block"`

	Abi abi.ABI
}

func (e Event) Name() string {
	return e.Name_
}

func (e Event) Outputs() []string {
	return e.Outputs_
}

func (e Event) OutputName() string {
	return e.Name_ + "_events"
}

type Transform struct {
	// These should be decoded in a later step with different evaluation contexts,
	// because they should provide access to things like inputs, outputs,
	// block numbers, tx hashes etc.
	Options hcl.Body `hcl:",remain"`
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
		Functions: Functions,
		Variables: map[string]cty.Value{
			"now": cty.NumberIntVal(time.Now().UnixMilli() / 1000),
		},
	})
}

// NewSchema returns a new DynamicSchema, loaded from confDir/schema.hcl.
// It will decode the top-level body with an initial evaluation context
// to provide access to custom functions. For each contract, it will also
// read and convert the json ABI file to an abi.ABI.
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

	schemaContext := InitialContext()
	s := &DynamicSchema{
		EvalContext: &schemaContext,
	}

	diags = gohcl.DecodeBody(file.Body, &schemaContext, s)
	if diags.HasErrors() {
		return nil, diags.Errs()[0]
	}

	for _, query := range s.Queries {
		queryContext := InitialContext()
		query.EvalContext = &queryContext

		for _, event := range query.Events {
			f, err := os.Open(path.Join(confDir, event.AbiPath))
			if err != nil {
				return nil, fmt.Errorf("ParseV2: reading ABI file: %w", err)
			}

			abi, err := abi.JSON(f)
			if err != nil {
				return nil, fmt.Errorf("ParseV2: parsing ABI")
			}

			event.Abi = abi
		}

		for _, contract := range query.Contracts {
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
	}

	// Evaluate top-level variables
	s.EvalVariables()

	return s, nil
}

func GenerateContextVars(cr types.CallResult) map[string]cty.Value {
	m := make(map[string]cty.Value)

	m["contract_address"], _ = gocty.ToCtyValue(cr.ContractAddress.String(), cty.String)
	m["blocknumber"], _ = gocty.ToCtyValue(cr.BlockNumber, cty.Number)
	m["timestamp"], _ = gocty.ToCtyValue(cr.Timestamp, cty.Number)
	m["block_hash"], _ = gocty.ToCtyValue(cr.BlockHash.String(), cty.String)

	if cr.Type != types.Method {
		m["tx_hash"], _ = gocty.ToCtyValue(cr.TxHash.String(), cty.String)
		m["tx_index"], _ = gocty.ToCtyValue(cr.TxIndex, cty.Number)
	}

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
		case common.Address:
			m[k], _ = gocty.ToCtyValue(v.(common.Address).String(), cty.String)
		case string:
			m[k], _ = gocty.ToCtyValue(v, cty.String)
		default:
			m[k], _ = gocty.ToCtyValue(v, cty.Number)
		}
	}

	return m
}
