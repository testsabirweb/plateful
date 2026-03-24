package queue

import (
	"context"

	"github.com/google/uuid"
)

// Publisher sends order lifecycle events to an async consumer.
type Publisher interface {
	PublishOrderCreated(ctx context.Context, orderID uuid.UUID) error
}
