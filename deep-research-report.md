# Executive summary

> **Repo note:** This file is **research notes** (citations, alternatives, a draft tree). The **canonical checklist and structure** for the Plateful codebase live in **`plan.md`**; if something disagrees with the repo, trust **`plan.md`** and the code.

It summarizes the same assignment as a complete, actionable implementation plan and TODO checklist for the “Golang + DevOps Take‑Home Assignment” described in the prompt (see **`plan.md`** in-repo for the live checklist). It prescribes a schema‑first GraphQL API built with `gqlgen` (code generation + type safety) citeturn0search0turn9search7, backed by PostgreSQL with type‑safe query generation via `sqlc` citeturn2search0turn2search1 and schema migrations via `golang-migrate` citeturn7view0turn7view1. It includes an asynchronous “SQS‑like” event flow using a Queue abstraction with two implementations: LocalStack SQS for realistic local infrastructure citeturn0search3turn6search6 and an in‑memory Go channel for fast tests. It also covers Docker packaging with a single `docker-compose up` entrypoint citeturn9search2turn9search18, Terraform module skeleton resources for ECS + SQS + RDS citeturn1search1turn4search0turn1search2turn4search1, a GitHub Actions CI pipeline citeturn3search3turn3search15, and required testing + observability (structured logs using `log/slog` citeturn3search0, Prometheus metrics with `promhttp` citeturn1search3turn8search2turn8search6, plus a Testcontainers integration test for GraphQL + Postgres citeturn3search14turn8search0).  

Where the prompt does not specify a concrete value (ports, instance sizes, AWS account IDs, etc.), this plan explicitly marks those as **UNSPECIFIED** and recommends parameterizing them instead of hardcoding.

## Scope & assumptions

**In scope (required deliverables)**  
You will implement:

- A Go GraphQL API with gqlgen (schema-first, generated server “boring bits”). citeturn0search0turn9search7  
- PostgreSQL persistence, with `sqlc` code generation configured via `sqlc.yaml`. citeturn2search0turn2search1  
- DB migrations using `golang-migrate` with `*.up.sql` and `*.down.sql` migration pairs and CLI usage. citeturn7view0turn7view1  
- Order status flow and **validated state transitions**.
- Async processing: publish on order creation; consume with a worker; demonstrate SQS pattern (LocalStack SQS preferred). citeturn0search3turn6search4  
- Dockerfile + docker-compose that starts API + Postgres + queue in one command; use Compose spec features as needed. citeturn9search2turn9search18  
- Terraform module skeleton for ECS service + SQS queue + RDS Postgres, using canonical AWS provider resources. citeturn1search1turn4search0turn1search2turn4search1  
- Basic CI (GitHub Actions) running tests, regeneration steps (sqlc/gqlgen), and Docker build. citeturn3search3turn3search15  
- Tests: unit tests for transition logic; at least one GraphQL integration test (with real Postgres). Testcontainers-go recommended. citeturn3search14turn8search0  
- Observability: structured logs (`slog`) and basic Prometheus metrics (request count/latency). citeturn3search0turn1search3turn8search2  

**Out of scope (explicitly not required by prompt)**  
No need to fully productionize: authn/authz, multi-tenant access control, robust retry semantics, DLQs, outbox pattern, tracing, full dashboards (Grafana) unless you do the bonus.

**Assumptions you must confirm/choose (prompt leaves unspecified)**  
Use the table below to record final decisions in your repo (README + env defaults). Treat all of these as **UNSPECIFIED** until you decide.

| Item | Status | Notes |
|---|---|---|
| API HTTP port | **UNSPECIFIED** | Parameterize via env (e.g., `HTTP_PORT`). |
| GraphQL endpoint path | **UNSPECIFIED** | Common: `/query` or `/graphql` (choose and document). |
| Metrics endpoint path | **UNSPECIFIED** | Common: `/metrics` per Prometheus examples. citeturn8search2 |
| Postgres version | **UNSPECIFIED** | Choose a docker image tag and document it. |
| Order ID type | **UNSPECIFIED** | Recommended: UUID (portable) or BIGSERIAL (simple). Decide and lock. |
| Timestamp scalar in GraphQL | **UNSPECIFIED** | Recommended: custom `scalar Time` mapped to Go `time.Time`. |
| AWS region (LocalStack + Terraform examples) | **UNSPECIFIED** | Common dev default: `us-east-1` (but do not assume in code). |
| ECS launch type + networking (Fargate/EC2, subnets, SGs) | **UNSPECIFIED** | Terraform module should accept these as variables. |
| RDS size/class/storage/backups | **UNSPECIFIED** | Terraform module should accept these as variables. |

## Repo layout & exact deliverables

**Repository file tree (target state)**  
This file tree is designed to keep generated code isolated and keep domain logic testable in isolation.

