package generate

import (
	"errors"
	"fmt"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
)

// This package should generate the packed ABI inputs

type Method interface {
	Name() string
	Inputs() map[string]string
}

func BuildCallMsg(to common.Address, method Method, abi abi.ABI) (ethereum.CallMsg, error) {
	input, err := BuildCallInput(method, abi)
	if err != nil {
		return ethereum.CallMsg{}, err
	}

	return ethereum.CallMsg{
		Data: input,
		To:   &to,
	}, nil
}

func BuildCallInput(method Method, abi abi.ABI) ([]byte, error) {
	var vals []interface{}
	abiInputs := abi.Methods[method.Name()].Inputs

	for _, abiArg := range abiInputs {
		for name, val := range method.Inputs() {
			// Some ABIs have no named inputs (when there is only a single input for example)
			if abiArg.Name == "" {
				newVal := ABIToGoType(ABIType(abiArg.Type.String()), val)
				vals = append(vals, newVal)
			} else if abiArg.Name == name {
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

func GetTopic(eventName string, abi abi.ABI) (common.Hash, error) {
	var topic common.Hash
	if event, ok := abi.Events[eventName]; !ok {
		return common.Hash{}, errors.New("GetTopic: no such event")
	} else {
		topic = event.ID
	}

	return topic, nil
}
