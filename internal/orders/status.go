// Package orders holds order domain types and state-transition rules.
package orders

import (
	"errors"
	"fmt"
)

// Status is the lifecycle state of an order (matches DB CHECK constraint).
type Status string

const (
	Pending   Status = "pending"
	Confirmed Status = "confirmed"
	Preparing Status = "preparing"
	Ready     Status = "ready"
	Delivered Status = "delivered"
	Cancelled Status = "cancelled"
)

var allStatuses = []Status{
	Pending, Confirmed, Preparing, Ready, Delivered, Cancelled,
}

// transitions lists allowed next states from each non-terminal state.
// Terminal: delivered, cancelled (empty slice).
var transitions = map[Status][]Status{
	Pending:   {Confirmed, Cancelled},
	Confirmed: {Preparing, Cancelled},
	Preparing: {Ready, Cancelled},
	Ready:     {Delivered, Cancelled},
	Delivered: {},
	Cancelled: {},
}

// ParseStatus returns Status if s is a known value.
func ParseStatus(s string) (Status, error) {
	st := Status(s)
	if !st.IsKnown() {
		return "", fmt.Errorf("%w: %q", ErrUnknownStatus, s)
	}
	return st, nil
}

// IsKnown reports whether s is one of the defined order statuses.
func (s Status) IsKnown() bool {
	for _, v := range allStatuses {
		if v == s {
			return true
		}
	}
	return false
}

// IsTerminal reports whether no further transitions are allowed.
func (s Status) IsTerminal() bool {
	return s == Delivered || s == Cancelled
}

// CanTransition reports whether a single-step transition from → to is allowed.
// Same-status updates are not considered valid transitions.
func CanTransition(from, to Status) bool {
	if from == to {
		return false
	}
	if !from.IsKnown() || !to.IsKnown() {
		return false
	}
	if from.IsTerminal() {
		return false
	}
	for _, next := range transitions[from] {
		if next == to {
			return true
		}
	}
	return false
}

// ErrInvalidTransition is returned when a status change violates the state machine.
var ErrInvalidTransition = errors.New("invalid status transition")

// ErrUnknownStatus is returned when a string is not a defined status.
var ErrUnknownStatus = errors.New("unknown order status")

// ValidateTransition returns nil if CanTransition(from, to), otherwise ErrInvalidTransition
// (wrapped with context) or an error if either status is unknown.
func ValidateTransition(from, to Status) error {
	if !from.IsKnown() || !to.IsKnown() {
		return fmt.Errorf("%w: from=%q to=%q", ErrUnknownStatus, from, to)
	}
	if CanTransition(from, to) {
		return nil
	}
	return fmt.Errorf("%w: %s → %s", ErrInvalidTransition, from, to)
}
