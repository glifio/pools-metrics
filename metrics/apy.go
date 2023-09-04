package metrics

import (
	"context"
	"math/big"

	pooltypes "github.com/glifio/go-pools/types"
	"github.com/glifio/go-pools/util"
)

func Apy(ctx context.Context, sdk pooltypes.PoolsSDK, blockNumber *big.Int) (*big.Float, error) {
	apy, err := sdk.Query().InfPoolApy(ctx, blockNumber)
	if err != nil {
		return nil, err
	}

	return util.ToFIL(apy), nil
}
