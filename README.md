# Plateful — Catering order API (Go + GraphQL + Postgres + SQS)

Backend service for a catering company: orders are stored in PostgreSQL, exposed via **GraphQL** (gqlgen), with **validated status transitions**, **async "order created" events** (SQS/LocalStack), **per-IP rate limiting**, **Prometheus metrics**, and a **Grafana dashboard**.

## Prerequisites

- **Go 1.25+**
- **Docker** (required for `docker compose`, and for **`go test ./...`** because of the Testcontainers integration test)

## Run everything (recommended)

```bash
docker compose up --build
```

This starts all services. The first run takes a minute — LocalStack needs ~15s to become healthy before the API and worker start.

## Services & endpoints

| Service | URL | Notes |
|---------|-----|-------|
| **GraphQL Playground** | http://localhost:8080/ | Interactive query UI with schema docs |
| **GraphQL API** | `POST` http://localhost:8080/query | All queries and mutations |
| **Prometheus metrics** | http://localhost:8080/metrics | Raw scrape endpoint |
| **Prometheus** | http://localhost:9090 | Query metrics directly |
| **Grafana dashboard** | http://localhost:3000 | Opens the Plateful API dashboard (no login) |
| **PostgreSQL** | `localhost:5432` | User/pass/db: `plateful` |
| **LocalStack SQS** | http://localhost:4566 | AWS-compatible SQS endpoint |

## Testing each component

### GraphQL API — browser playground

Open http://localhost:8080/ — the GraphQL Playground loads with full schema autocomplete. Paste any query or mutation from the [Example GraphQL](#example-graphql) section.

### GraphQL API — curl

