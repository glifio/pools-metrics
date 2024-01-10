package handler

import (
	"encoding/json"
	"fmt"
	"math/big"
	"net/http"
)

type ApyRes struct {
	Apy *big.Float `json:"apy"`
}

func Apy(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(&ApyRes{
		Apy: big.NewFloat(0),
	}); err != nil {
		http.Error(w, fmt.Sprintf("Error encoding apy to JSON: %v", err), http.StatusInternalServerError)
		return
	}
}