```text
.
├── cmd/
│   ├── api/
│   │   └── main.go
│   └── worker/
│       └── main.go
├── graph/
│   ├── schema.graphqls
│   ├── resolver.go
│   ├── generated/                # gqlgen runtime code output
│   └── model/                    # gqlgen models (optional; can use internal/domain instead)
├── internal/
│   ├── config/
│   │   └── config.go
│   ├── db/
│   │   ├── pool.go               # pgxpool init
│   │   └── migrate.go            # apply migrations (optional helper)
│   ├── store/
│   │   ├── sqlc/                 # sqlc generated package output (or put under db/sqlc)
│   │   ├── store.go              # interface wrapper around sqlc Queries
│   │   └── errors.go
│   ├── orders/
│   │   ├── status.go             # Status enum + transitions
│   │   ├── service.go            # order service (create/update/query/list)
│   │   └── errors.go             # domain errors (invalid transition, not found, conflict)
│   ├── queue/
│   │   ├── queue.go              # interface + message types
│   │   ├── sqs.go                # SQS impl (LocalStack for dev)
│   │   └── memory.go             # in-memory impl (tests/local fallback)
│   └── observability/
│       ├── logging.go            # slog setup
│       ├── metrics.go            # prometheus metrics registry + instruments
│       └── http_middleware.go    # request logging + metrics middleware
├── db/
│   ├── migrations/
│   │   ├── 000001_init_orders.up.sql
│   │   └── 000001_init_orders.down.sql
│   └── query/
│       └── orders.sql
├── infra/
│   └── terraform/
│       └── modules/
│           └── catering_service/
│               ├── main.tf
│               ├── variables.tf
│               ├── outputs.tf
│               └── README.md      # optional but recommended
├── .github/
│   └── workflows/
│       └── ci.yml
├── docker/
│   └── localstack/
│       └── ready.d/
│           └── 01-create-queues.sh
├── Dockerfile
├── docker-compose.yml
├── gqlgen.yml
├── sqlc.yaml
├── go.mod
├── go.sum
└── README.md
```

**Exact deliverables checklist (what must exist and work)**

| Deliverable | “Done” criteria |
|---|---|
| GraphQL API service | `docker-compose up` starts it; GraphQL endpoint responds; implements required queries/mutations. |
| Worker service | Runs, consumes order-created events, performs simulated action (log + counter), deletes message. |
| Postgres schema + migrations | `golang-migrate` compatible `up/down` migrations exist and can be applied. citeturn7view0turn7view1 |
| sqlc generated code | `sqlc generate` runs from repo root, producing type-safe code. citeturn2search0turn2search1 |
| docker-compose | One command launches Postgres + queue + API + worker; uses Compose spec. citeturn9search2turn9search18 |
| Dockerfile | Multi-stage build(s) producing minimal runtime images. citeturn4search2 |
| Terraform module skeleton | Defines `aws_sqs_queue`, `aws_ecs_task_definition/service`, `aws_db_instance` placeholders with comments. citeturn4search0turn4search1turn1search1turn1search2 |
| CI pipeline | Runs `go test ./...`, `sqlc generate`, `gqlgen generate`, and `docker build`. citeturn3search3turn3search15 |
| Tests | Unit tests for state machine; 1+ GraphQL integration test using real Postgres (Testcontainers). citeturn3search14turn8search0 |
| Observability | Structured logs (slog) + basic Prometheus metrics (request count/latency). citeturn3search0turn8search2turn8search6 |
| README.md | Setup instructions, architecture decisions, tradeoffs, “more time” section, sample GraphQL ops. |

**Actionable TODO list (no timeline; execute in this order to minimize rework)**

| Area | TODO items (mark complete in your repo) |
|---|---|
| Project bootstrap | Create Go module; standardize `make` targets (or scripts) for `generate`, `test`, `lint` (lint optional); wire minimal HTTP server + config loader. |
| DB + migrations | Decide Order ID type; write `init_orders` migration `up/down`; add indexes and constraints; ensure containerized Postgres + migrations apply cleanly via CLI. citeturn7view0turn7view1 |
| sqlc | Create `db/query/orders.sql` (insert/get/update/list); author `sqlc.yaml`; run `sqlc generate`; wrap generated `Queries` behind a small Store interface. citeturn2search0turn2search1turn0search1 |
| Domain/state machine | Implement `Status` constants and transition function; unit tests that exhaustively cover allowed/disallowed transitions. |
| GraphQL | Write `graph/schema.graphqls`; configure `gqlgen.yml`; run `gqlgen generate`; implement resolvers calling the service layer. citeturn0search0turn2search2 |
| Queue + worker | Build `Queue` interface; implement `memory` queue; implement SQS queue using AWS SDK for Go v2 with configurable endpoints (LocalStack); implement worker consumer loop. citeturn2search3turn6search0turn6search4 |
| Observability | Implement request logging middleware with `slog`; add Prometheus metrics endpoint and counters/histograms; instrument GraphQL handler. citeturn3search0turn8search2turn8search6 |
| Docker & compose | Add Dockerfile multi-stage builds; add docker-compose with Postgres + LocalStack + API + worker; include LocalStack init hook script to create SQS queue(s). citeturn4search2turn6search6turn9search18 |
| CI | Add GitHub Actions workflow using `actions/setup-go`; run generation + tests + docker build. citeturn3search15turn3search3 |
| Terraform | Create module skeleton resources and variables; comment design choices; no need to apply. citeturn4search0turn1search1turn1search2 |
| Tests | Add unit tests (state machine); add integration test (Testcontainers Postgres + migrations + GraphQL HTTP). citeturn3search14turn8search0turn7view1 |
| Docs | Write README with precise commands; include sample GraphQL operations and expected behavior. |

