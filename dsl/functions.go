package dsl

import (
	"math/big"
	"time"

	"github.com/zclconf/go-cty/cty"
	"github.com/zclconf/go-cty/cty/function"
	"github.com/zclconf/go-cty/cty/function/stdlib"
)

var Functions = map[string]function.Function{
	"upper":          stdlib.UpperFunc,
	"lower":          stdlib.LowerFunc,
	"abs":            stdlib.AbsoluteFunc,
	"parse_decimals": ParseDecimals,
	"format_date":    FormatDate,
	// "balance":        Balance,
	// "token_balance":  TokenBalance,
}

var ParseDecimals = function.New(&function.Spec{
	Params: []function.Parameter{
		{Name: "raw", Type: cty.Number},
		{Name: "decimals", Type: cty.Number},
	},
	Type: function.StaticReturnType(cty.Number),
	Impl: func(args []cty.Value, retType cty.Type) (cty.Value, error) {
		raw := args[0].AsBigFloat()
		decimalsInt, _ := args[1].AsBigFloat().Int64()
		decimals := new(big.Int).Exp(big.NewInt(10), big.NewInt(decimalsInt), nil)

		parsed, _ := raw.Quo(raw, new(big.Float).SetInt(decimals)).Float64()

		return cty.NumberFloatVal(parsed), nil
	},
})

// Formats the date according to a format and returns the Unix timestamp
// in seconds
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

// var Balance = function.New(&function.Spec{
// 	Params: []function.Parameter{
// 		{Name: "format", Type: cty.String},
// 		{Name: "date", Type: cty.String},
// 	},
// 	Type: function.StaticReturnType(cty.Number),
// 	Impl: func(args []cty.Value, retType cty.Type) (cty.Value, error) {
// 		address := args[0].AsString()
// 		b, err := chainservice.Balance(common.HexToAddress(address))
// 		if err != nil {
// 			return err
// 		}

// 		return cty.NumberIntVal(1000), nil
// 	},
// })

// var TokenBalance = function.New(&function.Spec{
// 	Params: []function.Parameter{
// 		{Name: "format", Type: cty.String},
// 		{Name: "date", Type: cty.String},
// 	},
// 	Type: function.StaticReturnType(cty.Number),
// 	Impl: func(args []cty.Value, retType cty.Type) (cty.Value, error) {
// 		address := args[0].AsString()
// 		b, err := chainservice.TokenBalance(common.HexToAddress(address))
// 		if err != nil {
// 			return err
// 		}

// 		return cty.NumberIntVal(1000), nil
// 	},
// })
