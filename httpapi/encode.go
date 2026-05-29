package httpapi

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"

	"github.com/swetjen/virtuous/internal/jsonlimit"
)

var ErrRequestBodyTooLarge = jsonlimit.ErrBodyTooLarge

// Encode writes a JSON response with the provided status code.
func Encode(w http.ResponseWriter, _ *http.Request, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

func Decode[T any](r *http.Request) (T, error) {
	return DecodeWithMaxBytes[T](r, jsonlimit.DefaultMaxBytes)
}

func DecodeWithMaxBytes[T any](r *http.Request, maxBytes int64) (T, error) {
	var v T
	body, err := jsonlimit.LimitReader(r, maxBytes)
	if err != nil {
		return v, fmt.Errorf("decode json: %w", err)
	}
	if err := json.NewDecoder(body).Decode(&v); err != nil {
		return v, fmt.Errorf("decode json: %w", err)
	}
	return v, nil
}

func IsRequestBodyTooLarge(err error) bool {
	return errors.Is(err, ErrRequestBodyTooLarge) || jsonlimit.IsBodyTooLarge(err)
}
