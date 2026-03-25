package queue

import (
	"context"

	"github.com/google/uuid"
)

// NoOpPublisher satisfies Publisher and drops events (used when SQS is disabled).
type NoOpPublisher struct{}

// PublishOrderCreated implements Publisher.
func (NoOpPublisher) PublishOrderCreated(ctx context.Context, orderID uuid.UUID) error {
	_ = ctx
	_ = orderID
	return nil
}
