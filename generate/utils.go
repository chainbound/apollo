package generate

type Type string

// Type Enum
const (
	Uint256 Type = "uint256"
	String  Type = "string"
	Address Type = "address"
)

var (
	types = map[Type]string{
		Uint256: "BIGINT",
		String:  "VARCHAR(55)", // strings are used for things like chain names
		Address: "VARCHAR(40)", // addresses should be stored as strings without 0x prefix
	}
)

func ConvertType(solType Type) string {
	return types[solType]
}
