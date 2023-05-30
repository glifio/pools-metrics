package handler

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/glifio/pools-metrics/common"
	m "github.com/glifio/pools-metrics/metrics"
)

type MetricsHandlerRes struct {
	PoolTotalAssets       string `json:"poolTotalAssets"`
	PoolTotalBorrwed      string `json:"poolTotalBorrowed"`
	TotalAgentCount       uint64 `json:"totalAgentCount"`
	TotalMinerCollaterals string `json:"totalMinerCollaterals"`
	TotalMinersCount      uint64 `json:"totalMinersCount"`
	TotalValueLocked      string `json:"totalValueLocked"`
}

func Metrics(w http.ResponseWriter, r *http.Request) {
	sdk, err := common.NewSDK(r)
	if err != nil {
		http.Error(w, fmt.Sprintf("Error initializing PoolsSDK: %v", err), http.StatusBadRequest)
		return
	}

	metrics, err := m.Metrics(r.Context(), sdk)
	if err != nil {
		http.Error(w, fmt.Sprintf("Error getting metrics: %v", err), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", "*")
	if err := json.NewEncoder(w).Encode(&MetricsHandlerRes{
		PoolTotalAssets:       metrics.PoolTotalAssets.String(),
		PoolTotalBorrwed:      metrics.PoolTotalBorrwed.String(),
		TotalAgentCount:       metrics.TotalAgentCount.Uint64(),
		TotalMinerCollaterals: metrics.TotalMinerCollaterals.String(),
		TotalMinersCount:      metrics.TotalMinersCount.Uint64(),
		TotalValueLocked:      metrics.TotalValueLocked.String(),
	}); err != nil {
		http.Error(w, fmt.Sprintf("Error encoding metrics to JSON: %v", err), http.StatusInternalServerError)
		return
	}
}
