# MyEnglish Backend -- Documentation Index

> A Go backend for a vocabulary-building app with a personal dictionary, reference catalog integration, content editing, spaced-repetition study, and topic organization -- served over GraphQL + REST.

## Reading Order

| # | File | What you'll learn | Time |
|---|---|---|---|
| 1 | [QUICKSTART.md](QUICKSTART.md) | Get the project running locally | 2 min |
| 2 | [ARCHITECTURE.md](ARCHITECTURE.md) | System overview, tech stack, project layout | 10 min |
| 3 | [COMPONENTS.md](COMPONENTS.md) | Deep dive into each service and adapter | 15 min |
| 4 | [DATA_FLOW.md](DATA_FLOW.md) | How requests flow through the system | 5 min |
| 5 | [API.md](API.md) | REST endpoints + GraphQL operations reference | as needed |
| 6 | [DECISIONS.md](DECISIONS.md) | Why things are built this way | 10 min |

**Total reading time: ~45 minutes** for full understanding.

## What's Not Documented

Topics intentionally omitted with pointers to source:

- **Individual SQL migrations** -- see `migrations/*.sql` for full schema DDL
- **Generated code** (gqlgen resolvers, sqlc queries, moq mocks) -- see `internal/transport/graphql/generated/` and `*_mock_test.go` files
- **SRS algorithm internals** -- the SM-2 variant is implemented in `internal/service/study/`; tuning params live in `config.yaml` under `srs:`
- **DataLoader batch logic** -- see `internal/transport/graphql/dataloader/`
- **Per-service `docs/` directories** -- some services have their own `docs/readme.md` with business-rule-level detail
- **Frontend** -- a separate `frontend/` directory exists at the repo root; it is outside the scope of this backend documentation