## Database, migrations, and sqlc

**Key design goals**  
- Support required GraphQL operations: `createOrder`, `updateOrderStatus`, `order(id)`, `orders(filter by status and date range)`.  
- Enforce valid statuses at least at the application layer; optionally also validate at DB layer.  
- Keep list queries fast via indexes on `(status, created_at)` or similar.

### Status handling choice (DB)

The prompt requires a fixed status flow. You can enforce validity in the DB in two reasonable ways:

- **Option A (recommended for take-home): `TEXT` + `CHECK` constraint**  
  Pros: easy migrations; no enum-alter complexity.  
  Cons: less “type-safe” in SQL than Postgres ENUM.

- **Option B: Postgres `ENUM` type**  
  Pros: strong DB-level typing.  
  Cons: evolving enums requires migration steps; for a take-home, might add complexity.

This plan uses **Option A** (TEXT + CHECK) so migrations stay simple; state transitions are still validated in Go (required) regardless.

### Orders table DDL (migration `000001_init_orders.up.sql`)

> **UNSPECIFIED:** Whether to use UUID vs BIGSERIAL IDs is not in the prompt. This DDL uses UUID (recommended) but you may switch to BIGSERIAL if you prefer. If you use UUID, you’ll typically enable `pgcrypto` to generate `gen_random_uuid()`.

```sql
-- 000001_init_orders.up.sql

-- If using UUIDs:
CREATE EXTENSION IF NOT EXISTS pgcrypto;

CREATE TABLE IF NOT EXISTS orders (
  id           uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  status       text NOT NULL,
  restaurant   text NOT NULL,
  customer     text NOT NULL,
  notes        text NOT NULL DEFAULT '',
  total_cents  integer NOT NULL DEFAULT 0,
  created_at   timestamptz NOT NULL DEFAULT now(),
  updated_at   timestamptz NOT NULL DEFAULT now()
);

-- Enforce only known statuses at the DB layer (state transitions enforced in Go).
ALTER TABLE orders
  ADD CONSTRAINT orders_status_check
  CHECK (status IN ('pending','confirmed','preparing','ready','delivered','cancelled'));

-- Helpful indexes for the required filter patterns:
CREATE INDEX IF NOT EXISTS orders_created_at_idx ON orders (created_at);
CREATE INDEX IF NOT EXISTS orders_status_created_at_idx ON orders (status, created_at);

-- Optional: keep updated_at current with a trigger.
-- (If you use a trigger, add it here. If not, ensure app sets updated_at on update.)
```

**Rationale for indexes**  
The list query must filter by status and/or created_at range. A composite index `(status, created_at)` supports queries where status is present and a time range is applied; a separate `created_at` index helps time-only filtering. (Exact index choices can be tuned later; this meets scope.)

### Down migration (`000001_init_orders.down.sql`)

```sql
-- 000001_init_orders.down.sql
DROP TABLE IF EXISTS orders;
-- DROP EXTENSION IF EXISTS pgcrypto; -- optional, avoid dropping if shared by other schemas
```

### golang-migrate wiring (CLI + optional in-app)

`golang-migrate/migrate` provides a CLI that reads migrations from a source and applies them in order citeturn7view0; their docs show basic usage like `migrate -source file://... -database ... up` citeturn7view0 and an equivalent pattern using `-path` (as used in their Postgres tutorial) citeturn7view1.

**Recommended repo commands (document in README)**  
Examples (replace values; many are **UNSPECIFIED**):

```bash
# Apply migrations to local Postgres (example)
migrate -path db/migrations -database "$DATABASE_URL" up

# Roll back all (example)
migrate -path db/migrations -database "$DATABASE_URL" down
```

The `migrate` README also documents using the library in Go (import `migrate/v4`, plus driver and source packages) citeturn7view0turn7view1. For integration tests, running migrations programmatically is often more deterministic than shelling out.

### sqlc configuration (`sqlc.yaml`)

`sqlc` is configured via `sqlc.yaml|yml` or `sqlc.json` in the directory you run `sqlc` from citeturn2search0. This plan uses v2 config and `pgx/v5` because sqlc explicitly supports emitting code that uses pgx types (and recommends setting `sql_package` to generate pgx-based code) citeturn5search2turn8search3.

```yaml
# sqlc.yaml
version: "2"
sql:
  - engine: "postgresql"
    schema: "db/migrations"
    queries: "db/query"
    gen:
      go:
        package: "sqlc"
        out: "internal/store/sqlc"
        sql_package: "pgx/v5"
        emit_json_tags: true
        emit_interface: false
        emit_empty_slices: true
```

### sqlc query file (`db/query/orders.sql`)

Use named parameters (`sqlc.arg`) for clarity citeturn2search1. For optional filters, `sqlc.narg()` forces nullable params so you can use patterns like `(@status IS NULL OR status = @status)` citeturn0search1.

