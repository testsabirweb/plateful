.PHONY: build test run-api run-worker migrate-up migrate-down

# golang-migrate CLI via go run (postgres driver). Requires DATABASE_URL, e.g.:
#   postgres://user:pass@localhost:5432/dbname?sslmode=disable
MIGRATE := go run -tags postgres github.com/golang-migrate/migrate/v4/cmd/migrate@v4.18.3

migrate-up:
	@test -n "$(DATABASE_URL)" || (echo "DATABASE_URL is required" && false)
	$(MIGRATE) -path db/migrations -database "$(DATABASE_URL)" up

migrate-down:
	@test -n "$(DATABASE_URL)" || (echo "DATABASE_URL is required" && false)
	$(MIGRATE) -path db/migrations -database "$(DATABASE_URL)" down 1

build:
	go build -o bin/api ./cmd/api
	go build -o bin/worker ./cmd/worker

test:
	go test ./...

run-api:
	go run ./cmd/api

run-worker:
	go run ./cmd/worker
