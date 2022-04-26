package generate

import (
	"fmt"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
)

// This package should generate the packed ABI inputs

func BuildCallMsg(to common.Address, method MethodV2, abi abi.ABI) (ethereum.CallMsg, error) {
	input, err := BuildCallInput(method, abi)
	if err != nil {
		return ethereum.CallMsg{}, err
	}
	return ethereum.CallMsg{
		Data: input,
		To:   &to,
	}, nil
}

func BuildCallInput(method MethodV2, abi abi.ABI) ([]byte, error) {
	var vals []interface{}
	for _, abiArg := range abi.Methods[method.Name()].Inputs {
		for name, val := range method.Args() {
			if abiArg.Name == name {
				newVal := ABIToGoType(ABIType(abiArg.Type.String()), val)
				vals = append(vals, newVal)
			}
		}
	}

	input, err := abi.Pack(method.Name(), vals...)
	if err != nil {
		return nil, fmt.Errorf("BuildCallInput: %w", err)
	}

	return input, nil
}
