-- name: CreateOrder :one
INSERT INTO orders (status, customer_name, notes, total_amount)
VALUES (
  sqlc.arg('status')::text,
  sqlc.narg('customer_name'),
  sqlc.narg('notes'),
  sqlc.narg('total_amount')
)
RETURNING id, status, created_at, updated_at, customer_name, notes, total_amount;

-- name: GetOrderByID :one
SELECT id, status, created_at, updated_at, customer_name, notes, total_amount
FROM orders
WHERE id = sqlc.arg('id')::uuid;

-- name: UpdateOrderStatus :one
UPDATE orders
SET
  status = sqlc.arg('new_status')::text,
  updated_at = now()
WHERE id = sqlc.arg('id')::uuid
  AND status = sqlc.arg('current_status')::text
RETURNING id, status, created_at, updated_at, customer_name, notes, total_amount;

-- name: GetOrdersByIDs :many
SELECT id, status, created_at, updated_at, customer_name, notes, total_amount
FROM orders
WHERE id = ANY(sqlc.arg('ids')::uuid[]);

-- name: ListOrders :many
SELECT id, status, created_at, updated_at, customer_name, notes, total_amount
FROM orders
WHERE (sqlc.narg('filter_status')::text IS NULL OR status = sqlc.narg('filter_status')::text)
  AND (sqlc.narg('created_from')::timestamptz IS NULL OR created_at >= sqlc.narg('created_from')::timestamptz)
  AND (sqlc.narg('created_to')::timestamptz IS NULL OR created_at <= sqlc.narg('created_to')::timestamptz)
ORDER BY created_at DESC;
