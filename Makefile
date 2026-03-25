.PHONY: build test run-api run-worker migrate-up migrate-down db-up compose-down migrate-local generate

# golang-migrate CLI (version aligned with go.mod github.com/golang-migrate/migrate/v4).
MIGRATE := go run -tags postgres github.com/golang-migrate/migrate/v4/cmd/migrate@v4.19.1

SQLC := go run github.com/sqlc-dev/sqlc/cmd/sqlc@v1.29.0
GQLGEN := go run github.com/99designs/gqlgen@v0.17.81

generate:
	$(SQLC) generate
	$(GQLGEN) generate

# Matches docker-compose.yml (postgres service). Use after: make db-up
DATABASE_URL_LOCAL ?= postgres://plateful:plateful@127.0.0.1:5432/plateful?sslmode=disable

# Start local Postgres only. Then: make migrate-local  OR  export DATABASE_URL && make migrate-up
db-up:
	docker compose up -d postgres
	@echo "Waiting for Postgres..."
	@until docker compose exec -T postgres pg_isready -U plateful -d plateful >/dev/null 2>&1; do sleep 1; done
	@echo "Ready. Example: export DATABASE_URL='$(DATABASE_URL_LOCAL)' && make migrate-up"

# Stops all compose services (postgres, localstack, api, worker, etc.)
compose-down:
	docker compose down

# Apply migrations using DATABASE_URL_LOCAL (no export needed)
migrate-local:
	$(MIGRATE) -path db/migrations -database "$(DATABASE_URL_LOCAL)" up

migrate-up:
	@test -n "$(DATABASE_URL)" || (echo "DATABASE_URL is required (or run: make db-up && make migrate-local)" && false)
	$(MIGRATE) -path db/migrations -database "$(DATABASE_URL)" up

migrate-down:
	@test -n "$(DATABASE_URL)" || (echo "DATABASE_URL is required" && false)
	$(MIGRATE) -path db/migrations -database "$(DATABASE_URL)" down 1

build:
	go build -o bin/api ./cmd/api
	go build -o bin/worker ./cmd/worker

test:
	go vet ./...
	go test ./... -count=1 -timeout=15m

run-api:
	go run ./cmd/api

run-worker:
	go run ./cmd/worker
