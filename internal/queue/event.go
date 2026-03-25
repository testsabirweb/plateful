package queue

import (
	"context"
	"encoding/json"

	"github.com/google/uuid"
)

// EventTypeOrderCreated is published after an order row is inserted.
const EventTypeOrderCreated = "OrderCreated"

// Event is the JSON envelope for queue messages (SQS body).
type Event struct {
	Type    string `json:"type"`
	OrderID string `json:"orderId"`
}

// Publisher sends order lifecycle events to an async consumer.
type Publisher interface {
	PublishOrderCreated(ctx context.Context, orderID uuid.UUID) error
}

// MarshalJSONEvent returns the canonical JSON body for SQS SendMessage.
func MarshalJSONEvent(e Event) ([]byte, error) {
	return json.Marshal(e)
}

// UnmarshalJSONEvent parses a message body into Event.
func UnmarshalJSONEvent(data []byte) (Event, error) {
	var e Event
	if err := json.Unmarshal(data, &e); err != nil {
		return Event{}, err
	}
	return e, nil
}
