package orders

import (
	"errors"
	"testing"
)

func TestCanTransition(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		from Status
		to   Status
		want bool
	}{
		// forward path
		{from: Pending, to: Confirmed, want: true},
		{from: Confirmed, to: Preparing, want: true},
		{from: Preparing, to: Ready, want: true},
		{from: Ready, to: Delivered, want: true},
		// cancel before terminal
		{from: Pending, to: Cancelled, want: true},
		{from: Confirmed, to: Cancelled, want: true},
		{from: Preparing, to: Cancelled, want: true},
		{from: Ready, to: Cancelled, want: true},
		// skips / backward
		{name: "skip ahead", from: Pending, to: Ready, want: false},
		{name: "backward", from: Confirmed, to: Pending, want: false},
		{name: "delivered to pending", from: Delivered, to: Pending, want: false},
		// terminal
		{name: "from delivered", from: Delivered, to: Ready, want: false},
		{name: "from cancelled", from: Cancelled, to: Pending, want: false},
		// no-op
		{name: "same status", from: Pending, to: Pending, want: false},
		// unknown
		{name: "unknown from", from: Status("nope"), to: Pending, want: false},
		{name: "unknown to", from: Pending, to: Status("nope"), want: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			n := tt.name
			if n == "" {
				n = string(tt.from) + "_to_" + string(tt.to)
			}
			got := CanTransition(tt.from, tt.to)
			if got != tt.want {
				t.Fatalf("%s: CanTransition(%q, %q) = %v, want %v", n, tt.from, tt.to, got, tt.want)
			}
		})
	}
}

func TestValidateTransition(t *testing.T) {
	t.Parallel()

	err := ValidateTransition(Pending, Confirmed)
	if err != nil {
		t.Fatalf("expected nil, got %v", err)
	}

	err = ValidateTransition(Delivered, Pending)
	if err == nil {
		t.Fatal("expected error")
	}
	if !errors.Is(err, ErrInvalidTransition) {
		t.Fatalf("want ErrInvalidTransition, got %v", err)
	}

	err = ValidateTransition(Status("x"), Pending)
	if err == nil {
		t.Fatal("expected error")
	}
	if !errors.Is(err, ErrUnknownStatus) {
		t.Fatalf("want ErrUnknownStatus, got %v", err)
	}
}

func TestParseStatus(t *testing.T) {
	t.Parallel()

	s, err := ParseStatus("ready")
	if err != nil || s != Ready {
		t.Fatalf("ParseStatus(ready) = %q, %v", s, err)
	}

	_, err = ParseStatus("invalid")
	if err == nil {
		t.Fatal("expected error")
	}
	if !errors.Is(err, ErrUnknownStatus) {
		t.Fatalf("want ErrUnknownStatus, got %v", err)
	}
}

func TestIsTerminal(t *testing.T) {
	t.Parallel()

	if !Delivered.IsTerminal() || !Cancelled.IsTerminal() {
		t.Fatal("delivered and cancelled should be terminal")
	}
	if Pending.IsTerminal() {
		t.Fatal("pending should not be terminal")
	}
}