```sql
-- db/query/orders.sql

-- name: InsertOrder :one
INSERT INTO orders (restaurant, customer, notes, total_cents, status)
VALUES (
  sqlc.arg(restaurant),
  sqlc.arg(customer),
  COALESCE(sqlc.arg(notes), ''),
  COALESCE(sqlc.arg(total_cents), 0),
  'pending'
)
RETURNING *;

-- name: GetOrder :one
SELECT * FROM orders
WHERE id = sqlc.arg(id);

-- name: UpdateOrderStatusCAS :one
-- Compare-and-set update: only update if current status matches expected.
UPDATE orders
SET status = sqlc.arg(new_status),
    updated_at = now()
WHERE id = sqlc.arg(id)
  AND status = sqlc.arg(expected_status)
RETURNING *;

-- name: ListOrders :many
SELECT * FROM orders
WHERE
  (sqlc.narg(status)::text IS NULL OR status = sqlc.narg(status)::text)
  AND (sqlc.narg(from_time)::timestamptz IS NULL OR created_at >= sqlc.narg(from_time)::timestamptz)
  AND (sqlc.narg(to_time)::timestamptz IS NULL OR created_at <= sqlc.narg(to_time)::timestamptz)
ORDER BY created_at DESC
LIMIT COALESCE(sqlc.narg(limit)::int, 50)
OFFSET COALESCE(sqlc.narg(offset)::int, 0);
```

Notes:
- `sqlc.arg` is a macro that annotates a parameter name and expands to a DB placeholder for your engine citeturn2search1.  
- `sqlc.narg()` forces nullable parameter generation when sqlc’s inference isn’t what you want citeturn0search1.  
- `LIMIT/OFFSET` are optional; not required by the prompt, but if you include them, document them as “extras.”

## Domain logic: state machine, concurrency, and error handling

### Status definitions

Required status flow in prompt:  
`pending > confirmed > preparing > ready > delivered > cancelled`  
Interpretation required: whether “cancelled” is reachable from any non-terminal status (common) or only from specific statuses. The prompt implies cancellation exists and transitions must be validated; this plan allows cancel from any pre-delivered state.

### Allowed transitions (exhaustive)

**Rule set used by this plan (document in README):**
- Happy path: pending → confirmed → preparing → ready → delivered
- Cancellation allowed from: pending/confirmed/preparing/ready
- Terminal states: delivered and cancelled (no outbound transitions)

**Transition matrix (Allowed = ✅, Disallowed = ❌)**

| From \ To | pending | confirmed | preparing | ready | delivered | cancelled |
|---|---:|---:|---:|---:|---:|---:|
| pending | ❌ | ✅ | ❌ | ❌ | ❌ | ✅ |
| confirmed | ❌ | ❌ | ✅ | ❌ | ❌ | ✅ |
| preparing | ❌ | ❌ | ❌ | ✅ | ❌ | ✅ |
| ready | ❌ | ❌ | ❌ | ❌ | ✅ | ✅ |
| delivered | ❌ | ❌ | ❌ | ❌ | ❌ | ❌ |
| cancelled | ❌ | ❌ | ❌ | ❌ | ❌ | ❌ |

### Go implementation (table-driven)

```go
// internal/orders/status.go
package orders

type Status string

const (
	StatusPending   Status = "pending"
	StatusConfirmed Status = "confirmed"
	StatusPreparing Status = "preparing"
	StatusReady     Status = "ready"
	StatusDelivered Status = "delivered"
	StatusCancelled Status = "cancelled"
)

var allowedTransitions = map[Status]map[Status]bool{
	StatusPending: {
		StatusConfirmed: true,
		StatusCancelled: true,
	},
	StatusConfirmed: {
		StatusPreparing: true,
		StatusCancelled: true,
	},
	StatusPreparing: {
		StatusReady:     true,
		StatusCancelled: true,
	},
	StatusReady: {
		StatusDelivered: true,
		StatusCancelled: true,
	},
	StatusDelivered: {},
	StatusCancelled: {},
}

func CanTransition(from, to Status) bool {
	if from == to {
		return false
	}
	next, ok := allowedTransitions[from]
	if !ok {
		return false
	}
	return next[to]
}
```

### Unit tests (exhaustive cases)

```go
// internal/orders/status_test.go
package orders

import "testing"

func TestCanTransition_Allowed(t *testing.T) {
	cases := []struct {
		from Status
		to   Status
	}{
		{StatusPending, StatusConfirmed},
		{StatusConfirmed, StatusPreparing},
		{StatusPreparing, StatusReady},
		{StatusReady, StatusDelivered},

		{StatusPending, StatusCancelled},
		{StatusConfirmed, StatusCancelled},
		{StatusPreparing, StatusCancelled},
		{StatusReady, StatusCancelled},
	}

	for _, tc := range cases {
		if !CanTransition(tc.from, tc.to) {
			t.Fatalf("expected allowed transition %s -> %s", tc.from, tc.to)
		}
	}
}

func TestCanTransition_Disallowed(t *testing.T) {
	cases := []struct {
		from Status
		to   Status
	}{
		// no self transitions
		{StatusPending, StatusPending},

		// regressions and skips
		{StatusDelivered, StatusPending},
		{StatusPending, StatusPreparing},
		{StatusConfirmed, StatusReady},

		// terminal states
		{StatusDelivered, StatusCancelled},
		{StatusCancelled, StatusPending},
		{StatusCancelled, StatusDelivered},
	}

	for _, tc := range cases {
		if CanTransition(tc.from, tc.to) {
			t.Fatalf("expected disallowed transition %s -> %s", tc.from, tc.to)
		}
	}
}
```

