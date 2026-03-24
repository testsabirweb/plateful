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

### Research notes (`deep-research-report.md`)

[`deep-research-report.md`](deep-research-report.md) is **supplementary** (tooling rationale, citations, alternative DDL sketches). **`plan.md` is the canonical checklist** for this repo. Where they differ, what is implemented in the tree wins; the table below records deliberate choices.

| Topic | Locked in this repo |
|-------|---------------------|
| HTTP / GraphQL | `HTTP_ADDR` (default `:8080`); GraphQL POST **`/query`**; Playground **`/`** |
| Metrics (§10) | **`/metrics`** when Prometheus is added |
| Postgres | **`postgres:16-alpine`** in Compose |
| Order IDs | **UUID** (`gen_random_uuid()` on PG 16+) |
| GraphQL `Time` | **`time.Time`** via gqlgen `graphql.Time` |
| AWS / LocalStack | **`us-east-1`** default in config for SDK |

| Research suggestion | This repo | Notes |
|---------------------|-----------|--------|
| Top-level `graph/` + `generated/` | `internal/graph/` (`generated.go`, `model/`, resolvers) | Same pattern; `internal/` keeps app packages import-private |
| `internal/store/sqlc/` or `db/sqlc/` | `internal/store/db` (package `storedb`) | Driven by `sqlc.yaml` `out` |
| `memory` queue + SQS | `Publisher` + **`SQSClient`** + **`NoOpPublisher`** | In-memory **channel** queue not added yet—optional for tests without LocalStack |
| `docker/localstack/ready.d/*.sh` | **Not used** | Queue ensured in Go (`ensureQueueURL`)—avoid duplicate init paths |
| `internal/observability/*` | **`http.go`** — Prometheus + HTTP middleware | `slog` still in `cmd/*`; metrics in §10 |
| `internal/orders/service.go` | **Not used** | Resolvers call `store` + `orders` directly—fine for scope; extract service if logic grows |

**Makefile / commands:** targets are all intentional—there is no stray `make` entry. `migrate-down` rolls back **one** migration (occasional use). `make generate` runs **sqlc + gqlgen** together so CI matches local dev.

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

### Target layout (actual)

```
cmd/
  api/
  worker/
internal/
  config/
  graph/          # schema.graphqls, gqlgen generated, resolvers, mappers
  orders/         # status + transitions
  queue/          # Publisher, SQS, events
  store/          # repository; store/db = sqlc output
db/
  migrations/
  query/
infra/
  terraform/
gqlgen.yml
sqlc.yaml
```

(`internal/observability/` appears in older sketches; we **do not** keep an empty package—§10 adds metrics/logging helpers when implemented.)

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

- [x] Define type and constants (`internal/orders/status.go`)

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

- [x] Implement transition map: `map[Status][]Status`
- [x] Implement:
  - `CanTransition(from, to)`
  - `ValidateTransition(from, to)`
  - Helpers: `ParseStatus`, `IsKnown`, `IsTerminal`

### Rules

- `delivered` → terminal
- `cancelled` → terminal
- No backward transitions
- Cancellation allowed before delivery

### Tests

- [x] Table-driven unit tests covering:
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

- [x] Initialize gqlgen (`gqlgen.yml`, `github.com/99designs/gqlgen`)
- [x] Define `internal/graph/schema.graphqls`
- [x] Run `go run github.com/99designs/gqlgen generate` (also `make generate`)
- [x] HTTP + playground: `cmd/api` serves `/` (GraphQL Playground) and `/query`; requires `DATABASE_URL`

### Resolver design

- Resolvers call `internal/store` + `orders.ValidateTransition` for status changes; mapping in `internal/graph/mappers.go`

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

- [x] On `createOrder`: insert into DB, then publish event to queue (`internal/queue`, `SQS_ENDPOINT` or no-op)

### TODO — worker

- [x] Poll queue (`internal/queue.SQSClient.ReceiveLoop` → long-poll SQS)
- [x] Process message: simulated action — log “simulated notification” + atomic **events_processed** counter
- [x] Delete message after success

### Implementation options

| Option | Notes |
|--------|--------|
| **Preferred** | LocalStack (SQS) — `SQS_ENDPOINT` (e.g. `http://localhost:4566`), `SQS_QUEUE_NAME` |
| **Fallback** | API uses `NoOpPublisher` when `SQS_ENDPOINT` unset |

---

## 7. Docker & Docker Compose

### Services

- `api`
- `worker`
- `postgres`
- `localstack` (SQS)

### TODO

- [x] Dockerfile: multi-stage build, targets `api` and `worker` (`Dockerfile`)
- [x] `docker-compose.yml` — **postgres**, **localstack**, **migrate** (one-shot `golang-migrate` image), **api**, **worker**; `docker compose up --build -d` brings up the stack (migrations run before api/worker)
- [x] Single-command local stack (see `docker compose up --build`)

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

- [x] Create module: `infra/terraform/modules/catering-service/`
- [x] Variables: subnets, SGs, container image, DB password, ECS IAM role ARNs (see `variables.tf`)
- [x] Outputs: `queue_url`, `ecs_service_name`, `rds_endpoint`, etc. (`outputs.tf`)
- [x] Comments in `main.tf` on omissions (ALB, IAM details, etc.)
- [x] `ci.auto.tfvars` — dummy values so `terraform validate` succeeds in CI (do not use for real apply)

---

## 9. CI pipeline (GitHub Actions)

### TODO

- [x] Create `.github/workflows/ci.yml`

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

- [x] `slog` (stderr) in `cmd/api`, `cmd/worker`
- [x] HTTP access logs via `internal/observability.HTTPMiddleware` (method, path, status, duration_ms)
- [x] Worker logs include `component=worker`

### Metrics

- [x] `http_requests_total`, `http_request_duration_seconds` (Prometheus)
- [x] `GET /metrics` (`promhttp`)
- Assignment: instrument metrics; **no** full Grafana setup required here (see [bonus](#13-bonus-optional) for Grafana/Prometheus add-on).

---

## 11. Testing

### Unit tests

- [x] State transition logic (`internal/orders`)

### Integration test

Assignment: **at least one** integration test for the GraphQL API (plus unit tests for state-transition logic — see [section 4](#4-domain-logic-state-machine)).

- [x] Postgres via **Testcontainers** (`postgres:16-alpine`)
- [x] **golang-migrate** applied from `db/migrations`
- [x] gqlgen handler in **httptest** — `createOrder`, then `order(id)` query

### TODO

- [x] Minimal integration test (`internal/graph/integration_test.go`); skipped when `go test -short`
- [x] `go test ./...` (with Docker) and `go test -short ./...` (without integration)

---

## 12. README

### Must include

- [x] Setup instructions (`README.md`)
- [x] How to run: `docker compose up --build`
- [x] How to test: `go test ./...`
- [x] Architecture explanation
- [x] Design decisions and trade-offs

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

- [x] `docker compose up --build` starts Postgres, LocalStack, migrate, api, worker
- [x] GraphQL integration test (Testcontainers) + unit tests
- [x] GraphQL operations work end-to-end (integration test + manual compose)
- [x] State transitions validated (domain tests)
- [x] Worker processes events (with LocalStack + SQS in compose)
- [x] `go test ./...` passes (Docker for integration); `go test -short` without integration
- [x] CI workflow exists (Go + codegen drift check + Docker build + Terraform fmt/validate)
- [x] README is clear and complete
