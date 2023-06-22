package metrics

import (
	"context"
	"math/big"

	"github.com/filecoin-project/go-address"
	pooltypes "github.com/glifio/go-pools/types"
	"github.com/glifio/go-pools/util"
)

func Miners(ctx context.Context, sdk pooltypes.PoolsSDK) (*big.Int, []address.Address, error) {
	agentCount, err := sdk.Query().AgentFactoryAgentCount(ctx)
	if err != nil {
		return nil, nil, err
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
		return nil, nil, err
	}
	var allMiners []address.Address
	for _, result := range results {
		agentMiners := result.([]address.Address)
		allMiners = append(allMiners, agentMiners...)
	}

	return big.NewInt(int64(len(allMiners))), allMiners, nil
}
