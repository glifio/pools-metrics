package handler

import (
	"fmt"
	"math/big"
	"net/http"

	"github.com/glifio/go-pools/constants"
	"github.com/glifio/pools-metrics/common"
)

var chainID = big.NewInt(constants.MainnetChainID)

func Tvl(w http.ResponseWriter, r *http.Request) {
	sdk, err := common.NewSDK(r)
	if err != nil {
		http.Error(w, fmt.Sprintf("Error initializing PoolsSDK: %v", err), http.StatusBadRequest)
		return
	}

	chainID := sdk.Query().ChainID()

	// chainID := sdk.Query().
	fmt.Fprintf(w, "<h1>Hello from Go! %v</h1>", chainID)
}
