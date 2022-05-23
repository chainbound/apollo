package chainservice

import (
	apolloTypes "github.com/chainbound/apollo/types"
	"github.com/ethereum/go-ethereum/accounts/abi"
)

func aggregateCallResults(results ...*apolloTypes.CallResult) *apolloTypes.CallResult {
	new := results[0]

	for i := 1; i < len(results); i++ {
		for k, v := range results[i].Inputs {
			new.Inputs[k] = v
		}

		for k, v := range results[i].Outputs {
			new.Outputs[k] = v
		}
	}

	return new
}

func matchABIValue(outputName string, outputs abi.Arguments, results []any) any {
	if len(results) == 1 {
		return results[0]
	}

	for i, o := range outputs {
		if o.Name == outputName {
			return results[i]
		}
	}

	return nil
}
