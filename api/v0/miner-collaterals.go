package handler

import (
	"fmt"
	"net/http"
)

type MinerCollateralsHandlerRes struct {
	TotalAgentCount       uint64 `json:"totalAgentCount"`
	TotalMinerCollaterals string `json:"totalMinerCollaterals"`
	TotalMinersCount      uint64 `json:"totalMinersCount"`
	Error                 string `json:"error"`
}

func MinerCollaterals(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Content-Type", "application/json")
	http.Error(w, fmt.Sprint("Endpoint deprecated - please use /miner-info instead"), http.StatusBadRequest)
	return
}
