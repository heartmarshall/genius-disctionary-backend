# MyEnglish — Business Domain Documentation

> A spaced-repetition language learning platform where users build personal English dictionaries, study vocabulary with scientifically-optimized flashcards (FSRS-5 algorithm), and track their learning progress over time.

## How This Document Was Created

This documentation was reverse-engineered from the codebase (`backend_v4/`). It represents what the code DOES, which may differ from original business requirements. Items marked with ❓ are inferred and should be verified with domain experts.

## Glossary

| Term | Meaning | Code location |
|---|---|---|
| Entry | A word or phrase in a user's personal dictionary | `internal/domain/entry.go` |
| Sense | A specific meaning/definition of a word (e.g., "bank" as financial institution vs. riverbank) | `internal/domain/entry.go` |
| Card | A flashcard linked 1:1 with a dictionary entry, tracked by the FSRS-5 algorithm | `internal/domain/card.go` |
| Reference Catalog | A shared, immutable knowledge base of words with definitions, examples, translations, pronunciations | `internal/domain/reference.go` |
| FSRS-5 | Free Spaced Repetition Scheduler v5 — the algorithm that decides when to show a card again | `internal/service/study/fsrs/` |
| Study Session | A bounded period of reviewing flashcards, with aggregated results | `internal/domain/card.go` |
| Topic | A user-created category for organizing dictionary entries (e.g., "Travel", "Business") | `internal/domain/organization.go` |
| Inbox | A quick-capture area for words/notes the user wants to process later | `internal/domain/organization.go` |
| Enrichment | An AI-powered process (LLM) that adds definitions, examples, and translations to catalog entries | `internal/domain/enrichment.go` |
| Review | The act of seeing a flashcard and self-grading recall quality | `internal/domain/card.go` |
| Lapse | When a user forgets a previously-learned card (grades "Again" on a Review card) | `internal/service/study/fsrs/scheduler.go` |
| Stability | FSRS metric: how long a memory is expected to last (in days) | `internal/service/study/fsrs/algorithm.go` |
| Difficulty | FSRS metric: how inherently hard a card is to remember (scale 1-10) | `internal/service/study/fsrs/algorithm.go` |
| Retrievability | FSRS metric: the probability of recalling a card at a given moment | `internal/service/study/fsrs/algorithm.go` |
| Streak | Consecutive days with at least one review completed | `internal/service/study/dashboard.go` |

## Reading Order

| # | File | What you'll learn | Time |
|---|---|---|---|
| 1 | [DOMAIN_MODEL.md](DOMAIN_MODEL.md) | What "things" exist in this business — users, words, cards, sessions | 10 min |
| 2 | [WORKFLOWS.md](WORKFLOWS.md) | How users register, build dictionaries, study, and track progress | 15 min |
| 3 | [BUSINESS_RULES.md](BUSINESS_RULES.md) | Validation limits, SRS constraints, and behavioral invariants | 10 min |
| 4 | [ROLES_AND_ACCESS.md](ROLES_AND_ACCESS.md) | User vs Admin permissions and what each can do | 5 min |
| 5 | [INTEGRATIONS.md](INTEGRATIONS.md) | OAuth providers, LLM enrichment, and reference data sources | 5 min |
| 6 | [UNKNOWNS.md](UNKNOWNS.md) | Open questions and ambiguities for domain experts | 5 min |

**Total reading time: ~50 minutes.**
