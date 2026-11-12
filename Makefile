# Veyro backend developer commands.
# Run from the backend/ directory. Configuration is read from the environment;
# .env files are not loaded automatically.

GO ?= go
APP_NAME ?= veyro-backend
DB_URL ?= postgres://postgres:postgres@localhost:5432/veyro?sslmode=disable
MIGRATE ?= migrate

.PHONY: all build api worker test test-race test-integration fmt vet lint run-api run-worker migrate-up migrate-down migrate-version sqlc clean help

all: build

build: api worker

api:
	$(GO) build -o bin/$(APP_NAME)-api ./cmd/api

worker:
	$(GO) build -o bin/$(APP_NAME)-worker ./cmd/worker

test:
	$(GO) test ./...

# Integration tests require an isolated, migrated database.
test-integration:
	@test -n "$(TEST_DATABASE_URL)" || (echo "Set TEST_DATABASE_URL to an isolated test database" && exit 1)
	TEST_DATABASE_URL="$(TEST_DATABASE_URL)" $(GO) test ./... -count=1

test-race:
	$(GO) test -race ./...

fmt:
	gofmt -w ./cmd ./internal

vet:
	$(GO) vet ./...

lint:
	@if command -v staticcheck >/dev/null 2>&1; then staticcheck ./...; else echo "staticcheck not installed, skipping"; fi

run-api:
	$(GO) run ./cmd/api

run-worker:
	$(GO) run ./cmd/worker

migrate-up:
	$(MIGRATE) -path db/migrations -database "$(DB_URL)" up

migrate-down:
	$(MIGRATE) -path db/migrations -database "$(DB_URL)" down 1

migrate-version:
	$(MIGRATE) -path db/migrations -database "$(DB_URL)" version

sqlc:
	@if command -v sqlc >/dev/null 2>&1; then cd db && sqlc generate; else echo "sqlc not installed"; fi

clean:
	rm -rf bin coverage.out

help:
	@echo "build, api, worker, test, test-integration, test-race, fmt, vet, lint,"
	@echo "run-api, run-worker, migrate-up, migrate-down, migrate-version, sqlc, clean"
