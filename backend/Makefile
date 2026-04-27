# Targets for the backend Go service.
#
# Install golangci-lint once with:
#     go install github.com/golangci/golangci-lint/v2/cmd/golangci-lint@latest

.PHONY: run test test-integration lint build tidy vet clean

run:
	go run ./cmd/server

# Unit tests only — no Postgres required.
test:
	go test -race ./...

# Integration tests against a real Postgres. Configure connection via
# PG_TEST_HOST/PORT/USER/PASSWORD (defaults: localhost:5432 postgres/postgres).
# pgtestdb creates a fresh DB per test from a cached, fully-migrated template.
test-integration:
	go test -race -tags=integration ./...

vet:
	go vet ./...

lint:
	golangci-lint run

build:
	go build -o bin/server ./cmd/server

tidy:
	go mod tidy

clean:
	rm -rf bin
