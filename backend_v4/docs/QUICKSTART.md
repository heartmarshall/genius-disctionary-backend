# Quick Start

## Prerequisites

- Go 1.24+
- Docker & Docker Compose
- Make

## Setup

```bash
# 1. Clone and navigate
cd backend_v4

# 2. Copy env template and fill in secrets
cp .env.example .env
# Edit .env: set DATABASE_DSN, AUTH_JWT_SECRET (min 32 chars)

# 3. Start PostgreSQL + run migrations
make docker-up
make migrate-up

# 4. Run the server
make run
```

## Verify

- **Liveness**: `curl http://localhost:8080/live` — should return `200 OK`
- **Readiness**: `curl http://localhost:8080/ready` — should return `200` if DB is connected
- **GraphQL Playground** (if enabled in config): open `http://localhost:8080/query` in browser

## Common Commands

```bash
make test               # Unit tests (race detector, no cache)
make test-e2e           # E2E tests (requires Docker)
make generate           # Regenerate sqlc + gqlgen code
make lint               # Run golangci-lint
make seed               # Populate reference catalog from datasets
```

## Troubleshooting

| Problem | Fix |
|---------|-----|
| `connection refused` on DB | Check `make docker-up` ran and container is healthy |
| `JWT secret too short` | Set `AUTH_JWT_SECRET` to 32+ characters in `.env` |
| `migration failed` | Run `make migrate-down` then `make migrate-up` |
| Code gen errors after schema change | Run `make generate` |
