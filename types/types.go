package types

import "github.com/ethereum/go-ethereum/common"

type Chain string

const (
	ETHEREUM Chain = "ethereum"
	AVAX     Chain = "avax"
	ARBITRUM Chain = "arbitrum"
	OPTIMISM Chain = "optimism"
	POLYGON  Chain = "polygon"
	FANTOM   Chain = "fantom"
)

// Main program options, provided as cli arguments
type ApolloOpts struct {
	Realtime   bool
	Db         bool
	Csv        bool
	Stdout     bool
	Interval   int64
	StartBlock int64
	EndBlock   int64
	RateLimit  int
	Chain      string
	LogLevel   int
	LogParts   int
}

type ResultType int

const (
	Event ResultType = iota
	GlobalEvent
	Method
)

type CallResult struct {
	Err             error
	Chain           Chain
	Type            ResultType
	QueryName       string
	Identifier      string
	ContractAddress common.Address
	BlockNumber     uint64
	BlockHash       common.Hash

	Timestamp uint64
	TxSender  common.Address
	TxIndex   uint
	TxHash    common.Hash
	Inputs    map[string]any
	Outputs   map[string]any
}
