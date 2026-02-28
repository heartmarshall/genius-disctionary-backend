# External Integrations

Each integration is described by its BUSINESS PURPOSE, not its technical protocol.

## Authentication: Google OAuth
**Business purpose**: Allow users to sign in with their Google accounts (one-tap login)
**What the system sends**: Authorization code from the user's browser
**What it receives**: Verified user identity (email, name, avatar, Google user ID)
**Business events triggered**: Account creation (if new user) or account linking (if email matches existing account)
**Configuration**: Requires Google Client ID, Client Secret, and Redirect URI
**Code**: `internal/auth/`, `internal/service/auth/login.go`

## Authentication: Apple Sign-In
**Business purpose**: Allow users to sign in with their Apple ID (required for iOS App Store compliance)
**What the system sends**: Authorization code from the user's device
**What it receives**: Verified user identity
**Configuration**: Requires Apple Key ID, Team ID, and Private Key
**Code**: `internal/auth/`, `internal/service/auth/login.go`

> Both OAuth providers are optional — the system only enables providers with fully configured credentials.

## AI Enrichment: Anthropic Claude (LLM)
**Business purpose**: Automatically enrich dictionary words with additional definitions, examples, and translations that aren't in the reference catalog
**How it works**:
1. When a user adds a word from the catalog, the system queues it for enrichment
2. A background processor (`cmd/enrich/`) picks up pending items and sends them to Claude
3. Claude generates additional senses, translations, and example sentences
4. Results are stored in the reference catalog (benefiting all users)

**Queue states**: `pending` → `processing` → `done` or `failed`
**Admin controls**: Admins can view queue stats, retry failures, and reset stuck items

**Code**: `internal/domain/enrichment.go`, `internal/service/enrichment/`, `cmd/enrich/`
**Dependency**: `github.com/anthropics/anthropic-sdk-go`

## Reference Data Sources (Seeded Offline)

These are not live integrations — they are data corpora imported during system setup via the seeder tool (`cmd/seeder/`). They populate the shared reference catalog.

| Source | Business purpose | What it provides |
|---|---|---|
| **CMU Pronouncing Dictionary** | Help users learn correct pronunciation | Phonetic transcriptions (IPA-like format) |
| **NGSL (New General Service List)** | Prioritize high-frequency vocabulary | Frequency rankings for common English words |
| **Tatoeba** | Show real-world usage of words in sentences | Example sentences with translations |
| **Wiktionary** | Provide comprehensive word definitions | Definitions, parts of speech, etymology |
| **WordNet** | Show how words relate to each other | Synonyms, antonyms, hypernyms, derived forms |

Each piece of data is tagged with a `source_slug` identifying its origin, enabling tracking which sources have been fetched for each word.

**Code**: `internal/app/seeder/`, `internal/domain/reference.go`

## Translation Service
**Business purpose**: Provide translations of words for learners whose native language isn't English
**Code**: `internal/adapter/provider/translate/`

> ❓ The exact translation provider is not clear from the code — it appears to be abstracted behind a provider interface. Could be Google Translate or similar.

## FreeDict
**Business purpose**: Open-source dictionary data to supplement other definition sources
**Code**: `internal/adapter/provider/freedict/`

## Database: PostgreSQL
**Business purpose**: Persistent storage for all user data, dictionary entries, study progress, and reference catalog
**Key facts**:
- Migrations managed by Goose (20 migration files)
- Connection pooling with configurable limits (default: 5-25 connections)
- All write operations use transactions for data consistency

**Code**: `internal/adapter/postgres/`, `migrations/`
