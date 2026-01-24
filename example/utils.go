package main

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
)

func Encode(w http.ResponseWriter, r *http.Request, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(v); err != nil {
		errorMsg := fmt.Errorf("encode json: %w", err)
		slog.Error("Encode() failed", "err", errorMsg)
		return
	}
}

func Decode[T any](r *http.Request) (T, error) {
	var v T
	if err := json.NewDecoder(r.Body).Decode(&v); err != nil {
		return v, fmt.Errorf("decode json: %w", err)
	}
	return v, nil
}
