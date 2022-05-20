package chainservice

import (
	"context"
	"fmt"
	"testing"

	"github.com/ethereum/go-ethereum/ethclient"
)

func TestBlockByTime(t *testing.T) {
	rpcUrl := "wss://arb-mainnet.g.alchemy.com/v2/4pRMjCWK8ZXbg0rqCBYT9surz2k26oH5"
	ctx := context.Background()
	client, err := ethclient.DialContext(ctx, rpcUrl)
	if err != nil {
		t.Fatal(err)
	}

	dater := NewBlockDater(client)

	target, err := dater.BlockNumberByTimestamp(ctx, 1653083262)
	if err != nil {
		t.Fatal(err)
	}

	// should be around 10767286
	fmt.Println("final target: ", target)
}
