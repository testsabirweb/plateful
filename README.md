# Plateful — Catering order API (Go + GraphQL + Postgres + SQS)

Backend service for a catering company: orders are stored in PostgreSQL, exposed via **GraphQL** (gqlgen), with **validated status transitions** and **async “order created” events** (SQS-compatible API; LocalStack in dev).

## Prerequisites

- **Go 1.25+**
- **Docker** (required for `docker compose`, and for **`go test ./...`** because of the Testcontainers integration test)

## Run everything (recommended)

From the repo root:

```bash
docker compose up --build
```

This starts **Postgres**, **LocalStack (SQS)**, a one-shot **migrate** job, the **API** on [http://localhost:8080](http://localhost:8080), and the **worker**.

- GraphQL HTTP: `POST http://localhost:8080/query`
- Playground: [http://localhost:8080/](http://localhost:8080/)
- Prometheus metrics: `GET http://localhost:8080/metrics`

### Environment (compose sets these for you)

| Variable | Purpose |
|----------|---------|
| `DATABASE_URL` | Postgres connection string |
| `HTTP_ADDR` | Listen address (default `:8080`) |
| `SQS_ENDPOINT` | e.g. `http://localstack:4566` |
| `SQS_QUEUE_NAME` | Queue name (default `plateful-orders`) |
| `AWS_REGION`, `AWS_ACCESS_KEY_ID`, `AWS_SECRET_ACCESS_KEY` | SDK config (dummy `test`/`test` for LocalStack) |

## Local development (without full compose)

```bash
make db-up
make migrate-local   # uses DATABASE_URL_LOCAL in Makefile
export DATABASE_URL='postgres://plateful:plateful@127.0.0.1:5432/plateful?sslmode=disable'
export SQS_ENDPOINT='http://127.0.0.1:4566'   # optional; omit for no-op queue
go run ./cmd/api
# other terminal:
go run ./cmd/worker   # requires SQS_ENDPOINT
```

## Code generation

```bash
make generate   # sqlc + gqlgen
```

## Tests

```bash
go test ./... -count=1
```

The suite includes a **GraphQL integration test** (`internal/graph/integration_test.go`) that starts Postgres in Docker via **Testcontainers**. **Docker must be running** for the default suite to pass.

Fast checks only (skip integration):

```bash
go test -short ./... -count=1
```

## Architecture (short)

- **cmd/api** — HTTP server: GraphQL + Playground + Prometheus `/metrics`; uses **pgxpool**, **sqlc** store, **gqlgen** resolvers, **SQS publisher** (or no-op).
- **cmd/worker** — Long-polls SQS, simulates a notification + increments a processed counter, deletes messages.
- **internal/orders** — Status state machine (`pending` → … → `delivered`; `cancelled` before delivery).
- **internal/store** — sqlc-generated queries + compare-and-set status updates.
- **internal/queue** — `OrderCreated` JSON events; LocalStack uses the AWS SDK with a custom endpoint.
- **infra/terraform/modules/catering-service** — Illustrative ECS + SQS + RDS (not applied in CI).

## Design decisions & trade-offs

- **TEXT + CHECK** for statuses in SQL keeps migrations simple; transitions are enforced in Go as well.
- **Compare-and-set** `UPDATE … WHERE status = $current` avoids lost updates under concurrency.
- **SQS + LocalStack** matches the real queue pattern; **NoOpPublisher** when `SQS_ENDPOINT` is unset so the API can run without async infrastructure.
- **gqlgen** for schema-first GraphQL and generated wiring.
- **Prometheus** client for request count/latency; **slog** for structured access logs (no Grafana in scope).

## Example GraphQL

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
query { order(id: "YOUR-UUID-HERE") { id status } }
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

**Invalid transition** (e.g. `DELIVERED` → `PENDING`) returns a GraphQL error with the underlying validation message.

## What I’d add with more time

- **Retries & DLQ** for queue consumers; **outbox** or transactional publish for exactly-once semantics with the DB.
- **Auth** (JWT/API keys) and tenant isolation.
- **Pagination** on `orders` and cursor-based filters.
- **Hardening**: rate limiting, stricter IAM/RDS/ECS in Terraform, secrets in AWS Secrets Manager.
- **Caching** (Redis) for hot reads if needed.
- **GraphQL dataloaders** (bonus) and **Grafana** for metrics (bonus).
