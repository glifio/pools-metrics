package metrics

import (
	"context"
	"fmt"
	"math/big"

	"github.com/filecoin-project/go-address"
	"github.com/filecoin-project/lotus/chain/types"
	pooltypes "github.com/glifio/go-pools/types"
	"github.com/glifio/go-pools/util"
)

type MetricData struct {
	PoolTotalAssets       *big.Int `json:"poolTotalAssets"`
	PoolTotalBorrwed      *big.Int `json:"poolTotalBorrowed"`
	TotalAgentCount       *big.Int `json:"totalAgentCount"`
	TotalMinerCollaterals *big.Int `json:"totalMinerCollaterals"`
	TotalMinersCount      *big.Int `json:"totalMinersCount"`
	TotalValueLocked      *big.Int `json:"totalValueLocked"`
}

func Metrics(ctx context.Context, sdk pooltypes.PoolsSDK) (*MetricData, error) {
	poolTotalAssetsFloat, err := sdk.Query().InfPoolTotalAssets(ctx)
	if err != nil {
		return nil, err
	}
	poolTotalAssets := util.ToAtto(poolTotalAssetsFloat)

	poolTotalBorrowedFloat, err := sdk.Query().InfPoolTotalBorrowed(ctx)
	if err != nil {
		return nil, err
	}
	poolTotalBorrowed := util.ToAtto(poolTotalBorrowedFloat)

	agentCount, err := sdk.Query().AgentFactoryAgentCount(ctx)
	if err != nil {
		return nil, err
	}

	// parallelize calls to the miner registry to get the list of every miner pledged in the system
	tasks := make([]util.TaskFunc, agentCount.Int64())
	for i := int64(0); i < agentCount.Int64(); i++ {
		index := big.NewInt(i + 1)
		tasks[i] = func() (interface{}, error) {
			// add one to the index because the agent ids start at 1
			return sdk.Query().MinerRegistryAgentMinersList(nil, index)
		}
	}

	results, err := util.Multiread(tasks)
	if err != nil {
		return nil, err
	}

	lapi, closer, err := sdk.Extern().ConnectLotusClient()
	if err != nil {
		return nil, err
	}
	defer closer()

	var allMiners []address.Address
	for _, result := range results {
		agentMiners := result.([]address.Address)
		allMiners = append(allMiners, agentMiners...)
	}

	tasks = make([]util.TaskFunc, len(allMiners))
	for i, minerAddr := range allMiners {
		tasks[i] = func() (interface{}, error) {
			state, err := lapi.StateReadState(ctx, minerAddr, types.EmptyTSK)
			if err != nil {
				return nil, err
			}
			bal, ok := new(big.Int).SetString(state.Balance.String(), 10)
			if !ok {
				return nil, fmt.Errorf("failed to convert balance to big.Int")
			}

			return bal, nil
		}
	}

	bals, err := util.Multiread(tasks)
	if err != nil {
		return nil, err
	}

	var totalMinerCollaterals = big.NewInt(0)
	for _, bal := range bals {
		totalMinerCollaterals.Add(totalMinerCollaterals, bal.(*big.Int))
	}

	tvl := new(big.Int).Add(poolTotalAssets, totalMinerCollaterals)
	tvl.Sub(tvl, poolTotalBorrowed)

	return &MetricData{
		PoolTotalAssets:       poolTotalAssets,
		PoolTotalBorrwed:      poolTotalBorrowed,
		TotalAgentCount:       agentCount,
		TotalMinerCollaterals: totalMinerCollaterals,
		TotalMinersCount:      big.NewInt(int64(len(allMiners))),
		TotalValueLocked:      tvl,
	}, nil
}
