package chainservice

import (
	"context"
	"fmt"
	"math"
	"math/big"
	"time"

	"github.com/chainbound/apollo/log"
	"github.com/rs/zerolog"

	"github.com/ethereum/go-ethereum/ethclient"
)

type BlockWrapper struct {
	Number    *big.Int
	Timestamp int64
}

// TODO: this should be a separate struct per chain
type BlockDater struct {
	client *ethclient.Client

	blockCache map[int64]BlockWrapper
	logger     zerolog.Logger

	BlockTime float64
	Latest    *BlockWrapper
	First     *BlockWrapper
}

func NewBlockDater(client *ethclient.Client) BlockDater {
	return BlockDater{
		client:     client,
		blockCache: make(map[int64]BlockWrapper),
		logger:     log.NewLogger("blockdater"),
	}
}

// BlockNumberByTimestamp returns the first block it finds that is under 60 seconds
// difference with the target timestamp.
func (b *BlockDater) BlockNumberByTimestamp(ctx context.Context, timestamp int64) (int64, error) {
	ts := time.Unix(timestamp, 0)

	if b.Latest == nil || b.First == nil || b.BlockTime == 0 {
		err := b.SetBoundaries(ctx)
		if err != nil {
			return 0, fmt.Errorf("error setting boundaries: %w", err)
		}

	}

	if ts.Before(time.Unix(b.First.Timestamp, 0)) {
		return b.First.Number.Int64(), nil
	}

	if ts.After(time.Unix(b.Latest.Timestamp, 0)) {
		return b.Latest.Number.Int64(), nil
	}

	predictedNum := int64(math.Ceil(float64(timestamp-b.First.Timestamp) / b.BlockTime))
	predicted, err := b.GetBlock(ctx, big.NewInt(predictedNum))
	if err != nil {
		return 0, err
	}

	target, err := b.FindTargetBlock(ctx, predicted, timestamp, 3*60)
	if err != nil {
		return 0, err
	}

	return target.Int64(), nil
}

func (b BlockDater) SecondsToBlockInterval(ctx context.Context, seconds int64) (int64, error) {
	if b.Latest == nil || b.First == nil || b.BlockTime == 0 {
		err := b.SetBoundaries(ctx)
		if err != nil {
			return 0, fmt.Errorf("error setting boundaries: %w", err)
		}
	}

	return int64(float64(seconds) / b.BlockTime), nil
}

var err error

// TODO: this should be a binary search
func (b *BlockDater) FindTargetBlock(ctx context.Context, currentBlock BlockWrapper, target, threshold int64) (*big.Int, error) {
	// blockFound := false

	var blockDiff int64

	for {
		blockDiff = int64(float64(target-currentBlock.Timestamp) / b.BlockTime)
		if target-threshold < currentBlock.Timestamp && currentBlock.Timestamp < target+threshold {
			return currentBlock.Number, nil
		}

		// if currentBlock.Timestamp < target-threshold {
		newBlockNum := currentBlock.Number.Int64() + blockDiff
		currentBlock, err = b.GetBlock(ctx, big.NewInt(newBlockNum))
		if err != nil {
			return nil, err
		}
	}
}

func (b *BlockDater) SetBoundaries(ctx context.Context) error {
	first, err := b.GetBlock(ctx, big.NewInt(1))
	if err != nil {
		return err
	}

	b.First = &first

	latest, err := b.GetBlock(ctx, nil)
	if err != nil {
		return err
	}

	b.Latest = &latest

	// Calculates the average block time
	b.BlockTime = float64(b.Latest.Timestamp-b.First.Timestamp) / float64(b.Latest.Number.Int64()-1)
	return nil
}

func (b *BlockDater) GetBlock(ctx context.Context, num *big.Int) (BlockWrapper, error) {
	safeNum := num
	if num == nil {
		safeNum = big.NewInt(0)
	}

	if cached, ok := b.blockCache[safeNum.Int64()]; ok {
		return cached, nil
	}

	block, err := b.client.BlockByNumber(ctx, num)
	if err != nil {
		return BlockWrapper{}, fmt.Errorf("getting block number %s: %w", num, err)
	}

	wrapper := BlockWrapper{
		Number:    block.Number(),
		Timestamp: int64(block.Time()),
	}

	b.blockCache[block.Number().Int64()] = wrapper

	b.logger.Trace().Msgf("got new block: %v", wrapper)

	return wrapper, nil
}
