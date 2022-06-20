package dsl

import (
	"math/big"
	"time"

	"github.com/chainbound/apollo/types"
	"github.com/ethereum/go-ethereum/common"
	"github.com/zclconf/go-cty/cty"
	"github.com/zclconf/go-cty/cty/function"
	"github.com/zclconf/go-cty/cty/function/stdlib"
)

// The initial functions provided by the DSL.
var Functions = map[string]function.Function{
	"upper":          stdlib.UpperFunc,
	"lower":          stdlib.LowerFunc,
	"abs":            stdlib.AbsoluteFunc,
	"parse_decimals": ParseDecimals,
	"format_date":    FormatDate,
}

// The definition of the `parse_decimals` function.
//
// Parses a raw blockchain value according to a number of decimals.
var ParseDecimals = function.New(&function.Spec{
	Params: []function.Parameter{
		{Name: "raw", Type: cty.Number},
		{Name: "decimals", Type: cty.Number},
	},
	Type: function.StaticReturnType(cty.Number),
	// The actual function implementation.
	Impl: func(args []cty.Value, retType cty.Type) (cty.Value, error) {
		// args are the arguments as a list.
		raw := args[0].AsBigFloat()
		decimalsInt, _ := args[1].AsBigFloat().Int64()

		divider := new(big.Int).Exp(big.NewInt(10), big.NewInt(decimalsInt), nil)
		parsed, _ := raw.Quo(raw, new(big.Float).SetInt(divider)).Float64()

		return cty.NumberFloatVal(parsed), nil
	},
})

// The definition of the `format_date` function
//
// Formats the date according to a format and returns the Unix timestamp
// in seconds.
var FormatDate = function.New(&function.Spec{
	Params: []function.Parameter{
		{Name: "format", Type: cty.String},
		{Name: "date", Type: cty.String},
	},
	Type: function.StaticReturnType(cty.Number),
	Impl: func(args []cty.Value, retType cty.Type) (cty.Value, error) {
		format := args[0].AsString()
		date := args[1].AsString()

		t, err := time.Parse(format, date)
		if err != nil {
			return cty.NilVal, err
		}

		return cty.NumberIntVal(t.UnixMilli() / 1000), nil
	},
})

// BuildChainFunctions builds the chain functions that can be used in the schema.
func BuildChainFunctions(provider ChainFunctionProvider, chain types.Chain, block *big.Int) map[string]function.Function {
	return map[string]function.Function{
		"balance": function.New(&function.Spec{
			Params: []function.Parameter{
				{Name: "address", Type: cty.String},
			},
			Type: function.StaticReturnType(cty.Number),
			Impl: func(args []cty.Value, retType cty.Type) (cty.Value, error) {
				address := args[0].AsString()
				b, err := provider.Balance(chain, common.HexToAddress(address), block)
				if err != nil {
					return cty.NilVal, err
				}

				return cty.NumberFloatVal(b), nil
			},
		}),

		"token_balance": function.New(&function.Spec{
			Params: []function.Parameter{
				{Name: "address", Type: cty.String},
				{Name: "token", Type: cty.String},
			},
			Type: function.StaticReturnType(cty.Number),
			Impl: func(args []cty.Value, retType cty.Type) (cty.Value, error) {
				address := args[0].AsString()
				token := args[1].AsString()
				b, err := provider.TokenBalance(chain, common.HexToAddress(address), common.HexToAddress(token), block)
				if err != nil {
					return cty.NilVal, err
				}

				return cty.NumberFloatVal(b), nil
			},
		}),

		// "get_price": function.New(&function.Spec{
		// 	Params: []function.Parameter{
		// 		{Name: "from", Type: cty.String},
		// 		{Name: "to", Type: cty.String},
		// 	},
		// 	Type: function.StaticReturnType(cty.Number),
		// 	Impl: func(args []cty.Value, retType cty.Type) (cty.Value, error) {
		// 		from := args[0].AsString()
		// 		to := args[1].AsString()
		// 		b, err := provider.Price(chain, common.HexToAddress(from), common.HexToAddress(to), block)
		// 		if err != nil {
		// 			return cty.NilVal, err
		// 		}

		// 		return cty.NumberFloatVal(b), nil
		// 	},
		// }),
	}
}
