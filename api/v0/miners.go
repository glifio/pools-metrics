package handler

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/filecoin-project/go-address"
	"github.com/glifio/pools-metrics/common"
	m "github.com/glifio/pools-metrics/metrics"
)

type MinersRes struct {
	Miners []address.Address `json:"miners"`
	Count  uint64            `json:"count"`
}

func Miners(w http.ResponseWriter, r *http.Request) {
	sdk, err := common.NewSDK(r)
	if err != nil {
		http.Error(w, fmt.Sprintf("Error initializing PoolsSDK: %v", err), http.StatusBadRequest)
		return
	}

	minerCount, miners, err := m.Miners(r.Context(), sdk)
	if err != nil {
		http.Error(w, fmt.Sprintf("Error getting miners: %v", err), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(&MinersRes{
		Miners: miners,
		Count:  minerCount.Uint64(),
	}); err != nil {
		http.Error(w, fmt.Sprintf("Error encoding metrics to JSON: %v", err), http.StatusInternalServerError)
		return
	}
}
