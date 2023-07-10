package metrics

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"math/big"
	"net/http"

	"github.com/ethereum/go-ethereum/common"
	"github.com/filecoin-project/go-address"
	"github.com/filecoin-project/go-state-types/abi"
	"github.com/filecoin-project/lotus/api"
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

func Metrics(ctx context.Context, sdk pooltypes.PoolsSDK, blockNumber *big.Int) (*MetricData, error) {
	poolTotalAssetsFloat, err := sdk.Query().InfPoolTotalAssets(ctx, blockNumber)
	if err != nil {
		return nil, err
	}
	poolTotalAssets := util.ToAtto(poolTotalAssetsFloat)

	poolTotalBorrowedFloat, err := sdk.Query().InfPoolTotalBorrowed(ctx, blockNumber)
	if err != nil {
		return nil, err
	}
	poolTotalBorrowed := util.ToAtto(poolTotalBorrowedFloat)

	agentCount, minerCount, totalMinerCollaterals, err := MinerCollaterals(ctx, sdk, blockNumber)

	tvl := new(big.Int).Add(poolTotalAssets, totalMinerCollaterals)

	return &MetricData{
		PoolTotalAssets:       poolTotalAssets,
		PoolTotalBorrwed:      poolTotalBorrowed,
		TotalAgentCount:       agentCount,
		TotalMinerCollaterals: totalMinerCollaterals,
		TotalMinersCount:      minerCount,
		TotalValueLocked:      tvl,
	}, nil
}

func AgentsLiquidAssets(ctx context.Context, sdk pooltypes.PoolsSDK, blockNumber *big.Int) (*big.Int, error) {
	resp, err := http.Get("https://events.glif.link/agent/list")
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var data []struct {
		TxHash  string         `json:"txHash"`
		ID      string         `json:"id"`
		Address common.Address `json:"address"`
		Height  *big.Int       `json:"height"`
	}
	err = json.Unmarshal(body, &data)
	if err != nil {
		return nil, err
	}

	tasks := make([]util.TaskFunc, len(data))
	for i, agent := range data {
		tasks[i] = createAgentLiquidAssetTask(ctx, sdk, agent.Address, blockNumber)
	}

	agentsLiquidAssets, err := util.Multiread(tasks)
	if err != nil {
		return nil, err
	}

	var totalAgentLiquidAssets = big.NewInt(0)
	for _, assets := range agentsLiquidAssets {
		totalAgentLiquidAssets.Add(totalAgentLiquidAssets, assets.(*big.Int))
	}

	return totalAgentLiquidAssets, nil
}

func MinerCollaterals(ctx context.Context, sdk pooltypes.PoolsSDK, blockNumber *big.Int) (agentCount *big.Int, minerCount *big.Int, minerCollaterals *big.Int, err error) {
	agentCount, err = sdk.Query().AgentFactoryAgentCount(ctx, blockNumber)
	if err != nil {
		return nil, nil, nil, err
	}

	// parallelize calls to the miner registry to get the list of every miner pledged in the system
	tasks := make([]util.TaskFunc, agentCount.Int64())
	for i := int64(0); i < agentCount.Int64(); i++ {
		index := big.NewInt(i + 1)
		tasks[i] = func() (interface{}, error) {
			// add one to the index because the agent ids start at 1
			return sdk.Query().MinerRegistryAgentMinersList(ctx, index, blockNumber)
		}
	}

	results, err := util.Multiread(tasks)
	if err != nil {
		return nil, nil, nil, err
	}

	lapi, closer, err := sdk.Extern().ConnectLotusClient()
	if err != nil {
		return nil, nil, nil, err
	}
	defer closer()

	var tsk types.TipSetKey = types.EmptyTSK

	if blockNumber != nil {
		ts, err := lapi.ChainGetTipSetByHeight(ctx, abi.ChainEpoch(blockNumber.Int64()), types.EmptyTSK)
		if err != nil {
			return nil, nil, nil, err
		}
		tsk = ts.Key()
	}

	var allMiners []address.Address
	for _, result := range results {
		agentMiners := result.([]address.Address)
		allMiners = append(allMiners, agentMiners...)
	}

	tasks = make([]util.TaskFunc, len(allMiners))
	for i, minerAddr := range allMiners {
		tasks[i] = createStateBalanceTask(ctx, lapi, minerAddr, tsk)
	}

	bals, err := util.Multiread(tasks)
	if err != nil {
		return nil, nil, nil, err
	}

	var totalMinerCollaterals = big.NewInt(0)
	for _, bal := range bals {
		totalMinerCollaterals.Add(totalMinerCollaterals, bal.(*big.Int))
	}

	totalIssuedFIL, err := sdk.Query().InfPoolTotalBorrowed(ctx, blockNumber)
	if err != nil {
		return nil, nil, nil, err
	}
	totalMinerCollaterals.Sub(totalMinerCollaterals, util.ToAtto(totalIssuedFIL))

	// count the assets held on agents as miner collaterals
	agentsLiquidAssets, err := AgentsLiquidAssets(ctx, sdk, blockNumber)
	if err != nil {
		return nil, nil, nil, err
	}
	totalMinerCollaterals.Add(totalMinerCollaterals, agentsLiquidAssets)

	return agentCount, big.NewInt(int64(len(allMiners))), totalMinerCollaterals, nil
}

func createStateBalanceTask(ctx context.Context, lapi *api.FullNodeStruct, addr address.Address, tsk types.TipSetKey) util.TaskFunc {
	return func() (interface{}, error) {
		state, err := lapi.StateReadState(ctx, addr, tsk)
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

func createAgentLiquidAssetTask(ctx context.Context, sdk pooltypes.PoolsSDK, agentAddr common.Address, blockNumber *big.Int) util.TaskFunc {
	return func() (interface{}, error) {
		return sdk.Query().AgentLiquidAssets(ctx, agentAddr, blockNumber)
	}
}
