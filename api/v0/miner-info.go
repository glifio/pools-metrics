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

type MinerInfoHandler struct {
	BorrowStart          string `json:"borrowStart"`
	BorrowCap            string `json:"borrowCap"`
	ExpectedDailyRewards string `json:"expectedDailyRewards"`
	Equity               string `json:"equity"`
	AnnualFeeRate        string `json:"annualFeeRate"`
	Denom                string `json:"denom"`
}

func MinerInfo(w http.ResponseWriter, r *http.Request) {
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

	borrowStart, borrowCap, edr, rate, err := m.MinerInfo(r.Context(), sdk, minerAddr)
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
	if err := json.NewEncoder(w).Encode(EncodeMinerInfo(borrowStart, borrowCap, edr, filRate, shouldConvert)); err != nil {
		http.Error(w, fmt.Sprintf("Error encoding to JSON: %v", err), http.StatusInternalServerError)
		return
	}
}

func EncodeMinerInfo(borrowStart *big.Int, borrowCap *big.Int, edr *big.Int, rate *big.Float, shouldConvert bool) *MinerInfoHandler {
	var res *MinerInfoHandler
	if !shouldConvert {
		res = &MinerInfoHandler{
			BorrowCap:            borrowCap.String(),
			BorrowStart:          borrowStart.String(),
			ExpectedDailyRewards: edr.String(),
			Equity:               borrowCap.String(),
			Denom:                "attofil",
		}
	} else {
		res = &MinerInfoHandler{
			BorrowCap:            common.FmtFILVal(borrowCap),
			BorrowStart:          common.FmtFILVal(borrowStart),
			ExpectedDailyRewards: common.FmtFILVal(edr),
			Equity:               common.FmtFILVal(borrowCap),
			Denom:                "fil",
		}
	}

	res.AnnualFeeRate = fmt.Sprintf("%0.03f%%", rate)

	return res
}