### Concurrency and atomic update patterns

**Problem:** Two clients may attempt to update an order concurrently.

**Plan:** Use a compare-and-set update in SQL that checks the current status in the `WHERE` clause, returning the updated row only if the status matched the expected value (`UpdateOrderStatusCAS` query above). This makes the update atomic at the database level and avoids lost updates.

**Service-layer algorithm for `updateOrderStatus(id, newStatus)`**
1. Load order (`GetOrder`) to retrieve current status.
2. Validate `CanTransition(current, newStatus)`; if invalid, return domain error.
3. Execute `UpdateOrderStatusCAS(id, expected=current, new=newStatus)`.
4. If update returns “no rows” (or sqlc returns an error indicating not found), map to a **conflict** error (“order status changed; please retry”).

**Error mapping guidance (GraphQL)**
- Not found → GraphQL error (message “not found”) + optional error extension code (e.g., `NOT_FOUND`).
- Invalid transition → error extension code `INVALID_TRANSITION`.
- Conflict (CAS failed) → error extension code `CONFLICT`.
- Validation errors on input → `BAD_REQUEST`.

## GraphQL API with gqlgen, schema, and mapping notes

### gqlgen configuration

`gqlgen` is schema-first citeturn0search0turn9search7 and is configured via `gqlgen.yml` in the repo root (or parent directories) citeturn2search2turn5search3.

A minimal `gqlgen.yml` (paths may be adjusted):

```yaml
# gqlgen.yml
schema:
  - graph/schema.graphqls

exec:
  filename: graph/generated/generated.go
  package: generated

model:
  filename: graph/model/models_gen.go
  package: model

resolver:
  layout: follow-schema
  dir: graph
  package: graph
```

### GraphQL schema (`graph/schema.graphqls`)

> **UNSPECIFIED:** GraphQL does not define built-in timestamp scalars; define a custom scalar and map it to Go `time.Time`.

```graphql
scalar Time

enum OrderStatus {
  pending
  confirmed
  preparing
  ready
  delivered
  cancelled
}

type Order {
  id: ID!
  status: OrderStatus!
  restaurant: String!
  customer: String!
  notes: String!
  totalCents: Int!
  createdAt: Time!
  updatedAt: Time!
}

input CreateOrderInput {
  restaurant: String!
  customer: String!
  notes: String
  totalCents: Int
}

input OrdersFilter {
  status: OrderStatus
  from: Time
  to: Time
  limit: Int
  offset: Int
}

type Query {
  order(id: ID!): Order
  orders(filter: OrdersFilter): [Order!]!
}

type Mutation {
  createOrder(input: CreateOrderInput!): Order!
  updateOrderStatus(id: ID!, status: OrderStatus!): Order!
}
```

### Go type mapping notes (important to prevent churn)

- **Order ID type:** GraphQL uses `ID!`. Decide whether this is a string uuid in Go (recommended) or an integer. If you use UUID in Postgres, you’ll likely model it as `uuid.UUID` (from `github.com/google/uuid`) or as `string`.  
  - If you use `uuid.UUID`, avoid leaking it into GraphQL models unless you want custom marshaling; simplest is to map DB UUID to string in GraphQL models.
- **Time scalar:** Map schema `Time` to Go `time.Time` and marshal as RFC3339 in gqlgen scalar config (implementation detail; no single required choice).
- **Status enum:** Map GraphQL `OrderStatus` to your domain `orders.Status`. Keep the conversion in one place to avoid repeated switch statements.

### gqlgen server wiring (HTTP)

If you use `handler.NewDefaultServer`, note that the package documentation explicitly says it is only suitable for examples and not recommended for production; `handler.New` with configured transports is the “real” path citeturn9search1. For a take-home, either choice is acceptable as long as you document the trade-off.

If you want a convenient local UI, gqlgen provides playground handlers (GraphiQL / Apollo sandbox handlers, etc.) citeturn8search5.

### Resolver responsibilities (keep resolvers thin)

Resolvers should:
- Validate inputs (e.g., required strings non-empty).
- Call `OrderService` (domain layer) for create/update/read/list.
- Convert domain/store models to GraphQL models.
- Translate domain errors into GraphQL errors with consistent codes.

## Async queue + worker: interface design, LocalStack SQS, and in-memory fallback

### Queue abstraction (interface)

Define a minimal interface with the exact operations you need (publish + consume + ack). Keep it general enough for both SQS and channel.

```go
// internal/queue/queue.go
package queue

import "context"

type Message struct {
	ID      string
	Body    []byte
	Receipt string // used by SQS delete; empty for in-memory
}

type Publisher interface {
	Publish(ctx context.Context, body []byte) error
}

type Consumer interface {
	Receive(ctx context.Context, max int) ([]Message, error)
	Delete(ctx context.Context, msg Message) error
}

type Queue interface {
	Publisher
	Consumer
}
```

