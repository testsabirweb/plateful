package graph

import (
	"fmt"
	"strconv"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/testsabirweb/plateful/internal/graph/model"
	"github.com/testsabirweb/plateful/internal/orders"
	storedb "github.com/testsabirweb/plateful/internal/store/db"
)

func orderToGQL(o storedb.Order) (*model.Order, error) {
	if !o.ID.Valid {
		return nil, fmt.Errorf("order has no id")
	}
	id := uuid.UUID(o.ID.Bytes)

	domainSt, err := orders.ParseStatus(o.Status)
	if err != nil {
		return nil, err
	}

	out := &model.Order{
		ID:        id.String(),
		CreatedAt: timestamptzToTime(o.CreatedAt),
		UpdatedAt: timestamptzToTime(o.UpdatedAt),
		Status:    domainStatusToGQL(domainSt),
	}
	if o.CustomerName.Valid {
		s := o.CustomerName.String
		out.CustomerName = &s
	}
	if o.Notes.Valid {
		s := o.Notes.String
		out.Notes = &s
	}
	if o.TotalAmount.Valid {
		f, err := o.TotalAmount.Float64Value()
		if err == nil && f.Valid {
			s := strconv.FormatFloat(f.Float64, 'f', 2, 64)
			out.TotalAmount = &s
		}
	}
	return out, nil
}

func timestamptzToTime(t pgtype.Timestamptz) time.Time {
	if !t.Valid {
		return time.Time{}
	}
	return t.Time
}

func domainStatusToGQL(s orders.Status) model.OrderStatus {
	switch s {
	case orders.Pending:
		return model.OrderStatusPending
	case orders.Confirmed:
		return model.OrderStatusConfirmed
	case orders.Preparing:
		return model.OrderStatusPreparing
	case orders.Ready:
		return model.OrderStatusReady
	case orders.Delivered:
		return model.OrderStatusDelivered
	case orders.Cancelled:
		return model.OrderStatusCancelled
	default:
		return model.OrderStatusPending
	}
}

func gqlStatusToDomain(s model.OrderStatus) (orders.Status, error) {
	switch s {
	case model.OrderStatusPending:
		return orders.Pending, nil
	case model.OrderStatusConfirmed:
		return orders.Confirmed, nil
	case model.OrderStatusPreparing:
		return orders.Preparing, nil
	case model.OrderStatusReady:
		return orders.Ready, nil
	case model.OrderStatusDelivered:
		return orders.Delivered, nil
	case model.OrderStatusCancelled:
		return orders.Cancelled, nil
	default:
		return "", fmt.Errorf("unknown GraphQL status %q", s)
	}
}

func parseUUID(s string) (pgtype.UUID, error) {
	u, err := uuid.Parse(s)
	if err != nil {
		return pgtype.UUID{}, err
	}
	return pgtype.UUID{Bytes: u, Valid: true}, nil
}

func optionalText(s *string) pgtype.Text {
	if s == nil {
		return pgtype.Text{}
	}
	return pgtype.Text{String: *s, Valid: true}
}

func optionalNumericString(s *string) (pgtype.Numeric, error) {
	var n pgtype.Numeric
	if s == nil || *s == "" {
		return n, nil
	}
	if err := n.Scan(*s); err != nil {
		return n, fmt.Errorf("totalAmount: %w", err)
	}
	return n, nil
}
