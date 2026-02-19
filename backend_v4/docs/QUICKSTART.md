# Quick Start

## Prerequisites

- Go 1.24+
- Docker & Docker Compose
- `make`

## Setup

```bash
# 1. Clone and enter the backend directory
cd backend_v4

# 2. Copy environment file and fill in secrets
cp .env.example .env   # then edit: AUTH_JWT_SECRET (>=32 chars), DB creds, OAuth keys

# 3. Start PostgreSQL + run migrations + start backend
make docker-up

# 4. (Alternative) Run locally against Dockerized DB
docker compose up -d postgres migrate
make run
```

## Verify

- **Health check**: `curl http://localhost:8080/health`
- **GraphQL Playground**: open `http://localhost:8080/` in a browser (enabled by default)
- **Readiness probe**: `curl http://localhost:8080/ready` -- returns 200 when DB is reachable

## Common Issues

| Problem | Fix |
|---|---|
| `connect to database: ...` | Ensure Postgres is running and `DATABASE_DSN` matches your `.env` |
| `jwt_secret must be at least 32 characters` | Set `AUTH_JWT_SECRET` in `.env` to a 32+ char string |
| Port 8080 in use | Change `SERVER_PORT` in `.env` or stop the conflicting process |
| Migrations fail | Run `make migrate-status` to see which migration is stuck |

## Useful Commands

```bash
make test              # unit tests with race detector
make test-e2e          # E2E tests (needs Docker for testcontainers)
make generate          # regenerate sqlc queries + go generate mocks
make migrate-create name=add_foo  # create a new migration
```
