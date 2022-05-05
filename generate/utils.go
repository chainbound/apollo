package generate

import (
	"math/big"

	"github.com/ethereum/go-ethereum/common"
)

type ABIType string

// Type Enum
const (
	Uint256 ABIType = "uint256"
	String  ABIType = "string"
	Address ABIType = "address"
)

var (
	sqlTypes = map[ABIType]string{
		Uint256: "NUMERIC",
		String:  "VARCHAR(55)", // strings are used for things like chain names
		Address: "VARCHAR(42)", // addresses should be stored as strings without 0x prefix
	}
)

func ABIToSQLType(abiType ABIType) string {
	if sqlType, ok := sqlTypes[abiType]; ok {
		return sqlType
	}

	// By default, return BIGINT
	return "NUMERIC"
}

func ABIToGoType(abiType ABIType, val string) any {
	switch abiType {
	case Uint256:
		v, _ := new(big.Int).SetString(val, 10)
		return v
	case String:
		return val
	case Address:
		return common.HexToAddress(val)
	default:
		v, _ := new(big.Int).SetString(val, 10)
		return v
	}
}
