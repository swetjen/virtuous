// Package media provides RPC fixture handlers for client-spec tests.
package media

import "context"

// Asset is the fixture response payload.
type Asset struct {
	ID string `json:"id"`
}

// GetAsset is a fixture RPC handler with no request payload.
func GetAsset(_ context.Context) (Asset, int) {
	return Asset{}, 200
}
