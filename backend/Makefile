# Targets for the backend Go service.
#
# Install golangci-lint once with:
#     go install github.com/golangci/golangci-lint/v2/cmd/golangci-lint@latest

.PHONY: run test test-integration lint build tidy vet clean \
        db-up db-down db-status db-shell db-check db-clean db-bootstrap

# ---------- application ----------

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

# ---------- local Postgres (native systemd) ----------
#
# These targets assume the Ubuntu/Debian native install:
#     sudo apt install postgresql
#     sudo -u postgres psql -c "ALTER USER postgres PASSWORD 'postgres';"
#
# If you use Docker instead, ignore these and manage the container with
# docker start/stop screentime-pg.

PG_HOST     ?= localhost
PG_PORT     ?= 5432
PG_USER     ?= postgres
PG_PASSWORD ?= postgres
PG_ADMIN_DB ?= postgres

PSQL_ADMIN := PGPASSWORD=$(PG_PASSWORD) psql -h $(PG_HOST) -p $(PG_PORT) -U $(PG_USER) -d $(PG_ADMIN_DB)

# Start the system Postgres service.
db-up:
	sudo systemctl start postgresql

# Stop the system Postgres service.
db-down:
	sudo systemctl stop postgresql

db-status:
	systemctl status postgresql --no-pager

# Quick reachability check; non-zero exit if Postgres is down.
db-check:
	pg_isready -h $(PG_HOST) -p $(PG_PORT) -U $(PG_USER)

# Open a psql shell as the test user.
db-shell:
	$(PSQL_ADMIN)

# Drop all pgtestdb-created databases (templates and any leaked
# instances). Safe to run any time; the next test run rebuilds the
# template from migrations.
db-clean:
	@$(PSQL_ADMIN) -t -A -c "SELECT datname FROM pg_database WHERE datname LIKE 'testdb_tpl_%'" | \
	while read -r db; do \
	    [ -z "$$db" ] && continue; \
	    echo "drop $$db"; \
	    $(PSQL_ADMIN) -c "UPDATE pg_database SET datistemplate = false WHERE datname = '$$db'" >/dev/null; \
	    $(PSQL_ADMIN) -c "DROP DATABASE IF EXISTS \"$$db\"" >/dev/null; \
	done

# One-shot first-time setup: ensure the postgres role has the password
# our tests expect. Idempotent.
db-bootstrap:
	sudo -u postgres psql -c "ALTER USER postgres PASSWORD '$(PG_PASSWORD)';"
	@echo "ok — try: make db-check"
