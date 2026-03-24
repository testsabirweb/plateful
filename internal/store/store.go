package store

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"

	storedb "github.com/testsabirweb/plateful/internal/store/db"
)

// Store wraps sqlc queries with a small repository API.
type Store struct {
	q *storedb.Queries
}

// New returns a Store bound to the given pool.
func New(pool *pgxpool.Pool) *Store {
	return &Store{q: storedb.New(pool)}
}

// CreateOrder inserts a new order.
func (s *Store) CreateOrder(ctx context.Context, arg storedb.CreateOrderParams) (storedb.Order, error) {
	return s.q.CreateOrder(ctx, arg)
}

// GetOrderByID returns an order by id or ErrNotFound.
func (s *Store) GetOrderByID(ctx context.Context, id pgtype.UUID) (storedb.Order, error) {
	o, err := s.q.GetOrderByID(ctx, id)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return storedb.Order{}, ErrNotFound
		}
		return storedb.Order{}, err
	}
	return o, nil
}

// ListOrders returns orders matching optional filters.
func (s *Store) ListOrders(ctx context.Context, arg storedb.ListOrdersParams) ([]storedb.Order, error) {
	return s.q.ListOrders(ctx, arg)
}

// UpdateOrderStatus updates status using compare-and-set on the current status.
// Returns ErrStatusConflict when no row matches id + current status.
func (s *Store) UpdateOrderStatus(ctx context.Context, arg storedb.UpdateOrderStatusParams) (storedb.Order, error) {
	o, err := s.q.UpdateOrderStatus(ctx, arg)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return storedb.Order{}, ErrStatusConflict
		}
		return storedb.Order{}, err
	}
	return o, nil
}