### Message format

Use JSON so it’s easy to inspect in LocalStack and logs.

```json
{
  "type": "OrderCreated",
  "orderId": "<string>",
  "createdAt": "<RFC3339 timestamp>"
}
```

> **UNSPECIFIED:** exact field casing; pick one and keep it stable.

### Worker behavior

**Worker loop (required)**
1. Poll queue (long-poll if SQS) for up to N messages.
2. For each message:
   - Parse JSON.
   - If `type == OrderCreated`: perform simulated action:
     - log a structured entry “notification_sent” (or “analytics_incremented”)
     - increment a process counter metric
   - On success: delete/ack the message.
   - On failure: log error; for SQS you can leave it un-deleted so it’s retried after visibility timeout (**do not implement advanced retry logic** in scope).

### LocalStack SQS implementation (recommended for demo)

LocalStack supports SQS emulation locally citeturn0search3 and provides initialization hooks that run scripts at lifecycle stages (including `ready.d` when LocalStack is ready). LocalStack documents the `/etc/localstack/init/<stage>.d` directories and that they can contain executable shell/Python scripts citeturn6search6turn1search0.

**SQS client in Go**  
Use AWS SDK for Go v2 SQS client citeturn2search7turn2search3. Configure custom endpoints for LocalStack (advanced topic, but supported), as described in AWS SDK v2 endpoint configuration docs citeturn6search0. LocalStack also documents AWS SDK (Go) integration and recommends setting endpoints when creating clients citeturn6search4.

Implementation notes:
- Read `AWS_REGION`, `AWS_ACCESS_KEY_ID`, `AWS_SECRET_ACCESS_KEY`, and an endpoint env var like `AWS_ENDPOINT_URL` (**UNSPECIFIED** names are up to you; choose and document).
- For LocalStack, static fake creds are fine (LocalStack accepts them).
- Set SQS endpoint to LocalStack URL; region still required by SDK. citeturn5search4turn6search0

### In-memory queue implementation (fallback + tests)

Use a buffered channel + goroutine-safe wrapper.
- `Publish`: enqueue `Message{ID: ..., Body: ...}`
- `Receive`: drain up to `max` messages non-blocking (or with context-aware wait).
- `Delete`: no-op (or track in-memory acknowledgments).

This is useful in unit tests for the service “publish on create” without starting LocalStack.

## Dockerfile + docker-compose + LocalStack init hooks

### docker-compose services

Docker Compose is configured via the Compose file and spec citeturn9search2, and you can manage startup ordering with `depends_on` plus health checks (recommended) citeturn9search18.

**Services table**

| Service | Purpose | Depends on | Notes |
|---|---|---|---|
| `postgres` | DB storage | — | Expose port **UNSPECIFIED**; use named volume. |
| `localstack` | SQS emulation | — | Use init hooks to create queue(s). citeturn6search6 |
| `api` | GraphQL server | postgres, localstack | Env: DB URL, AWS region/endpoint, queue URL/name. |
| `worker` | SQS consumer | postgres?, localstack | Needs queue URL/name; DB optional unless you write analytics to DB. |

> **UNSPECIFIED:** exact ports, image tags, and container names. Parameterize and document defaults.

### LocalStack init hook to create the queue

Mount a script at `docker/localstack/ready.d/01-create-queues.sh` to `/etc/localstack/init/ready.d/01-create-queues.sh` inside the container. LocalStack will execute scripts in `ready.d` when it becomes ready citeturn6search6turn1search0.

Example script (uses `awslocal`; LocalStack commonly includes it; verify in your chosen image tag):

```bash
#!/usr/bin/env bash
set -euo pipefail

# UNPECIFIED: queue name; parameterize with env if desired.
QUEUE_NAME="${QUEUE_NAME:-orders-created}"

awslocal sqs create-queue --queue-name "$QUEUE_NAME"

# Optional: output the queue URL for debugging
awslocal sqs get-queue-url --queue-name "$QUEUE_NAME"
```

### docker-compose.yml skeleton

```yaml
services:
  postgres:
    image: postgres:<UNSPECIFIED>
    environment:
      POSTGRES_USER: <UNSPECIFIED>
      POSTGRES_PASSWORD: <UNSPECIFIED>
      POSTGRES_DB: <UNSPECIFIED>
    volumes:
      - pgdata:/var/lib/postgresql/data
    healthcheck:
      test: ["CMD-SHELL", "pg_isready -U $$POSTGRES_USER -d $$POSTGRES_DB"]
      interval: 5s
      timeout: 3s
      retries: 20

  localstack:
    image: localstack/localstack:<UNSPECIFIED>
    environment:
      SERVICES: sqs
      AWS_DEFAULT_REGION: <UNSPECIFIED>
      # Other LocalStack envs are UNSPECIFIED; set only what you need.
    volumes:
      - ./docker/localstack/ready.d:/etc/localstack/init/ready.d
      - /var/run/docker.sock:/var/run/docker.sock
    healthcheck:
      test: ["CMD-SHELL", "curl -s http://localhost:4566/_localstack/health | grep -q '\"sqs\"'"]
      interval: 5s
      timeout: 3s
      retries: 30

  api:
    build:
      context: .
      target: api
    environment:
      DATABASE_URL: <UNSPECIFIED>
      AWS_REGION: <UNSPECIFIED>
      AWS_ENDPOINT_URL: <UNSPECIFIED> # e.g., http://localstack:4566
      SQS_QUEUE_NAME: <UNSPECIFIED>
    depends_on:
      postgres:
        condition: service_healthy
      localstack:
        condition: service_healthy

  worker:
    build:
      context: .
      target: worker
    environment:
      AWS_REGION: <UNSPECIFIED>
      AWS_ENDPOINT_URL: <UNSPECIFIED>
      SQS_QUEUE_NAME: <UNSPECIFIED>
    depends_on:
      localstack:
        condition: service_healthy

volumes:
  pgdata:
```