See the full [curl examples](#curl-examples) section below.

### Worker (SQS consumer)

The worker has no HTTP interface. Watch it process events in real time:

```bash
docker compose logs -f worker
```

Every `createOrder` call publishes a message to SQS; you'll see the worker log a simulated notification:

```
worker  | notification: order <uuid> created for Alice
```

### PostgreSQL — direct query

```bash
docker compose exec postgres psql -U plateful -d plateful
```

```sql
SELECT id, status, customer_name, total_amount, created_at FROM orders ORDER BY created_at DESC;
```

### LocalStack SQS — queue inspection

```bash
# List queues
aws --endpoint-url=http://localhost:4566 --region us-east-1 sqs list-queues

# Queue attributes (message count, etc.)
aws --endpoint-url=http://localhost:4566 --region us-east-1 sqs get-queue-attributes \
  --queue-url http://sqs.us-east-1.localhost.localstack.cloud:4566/000000000000/plateful-orders \
  --attribute-names All
```

Set these env vars once to avoid repeating the flags:

```bash
export AWS_DEFAULT_REGION=us-east-1
export AWS_ACCESS_KEY_ID=test
export AWS_SECRET_ACCESS_KEY=test
```

### Prometheus

Open http://localhost:9090 and run a PromQL query, e.g.:

```
rate(http_requests_total[1m])
```

Or check that the API target is being scraped: http://localhost:9090/targets

### Grafana dashboard

Open http://localhost:3000 — the **Plateful API** dashboard loads automatically (no login required). It shows:

- Request rate by path/method
- Request rate by status code
- Latency p50 / p95 / p99
- Total requests and error rate

Send a few requests first to populate the graphs:

```bash
for i in $(seq 1 10); do
  curl -s -X POST http://localhost:8080/query \
    -H "Content-Type: application/json" \
    -d '{"query":"{ orders { id status } }"}' > /dev/null
done
```

### Rate limiter

The API enforces **10 requests/sec per IP** with a burst of 30. To trigger a 429:

```bash
for i in $(seq 1 50); do
  curl -s -o /dev/null -w "%{http_code}\n" http://localhost:8080/query \
    -H "Content-Type: application/json" \
    -d '{"query":"{ orders { id } }"}' &
done
wait
```

---

## Environment variables

Compose sets all of these automatically. For local development without compose:

| Variable | Default | Purpose |
|----------|---------|---------|
| `DATABASE_URL` | — | Postgres connection string (required) |
| `HTTP_ADDR` | `:8080` | API listen address |
| `SQS_ENDPOINT` | — | e.g. `http://localhost:4566`; omit for no-op publisher |
| `SQS_QUEUE_NAME` | `plateful-orders` | SQS queue name |
| `AWS_REGION` | `us-east-1` | AWS SDK region |
| `AWS_ACCESS_KEY_ID` | `test` | Dummy value for LocalStack |
| `AWS_SECRET_ACCESS_KEY` | `test` | Dummy value for LocalStack |

## Local development (without full compose)

```bash
make db-up
make migrate-local
export DATABASE_URL='postgres://plateful:plateful@127.0.0.1:5432/plateful?sslmode=disable'
export SQS_ENDPOINT='http://127.0.0.1:4566'   # optional; omit for no-op queue
go run ./cmd/api
# other terminal:
go run ./cmd/worker
```

## Code generation

```bash
make generate   # sqlc + gqlgen
```

## Tests

```bash
make test    # go vet + go test (15m timeout; needs Docker for Testcontainers)
```

The suite includes a **GraphQL integration test** (`internal/graph/integration_test.go`) that spins up a real Postgres container via Testcontainers. **Docker must be running.**

Fast checks only (skips integration test):

```bash
go test -short ./... -count=1
```

To tear down the compose stack: `make compose-down`.

---

## Architecture

```
Browser / curl
      │
      ▼
┌─────────────────────────────────────┐
│  cmd/api  :8080                     │
│  ├── /          GraphQL Playground  │
│  ├── /query     GraphQL endpoint    │
│  └── /metrics   Prometheus scrape   │
└────────┬─────────────────────────────┘
         │ reads/writes
         ▼
┌──────────────────┐    publishes event on createOrder
│  PostgreSQL :5432│◄────────────────────────────────┐
└──────────────────┘                                 │
                                            ┌────────┴──────┐
                                            │   cmd/api     │
                                            └────────┬──────┘
                                                     │ sends to
                                                     ▼
                                            ┌─────────────────┐
                                            │  LocalStack SQS │
                                            │  :4566          │
                                            └────────┬────────┘
                                                     │ long-polls
                                                     ▼
                                            ┌─────────────────┐
                                            │   cmd/worker    │
                                            │ logs + deletes  │
                                            └─────────────────┘

Prometheus :9090 ──scrapes /metrics──► cmd/api
Grafana    :3000 ──queries──────────► Prometheus
```

### Package layout

| Package | Responsibility |
|---------|---------------|
| `cmd/api` | HTTP server: rate limiter → observability middleware → GraphQL handler |
| `cmd/worker` | SQS long-poll consumer with graceful shutdown |
| `internal/orders` | Status state machine — pure domain logic, no I/O |
| `internal/store` | Repository layer over sqlc-generated queries |
| `internal/graph` | gqlgen resolvers, mappers, dataloader middleware |
| `internal/queue` | `Publisher` interface, `SQSClient`, `NoOpPublisher` |
| `internal/config` | Environment variable loader |
| `internal/observability` | Prometheus metrics middleware, rate limiter |
| `infra/terraform` | Illustrative ECS + SQS + RDS module (not applied) |

---

## Design decisions & trade-offs

- **TEXT + CHECK** for statuses in SQL keeps migrations simple; transitions are also enforced in Go (`internal/orders`).
- **Compare-and-set** `UPDATE … WHERE status = $current` prevents lost updates under concurrency without explicit locking.
- **SQS + LocalStack** matches the real AWS queue pattern; `NoOpPublisher` fallback when `SQS_ENDPOINT` is unset so the API runs without async infrastructure.
- **gqlgen** schema-first: schema is the source of truth; resolvers and models are generated.
- **sqlc** type-safe queries: SQL is the source of truth for the data layer; no ORM magic.
- **Graceful shutdown**: both `cmd/api` (10s drain) and `cmd/worker` trap `SIGTERM` cleanly.
- **Dataloaders** (`dataloadgen`): `order(id)` lookups are batched within a request tick to eliminate N+1 DB calls; falls back to direct lookup when middleware is absent.
- **Rate limiter**: per-IP token bucket (10 req/s, burst 30) using `golang.org/x/time/rate`; stale entries pruned every minute.

---

## Example GraphQL

Use these in the **Playground** at http://localhost:8080/ or via curl (see [curl examples](#curl-examples)).

**Create order**

```graphql
mutation {
  createOrder(input: { customerName: "Alice", notes: "VIP", totalAmount: "12.50" }) {
    id
    status
    createdAt
  }
}
```

**Get order**

```graphql
query {
  order(id: "YOUR-UUID-HERE") { id status customerName totalAmount }
}
```

**List orders with filter**

```graphql
query {
  orders(filter: { status: PENDING }) { id status customerName createdAt }
}
```

**Update status** (valid single step, e.g. pending → confirmed)

```graphql
mutation {
  updateOrderStatus(id: "YOUR-UUID-HERE", status: CONFIRMED) {
    id
    status
  }
}
```

**Invalid transition** (e.g. `DELIVERED` → `PENDING`) returns a GraphQL error with the validation message.

---

## curl examples

All requests go to `POST http://localhost:8080/query`. Start the stack first: `docker compose up --build -d`.

**Capture the order ID in a variable (recommended starting point)**

```bash
ORDER_ID=$(curl -s -X POST http://localhost:8080/query \
  -H "Content-Type: application/json" \
  -d '{"query":"mutation { createOrder(input: { customerName: \"Alice\", notes: \"VIP table\", totalAmount: \"129.99\" }) { id status customerName notes totalAmount createdAt } }"}' \
  | jq -r '.data.createOrder.id')

echo "Created order: $ORDER_ID"
```

**Get a single order**

```bash
curl -s -X POST http://localhost:8080/query \
  -H "Content-Type: application/json" \
  -d "{\"query\":\"{ order(id: \\\"$ORDER_ID\\\") { id status customerName notes totalAmount createdAt } }\"}" | jq
```

**List all orders**

```bash
curl -s -X POST http://localhost:8080/query \
  -H "Content-Type: application/json" \
  -d '{"query":"{ orders { id status customerName totalAmount createdAt } }"}' | jq
```

**List orders filtered by status**

```bash
curl -s -X POST http://localhost:8080/query \
  -H "Content-Type: application/json" \
  -d '{"query":"{ orders(filter: { status: PENDING }) { id status customerName } }"}' | jq
```

**Walk through the status machine**

```bash
# pending → confirmed
curl -s -X POST http://localhost:8080/query \
  -H "Content-Type: application/json" \
  -d "{\"query\":\"mutation { updateOrderStatus(id: \\\"$ORDER_ID\\\", status: CONFIRMED) { id status } }\"}" | jq

# confirmed → preparing
curl -s -X POST http://localhost:8080/query \
  -H "Content-Type: application/json" \
  -d "{\"query\":\"mutation { updateOrderStatus(id: \\\"$ORDER_ID\\\", status: PREPARING) { id status } }\"}" | jq

# preparing → ready
curl -s -X POST http://localhost:8080/query \
  -H "Content-Type: application/json" \
  -d "{\"query\":\"mutation { updateOrderStatus(id: \\\"$ORDER_ID\\\", status: READY) { id status } }\"}" | jq

# ready → delivered
curl -s -X POST http://localhost:8080/query \
  -H "Content-Type: application/json" \
  -d "{\"query\":\"mutation { updateOrderStatus(id: \\\"$ORDER_ID\\\", status: DELIVERED) { id status } }\"}" | jq
```

**Cancel an order** (valid from any non-terminal state)

```bash
curl -s -X POST http://localhost:8080/query \
  -H "Content-Type: application/json" \
  -d "{\"query\":\"mutation { updateOrderStatus(id: \\\"$ORDER_ID\\\", status: CANCELLED) { id status } }\"}" | jq
```

**Trigger an invalid transition** (should return a GraphQL error)

```bash
curl -s -X POST http://localhost:8080/query \
  -H "Content-Type: application/json" \
  -d "{\"query\":\"mutation { updateOrderStatus(id: \\\"$ORDER_ID\\\", status: PENDING) { id status } }\"}" | jq
```

**Check Prometheus metrics**

```bash
curl -s http://localhost:8080/metrics | grep http_requests
```

---

## What I'd add with more time

- **Retries & DLQ** for queue consumers; **outbox** or transactional publish for exactly-once semantics with the DB.
- **Auth** (JWT/API keys) and tenant isolation.
- **Pagination** on `orders` (cursor-based).
- **Hardening**: stricter IAM/RDS/ECS in Terraform, secrets in AWS Secrets Manager, TLS.
- **Caching** (Redis) for hot reads.
