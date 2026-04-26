# Targets for the backend Go service.
#
# Install golangci-lint once with:
#     go install github.com/golangci/golangci-lint/v2/cmd/golangci-lint@latest

.PHONY: run test lint build tidy vet clean

run:
	go run ./cmd/server

test:
	go test -race ./...

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