### Dockerfile: multi-stage build notes

Docker multi-stage builds allow you to build binaries in one stage and copy only the final artifacts into a small runtime image citeturn4search2. For this repo, build two binaries: `api` and `worker`.

**Dockerfile example (two targets, `api` and `worker`)**

```dockerfile
# syntax=docker/dockerfile:1

FROM golang:<UNSPECIFIED> AS build
WORKDIR /src

COPY go.mod go.sum ./
RUN go mod download

COPY . .

# Build API
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o /out/api ./cmd/api

# Build Worker
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o /out/worker ./cmd/worker

FROM gcr.io/distroless/static:nonroot AS api
WORKDIR /
COPY --from=build /out/api /api
USER nonroot:nonroot
ENTRYPOINT ["/api"]

FROM gcr.io/distroless/static:nonroot AS worker
WORKDIR /
COPY --from=build /out/worker /worker
USER nonroot:nonroot
ENTRYPOINT ["/worker"]
```

> **UNSPECIFIED:** base images and Go version. Choose and document.

## Terraform module + CI + testing + observability + README

### Terraform module skeleton (does not need to apply)

Use canonical AWS provider resources:
- `aws_sqs_queue` citeturn4search0  
- `aws_ecs_task_definition` citeturn4search1  
- `aws_ecs_service` citeturn1search1  
- `aws_db_instance` citeturn1search2  

Also follow a recommended module structure and variable/outputs pattern (HashiCorp module creation guidance). citeturn9search5

**Module variables table (example; all values remain UNSPECIFIED by prompt)**

| Variable | Type | Why it exists |
|---|---|---|
| `name` | string | Prefix for ECS service, SQS queue, DB identifier. |
| `aws_region` | string | Required by provider; keep explicit. |
| `ecs_cluster_arn` | string | Don’t create a cluster; accept as input. |
| `subnet_ids` | list(string) | ECS networking, RDS subnet group. |
| `security_group_ids` | list(string) | For ECS tasks and/or RDS. |
| `container_image_api` | string | Image URI for API container. |
| `container_image_worker` | string | Image URI for Worker container. |
| `db_instance_class` | string | RDS sizing. |
| `db_allocated_storage` | number | RDS storage. |
| `db_username` | string | Master username. |
| `db_password` | string (sensitive) | Master password. |
| `db_name` | string | Initial database. |
| `sqs_queue_name` | string | Queue naming override. |

**Terraform skeleton (high-level)**  
Keep networking/ALB/IAM intentionally minimal; document what’s missing.

```hcl
# infra/terraform/modules/catering_service/main.tf

resource "aws_sqs_queue" "orders_created" {
  name = var.sqs_queue_name

  # Comments: for prod, consider KMS encryption, DLQ/redrive, retention tuning, FIFO if ordering is required.
}

resource "aws_db_instance" "postgres" {
  identifier = "${var.name}-db"

  engine         = "postgres"
  instance_class = var.db_instance_class

  allocated_storage = var.db_allocated_storage
  db_name           = var.db_name
  username          = var.db_username
  password          = var.db_password

  # Comments: production typically requires subnet group, parameter group, backups, multi-AZ, maintenance windows,
  # deletion protection, storage encryption, and SG rules. Keep as variables/skeleton for take-home.
}

resource "aws_ecs_task_definition" "api" {
  family                   = "${var.name}-api"
  requires_compatibilities = ["FARGATE"] # UNSPECIFIED; choose EC2/Fargate
  network_mode             = "awsvpc"
  cpu                      = "256"  # UNSPECIFIED
  memory                   = "512"  # UNSPECIFIED

  container_definitions = jsonencode([
    {
      name  = "api"
      image = var.container_image_api
      essential = true
      environment = [
        # Provide DATABASE_URL, SQS queue URL/name, region, etc.
      ]
      portMappings = [
        # UNSPECIFIED: container port mapping
      ]
    }
  ])
}

resource "aws_ecs_service" "api" {
  name            = "${var.name}-api"
  cluster         = var.ecs_cluster_arn
  task_definition = aws_ecs_task_definition.api.arn
  desired_count   = 1

  # Comments: for production, add load balancer, autoscaling, deployment circuit breaker, IAM roles, etc.
}
```

### CI pipeline (GitHub Actions)

