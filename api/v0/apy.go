package handler

import (
	"encoding/json"
	"fmt"
	"math/big"
	"net/http"

	"github.com/glifio/pools-metrics/common"
	m "github.com/glifio/pools-metrics/metrics"
)

type ApyRes struct {
	Apy *big.Float `json:"apy"`
}

func Apy(w http.ResponseWriter, r *http.Request) {
	sdk, err := common.NewSDK(r)
	if err != nil {
		http.Error(w, fmt.Sprintf("Error initializing PoolsSDK: %v", err), http.StatusBadRequest)
		return
	}

	blockNumber, err := common.GetBlockNumberQP(r)
	if err != nil {
		http.Error(w, fmt.Sprintf("Error getting block number: %v", err), http.StatusBadRequest)
		return
	}

	apy, err := m.Apy(r.Context(), sdk, blockNumber)
	if err != nil {
		http.Error(w, fmt.Sprintf("Error getting apy: %v", err), http.StatusInternalServerError)
		return
	}

	apy.Mul(apy, big.NewFloat(100))

	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(&ApyRes{
		Apy: apy,
	}); err != nil {
		http.Error(w, fmt.Sprintf("Error encoding apy to JSON: %v", err), http.StatusInternalServerError)
		return
	}
}
