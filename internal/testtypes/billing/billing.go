// Package billing provides RPC fixture handlers for client-spec tests.
package billing

import (
	"context"

	"github.com/jackc/pgx/v5/pgtype"
)

// InvoiceQuery is the fixture request payload.
type InvoiceQuery struct {
	Limit  int    `json:"limit"`
	Cursor string `json:"cursor,omitempty" doc:"Opaque page cursor"`
}

// Invoice exercises optional, nullable, and documented fields.
type Invoice struct {
	ID     string      `json:"id" doc:"Invoice id"`
	Amount float64     `json:"amount"`
	Note   *string     `json:"note,omitempty" doc:"Optional note"`
	PaidAt pgtype.Text `json:"paidAt"`
}

// Page exercises a generic type name in the schema graph.
type Page[T any] struct {
	Items []T `json:"items"`
	Total int `json:"total"`
}

// ListInvoices is a fixture RPC handler.
func ListInvoices(_ context.Context, _ InvoiceQuery) (Page[Invoice], int) {
	return Page[Invoice]{}, 200
}

// ListInvoicesLegacy is a fixture RPC handler registered as deprecated.
func ListInvoicesLegacy(_ context.Context) (Page[Invoice], int) {
	return Page[Invoice]{}, 200
}
