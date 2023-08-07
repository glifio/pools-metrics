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
	PoolTotalAssets           string `json:"poolTotalAssets"`
	PoolTotalBorrowed         string `json:"poolTotalBorrowed"`
	PoolTotalBorrowableAssets string `json:"poolTotalBorrowableAssets"`
	PoolExitReserve           string `json:"poolExitReserve"`
	TotalAgentCount           uint64 `json:"totalAgentCount"`
	TotalMinerCollaterals     string `json:"totalMinerCollaterals"`
	TotalMinersCount          uint64 `json:"totalMinersCount"`
	TotalMinersSectors        string `json:"totalMinersSectors"`
	TotalMinerQAP             string `json:"totalMinerQAP"`
	TotalMinerRBP             string `json:"totalMinerRBP"`
	TotalValueLocked          string `json:"totalValueLocked"`

	Denom       string `json:"denom"`
	BlockNumber uint64 `json:"blockNumber"`
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

	blockNumber, err := common.GetBlockNumberQP(r)
	if err != nil {
		http.Error(w, fmt.Sprintf("Error getting block number: %v", err), http.StatusBadRequest)
		return
	}

	metrics, err := m.Metrics(r.Context(), sdk, blockNumber)
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
			PoolTotalAssets:           metrics.PoolTotalAssets.String(),
			PoolTotalBorrowed:         metrics.PoolTotalBorrowed.String(),
			PoolTotalBorrowableAssets: metrics.PoolTotalBorrowableAssets.String(),
			PoolExitReserve:           metrics.PoolExitReserve.String(),
			TotalMinerCollaterals:     metrics.TotalMinerCollaterals.String(),
			TotalValueLocked:          metrics.TotalValueLocked.String(),
			Denom:                     "attofil",
		}
	} else {
		res = &MetricsHandlerRes{
			PoolTotalAssets:           common.FmtFILVal(metrics.PoolTotalAssets),
			PoolTotalBorrowed:         common.FmtFILVal(metrics.PoolTotalBorrowed),
			PoolTotalBorrowableAssets: common.FmtFILVal(metrics.PoolTotalBorrowableAssets),
			PoolExitReserve:           common.FmtFILVal(metrics.PoolExitReserve),
			TotalMinerCollaterals:     common.FmtFILVal(metrics.TotalMinerCollaterals),
			TotalValueLocked:          common.FmtFILVal(metrics.TotalValueLocked),
			Denom:                     "fil",
		}
	}

	res.TotalAgentCount = metrics.TotalAgentCount.Uint64()
	res.TotalMinersCount = metrics.TotalMinersCount.Uint64()

	return res
}