GitHub’s docs recommend `actions/setup-go` for consistent Go installs on runners citeturn3search3turn3search15.

**CI steps required by prompt**
- `go test ./...`
- `sqlc generate`
- `gqlgen generate`
- `docker build`

Skeleton workflow:

```yaml
name: ci

on:
  push:
  pull_request:

jobs:
  test-build:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4

      - uses: actions/setup-go@v6
        with:
          go-version-file: go.mod

      - name: Install tools
        run: |
          go install github.com/sqlc-dev/sqlc/cmd/sqlc@latest
          go install github.com/99designs/gqlgen@latest

      - name: Generate sqlc
        run: sqlc generate

      - name: Generate gqlgen
        run: gqlgen generate

      - name: Go tests
        run: go test ./...

      - name: Docker build (API image target)
        run: docker build --target api -t catering-api:ci .

      - name: Docker build (Worker image target)
        run: docker build --target worker -t catering-worker:ci .
```

> **UNSPECIFIED:** you may pin versions rather than `@latest`.

### Testing plan (unit + integration)

**Unit tests (required)**
- `internal/orders/status_test.go`: exhaustive transition tests (provided earlier).

**Integration test (required: at least one GraphQL integration)**  
Use Testcontainers for Go to start a Postgres database container in tests citeturn3search14turn8search0, apply migrations (programmatically or via CLI), start the GraphQL server in-process, and perform an HTTP POST GraphQL request.

Testcontainers Postgres module documentation provides a supported approach to starting Postgres dependencies for tests citeturn8search0.

Migration application: golang-migrate docs include a minimal example applying migrations inside Go via `migrate.New(...).Up()` citeturn7view1.

**Integration test structure**
- Start Postgres container.
- Construct `DATABASE_URL` from container details.
- Apply migrations using migrate library.
- Start API server on an ephemeral port.
- Execute GraphQL operations:
  1) `createOrder`
  2) `order(id)` query
  3) `updateOrderStatus` valid transition
  4) `updateOrderStatus` invalid transition (assert error)

### Observability instrumentation

**Structured logging with `log/slog`**  
Go’s `log/slog` package provides structured logging with key-value attributes and leveled log methods citeturn3search0. Use JSON handler for container logs.

**Metrics with Prometheus client**  
Prometheus Go guides recommend exposing a `/metrics` endpoint and using `promhttp.Handler()`/`HandlerFor()` to expose registered metrics citeturn8search2turn1search3. The `promhttp` package also provides HTTP middleware helpers named `InstrumentHandlerX` citeturn8search6.

**Metrics to implement (minimum per prompt)**
- Request count (CounterVec labeled by method/path/status)
- Request latency (HistogramVec labeled similarly)
- Custom counters:
  - `orders_created_total`
  - `orders_status_updates_total`
  - `worker_orders_processed_total`

### README.md content outline (must be explicit and runnable)

Your README must include:

- Project overview + architecture (API, Postgres, queue, worker).
- Setup prerequisites (**UNSPECIFIED**: Go version, Docker version).
- Exact commands:
  - `docker-compose up --build`
  - `go test ./...`
  - `sqlc generate`
  - `gqlgen generate`
  - `migrate -path db/migrations -database "$DATABASE_URL" up` (if used manually) citeturn7view1turn7view0
- Environment variables table (with defaults marked UNSPECIFIED if you don’t choose defaults).
- Sample GraphQL operations (copy/paste):
  - `createOrder`
  - `order(id)`
  - `orders(filter)`
  - `updateOrderStatus`
- Trade-offs:
  - SQS reliability/DLQ omitted
  - Outbox pattern not implemented
  - Auth omitted
- “What I’d add with more time” section:
  - Rate limiter middleware (bonus)
  - Prometheus/Grafana compose extension (bonus)
  - GraphQL dataloaders for batching/N+1 avoidance (bonus; gqlgen has a reference guide) citeturn4search11  

**Suggested sample GraphQL snippets (include in README)**  
(Ensure endpoint path is consistent; **UNSPECIFIED** until you decide.)

```graphql
mutation CreateOrder {
  createOrder(input: {
    restaurant: "Spice Kitchen"
    customer: "Ada Lovelace"
    notes: "No onions"
    totalCents: 2599
  }) {
    id
    status
    createdAt
  }
}

mutation Confirm {
  updateOrderStatus(id: "<ID>", status: confirmed) {
    id
    status
    updatedAt
  }
}

query Filtered {
  orders(filter: { status: confirmed, from: "2026-03-01T00:00:00Z", to: "2026-03-31T23:59:59Z" }) {
    id
    status
    createdAt
  }
}
```

### What to add with more time (explicit list)

- **Rate limiter middleware** (bonus): per-IP token bucket, configurable limits, metrics for throttled requests.
- **Grafana/Prometheus compose extension** (bonus): add `prometheus` scrape config + `grafana` dashboard; keep it opt-in with compose profiles.
- **GraphQL dataloaders** (bonus): implement batching for `orders` lookups or nested fields; gqlgen provides a dataloader reference to reduce N+1 queries citeturn4search11.
- Transactional outbox pattern for “publish after commit” reliability (document-only in take-home; implement if time).