# API Reference

## REST Endpoints

### Health Probes

| Method | Path | Auth | Response |
|---|---|---|---|
| GET | `/live` | No | `200 OK` — server is running |
| GET | `/ready` | No | `200` if DB connected, `503` if not |
| GET | `/health` | No | `{ status, version, components: { database: { status, latency } } }` |

### Authentication

All auth endpoints return: `{ accessToken, refreshToken, user: { id, email, username, name, avatarUrl, role } }`

| Method | Path | Body | Notes |
|---|---|---|---|
| POST | `/auth/register` | `{ email, username, password }` | Rate limit: 5/min per IP |
| POST | `/auth/login` | `{ provider, code }` | OAuth login (Google). Rate limit: 10/min |
| POST | `/auth/login/password` | `{ email, password }` | Rate limit: 10/min |
| POST | `/auth/refresh` | `{ refreshToken }` | Rotates token. Rate limit: 20/min |
| POST | `/auth/logout` | — | Requires Bearer token. Revokes all sessions |

**Validation error format** (400):
```json
{
  "error": "validation error",
  "code": "VALIDATION",
  "fields": [
    { "field": "email", "message": "invalid email format" },
    { "field": "password", "message": "must be 8-72 characters" }
  ]
}
```

### Admin (requires `admin` role)

| Method | Path | Params/Body | Response |
|---|---|---|---|
| GET | `/admin/users` | `?limit=&offset=` | `{ users: [], total }` |
| PUT | `/admin/users/{id}/role` | `{ role: "admin"\|"user" }` | Updated `User` |
| GET | `/admin/enrichment/stats` | — | `{ pending, processing, done, failed, total }` |
| GET | `/admin/enrichment/queue` | `?status=&limit=&offset=` | `[EnrichmentQueueItem]` |
| POST | `/admin/enrichment/enqueue` | `{ refEntryId }` | `{ status, refEntryId }` |
| POST | `/admin/enrichment/retry` | — | `{ retried: int }` |
| POST | `/admin/enrichment/reset-processing` | — | `{ reset: int }` |

---

## GraphQL API

**Endpoint**: `POST /query`
**Auth**: Bearer token in `Authorization` header (required for all operations)

### Dictionary

```graphql
# Search reference catalog (autocomplete)
query { searchCatalog(query: "exam", limit: 10) { id, text, senses { definition } } }

# Preview full catalog entry (fetches from API if missing)
query { previewRefEntry(text: "ephemeral") { id, text, senses { ... }, pronunciations { ... } } }

# List user's entries (cursor pagination)
query {
  dictionary(input: {
    search: "run"
    sortField: CREATED_AT
    sortDirection: DESC
    first: 20
    after: "cursor..."
    hasCard: true
    partOfSpeech: VERB
    topicId: "uuid"
    status: REVIEW
  }) {
    edges { node { id, text, senses { definition }, card { state, due } }, cursor }
    pageInfo { hasNextPage, endCursor }
    totalCount
  }
}

# Single entry with all nested data
query { dictionaryEntry(id: "uuid") { id, text, notes, senses { ... }, card { ... }, topics { ... } } }

# Trash
query { deletedEntries(limit: 20, offset: 0) { entries { id, text, deletedAt }, totalCount } }
```

```graphql
# Create from catalog
mutation { createEntryFromCatalog(input: {
  refEntryId: "uuid", senseIds: ["uuid"], createCard: true, notes: "..."
}) { entry { id } } }

# Create custom
mutation { createEntryCustom(input: {
  text: "serendipity",
  senses: [{ definition: "...", partOfSpeech: NOUN, translations: ["..."], examples: [{ sentence: "..." }] }],
  createCard: true
}) { entry { id } } }

# Soft delete / restore
mutation { deleteEntry(id: "uuid") { success } }
mutation { restoreEntry(id: "uuid") { entry { id } } }

# Import
mutation { importEntries(input: { items: [{ text: "word", translations: ["..."] }] }) { created, skipped, errors } }
```

### Content Editing

