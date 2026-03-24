package store

import "errors"

var (
	// ErrNotFound is returned when no row matches the query (e.g. unknown order id).
	ErrNotFound = errors.New("order not found")
	// ErrStatusConflict is returned when compare-and-set status update affects 0 rows.
	ErrStatusConflict = errors.New("order status conflict")
)
