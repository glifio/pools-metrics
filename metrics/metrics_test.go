package metrics

import (
	"context"
	"math/big"
	"testing"

	"github.com/glifio/go-pools/constants"
	psdk "github.com/glifio/go-pools/sdk"
	"github.com/glifio/pools-metrics/common"
)

func TestMetrics(t *testing.T) {
	ctx := context.Background()

	chainID := big.NewInt(constants.MainnetChainID)
	extern, err := common.GetExtern(chainID)

	sdk, err := psdk.New(ctx, chainID, extern)
	metrics, err := Metrics(ctx, sdk)
	if err != nil {
		t.Fatal(err)
	}

	if metrics.PoolTotalAssets.Cmp(big.NewInt(0)) != 1 {
		t.Fatal("PoolTotalAssets should be greater than 0")
	}
	if metrics.PoolTotalBorrwed.Cmp(big.NewInt(0)) != 1 {
		t.Fatal("PoolTotalBorrwed should be greater than 0")
	}
	if metrics.TotalAgentCount.Cmp(big.NewInt(0)) != 1 {
		t.Fatal("TotalAgentCount should be greater than 0")
	}
	if metrics.TotalMinerCollaterals.Cmp(big.NewInt(0)) != 1 {
		t.Fatal("TotalMinerCollaterals should be greater than 0")
	}
	if metrics.TotalMinersCount.Cmp(big.NewInt(0)) != 1 {
		t.Fatal("TotalMinersCount should be greater than 0")
	}
}