package handler

import (
	"encoding/json"
	"fmt"
	"math/big"
	"net/http"
	"strings"

	"github.com/filecoin-project/go-address"
	"github.com/glifio/go-pools/constants"
	"github.com/glifio/go-pools/util"
	"github.com/glifio/pools-metrics/common"
	m "github.com/glifio/pools-metrics/metrics"
)

type MinerMaxBorrowHandler struct {
	MaxBorrow     string `json:"maxBorrow"`
	AnnualFeeRate string `json:"annualFeeRate"`
	Denom         string `json:"denom"`
}

func MinerMaxBorrow(w http.ResponseWriter, r *http.Request) {
	sdk, err := common.NewSDK(r)
	if err != nil {
		http.Error(w, fmt.Sprintf("Error initializing PoolsSDK: %v", err), http.StatusBadRequest)
		return
	}

	minerAddr, err := address.NewFromString(r.URL.Query().Get("miner"))
	if err != nil {
		http.Error(w, fmt.Sprintf("Error parsing miner address: %v", err), http.StatusBadRequest)
		return
	}

	maxBorrow, rate, err := m.MinerMaxBorrow(r.Context(), sdk, minerAddr)
	if err != nil {
		http.Error(w, fmt.Sprintf("Error getting miner max borrow: %v", err), http.StatusInternalServerError)
		return
	}

	// annualize rate
	rate.Mul(rate, big.NewInt(constants.EpochsInYear))
	rate.Div(rate, constants.WAD)
	filRate := util.ToFIL(rate)
	// make a rate a percentage
	filRate.Mul(filRate, big.NewFloat(100))

	var shouldConvert bool = false
	if strings.ToLower(r.URL.Query().Get("denom")) == "fil" {
		shouldConvert = true
	}

	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", "*")
	if err := json.NewEncoder(w).Encode(encodeMinerMaxBorrow(maxBorrow, filRate, shouldConvert)); err != nil {
		http.Error(w, fmt.Sprintf("Error encoding to JSON: %v", err), http.StatusInternalServerError)
		return
	}
}

func encodeMinerMaxBorrow(maxBorrow *big.Int, rate *big.Float, shouldConvert bool) *MinerMaxBorrowHandler {
	var res *MinerMaxBorrowHandler
	if !shouldConvert {
		res = &MinerMaxBorrowHandler{
			MaxBorrow: maxBorrow.String(),
			Denom:     "attofil",
		}
	} else {
		res = &MinerMaxBorrowHandler{
			MaxBorrow: common.FmtFILVal(maxBorrow),
			Denom:     "fil",
		}
	}

	res.AnnualFeeRate = fmt.Sprintf("%0.03f%%", rate)

	return res
}
