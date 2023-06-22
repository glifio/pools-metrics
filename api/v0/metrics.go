package handler

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/glifio/pools-metrics/common"
	m "github.com/glifio/pools-metrics/metrics"
)

type MetricsHandlerRes struct {
	PoolTotalAssets       string `json:"poolTotalAssets"`
	PoolTotalBorrowed     string `json:"poolTotalBorrowed"`
	TotalAgentCount       uint64 `json:"totalAgentCount"`
	TotalMinerCollaterals string `json:"totalMinerCollaterals"`
	TotalMinersCount      uint64 `json:"totalMinersCount"`
	TotalValueLocked      string `json:"totalValueLocked"`
	Denom                 string `json:"denom"`
}

func Metrics(w http.ResponseWriter, r *http.Request) {
	sdk, err := common.NewSDK(r)
	if err != nil {
		http.Error(w, fmt.Sprintf("Error initializing PoolsSDK: %v", err), http.StatusBadRequest)
		return
	}

	var shouldConvert bool = false
	if strings.ToLower(r.URL.Query().Get("denom")) == "fil" {
		shouldConvert = true
	}

	metrics, err := m.Metrics(r.Context(), sdk)
	if err != nil {
		http.Error(w, fmt.Sprintf("Error getting metrics: %v", err), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", "*")
	if err := json.NewEncoder(w).Encode(encodeMetrics(metrics, shouldConvert)); err != nil {
		http.Error(w, fmt.Sprintf("Error encoding metrics to JSON: %v", err), http.StatusInternalServerError)
		return
	}
}

func encodeMetrics(metrics *m.MetricData, shouldConvert bool) *MetricsHandlerRes {
	var res *MetricsHandlerRes
	if !shouldConvert {
		res = &MetricsHandlerRes{
			PoolTotalAssets:       metrics.PoolTotalAssets.String(),
			PoolTotalBorrowed:     metrics.PoolTotalBorrwed.String(),
			TotalMinerCollaterals: metrics.TotalMinerCollaterals.String(),
			TotalValueLocked:      metrics.TotalValueLocked.String(),
			Denom:                 "attofil",
		}
	} else {
		res = &MetricsHandlerRes{
			PoolTotalAssets:       common.FmtFILVal(metrics.PoolTotalAssets),
			PoolTotalBorrowed:     common.FmtFILVal(metrics.PoolTotalBorrwed),
			TotalMinerCollaterals: common.FmtFILVal(metrics.TotalMinerCollaterals),
			TotalValueLocked:      common.FmtFILVal(metrics.TotalValueLocked),
			Denom:                 "fil",
		}
	}

	res.TotalAgentCount = metrics.TotalAgentCount.Uint64()
	res.TotalMinersCount = metrics.TotalMinersCount.Uint64()

	return res
}
