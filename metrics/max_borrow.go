package metrics

import (
	"context"
	"math/big"

	"github.com/filecoin-project/go-address"
	"github.com/glifio/go-pools/mstat"
	psdk "github.com/glifio/go-pools/sdk"
	pooltypes "github.com/glifio/go-pools/types"
	"github.com/glifio/go-pools/vc"
)

func MinerMaxBorrow(ctx context.Context, sdk pooltypes.PoolsSDK, miner address.Address) (*big.Int, *big.Int, error) {
	lapi, closer, err := sdk.Extern().ConnectLotusClient()
	if err != nil {
		return nil, nil, err
	}
	defer closer()

	ts, err := sdk.Query().ChainHead(ctx)
	if err != nil {
		return nil, nil, err
	}

	edr, err := mstat.ComputeEDRLazy1(ctx, miner, ts, lapi)
	if err != nil {
		return nil, nil, err
	}

	agentValue, err := lapi.WalletBalance(ctx, miner)
	if err != nil {
		return nil, nil, err
	}

	agentVal, ok := new(big.Int).SetString(agentValue.String(), 10)
	if !ok {
		return nil, nil, err
	}

	agentData := &vc.AgentData{
		AgentValue:                  agentVal,
		CollateralValue:             big.NewInt(0),
		ExpectedDailyFaultPenalties: big.NewInt(0),
		ExpectedDailyRewards:        edr,
		Gcred:                       big.NewInt(100),
		QaPower:                     big.NewInt(0),
		Principal:                   big.NewInt(0),
		FaultySectors:               big.NewInt(0),
		LiveSectors:                 big.NewInt(0),
		GreenScore:                  big.NewInt(0),
	}

	nullishCred, err := vc.NullishVerifiableCredential(*agentData)
	if err != nil {
		return nil, nil, err
	}

	rate, err := sdk.Query().InfPoolGetRate(ctx, *nullishCred)
	if err != nil {
		return nil, nil, err
	}

	return psdk.MaxBorrowFromAgentData(agentData, rate), rate, nil
}
