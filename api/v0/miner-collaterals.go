package handler

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/glifio/pools-metrics/common"
	m "github.com/glifio/pools-metrics/metrics"
)

type MinerCollateralsHandlerRes struct {
	TotalAgentCount       uint64 `json:"totalAgentCount"`
	TotalMinerCollaterals string `json:"totalMinerCollaterals"`
	TotalMinersCount      uint64 `json:"totalMinersCount"`
}

func MinerCollaterals(w http.ResponseWriter, r *http.Request) {
	sdk, err := common.NewSDK(r)
	if err != nil {
		http.Error(w, fmt.Sprintf("Error initializing PoolsSDK: %v", err), http.StatusBadRequest)
		return
	}

	agentCount, minerCount, minerCollaterals, err := m.MinerCollaterals(r.Context(), sdk)
	if err != nil {
		http.Error(w, fmt.Sprintf("Error getting metrics: %v", err), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(&MinerCollateralsHandlerRes{
		TotalAgentCount:       agentCount.Uint64(),
		TotalMinerCollaterals: minerCollaterals.String(),
		TotalMinersCount:      minerCount.Uint64(),
	}); err != nil {
		http.Error(w, fmt.Sprintf("Error encoding metrics to JSON: %v", err), http.StatusInternalServerError)
		return
	}
}
