package metrics

import (
	"context"
	"math/big"
	"testing"

	"github.com/glifio/go-pools/constants"
	psdk "github.com/glifio/go-pools/sdk"
	"github.com/glifio/pools-metrics/common"
)

func TestApy(t *testing.T) {
	ctx := context.Background()

	chainID := big.NewInt(constants.MainnetChainID)
	extern, err := common.GetExtern(chainID)

	sdk, err := psdk.New(ctx, chainID, extern)
	apy, err := Apy(ctx, sdk, nil)
	if err != nil {
		t.Fatal(err)
	}

	if apy.Cmp(big.NewFloat(0)) != 1 {
		t.Fatal("apy should be greater than 0")
	}
}
