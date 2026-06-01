# Aurganize Work — Backend Makefile
# All commands are run from the project root (backend/).

SHELL := /bin/bash
.DEFAULT_GOAL := help

# Load .env if present so commands inherit DATABASE_URL etc.
ifneq (,$(wildcard .env))
	include .env
	export
endif

# === Help ===
.PHONY: help
help: ## Show this help
	@awk 'BEGIN {FS = ":.*##"; printf "Targets:\n"} \
		/^[a-zA-Z_-]+:.*?##/ {printf "  \033[36m%-22s\033[0m %s\n", $$1, $$2}' \
		$(MAKEFILE_LIST)

# === Local infra ===
.PHONY: up
up: ## Start local Postgres (dev + test)
	docker compose up -d
	@echo "Waiting for Postgres to be ready..."
	@until docker exec aurganize-postgres pg_isready -U aurganize > /dev/null 2>&1; \
	  do sleep 1; done
	@echo "Postgres is ready."

.PHONY: down
down: ## Stop local Postgres
	docker compose down

.PHONY: nuke
nuke: ## Stop Postgres and DESTROY all local data (dev DB only; test DB is already ephemeral)
	docker compose down -v
	rm -rf .postgres-data

# === Tooling ===
.PHONY: tools
tools: ## Install Go-based CLI tools (sqlc, goose, golangci-lint, govulncheck)
	go install github.com/sqlc-dev/sqlc/cmd/sqlc@latest
	go install github.com/pressly/goose/v3/cmd/goose@latest
	go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
	go install golang.org/x/vuln/cmd/govulncheck@latest

# === Database ===

# === Database roles ===
.PHONY: setup-roles
setup-roles: ## Create/refresh aurganize_app and aurganize_auth roles
	bash scripts/setup_db_roles.sh


# === Database (migrations) ===
.PHONY: migrate-up
migrate-up: ## Apply all pending migrations to dev DB
	goose -dir migrations postgres "$(DATABASE_OWNER_URL)" up

.PHONY: migrate-down
migrate-down: ## Roll back the last migration on dev DB
	goose -dir migrations postgres "$(DATABASE_OWNER_URL)" down

.PHONY: migrate-status
migrate-status: ## Show migration status on dev DB
	goose -dir migrations postgres "$(DATABASE_OWNER_URL)" status

.PHONY: migrate-create
migrate-create: ## Create a new migration. Usage: make migrate-create name=add_clients_table
	@if [ -z "$(name)" ]; then echo "Usage: make migrate-create name=<snake_case_name>"; exit 1; fi
	goose -dir migrations -s create $(name) sql

# === Test Database ===
.PHONY: test-migrate-up
test-migrate-up: ## Apply all pending migrations to dev DB
	goose -dir migrations postgres "$(DATABASE_URL_TEST)" up

.PHONY: test-migrate-down
test-migrate-down: ## Roll back the last migration on dev DB
	goose -dir migrations postgres "$(DATABASE_URL_TEST)" down

.PHONY: test-migrate-status
test-migrate-status: ## Show migration status on dev DB
	goose -dir migrations postgres "$(DATABASE_URL_TEST)" status

# === Codegen ===
.PHONY: sqlc
sqlc: ## Regenerate sqlc Go code from SQL queries
	sqlc generate

# === App ===
.PHONY: run
run: ## Run the API server (live config from .env)
	go run ./cmd/api

.PHONY: build
build: ## Build all binaries into ./bin
	mkdir -p bin
	go build -o bin/api ./cmd/api

# === Quality ===
.PHONY: fmt
fmt: ## Format Go code
	gofmt -s -w .

.PHONY: lint
lint: ## Lint Go code
	golangci-lint run ./...

.PHONY: vet
vet: ## Run go vet
	go vet ./...

.PHONY: vuln
vuln: ## Scan dependencies for known vulnerabilities
	govulncheck ./...

.PHONY: test
test: ## Run all tests (unit + integration). Requires `make up`.
	DATABASE_URL_TEST=postgres://aurganize:aurganize@localhost:5433/aurganize_test?sslmode=disable \
	go test -race -count=1 ./...

.PHONY: test-unit
test-unit: ## Run unit tests only (no DB)
	go test -race -count=1 -short ./...

# === Convenience ===
.PHONY: psql
psql: ## Open psql to dev DB
	psql "$(DATABASE_URL)"

.PHONY: psql-test
psql-test: ## Open psql to test DB
	psql "postgres://aurganize:aurganize@localhost:5433/aurganize_test?sslmode=disable"

.PHONY: psql-app
psql-app: ## Open psql as the application role (RLS-respecting)
	psql "$(DATABASE_URL)"

.PHONY: psql-auth
psql-auth: ## Open psql as the auth role (read-only, BYPASSRLS)
	psql "$(DATABASE_AUTH_URL)"