```graphql
# Senses
mutation { addSense(input: { entryId: "uuid", definition: "...", partOfSpeech: NOUN }) { sense { id } } }
mutation { updateSense(input: { senseId: "uuid", definition: "..." }) { sense { id } } }
mutation { deleteSense(id: "uuid") { success } }
mutation { reorderSenses(input: { entryId: "uuid", items: [{ id: "uuid", position: 0 }] }) { success } }

# Translations (same pattern: add, update, delete, reorder)
mutation { addTranslation(input: { senseId: "uuid", text: "..." }) { translation { id } } }

# Examples
mutation { addExample(input: { senseId: "uuid", sentence: "...", translation: "..." }) { example { id } } }

# User images
mutation { addUserImage(input: { entryId: "uuid", url: "https://...", caption: "..." }) { image { id } } }
```

### Study (SRS)

```graphql
# Study queue (due + new cards)
query { studyQueue(limit: 50) { id, text, senses { definition, translations { text } }, card { state, due } } }

# Dashboard
query { dashboard {
  dueCount, newCount, reviewedToday, newToday, streak, overdueCount
  statusCounts { new, learning, review, relearning, total }
  activeSession { id, status }
} }

# Review a card
mutation { reviewCard(input: { cardId: "uuid", grade: GOOD, durationMs: 5000 }) {
  card { id, state, stability, difficulty, due, reps, lapses }
} }

# Undo last review (within 10 min)
mutation { undoReview(cardId: "uuid") { card { id, state, due } } }

# Session lifecycle
mutation { startStudySession { session { id, status } } }
mutation { finishStudySession { session { id, result { totalReviews, accuracyRate, gradeCounts { again, hard, good, easy } } } } }

# Card management
mutation { createCard(entryId: "uuid") { card { id } } }
mutation { batchCreateCards(entryIds: ["uuid1", "uuid2"]) { created } }
mutation { deleteCard(id: "uuid") { success } }

# Card history & stats
query { cardHistory(input: { cardId: "uuid", limit: 20 }) { logs { grade, reviewedAt, durationMs }, total } }
query { cardStats(cardId: "uuid") { totalReviews, accuracyRate, averageDurationMs, gradeDistribution { again, hard, good, easy } } }
```

### Organization

```graphql
# Topics
query { topics { id, name, description, entryCount } }
mutation { createTopic(input: { name: "Travel", description: "..." }) { topic { id } } }
mutation { updateTopic(input: { topicId: "uuid", name: "..." }) { topic { id } } }
mutation { deleteTopic(id: "uuid") { success } }
mutation { linkEntryToTopic(input: { topicId: "uuid", entryId: "uuid" }) { success } }
mutation { batchLinkEntriesToTopic(input: { topicId: "uuid", entryIds: ["uuid1", "uuid2"] }) { linked } }

# Inbox
query { inboxItems(limit: 20, offset: 0) { items { id, text, context, createdAt }, totalCount } }
mutation { createInboxItem(input: { text: "ephemeral", context: "Heard in podcast" }) { item { id } } }
mutation { deleteInboxItem(id: "uuid") { success } }
mutation { clearInbox { deleted } }
```

### User

```graphql
query { me { id, email, username, name, role } }
query { userSettings { newCardsPerDay, reviewsPerDay, maxIntervalDays, desiredRetention, timezone } }

mutation { updateProfile(input: { name: "John" }) { user { id, name } } }
mutation { updateSettings(input: { newCardsPerDay: 30, desiredRetention: 0.85, timezone: "Europe/London" }) { settings { ... } } }
```

---

## Key Types

```graphql
enum CardState     { NEW, LEARNING, REVIEW, RELEARNING }
enum ReviewGrade   { AGAIN, HARD, GOOD, EASY }
enum PartOfSpeech  { NOUN, VERB, ADJECTIVE, ADVERB, PRONOUN, PREPOSITION, CONJUNCTION, INTERJECTION, PHRASE, IDIOM, OTHER }
enum SessionStatus { ACTIVE, FINISHED, ABANDONED }

scalar UUID        # google/uuid format
scalar DateTime    # RFC 3339
```

> For the complete GraphQL schema, see `internal/transport/graphql/schema/*.graphql`
