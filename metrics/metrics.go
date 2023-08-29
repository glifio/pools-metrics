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
	PoolTotalAssets           *big.Int `json:"poolTotalAssets"`
	PoolTotalBorrowed         *big.Int `json:"poolTotalBorrowed"`
	PoolTotalBorrowableAssets *big.Int `json:"poolTotalBorrowableAssets"`
	PoolExitReserve           *big.Int `json:"poolExitReserve"`
	TotalAgentCount           *big.Int `json:"totalAgentCount"`
	TotalMinerCollaterals     *big.Int `json:"totalMinerCollaterals"`
	TotalMinersCount          *big.Int `json:"totalMinersCount"`
	TotalValueLocked          *big.Int `json:"totalValueLocked"`
	TotalMinersSectors        *big.Int `json:"totalMinersSectors"`
	TotalMinerQAP             *big.Int `json:"totalMinerQAP"`
	TotalMinerRBP             *big.Int `json:"totalMinerRBP"`
}

func Metrics(ctx context.Context, sdk pooltypes.PoolsSDK, blockNumber *big.Int) (*MetricData, error) {
	poolTotalAssetsFloat, err := sdk.Query().InfPoolTotalAssets(ctx, blockNumber)
	if err != nil {
		return nil, err
	}
	poolTotalAssets := util.ToAtto(poolTotalAssetsFloat)

	poolTotalBorrowable, err := sdk.Query().InfPoolBorrowableLiquidity(ctx, blockNumber)
	if err != nil {
		return nil, err
	}

	poolExitReserves, _, err := sdk.Query().InfPoolExitReserve(ctx, blockNumber)
	if err != nil {
		return nil, err
	}

	poolTotalBorrowedFloat, err := sdk.Query().InfPoolTotalBorrowed(ctx, blockNumber)
	if err != nil {
		return nil, err
	}
	poolTotalBorrowed := util.ToAtto(poolTotalBorrowedFloat)

	agentCount, minerCount, totalMinerCollaterals, totalMinerSectors, totalMinerQAP, totalMinerRBP, err := MinerCollaterals(ctx, sdk, blockNumber)
	if err != nil {
		return nil, err
	}

	tvl := new(big.Int).Add(poolTotalAssets, totalMinerCollaterals)

	return &MetricData{
		PoolTotalAssets:           poolTotalAssets,
		PoolTotalBorrowed:         poolTotalBorrowed,
		PoolTotalBorrowableAssets: util.ToAtto(poolTotalBorrowable),
		PoolExitReserve:           poolExitReserves,
		TotalAgentCount:           agentCount,
		TotalMinerCollaterals:     totalMinerCollaterals,
		TotalMinersCount:          minerCount,
		TotalMinersSectors:        totalMinerSectors,
		TotalMinerQAP:             totalMinerQAP,
		TotalMinerRBP:             totalMinerRBP,
		TotalValueLocked:          tvl,
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

func MinerCollaterals(ctx context.Context, sdk pooltypes.PoolsSDK, blockNumber *big.Int) (agentCount *big.Int, minerCount *big.Int, minerCollaterals *big.Int, totalMinerSectors *big.Int, totalMinerQAP *big.Int, totalMinerRBP *big.Int, err error) {
	agentCount, err = sdk.Query().AgentFactoryAgentCount(ctx, blockNumber)
	if err != nil {
		return nil, nil, nil, nil, nil, nil, err
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
		return nil, nil, nil, nil, nil, nil, err
	}

	lapi, closer, err := sdk.Extern().ConnectLotusClient()
	if err != nil {
		return nil, nil, nil, nil, nil, nil, err
	}
	defer closer()

	var tsk types.TipSetKey = types.EmptyTSK
	if blockNumber != nil {
		ts, err := lapi.ChainGetTipSetByHeight(ctx, abi.ChainEpoch(blockNumber.Int64()), types.EmptyTSK)
		if err != nil {
			return nil, nil, nil, nil, nil, nil, err
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
		return nil, nil, nil, nil, nil, nil, err
	}

	var totalMinerCollaterals = big.NewInt(0)
	for _, bal := range bals {
		totalMinerCollaterals.Add(totalMinerCollaterals, bal.(*big.Int))
	}

	tasks = make([]util.TaskFunc, len(allMiners))
	for i, minerAddr := range allMiners {
		tasks[i] = createSectorPowerTask(ctx, lapi, minerAddr, tsk)
	}

	sectorPows, err := util.Multiread(tasks)
	if err != nil {
		return nil, nil, nil, nil, nil, nil, err
	}

	totalMinerSectors = big.NewInt(0)
	totalMinerQAP = big.NewInt(0)
	totalMinerRBP = big.NewInt(0)
	for _, sectorPow := range sectorPows {
		minerSectorPow := sectorPow.(*MinerSectorsPower)
		totalMinerSectors.Add(totalMinerSectors, minerSectorPow.sectors)
		totalMinerQAP.Add(totalMinerQAP, minerSectorPow.qap)
		totalMinerRBP.Add(totalMinerRBP, minerSectorPow.rbp)
	}

	totalIssuedFIL, err := sdk.Query().InfPoolTotalBorrowed(ctx, blockNumber)
	if err != nil {
		return nil, nil, nil, nil, nil, nil, err
	}
	totalMinerCollaterals.Sub(totalMinerCollaterals, util.ToAtto(totalIssuedFIL))

	// count the assets held on agents as miner collaterals
	agentsLiquidAssets, err := AgentsLiquidAssets(ctx, sdk, blockNumber)
	if err != nil {
		return nil, nil, nil, nil, nil, nil, err
	}
	totalMinerCollaterals.Add(totalMinerCollaterals, agentsLiquidAssets)

	return agentCount, big.NewInt(int64(len(allMiners))), totalMinerCollaterals, totalMinerSectors, totalMinerQAP, totalMinerRBP, nil
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

type MinerSectorsPower struct {
	miner   address.Address
	sectors *big.Int
	qap     *big.Int
	rbp     *big.Int
}

func createSectorPowerTask(ctx context.Context, lapi *api.FullNodeStruct, addr address.Address, tsk types.TipSetKey) util.TaskFunc {
	return func() (interface{}, error) {

		pow, err := lapi.StateMinerPower(ctx, addr, tsk)
		if err != nil {
			return nil, err
		}

		return &MinerSectorsPower{
			miner:   addr,
			sectors: big.NewInt(0),
			qap:     pow.MinerPower.QualityAdjPower.Int,
			rbp:     pow.MinerPower.RawBytePower.Int,
		}, nil
	}
}
