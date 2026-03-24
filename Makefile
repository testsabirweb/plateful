.PHONY: build test run-api run-worker

build:
	go build -o bin/api ./cmd/api
	go build -o bin/worker ./cmd/worker

test:
	go test ./...

run-api:
	go run ./cmd/api

run-worker:
	go run ./cmd/worker
