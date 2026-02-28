# MyEnglish Backend — Documentation Index

> A Go backend for a personal English vocabulary learning app with spaced repetition (FSRS-5), dictionary management, and reference catalog integration.

## Reading Order

| # | File | What you'll learn | Time |
|---|---|---|---|
| 1 | [QUICKSTART.md](QUICKSTART.md) | Get the project running locally | 2 min |
| 2 | [ARCHITECTURE.md](ARCHITECTURE.md) | System overview, tech stack, project structure | 10 min |
| 3 | [COMPONENTS.md](COMPONENTS.md) | Deep dive into services, repos, and middleware | 15 min |
| 4 | [DATA_FLOW.md](DATA_FLOW.md) | How requests, data, and SRS scheduling flow | 5 min |
| 5 | [API.md](API.md) | REST + GraphQL API reference | as needed |
| 6 | [DECISIONS.md](DECISIONS.md) | Why things are built this way | 10 min |

**Total reading time: ~45 minutes** for full understanding.

## What's Not Documented

Topics intentionally omitted with pointers to source:

- **Seeder pipeline** (Wiktionary, NGSL, WordNet, Tatoeba imports) — see `cmd/seeder/` and `internal/app/seeder/`
- **Enrichment worker** (LLM-powered catalog enrichment) — see `internal/service/enrichment/`
- **Frontend** (React 19 + Vite + Tailwind) — see `../frontend/`
- **Individual sqlc query SQL** — see `internal/adapter/postgres/*/query/*.sql`
- **E2E test scenarios** — see `tests/e2e/`
- **Dataset processing** — see `../datasets/`
