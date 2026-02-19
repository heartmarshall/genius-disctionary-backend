# API Reference

## REST Endpoints

### Health

| Method | Path | Auth | Description |
|---|---|---|---|
| GET | `/live` | No | Liveness probe -- always 200 |
| GET | `/ready` | No | Readiness probe -- pings DB, 200 or 503 |
| GET | `/health` | No | Full health: DB latency, version, status |

### Authentication

All auth endpoints are public (CORS only, no JWT required).

| Method | Path | Description |
|---|---|---|
| POST | `/auth/register` | Create account with email + password |
| POST | `/auth/login` | OAuth login (Google/Apple) |
| POST | `/auth/login/password` | Email + password login |
| POST | `/auth/refresh` | Exchange refresh token for new token pair |
| POST | `/auth/logout` | Revoke refresh token |

**Register request:**
```json
{ "email": "user@example.com", "username": "jdoe", "password": "secret123" }
```

**Auth response (all login/register/refresh):**
```json
{
  "accessToken": "eyJhbG...",
  "refreshToken": "base64-encoded-token",
  "user": { "id": "uuid", "email": "...", "username": "...", "name": "...", "avatarUrl": "..." }
}
```

---

## GraphQL Endpoint

**URL**: `POST /query`
**Auth**: Bearer token in `Authorization` header (some queries work anonymously)
**Playground**: `GET /` (when `GRAPHQL_PLAYGROUND_ENABLED=true`)

### Queries

#### Dictionary

| Query | Auth | Description |
|---|---|---|
| `searchCatalog(query, limit)` | No | Search reference catalog |
| `previewRefEntry(text)` | No | Full preview of a catalog entry |
| `dictionary(input: DictionaryFilterInput)` | Yes | User's entries with filtering, sorting, cursor/offset pagination |
| `dictionaryEntry(id: UUID!)` | Yes | Single entry with all nested data |
| `deletedEntries(limit, offset)` | Yes | Soft-deleted entries (trash bin) |
| `exportEntries` | Yes | Full dictionary export |

#### Study

| Query | Auth | Description |
|---|---|---|
| `studyQueue(limit)` | Yes | Due + new cards for review |
| `dashboard` | Yes | Due count, new count, streak, status distribution |
| `cardHistory(input)` | Yes | Paginated review log for a card |
| `cardStats(cardId)` | Yes | Accuracy, grade distribution, interval |

#### Organization

| Query | Auth | Description |
|---|---|---|
| `topics` | Yes | All user topics |
| `inboxItems(limit, offset)` | Yes | Paginated inbox items |
| `inboxItem(id)` | Yes | Single inbox item |

#### User

| Query | Auth | Description |
|---|---|---|
| `me` | Yes | Current user profile + settings |

### Mutations

#### Dictionary

| Mutation | Description |
|---|---|
| `createEntryFromCatalog(input)` | Create entry linked to reference catalog |
| `createEntryCustom(input)` | Create fully custom entry |
| `updateEntryNotes(input)` | Edit entry notes |
| `deleteEntry(id)` | Soft-delete entry |
| `restoreEntry(id)` | Restore soft-deleted entry |
| `batchDeleteEntries(ids)` | Bulk soft-delete |
| `importEntries(input)` | Bulk import with deduplication |

#### Content Editing

| Mutation | Description |
|---|---|
| `addSense(input)` | Add sense to entry |
| `updateSense(input)` | Edit sense definition/POS/CEFR |
| `deleteSense(id)` | Remove sense |
| `reorderSenses(input)` | Reorder senses by position |
| `addTranslation(input)` | Add translation to sense |
| `updateTranslation(input)` | Edit translation text |
| `deleteTranslation(id)` | Remove translation |
| `reorderTranslations(input)` | Reorder translations |
| `addExample(input)` | Add usage example to sense |
| `updateExample(input)` | Edit example |
| `deleteExample(id)` | Remove example |
| `reorderExamples(input)` | Reorder examples |
| `addUserImage(input)` | Upload image to entry |
| `deleteUserImage(id)` | Remove user image |

#### Study

| Mutation | Description |
|---|---|
| `reviewCard(input)` | Grade a card (AGAIN/HARD/GOOD/EASY) |
| `undoReview(cardId)` | Revert last review |
| `createCard(entryId)` | Create flashcard for entry |
| `deleteCard(id)` | Remove flashcard |
| `batchCreateCards(entryIds)` | Bulk create cards |
| `startStudySession` | Begin new session |
| `finishStudySession(input)` | End session with results |
| `abandonStudySession` | Abandon current session |

#### Organization

| Mutation | Description |
|---|---|
| `createTopic(input)` | Create entry group |
| `updateTopic(input)` | Edit topic name/description |
| `deleteTopic(id)` | Remove topic |
| `linkEntryToTopic(input)` | Add entry to topic |
| `unlinkEntryFromTopic(input)` | Remove entry from topic |
| `batchLinkEntries(input)` | Bulk link entries to topic |

#### Inbox

| Mutation | Description |
|---|---|
| `createInboxItem(input)` | Add quick note |
| `deleteInboxItem(id)` | Remove note |
| `deleteAllInboxItems` | Clear inbox |

#### User

| Mutation | Description |
|---|---|
| `updateSettings(input)` | Update SRS preferences |

### Key Input Types

**DictionaryFilterInput** (used by `dictionary` query):
```graphql
input DictionaryFilterInput {
  search: String           # text search
  hasCard: Boolean         # filter by card existence
  partOfSpeech: PartOfSpeech
  topicId: UUID
  status: LearningStatus
  sortBy: EntrySortField   # TEXT, CREATED_AT, UPDATED_AT
  sortOrder: SortDirection  # ASC, DESC
  limit: Int
  cursor: String           # cursor-based pagination
  offset: Int              # offset-based pagination
}
```

**ReviewCardInput**:
```graphql
input ReviewCardInput {
  cardId: UUID!
  grade: ReviewGrade!      # AGAIN, HARD, GOOD, EASY
  durationMs: Int          # time spent reviewing
}
```

### Error Codes

GraphQL errors use extension codes:

| Code | Domain Error | HTTP Analogy |
|---|---|---|
| `NOT_FOUND` | `ErrNotFound` | 404 |
| `ALREADY_EXISTS` | `ErrAlreadyExists` | 409 |
| `VALIDATION` | `ErrValidation` | 422 (includes field details) |
| `UNAUTHENTICATED` | `ErrUnauthorized` | 401 |
| `FORBIDDEN` | `ErrForbidden` | 403 |
| `CONFLICT` | `ErrConflict` | 409 |
| `INTERNAL` | Unexpected error | 500 (generic message to client) |

> For the complete GraphQL schema, see `internal/transport/graphql/schema/*.graphql`.
