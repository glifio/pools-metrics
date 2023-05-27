package common

import (
	"errors"
	"math/big"
	"net/http"
	"net/url"

	"github.com/glifio/go-pools/constants"
	"github.com/glifio/go-pools/deploy"
	psdk "github.com/glifio/go-pools/sdk"
	"github.com/glifio/go-pools/types"
	pooltypes "github.com/glifio/go-pools/types"
)

// default to mainnet
var chainID = big.NewInt(constants.MainnetChainID)

func SupportedNetwork(chainID *big.Int) bool {
	switch chainID.Int64() {
	case constants.MainnetChainID:
		return true
	case constants.CalibnetChainID:
		return true
	case constants.LocalnetChainID:
		return true
	default:
		return false
	}
}

func GetChainID(qparams url.Values) (*big.Int, error) {
	chainIDStr := qparams.Get("chainID")
	if chainIDStr != "" {
		id, ok := new(big.Int).SetString(chainIDStr, 10)
		if !ok {
			return nil, errors.New("Error getting chainID")
		}
		if !SupportedNetwork(id) {
			return nil, errors.New("Unsupported chainID")
		}
		chainID = id
	}

	return chainID, nil
}

func GetExtern(chainID *big.Int) (pooltypes.Extern, error) {
	switch chainID.Int64() {
	case constants.MainnetChainID:
		return deploy.Extern, nil
	case constants.CalibnetChainID:
		return deploy.TestExtern, nil
	default:
		return types.Extern{}, errors.New("Unsupported chainID - add Extern type")
	}
}

func NewSDK(r *http.Request) (pooltypes.PoolsSDK, error) {
	chainID, err := GetChainID(r.URL.Query())
	if err != nil {
		return nil, err
	}

	extern, err := GetExtern(chainID)
	if err != nil {
		return nil, err
	}
	ctx := r.Context()

	sdk, err := psdk.New(ctx, chainID, extern)
	return sdk, nil
}
