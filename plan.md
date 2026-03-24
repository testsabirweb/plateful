# Golang + DevOps Take-Home Assignment Plan

This document outlines the implementation plan and TODO checklist for building a backend service for a catering company.

The goal is to deliver a **production-grade, minimal but well-structured system** that demonstrates:

- Strong backend fundamentals (Go, GraphQL, SQL)
- Clean architecture and state modeling
- Async/event-driven patterns
- DevOps readiness (Docker, Terraform, CI)
- Testing and observability

### Scope & submission (from brief)

- **Time budget:** 24–48 hours — scope accordingly; document what you would improve with more time.
- **Stack notes:** GraphQL via **gqlgen** (or similar); migrations via **golang-migrate** (or similar).
- **Submission:** GitHub repo (public or private invite); runnable with `docker-compose up`, testable with `go test ./...`.
- **README** must cover setup, architecture decisions, trade-offs, and “what I’d add with more time” (see [section 12 — README](#12-readme)).

### Progress workflow

After each implementation step: **commit** the changes, then **update this file** — check off completed TODOs (and add short notes where something was deferred or done out of order).

---

## Table of contents

| # | Section |
|---|---------|
| 0 | [High-level architecture](#0-high-level-architecture) |
| 1 | [Project setup](#1-project-setup) |
| 2 | [Database design & migrations](#2-database-design--migrations) |
| 3 | [SQLC setup & queries](#3-sqlc-setup--queries) |
| 4 | [Domain logic (state machine)](#4-domain-logic-state-machine) |
| 5 | [GraphQL API (gqlgen)](#5-graphql-api-gqlgen) |
| 6 | [Async processing (queue + worker)](#6-async-processing-queue--worker) |
| 7 | [Docker & Docker Compose](#7-docker--docker-compose) |
| 8 | [Terraform module](#8-terraform-module) |
| 9 | [CI pipeline (GitHub Actions)](#9-ci-pipeline-github-actions) |
| 10 | [Observability](#10-observability) |
| 11 | [Testing](#11-testing) |
| 12 | [README](#12-readme) |
| 13 | [Bonus (optional)](#13-bonus-optional) |
| 14 | [Assignment requirements checklist](#14-assignment-requirements-checklist) |
| — | [Definition of done](#definition-of-done) |

---

## 0. High-level architecture

### Components

- **API Service (Go + GraphQL)**  
  Handles queries/mutations, applies business logic, persists data via sqlc.

- **PostgreSQL**  
  Stores orders.

- **Queue (SQS via LocalStack or in-memory fallback)**  
  Receives `OrderCreated` events.

- **Worker Service**  
  Consumes queue messages; simulates async processing.

- **Infra (Terraform)**  
  ECS service, SQS queue, RDS Postgres.

---

## 1. Project setup

### Target layout

```
cmd/
  api/
  worker/
internal/
  orders/
  store/
  queue/
  observability/
  config/
  graph/
db/
  migrations/
  query/
infra/
  terraform/
```

### TODO

- [x] Initialize Go module
- [x] Create project structure (see layout above)
- [x] Add Makefile:
  - [x] `make test`
  - [x] `make generate` (sqlc)
  - [x] `make run-api` / `make run-worker` (no single `make run`; optional to add later)
- [x] Setup environment config (env vars) — `internal/config` (`HTTP_ADDR`, `DATABASE_URL`)

---

## 2. Database design & migrations

### Orders table

| Field | Type | Notes |
|-------|------|--------|
| `id` | UUID or BIGSERIAL | Primary key |
| `status` | TEXT or ENUM | See status flow |
| `created_at` | TIMESTAMP | |
| `updated_at` | TIMESTAMP | |
| `customer_name` | optional | |
| `notes` | optional | |
| `total_amount` | optional | |

### Status flow

Assignment wording: `pending → confirmed → preparing → ready → delivered` (with **cancelled** as a terminal outcome; validate transitions — e.g. cannot go from **delivered** back to **pending**).

```
pending → confirmed → preparing → ready → delivered
   ↘
cancelled
```

### TODO

- [x] Choose status representation:
  - [ ] ENUM (strict DB-level validation)
  - [x] TEXT + CHECK constraint (simpler migrations)
- [x] Create migration files using golang-migrate:
  - [x] `000001_create_orders_table.up.sql`
  - [x] `000001_create_orders_table.down.sql`
- [x] Add indexes:
  - [x] `status`
  - [x] `created_at`
- [x] Verify migrations run via CLI and Docker (`make migrate-up` / `make migrate-local` with Postgres)

---

## 3. SQLC setup & queries

### Configuration

- [x] Create `sqlc.yaml`
- [x] Configure:
  - schema: `db/migrations`
  - queries: `db/query`
  - engine: `postgresql`

### Queries to implement

- [x] `CreateOrder`
- [x] `GetOrderByID`
- [x] `UpdateOrderStatus`
- [x] `ListOrders` (with filters)

### Filtering strategy

Use optional filters: `status`, date range (`from`, `to`).

SQL pattern:

```sql
WHERE (status = $1 OR $1 IS NULL)
AND (created_at >= $2 OR $2 IS NULL)
AND (created_at <= $3 OR $3 IS NULL)
```

### TODO

- [x] Use `sqlc.narg()` for nullable params (`ListOrders` filters)
- [x] Generate code: `sqlc generate` / `make generate`
- [x] Wrap generated queries in repository layer (`internal/store` → `internal/store/db`)

### Concurrency safety (important)

- [x] Implement compare-and-set update (`UpdateOrderStatus` in `db/query/orders.sql`)

```sql
UPDATE orders
SET status = $new
WHERE id = $id AND status = $current
```

- [x] Handle 0 rows affected → conflict error (`internal/store`: `ErrStatusConflict`)

---

## 4. Domain logic (state machine)

### TODO

- [ ] Define type and constants:

```go
type Status string

const (
  Pending   Status = "pending"
  Confirmed Status = "confirmed"
  Preparing Status = "preparing"
  Ready     Status = "ready"
  Delivered Status = "delivered"
  Cancelled Status = "cancelled"
)
```

- [ ] Implement transition map: `map[Status][]Status`
- [ ] Implement:
  - `CanTransition(from, to)`
  - `ValidateTransition(from, to)`

### Rules

- `delivered` → terminal
- `cancelled` → terminal
- No backward transitions
- Cancellation allowed before delivery

### Tests

- [ ] Table-driven unit tests covering:
  - Valid transitions
  - Invalid transitions
  - Terminal states

---

## 5. GraphQL API (gqlgen)

### Schema

**Types**

- `Order`
- `OrderStatus` (enum)
- `CreateOrderInput`
- `OrdersFilter`

**Queries**

- `order(id)`
- `orders(filter)`

**Mutations**

- `createOrder(input)`
- `updateOrderStatus(id, status)`

### TODO

- [ ] Initialize gqlgen
- [ ] Define `schema.graphqls`
- [ ] Run `go run github.com/99designs/gqlgen generate`

### Resolver design

- Keep resolvers thin: validate input → call service → return result
- Map domain → GraphQL types

---

## 6. Async processing (queue + worker)

### Queue interface

```go
type Queue interface {
  Publish(ctx, event)
  Consume(ctx) (event)
}
```

### Event payload (example)

```json
{
  "type": "OrderCreated",
  "orderId": "..."
}
```

### TODO — API

- [ ] On `createOrder`: insert into DB, then publish event to queue

### TODO — worker

- [ ] Poll queue
- [ ] Process message: simulated action — e.g. log “notification sent”, **or** update an analytics counter (per brief)
- [ ] Delete message after success

### Implementation options

| Option | Notes |
|--------|--------|
| **Preferred** | LocalStack (SQS) |
| **Fallback** | In-memory Go channel |

---

## 7. Docker & Docker Compose

### Services

- `api`
- `worker`
- `postgres`
- `localstack` (SQS)

### TODO

- [ ] Write Dockerfile: multi-stage build, small final image
- [x] **Partial:** `docker-compose.yml` with **Postgres only** for local DB + migrations (`make db-up`, `make migrate-local`). Full stack (api, worker, LocalStack) still TODO.
- [ ] Ensure `docker-compose up --build` works with no manual steps (full stack)

---

## 8. Terraform module

### Goal

Create a **non-applied** module that shows infra design.

### Resources (illustrative)

- `aws_sqs_queue`
- `aws_ecs_task_definition`
- `aws_ecs_service`
- `aws_db_instance` (Postgres)

### TODO

- [ ] Create module: `infra/terraform/modules/catering-service/`
- [ ] Add variables: `vpc_id`, `subnet_ids`, `image`, DB config
- [ ] Add outputs: `queue_url`, `service_name`
- [ ] Add comments: design decisions, trade-offs, what’s omitted (ALB, IAM, etc.)

---

## 9. CI pipeline (GitHub Actions)

### TODO

- [ ] Create `.github/workflows/ci.yml`

### Steps

Per brief: run tests, generate **sqlc** output, build Docker image. Recommended full sequence:

1. Checkout code
2. Setup Go
3. `go test ./...`
4. `sqlc generate`
5. `gqlgen generate` (keeps generated GraphQL code in sync with schema)
6. Build Docker image

---

## 10. Observability

### Logging

- Use `slog` or `zerolog`
- Add: request logs, error logs, worker logs

### Metrics

- Add: request count, request latency
- Expose: `/metrics`
- Assignment: instrument metrics; **no** full Grafana setup required here (see [bonus](#13-bonus-optional) for Grafana/Prometheus add-on).

---

## 11. Testing

### Unit tests

- State transition logic

### Integration test

Assignment: **at least one** integration test for the GraphQL API (plus unit tests for state-transition logic — see [section 4](#4-domain-logic-state-machine)).

- Spin up Postgres (Testcontainers preferred)
- Apply migrations
- Start API
- Run GraphQL: `createOrder`, fetch order

### TODO

- [ ] Keep tests minimal but realistic
- [ ] Ensure `go test ./...` runs cleanly

---

## 12. README

### Must include

- Setup instructions
- How to run: `docker-compose up`
- How to test: `go test ./...`
- Architecture explanation
- Design decisions and trade-offs

### Demo section

Example GraphQL queries:

- Create order
- Fetch order
- Update status
- Invalid transition

### Final section

“What I’d add with more time” — explicit examples from the original plan:

- Retries & DLQ
- Auth
- Pagination
- Production infra
- Caching

(Add others as needed.)

---

## 13. Bonus (optional)

### TODO

- [ ] Rate limiter middleware
- [ ] Prometheus + Grafana (docker-compose)
- [ ] GraphQL dataloaders (batching)

---

## 14. Assignment requirements checklist

Quick mapping to the take-home brief (Parts 1–4):

| Part | Requirement | Planned in |
|------|-------------|------------|
| 1 | `createOrder`, `updateOrderStatus`, `order`, `orders(filter)` with status + date range | [§5](#5-graphql-api-gqlgen), [§3](#3-sqlc-setup--queries) |
| 1 | PostgreSQL + **sqlc** + **golang-migrate** | [§2](#2-database-design--migrations), [§3](#3-sqlc-setup--queries) |
| 1 | Order status flow + validate transitions | [§2](#2-database-design--migrations), [§4](#4-domain-logic-state-machine) |
| 2 | Publish on create; worker simulates notification/analytics; local SQS stand-in | [§6](#6-async-processing-queue--worker) |
| 3 | Dockerfile + `docker-compose` one command | [§7](#7-docker--docker-compose) |
| 3 | Terraform module (ECS, SQS, RDS) + comments | [§8](#8-terraform-module) |
| 3 | CI: tests, sqlc generate, Docker build | [§9](#9-ci-pipeline-github-actions) |
| 4 | Unit tests (transitions) + ≥1 GraphQL integration test | [§4](#4-domain-logic-state-machine), [§11](#11-testing) |
| 4 | Structured logging + basic metrics (count, latency) | [§10](#10-observability) |
| — | README: setup, architecture, trade-offs, more time | [§12](#12-readme) |

---

## Definition of done

- [ ] `docker-compose up` works
- [ ] API is accessible
- [ ] GraphQL operations work end-to-end
- [ ] State transitions validated
- [ ] Worker processes events
- [ ] Tests pass
- [ ] CI passes
- [ ] README is clear and complete
