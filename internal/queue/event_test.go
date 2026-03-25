package queue

import (
	"testing"

	"github.com/google/uuid"
)

func TestMarshalRoundTrip(t *testing.T) {
	t.Parallel()
	id := uuid.MustParse("550e8400-e29b-41d4-a716-446655440000")
	b, err := MarshalJSONEvent(Event{Type: EventTypeOrderCreated, OrderID: id.String()})
	if err != nil {
		t.Fatal(err)
	}
	e, err := UnmarshalJSONEvent(b)
	if err != nil {
		t.Fatal(err)
	}
	if e.Type != EventTypeOrderCreated || e.OrderID != id.String() {
		t.Fatalf("got %+v", e)
	}
}
