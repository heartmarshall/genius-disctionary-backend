# MyEnglish Backend v4 — Makefile
# Run `make` or `make help` to see available targets.

.DEFAULT_GOAL := help

# ---------------------------------------------------------------------------
# Variables (override via environment)
# ---------------------------------------------------------------------------
DATABASE_DSN ?= postgres://myenglish:myenglish@localhost:5432/myenglish?sslmode=disable

# ---------------------------------------------------------------------------
# Targets
# ---------------------------------------------------------------------------

##@ Build

build: ## Build the binary
	go build -o bin/server ./cmd/server/

run: ## Run the server
	go run ./cmd/server/

##@ Tests

test: ## Run all unit tests (with race detector, no cache)
	go test ./... -race -count=1

test-cover: ## Run tests with coverage report (opens HTML)
	go test ./... -coverprofile=coverage.out && go tool cover -html=coverage.out

test-integration: ## Run integration tests (require Docker)
	go test ./... -race -count=1 -tags=integration

##@ Generation

generate: ## Generate code (sqlc + go generate)
	@find internal/adapter/postgres -name "sqlc.yaml" -execdir go tool sqlc generate \;
	go generate ./...

##@ Migrations

migrate-up: ## Apply all pending migrations
	go tool goose -dir migrations/ postgres "$$DATABASE_DSN" up

migrate-down: ## Rollback the last migration
	go tool goose -dir migrations/ postgres "$$DATABASE_DSN" down

migrate-status: ## Show migration status
	go tool goose -dir migrations/ postgres "$$DATABASE_DSN" status

migrate-create: ## Create a new migration (usage: make migrate-create name=<name>)
	go tool goose -dir migrations/ create $(name) sql

##@ Docker

docker-up: ## Start dev environment (docker compose)
	docker compose up -d

docker-down: ## Stop dev environment
	docker compose down

docker-logs: ## Follow dev environment logs
	docker compose logs -f

##@ Quality

lint: ## Run linter (golangci-lint)
	golangci-lint run

##@ Help

help: ## Show this help message
	@awk 'BEGIN {FS = ":.*##"; printf "\nUsage:\n  make \033[36m<target>\033[0m\n"} /^[a-zA-Z_-]+:.*?##/ { printf "  \033[36m%-20s\033[0m %s\n", $$1, $$2 } /^##@/ { printf "\n\033[1m%s\033[0m\n", substr($$0, 5) }' $(MAKEFILE_LIST)

# ---------------------------------------------------------------------------
# Phony
# ---------------------------------------------------------------------------
.PHONY: build run test test-cover test-integration generate \
        migrate-up migrate-down migrate-status migrate-create \
        docker-up docker-down docker-logs lint help
