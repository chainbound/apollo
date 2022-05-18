package dsl

import (
	"math/big"

	"github.com/zclconf/go-cty/cty"
	"github.com/zclconf/go-cty/cty/function"
)

var ParseDecimals = function.New(&function.Spec{
	Params: []function.Parameter{
		{Type: cty.Number},
		{Type: cty.Number},
	},
	Type: func(args []cty.Value) (cty.Type, error) {
		return cty.Number, nil
	},
	Impl: func(args []cty.Value, retType cty.Type) (cty.Value, error) {
		raw := args[0].AsBigFloat()
		decimalsInt, _ := args[1].AsBigFloat().Int64()
		decimals := new(big.Int).Exp(big.NewInt(10), big.NewInt(decimalsInt), nil)

		parsed, _ := raw.Quo(raw, new(big.Float).SetInt(decimals)).Float64()

		return cty.NumberFloatVal(parsed), nil
	},
})
