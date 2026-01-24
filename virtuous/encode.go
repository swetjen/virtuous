package virtuous

import (
	"encoding/json"
	"net/http"
)

// Encode writes a JSON response with the provided status code.
func Encode(w http.ResponseWriter, _ *http.Request, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}
