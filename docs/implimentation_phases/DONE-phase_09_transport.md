# Фаза 9: Транспортный слой (GraphQL)

> **Дата:** 2026-02-15
> **Назначение:** Определение GraphQL-схемы, реализация резолверов, обработка ошибок, маппинг ввода.
> DataLoaders уже реализованы в Фазе 3 (TASK-3.15). HTTP-сервер, middleware-стек, health endpoints и bootstrap wiring — Фаза 10.

---

## Документы-источники

| Документ | Секции |
|----------|--------|
| `docs/code_conventions_v4.md` | §1 (структура), §2 (ошибки), §3 (валидация), §4 (контекст), §10 (GraphQL), §12 (naming) |
| `docs/infra/infra_spec_v4.md` | §4 (HTTP-сервер, middleware), §9 (зависимости) |
| `docs/services/service_layer_spec_v4.md` | §3 (паттерны), §5 (лимиты), §6 (карта сервисов) |
| `docs/services/dictionary_service_spec_v4.md` | Все секции — 13 операций |
| `docs/services/content_service_spec_v4.md` | Все секции — 14 операций |
| `docs/services/study_service_spec_v4_v1.1.md` | Все секции — 12 операций |
| `docs/services/topic_service_spec_v4.md` | Все секции — 7 операций |
| `docs/services/inbox_service_spec_v4.md` | Все секции — 5 операций |
| `docs/services/auth_service_spec_v4.md` | §4 (ValidateToken) — для auth middleware |
| `docs/services/business_scenarios_v4.md` | U1-U4 (User), все сценарии |
| `docs/repo/repo_layer_spec_v4.md` | §18 (DataLoaders) |
| `docs/data_model_v4.md` | Все секции — типы для GraphQL-схемы |

---

## Предусловия (от завершённых фаз)

**Фаза 1 (Skeleton + Domain):**
- Все доменные модели: `Entry`, `Sense`, `Translation`, `Example`, `Card`, `ReviewLog`, `StudySession`, `Topic`, `InboxItem`, `User`, `UserSettings`, `RefEntry`, `RefSense`, etc.
- Все енумы: `LearningStatus`, `ReviewGrade`, `PartOfSpeech`, `SessionStatus`, `EntityType`, `AuditAction`, `OAuthProvider`
- Sentinel errors: `ErrNotFound`, `ErrAlreadyExists`, `ErrValidation`, `ErrUnauthorized`, `ErrForbidden`, `ErrConflict`
- `ValidationError` с `[]FieldError`
- `pkg/ctxutil/` — `WithUserID`, `UserIDFromCtx`, `WithRequestID`, `RequestIDFromCtx`
- `internal/config/` — `Config` со всеми секциями включая `GraphQLConfig`

**Фаза 3 (Repository Layer):**
- DataLoaders (TASK-3.15): 9 загрузчиков в `internal/transport/graphql/dataloader/`
- `dataloader.Middleware(repos)` — per-request middleware
- `dataloader.FromContext(ctx)` — доступ к загрузчикам

**Фазы 4–8 (Service Layer):**
- `internal/service/dictionary/` — 13 операций (SearchCatalog, PreviewRefEntry, CreateEntryFromCatalog, CreateEntryCustom, FindEntries, GetEntry, UpdateNotes, DeleteEntry, FindDeletedEntries, RestoreEntry, BatchDeleteEntries, ImportEntries, ExportEntries)
- `internal/service/content/` — 14 операций (AddSense, UpdateSense, DeleteSense, ReorderSenses, AddTranslation, UpdateTranslation, DeleteTranslation, ReorderTranslations, AddExample, UpdateExample, DeleteExample, ReorderExamples, AddUserImage, DeleteUserImage)
- `internal/service/study/` — 12 операций (GetStudyQueue, ReviewCard, UndoReview, StartSession, FinishSession, AbandonSession, CreateCard, DeleteCard, BatchCreateCards, GetDashboard, GetCardHistory, GetCardStats)
- `internal/service/topic/` — 7 операций (CreateTopic, UpdateTopic, DeleteTopic, ListTopics, LinkEntry, UnlinkEntry, BatchLinkEntries)
- `internal/service/inbox/` — 5 операций (CreateItem, ListItems, GetItem, DeleteItem, DeleteAll)
- `internal/service/auth/` — ValidateToken (для middleware, Фаза 4)
- `internal/service/user/` — GetProfile, GetSettings, UpdateSettings (Фаза 4)

---

## Фиксированные решения

| # | Решение | Обоснование |
|---|---------|-------------|
| 1 | **gqlgen** (schema-first) — схема в `.graphql` файлах, Go-код генерируется | Соответствует tech stack проекта. Schema-first даёт контроль над API |
| 2 | **Autobind к `domain/`** — gqlgen маппит GraphQL-типы на доменные структуры | Избегаем дублирования типов. `DictionaryEntry` → `domain.Entry`, `Sense` → `domain.Sense` и т.д. |
| 3 | **Custom scalars: UUID и DateTime** — `github.com/google/uuid.UUID` и `time.Time` | UUID для всех идентификаторов, DateTime для временных полей |
| 4 | **Field resolvers для вложенных типов** — `DictionaryEntry.senses`, `Sense.translations` и т.д. через DataLoaders | N+1 prevention. Если данные уже загружены (struct field != nil), возвращаем их напрямую |
| 5 | **Relay-style Connection для dictionary** — `DictionaryConnection` с `edges`, `pageInfo`, `totalCount` | Dictionary поддерживает cursor-based pagination. Для остальных (inbox, deleted) — простые списки |
| 6 | **Payload types для всех мутаций** — `CreateEntryPayload`, `DeleteEntryPayload` и т.д. | Консистентный паттерн: мутация возвращает payload, а не голый тип. Позволяет добавлять metadata позже |
| 7 | **Error codes в extensions** — `NOT_FOUND`, `VALIDATION`, `UNAUTHENTICATED`, `ALREADY_EXISTS`, `CONFLICT`, `INTERNAL` | Клиент парсит `extensions.code`. Validation errors дополнительно содержат `extensions.fields` |
| 8 | **Transport может импортировать `service/`** для input/result типов | Конвенции запрещают `transport/ → adapter/`, но НЕ `transport/ → service/`. Input-типы сервисов — часть его публичного API |
| 9 | **Один Resolver struct** с 6 сервисными интерфейсами | Стандартный gqlgen-паттерн. Thin wrapper structs (`queryResolver`, `mutationResolver`) для каждого resolver interface |
| 10 | **Schema split по домену** — 8 `.graphql` файлов | `schema.graphql` (root), `enums.graphql`, `dictionary.graphql`, `content.graphql`, `study.graphql`, `organization.graphql`, `user.graphql`, `pagination.graphql` |
| 11 | **gqlgen layout: follow-schema** — генерирует `*.resolvers.go` по имени schema-файла | `dictionary.resolvers.go`, `content.resolvers.go` и т.д. Автоматическая организация |
| 12 | **Все мутации требуют аутентификации** — проверка userID в каждом resolver-методе | Auth middleware (Фаза 4) пропускает anonymous requests. Resolver сам проверяет userID через service |
| 13 | **Queries могут быть anonymous** — `searchCatalog`, `previewRefEntry` не требуют auth | RefCatalog — shared данные. Остальные queries требуют auth (проверяется в service) |
| 14 | **Input mapping в resolvers** — GraphQL input → service input struct | Маппинг явный, в каждом resolver-методе. Никаких magic-конвертеров |
| 15 | **Тесты резолверов** — unit-тесты с замоканными сервисами | Проверяют: input mapping, error propagation, payload construction. Не тестируют бизнес-логику |
| 16 | **`generate.go`** в пакете `graphql/` с `//go:generate` директивой | Запуск `go generate ./internal/transport/graphql/...` для gqlgen |
| 17 | **Не используем `@goField` директивы** — field resolvers конфигурируются в `gqlgen.yml` | Чище: вся конфигурация в одном месте |
| 18 | **DateTime формат** — RFC3339 (`2006-01-02T15:04:05Z07:00`) | Стандарт для GraphQL. gqlgen маршалит `time.Time` в RFC3339 по умолчанию |
| 19 | **Nullable поля в GraphQL** соответствуют pointer-типам в domain | `notes: String` (nullable) → `*string` в Go. `text: String!` (required) → `string` |
| 20 | **REST-эндпоинты auth — Фаза 4, health — Фаза 10** | Фаза 9 фокусируется исключительно на GraphQL транспорте |

---

## Файловая структура (результат фазы)

```
internal/transport/graphql/
├── schema/
│   ├── schema.graphql           # Scalars, root Query и Mutation
│   ├── enums.graphql            # Все GraphQL-енумы
│   ├── pagination.graphql       # PageInfo, Connection pattern
│   ├── dictionary.graphql       # Dictionary + Reference Catalog типы, queries, mutations
│   ├── content.graphql          # Content mutations, inputs, payloads
│   ├── study.graphql            # Study типы, queries, mutations
│   ├── organization.graphql     # Topic + Inbox типы, queries, mutations
│   └── user.graphql             # User типы, queries, mutations
├── model/
│   └── scalars.go               # UUID и DateTime marshalers
├── generated/
│   ├── generated.go             # Сгенерированный gqlgen код
│   └── models_gen.go            # Сгенерированные модели (inputs, payloads)
├── resolver/
│   ├── resolver.go              # Resolver struct, интерфейсы, конструктор
│   ├── schema.resolvers.go      # (generated stub)
│   ├── dictionary.resolvers.go  # Dictionary + RefCatalog resolvers
│   ├── content.resolvers.go     # Content resolvers
│   ├── study.resolvers.go       # Study resolvers
│   ├── organization.resolvers.go # Topic + Inbox resolvers
│   ├── user.resolvers.go        # User resolvers
│   ├── fieldresolvers.go        # Entry/Sense field resolvers (DataLoaders)
│   └── resolver_test.go         # Unit-тесты резолверов
├── errpresenter.go              # Error presenter: domain → GraphQL errors
├── errpresenter_test.go         # Тесты error presenter
├── gqlgen.yml                   # Конфигурация gqlgen
├── generate.go                  # //go:generate директива
└── dataloader/                  # Уже существует (Фаза 3)
    ├── dataloader.go
    ├── loaders.go
    └── middleware.go
```

---

## Задачи

### TASK-9.1: Конфигурация gqlgen и GraphQL-схема

**Зависит от:** Фаза 1 (domain models), Фаза 3 (DataLoaders)

**Контекст:**
- `code_conventions_v4.md` — §10 (GraphQL конвенции)
- `data_model_v4.md` — все типы для схемы
- Все спецификации сервисов — публичные операции

**Что сделать:**

Добавить gqlgen в зависимости, создать конфигурацию, определить полную GraphQL-схему (8 файлов), реализовать custom scalars, запустить генерацию кода.

---

#### Зависимость gqlgen

Добавить в `go.mod`:

```bash
go get github.com/99designs/gqlgen
```

Добавить в `backend_v4/tools.go`:

```go
//go:build tools

package main

import (
    _ "github.com/99designs/gqlgen"
    _ "github.com/pressly/goose/v3/cmd/goose"
    _ "github.com/sqlc-dev/sqlc/cmd/sqlc"
)
```

---

#### `generate.go`

```go
package graphql

//go:generate go run github.com/99designs/gqlgen generate
```

---

#### `gqlgen.yml`

```yaml
schema:
  - internal/transport/graphql/schema/*.graphql

exec:
  filename: internal/transport/graphql/generated/generated.go
  package: generated

model:
  filename: internal/transport/graphql/generated/models_gen.go
  package: generated

resolver:
  layout: follow-schema
  dir: internal/transport/graphql/resolver
  package: resolver
  filename_template: "{name}.resolvers.go"

autobind:
  - "github.com/heartmarshall/myenglish-backend/internal/domain"

models:
  # Custom Scalars
  UUID:
    model:
      - "github.com/google/uuid.UUID"
  DateTime:
    model:
      - "github.com/heartmarshall/myenglish-backend/internal/transport/graphql/model.DateTime"

  # Domain type bindings
  DictionaryEntry:
    model:
      - "github.com/heartmarshall/myenglish-backend/internal/domain.Entry"
    fields:
      senses:
        resolver: true
      pronunciations:
        resolver: true
      catalogImages:
        resolver: true
      userImages:
        resolver: true
      card:
        resolver: true
      topics:
        resolver: true
  Sense:
    model:
      - "github.com/heartmarshall/myenglish-backend/internal/domain.Sense"
    fields:
      translations:
        resolver: true
      examples:
        resolver: true
  Translation:
    model:
      - "github.com/heartmarshall/myenglish-backend/internal/domain.Translation"
  Example:
    model:
      - "github.com/heartmarshall/myenglish-backend/internal/domain.Example"
  UserImage:
    model:
      - "github.com/heartmarshall/myenglish-backend/internal/domain.UserImage"
  Pronunciation:
    model:
      - "github.com/heartmarshall/myenglish-backend/internal/domain.RefPronunciation"
  CatalogImage:
    model:
      - "github.com/heartmarshall/myenglish-backend/internal/domain.RefImage"
  Card:
    model:
      - "github.com/heartmarshall/myenglish-backend/internal/domain.Card"
  ReviewLog:
    model:
      - "github.com/heartmarshall/myenglish-backend/internal/domain.ReviewLog"
  StudySession:
    model:
      - "github.com/heartmarshall/myenglish-backend/internal/domain.StudySession"
  SessionResult:
    model:
      - "github.com/heartmarshall/myenglish-backend/internal/domain.SessionResult"
  GradeCounts:
    model:
      - "github.com/heartmarshall/myenglish-backend/internal/domain.GradeCounts"
  Dashboard:
    model:
      - "github.com/heartmarshall/myenglish-backend/internal/domain.Dashboard"
  CardStatusCounts:
    model:
      - "github.com/heartmarshall/myenglish-backend/internal/domain.CardStatusCounts"
  CardStats:
    model:
      - "github.com/heartmarshall/myenglish-backend/internal/domain.CardStats"
  Topic:
    model:
      - "github.com/heartmarshall/myenglish-backend/internal/domain.Topic"
  InboxItem:
    model:
      - "github.com/heartmarshall/myenglish-backend/internal/domain.InboxItem"
  User:
    model:
      - "github.com/heartmarshall/myenglish-backend/internal/domain.User"
    fields:
      settings:
        resolver: true
  UserSettings:
    model:
      - "github.com/heartmarshall/myenglish-backend/internal/domain.UserSettings"
  RefEntry:
    model:
      - "github.com/heartmarshall/myenglish-backend/internal/domain.RefEntry"
  RefSense:
    model:
      - "github.com/heartmarshall/myenglish-backend/internal/domain.RefSense"
  RefTranslation:
    model:
      - "github.com/heartmarshall/myenglish-backend/internal/domain.RefTranslation"
  RefExample:
    model:
      - "github.com/heartmarshall/myenglish-backend/internal/domain.RefExample"
  RefPronunciation:
    model:
      - "github.com/heartmarshall/myenglish-backend/internal/domain.RefPronunciation"
  RefImage:
    model:
      - "github.com/heartmarshall/myenglish-backend/internal/domain.RefImage"

  # Enum bindings
  LearningStatus:
    model:
      - "github.com/heartmarshall/myenglish-backend/internal/domain.LearningStatus"
  ReviewGrade:
    model:
      - "github.com/heartmarshall/myenglish-backend/internal/domain.ReviewGrade"
  PartOfSpeech:
    model:
      - "github.com/heartmarshall/myenglish-backend/internal/domain.PartOfSpeech"
  SessionStatus:
    model:
      - "github.com/heartmarshall/myenglish-backend/internal/domain.SessionStatus"

  # Export types binding
  ExportResult:
    model:
      - "github.com/heartmarshall/myenglish-backend/internal/service/dictionary.ExportResult"
  ExportItem:
    model:
      - "github.com/heartmarshall/myenglish-backend/internal/service/dictionary.ExportItem"
  ExportSense:
    model:
      - "github.com/heartmarshall/myenglish-backend/internal/service/dictionary.ExportSense"
  ExportExample:
    model:
      - "github.com/heartmarshall/myenglish-backend/internal/service/dictionary.ExportExample"
```

> **Примечание:** Путь к `gqlgen.yml` — `internal/transport/graphql/gqlgen.yml`. Запуск: `cd backend_v4 && go generate ./internal/transport/graphql/...`. В gqlgen.yml пути указаны относительно корня модуля.

---

#### `model/scalars.go` — Custom Scalars

```go
package model

import (
    "fmt"
    "io"
    "time"

    "github.com/99designs/gqlgen/graphql"
)

// DateTime wraps time.Time for GraphQL scalar marshaling.
type DateTime time.Time

func MarshalDateTime(t time.Time) graphql.Marshaler {
    return graphql.WriterFunc(func(w io.Writer) {
        io.WriteString(w, `"`+t.Format(time.RFC3339)+`"`)
    })
}

func UnmarshalDateTime(v interface{}) (time.Time, error) {
    switch v := v.(type) {
    case string:
        return time.Parse(time.RFC3339, v)
    default:
        return time.Time{}, fmt.Errorf("DateTime must be a string in RFC3339 format")
    }
}
```

> **UUID scalar** не требует custom marshaler — gqlgen умеет работать с `uuid.UUID` из `google/uuid` через autobind. При необходимости добавить marshaler аналогично DateTime.

---

#### Схема: `schema/schema.graphql`

```graphql
scalar UUID
scalar DateTime

type Query

type Mutation
```

---

#### Схема: `schema/enums.graphql`

```graphql
enum LearningStatus {
  NEW
  LEARNING
  REVIEW
  MASTERED
}

enum ReviewGrade {
  AGAIN
  HARD
  GOOD
  EASY
}

enum PartOfSpeech {
  NOUN
  VERB
  ADJECTIVE
  ADVERB
  PRONOUN
  PREPOSITION
  CONJUNCTION
  INTERJECTION
  PHRASE
  IDIOM
  OTHER
}

enum SessionStatus {
  ACTIVE
  FINISHED
  ABANDONED
}

enum EntrySortField {
  TEXT
  CREATED_AT
  UPDATED_AT
}

enum SortDirection {
  ASC
  DESC
}
```

---

#### Схема: `schema/pagination.graphql`

```graphql
type PageInfo {
  hasNextPage: Boolean!
  hasPreviousPage: Boolean!
  startCursor: String
  endCursor: String
}

type DictionaryConnection {
  edges: [DictionaryEdge!]!
  pageInfo: PageInfo!
  totalCount: Int!
}

type DictionaryEdge {
  node: DictionaryEntry!
  cursor: String!
}

"""Простой список с общим количеством (для offset-пагинации)."""
type DeletedEntriesList {
  entries: [DictionaryEntry!]!
  totalCount: Int!
}

type InboxItemList {
  items: [InboxItem!]!
  totalCount: Int!
}
```

---

#### Схема: `schema/dictionary.graphql`

```graphql
# ============================================================
#  OUTPUT TYPES — Dictionary
# ============================================================

type DictionaryEntry {
  id: UUID!
  text: String!
  textNormalized: String!
  notes: String
  createdAt: DateTime!
  updatedAt: DateTime!
  deletedAt: DateTime
  # Field resolvers (DataLoaders):
  senses: [Sense!]!
  pronunciations: [Pronunciation!]!
  catalogImages: [CatalogImage!]!
  userImages: [UserImage!]!
  card: Card
  topics: [Topic!]!
}

type Sense {
  id: UUID!
  definition: String
  partOfSpeech: PartOfSpeech
  cefrLevel: String
  sourceSlug: String!
  position: Int!
  # Field resolvers (DataLoaders):
  translations: [Translation!]!
  examples: [Example!]!
}

type Translation {
  id: UUID!
  text: String
  sourceSlug: String!
  position: Int!
}

type Example {
  id: UUID!
  sentence: String
  translation: String
  sourceSlug: String!
  position: Int!
}

type Pronunciation {
  id: UUID!
  transcription: String!
  audioUrl: String
  region: String
}

type CatalogImage {
  id: UUID!
  url: String!
  caption: String
}

type UserImage {
  id: UUID!
  url: String!
  caption: String
  createdAt: DateTime!
}

# ============================================================
#  OUTPUT TYPES — Reference Catalog
# ============================================================

type RefEntry {
  id: UUID!
  text: String!
  textNormalized: String!
  senses: [RefSense!]!
  pronunciations: [RefPronunciation!]!
  images: [RefImage!]!
}

type RefSense {
  id: UUID!
  definition: String
  partOfSpeech: PartOfSpeech
  cefrLevel: String
  sourceSlug: String!
  position: Int!
  translations: [RefTranslation!]!
  examples: [RefExample!]!
}

type RefTranslation {
  id: UUID!
  text: String!
  sourceSlug: String!
}

type RefExample {
  id: UUID!
  sentence: String!
  translation: String
  sourceSlug: String!
}

type RefPronunciation {
  id: UUID!
  transcription: String!
  audioUrl: String
  region: String
}

type RefImage {
  id: UUID!
  url: String!
  caption: String
}

# ============================================================
#  OUTPUT TYPES — Export
# ============================================================

type ExportResult {
  items: [ExportItem!]!
  exportedAt: DateTime!
}

type ExportItem {
  text: String!
  notes: String
  senses: [ExportSense!]!
  topicNames: [String!]!
}

type ExportSense {
  definition: String
  partOfSpeech: PartOfSpeech
  translations: [String!]!
  examples: [ExportExample!]!
}

type ExportExample {
  sentence: String!
  translation: String
}

# ============================================================
#  INPUT TYPES
# ============================================================

input DictionaryFilterInput {
  search: String
  hasCard: Boolean
  partOfSpeech: PartOfSpeech
  topicId: UUID
  status: LearningStatus
  sortField: EntrySortField
  sortDirection: SortDirection
  """Cursor-based: количество записей."""
  first: Int
  """Cursor-based: курсор после которого загружать."""
  after: String
  """Offset-based: лимит записей."""
  limit: Int
  """Offset-based: смещение."""
  offset: Int
}

input CreateEntryFromCatalogInput {
  refEntryId: UUID!
  senseIds: [UUID!]!
  translationIds: [UUID!]
  exampleIds: [UUID!]
  pronunciationIds: [UUID!]
  imageIds: [UUID!]
  notes: String
  createCard: Boolean
  topicId: UUID
}

input CreateEntryCustomInput {
  text: String!
  senses: [CustomSenseInput!]!
  notes: String
  createCard: Boolean
  topicId: UUID
}

input CustomSenseInput {
  definition: String
  partOfSpeech: PartOfSpeech
  translations: [String!]
  examples: [CustomExampleInput!]
}

input CustomExampleInput {
  sentence: String!
  translation: String
}

input UpdateEntryNotesInput {
  entryId: UUID!
  notes: String
}

input ImportEntriesInput {
  items: [ImportItemInput!]!
}

input ImportItemInput {
  text: String!
  notes: String
  senses: [ImportSenseInput!]!
  topicName: String
}

input ImportSenseInput {
  definition: String
  partOfSpeech: PartOfSpeech
  translations: [String!]
  examples: [ImportExampleInput!]
}

input ImportExampleInput {
  sentence: String!
  translation: String
}

# ============================================================
#  PAYLOAD TYPES
# ============================================================

type CreateEntryPayload {
  entry: DictionaryEntry!
}

type UpdateEntryPayload {
  entry: DictionaryEntry!
}

type DeleteEntryPayload {
  entryId: UUID!
}

type RestoreEntryPayload {
  entry: DictionaryEntry!
}

type BatchDeletePayload {
  deletedCount: Int!
  errors: [BatchError!]!
}

type BatchError {
  id: UUID!
  message: String!
}

type ImportPayload {
  importedCount: Int!
  skippedCount: Int!
  errors: [ImportError!]!
}

type ImportError {
  index: Int!
  text: String!
  message: String!
}

# ============================================================
#  QUERIES
# ============================================================

extend type Query {
  """Поиск в Reference Catalog (автокомплит). Не требует авторизации."""
  searchCatalog(query: String!, limit: Int): [RefEntry!]!

  """Полный preview слова из каталога. Не требует авторизации."""
  previewRefEntry(text: String!): RefEntry

  """Поиск/фильтрация словаря пользователя. Поддерживает cursor и offset."""
  dictionary(input: DictionaryFilterInput!): DictionaryConnection!

  """Одна запись словаря по ID (вложенные данные через DataLoaders)."""
  dictionaryEntry(id: UUID!): DictionaryEntry

  """Корзина: soft-deleted записи."""
  deletedEntries(limit: Int, offset: Int): DeletedEntriesList!

  """Экспорт всего словаря в структурированном формате."""
  exportEntries: ExportResult!
}

# ============================================================
#  MUTATIONS
# ============================================================

extend type Mutation {
  """Создание записи из Reference Catalog."""
  createEntryFromCatalog(input: CreateEntryFromCatalogInput!): CreateEntryPayload!

  """Создание пользовательской записи (без каталога)."""
  createEntryCustom(input: CreateEntryCustomInput!): CreateEntryPayload!

  """Обновление заметок записи."""
  updateEntryNotes(input: UpdateEntryNotesInput!): UpdateEntryPayload!

  """Soft delete записи."""
  deleteEntry(id: UUID!): DeleteEntryPayload!

  """Восстановление из корзины."""
  restoreEntry(id: UUID!): RestoreEntryPayload!

  """Массовое soft delete."""
  batchDeleteEntries(ids: [UUID!]!): BatchDeletePayload!

  """Импорт записей (chunked)."""
  importEntries(input: ImportEntriesInput!): ImportPayload!
}
```

---

#### Схема: `schema/content.graphql`

```graphql
# ============================================================
#  INPUT TYPES — Content
# ============================================================

input AddSenseInput {
  entryId: UUID!
  definition: String
  partOfSpeech: PartOfSpeech
  translations: [String!]
  examples: [AddSenseExampleInput!]
}

input AddSenseExampleInput {
  sentence: String!
  translation: String
}

input UpdateSenseInput {
  senseId: UUID!
  definition: String
  partOfSpeech: PartOfSpeech
  cefrLevel: String
}

input ReorderSensesInput {
  entryId: UUID!
  items: [ReorderItemInput!]!
}

input ReorderItemInput {
  id: UUID!
  position: Int!
}

input AddTranslationInput {
  senseId: UUID!
  text: String!
}

input UpdateTranslationInput {
  translationId: UUID!
  text: String!
}

input ReorderTranslationsInput {
  senseId: UUID!
  items: [ReorderItemInput!]!
}

input AddExampleInput {
  senseId: UUID!
  sentence: String!
  translation: String
}

input UpdateExampleInput {
  exampleId: UUID!
  sentence: String
  translation: String
}

input ReorderExamplesInput {
  senseId: UUID!
  items: [ReorderItemInput!]!
}

input AddUserImageInput {
  entryId: UUID!
  url: String!
  caption: String
}

# ============================================================
#  PAYLOAD TYPES — Content
# ============================================================

type AddSensePayload {
  sense: Sense!
}

type UpdateSensePayload {
  sense: Sense!
}

type DeleteSensePayload {
  senseId: UUID!
}

type ReorderPayload {
  success: Boolean!
}

type AddTranslationPayload {
  translation: Translation!
}

type UpdateTranslationPayload {
  translation: Translation!
}

type DeleteTranslationPayload {
  translationId: UUID!
}

type AddExamplePayload {
  example: Example!
}

type UpdateExamplePayload {
  example: Example!
}

type DeleteExamplePayload {
  exampleId: UUID!
}

type AddUserImagePayload {
  image: UserImage!
}

type DeleteUserImagePayload {
  imageId: UUID!
}

# ============================================================
#  MUTATIONS — Content
# ============================================================

extend type Mutation {
  addSense(input: AddSenseInput!): AddSensePayload!
  updateSense(input: UpdateSenseInput!): UpdateSensePayload!
  deleteSense(id: UUID!): DeleteSensePayload!
  reorderSenses(input: ReorderSensesInput!): ReorderPayload!

  addTranslation(input: AddTranslationInput!): AddTranslationPayload!
  updateTranslation(input: UpdateTranslationInput!): UpdateTranslationPayload!
  deleteTranslation(id: UUID!): DeleteTranslationPayload!
  reorderTranslations(input: ReorderTranslationsInput!): ReorderPayload!

  addExample(input: AddExampleInput!): AddExamplePayload!
  updateExample(input: UpdateExampleInput!): UpdateExamplePayload!
  deleteExample(id: UUID!): DeleteExamplePayload!
  reorderExamples(input: ReorderExamplesInput!): ReorderPayload!

  addUserImage(input: AddUserImageInput!): AddUserImagePayload!
  deleteUserImage(id: UUID!): DeleteUserImagePayload!
}
```

---

#### Схема: `schema/study.graphql`

```graphql
# ============================================================
#  OUTPUT TYPES — Study
# ============================================================

type Card {
  id: UUID!
  entryId: UUID!
  status: LearningStatus!
  nextReviewAt: DateTime
  intervalDays: Int!
  easeFactor: Float!
  createdAt: DateTime!
  updatedAt: DateTime!
}

type ReviewLog {
  id: UUID!
  cardId: UUID!
  grade: ReviewGrade!
  durationMs: Int
  reviewedAt: DateTime!
}

type StudySession {
  id: UUID!
  status: SessionStatus!
  startedAt: DateTime!
  finishedAt: DateTime
  result: SessionResult
}

type SessionResult {
  totalReviews: Int!
  gradeCounts: GradeCounts!
  averageDurationMs: Int!
}

type GradeCounts {
  again: Int!
  hard: Int!
  good: Int!
  easy: Int!
}

type Dashboard {
  dueCount: Int!
  newCount: Int!
  reviewedToday: Int!
  streak: Int!
  statusCounts: CardStatusCounts!
  overdueCount: Int!
  activeSession: StudySession
}

type CardStatusCounts {
  new: Int!
  learning: Int!
  review: Int!
  mastered: Int!
}

type CardStats {
  totalReviews: Int!
  averageDurationMs: Int!
  accuracy: Float!
  gradeDistribution: GradeCounts!
}

# ============================================================
#  INPUT TYPES — Study
# ============================================================

input ReviewCardInput {
  cardId: UUID!
  grade: ReviewGrade!
  durationMs: Int
}

input FinishSessionInput {
  sessionId: UUID!
}

input GetCardHistoryInput {
  cardId: UUID!
  limit: Int
  offset: Int
}

# ============================================================
#  PAYLOAD TYPES — Study
# ============================================================

type ReviewCardPayload {
  card: Card!
  reviewLog: ReviewLog!
}

type UndoReviewPayload {
  card: Card!
}

type CreateCardPayload {
  card: Card!
}

type DeleteCardPayload {
  cardId: UUID!
}

type BatchCreateCardsPayload {
  createdCount: Int!
  skippedCount: Int!
  errors: [BatchCreateCardError!]!
}

type BatchCreateCardError {
  entryId: UUID!
  message: String!
}

type StartSessionPayload {
  session: StudySession!
}

type FinishSessionPayload {
  session: StudySession!
}

type AbandonSessionPayload {
  session: StudySession!
}

# ============================================================
#  QUERIES — Study
# ============================================================

extend type Query {
  """Очередь повторения: overdue + new cards (с лимитом)."""
  studyQueue(limit: Int): [DictionaryEntry!]!

  """Dashboard: статистика, due counts, streak."""
  dashboard: Dashboard!

  """История повторений карточки."""
  cardHistory(input: GetCardHistoryInput!): [ReviewLog!]!

  """Статистика карточки: accuracy, grade distribution."""
  cardStats(cardId: UUID!): CardStats!
}

# ============================================================
#  MUTATIONS — Study
# ============================================================

extend type Mutation {
  reviewCard(input: ReviewCardInput!): ReviewCardPayload!
  undoReview(cardId: UUID!): UndoReviewPayload!
  createCard(entryId: UUID!): CreateCardPayload!
  deleteCard(id: UUID!): DeleteCardPayload!
  batchCreateCards(entryIds: [UUID!]!): BatchCreateCardsPayload!
  startStudySession: StartSessionPayload!
  finishStudySession(input: FinishSessionInput!): FinishSessionPayload!
  abandonStudySession: AbandonSessionPayload!
}
```

---

#### Схема: `schema/organization.graphql`

```graphql
# ============================================================
#  OUTPUT TYPES — Organization
# ============================================================

type Topic {
  id: UUID!
  name: String!
  description: String
  entryCount: Int!
  createdAt: DateTime!
  updatedAt: DateTime!
}

type InboxItem {
  id: UUID!
  text: String!
  context: String
  createdAt: DateTime!
}

# ============================================================
#  INPUT TYPES — Topic
# ============================================================

input CreateTopicInput {
  name: String!
  description: String
}

input UpdateTopicInput {
  topicId: UUID!
  name: String
  description: String
}

input LinkEntryInput {
  topicId: UUID!
  entryId: UUID!
}

input UnlinkEntryInput {
  topicId: UUID!
  entryId: UUID!
}

input BatchLinkEntriesInput {
  topicId: UUID!
  entryIds: [UUID!]!
}

# ============================================================
#  INPUT TYPES — Inbox
# ============================================================

input CreateInboxItemInput {
  text: String!
  context: String
}

# ============================================================
#  PAYLOAD TYPES — Topic
# ============================================================

type CreateTopicPayload {
  topic: Topic!
}

type UpdateTopicPayload {
  topic: Topic!
}

type DeleteTopicPayload {
  topicId: UUID!
}

type LinkEntryPayload {
  success: Boolean!
}

type UnlinkEntryPayload {
  success: Boolean!
}

type BatchLinkPayload {
  linked: Int!
  skipped: Int!
}

# ============================================================
#  PAYLOAD TYPES — Inbox
# ============================================================

type CreateInboxItemPayload {
  item: InboxItem!
}

type DeleteInboxItemPayload {
  itemId: UUID!
}

type ClearInboxPayload {
  deletedCount: Int!
}

# ============================================================
#  QUERIES — Organization
# ============================================================

extend type Query {
  """Все темы пользователя (сортировка по имени, с EntryCount)."""
  topics: [Topic!]!

  """Список inbox items с пагинацией."""
  inboxItems(limit: Int, offset: Int): InboxItemList!

  """Один inbox item по ID."""
  inboxItem(id: UUID!): InboxItem
}

# ============================================================
#  MUTATIONS — Organization
# ============================================================

extend type Mutation {
  createTopic(input: CreateTopicInput!): CreateTopicPayload!
  updateTopic(input: UpdateTopicInput!): UpdateTopicPayload!
  deleteTopic(id: UUID!): DeleteTopicPayload!
  linkEntryToTopic(input: LinkEntryInput!): LinkEntryPayload!
  unlinkEntryFromTopic(input: UnlinkEntryInput!): UnlinkEntryPayload!
  batchLinkEntriesToTopic(input: BatchLinkEntriesInput!): BatchLinkPayload!

  createInboxItem(input: CreateInboxItemInput!): CreateInboxItemPayload!
  deleteInboxItem(id: UUID!): DeleteInboxItemPayload!
  clearInbox: ClearInboxPayload!
}
```

---

#### Схема: `schema/user.graphql`

```graphql
# ============================================================
#  OUTPUT TYPES — User
# ============================================================

type User {
  id: UUID!
  email: String!
  name: String
  avatarUrl: String
  oauthProvider: String!
  createdAt: DateTime!
  """Field resolver — загружает настройки пользователя."""
  settings: UserSettings!
}

type UserSettings {
  newCardsPerDay: Int!
  reviewsPerDay: Int!
  maxIntervalDays: Int!
  timezone: String!
}

# ============================================================
#  INPUT TYPES — User
# ============================================================

input UpdateSettingsInput {
  newCardsPerDay: Int
  reviewsPerDay: Int
  maxIntervalDays: Int
  timezone: String
}

# ============================================================
#  PAYLOAD TYPES — User
# ============================================================

type UpdateSettingsPayload {
  settings: UserSettings!
}

# ============================================================
#  QUERIES — User
# ============================================================

extend type Query {
  """Текущий пользователь (требует авторизации)."""
  me: User!
}

# ============================================================
#  MUTATIONS — User
# ============================================================

extend type Mutation {
  updateSettings(input: UpdateSettingsInput!): UpdateSettingsPayload!
}
```

---

#### Сводка GraphQL API

| Категория | Queries | Mutations | Итого |
|-----------|---------|-----------|-------|
| Dictionary + RefCatalog | 6 | 7 | 13 |
| Content | 0 | 14 | 14 |
| Study | 4 | 8 | 12 |
| Organization (Topic + Inbox) | 3 | 9 | 12 |
| User | 1 | 1 | 2 |
| **Итого** | **14** | **39** | **53** |

Field resolvers: 10 (8 на DictionaryEntry/Sense + 1 на User.settings + 1 на Card при необходимости).

---

**Acceptance criteria TASK-9.1:**
- [ ] gqlgen добавлен в `go.mod` и `tools.go`
- [ ] `gqlgen.yml` создан с autobind, model bindings, field resolvers
- [ ] 8 `.graphql` файлов в `schema/` покрывают все 53 операции
- [ ] `model/scalars.go` с DateTime marshaler
- [ ] `generate.go` с `//go:generate` директивой
- [ ] `go generate ./internal/transport/graphql/...` выполняется без ошибок
- [ ] `generated/generated.go` и `generated/models_gen.go` сгенерированы
- [ ] Resolver stubs сгенерированы в `resolver/`
- [ ] `go build ./...` компилируется (stubs возвращают `panic("not implemented")`)

---

### TASK-9.2: Resolver Foundation и Error Presentation

**Зависит от:** TASK-9.1 (сгенерированный код)

**Контекст:**
- `code_conventions_v4.md` — §1 (интерфейсы потребителя), §2 (обработка ошибок)
- `services/service_layer_spec_v4.md` — §6 (карта сервисов)
- Все спецификации сервисов — публичные методы и типы

**Что сделать:**

Реализовать `resolver.go` с Resolver struct и приватными интерфейсами сервисов, error presenter для маппинга domain-ошибок в GraphQL, field resolvers для DataLoaders.

---

#### `resolver/resolver.go` — Resolver struct и интерфейсы

```go
package resolver

import (
    "context"
    "log/slog"

    "github.com/google/uuid"
    "github.com/heartmarshall/myenglish-backend/internal/domain"
    "github.com/heartmarshall/myenglish-backend/internal/service/content"
    "github.com/heartmarshall/myenglish-backend/internal/service/dictionary"
    "github.com/heartmarshall/myenglish-backend/internal/service/inbox"
    "github.com/heartmarshall/myenglish-backend/internal/service/study"
    "github.com/heartmarshall/myenglish-backend/internal/service/topic"
    "github.com/heartmarshall/myenglish-backend/internal/transport/graphql/generated"
)

// dictionaryService определяет что resolver-у нужно от Dictionary.
type dictionaryService interface {
    SearchCatalog(ctx context.Context, query string, limit int) ([]*domain.RefEntry, error)
    PreviewRefEntry(ctx context.Context, text string) (*domain.RefEntry, error)
    CreateEntryFromCatalog(ctx context.Context, input dictionary.CreateFromCatalogInput) (*domain.Entry, error)
    CreateEntryCustom(ctx context.Context, input dictionary.CreateCustomInput) (*domain.Entry, error)
    FindEntries(ctx context.Context, input dictionary.FindInput) (*dictionary.FindResult, error)
    GetEntry(ctx context.Context, entryID uuid.UUID) (*domain.Entry, error)
    UpdateNotes(ctx context.Context, input dictionary.UpdateNotesInput) (*domain.Entry, error)
    DeleteEntry(ctx context.Context, entryID uuid.UUID) error
    FindDeletedEntries(ctx context.Context, limit, offset int) ([]*domain.Entry, int, error)
    RestoreEntry(ctx context.Context, entryID uuid.UUID) (*domain.Entry, error)
    BatchDeleteEntries(ctx context.Context, entryIDs []uuid.UUID) (*dictionary.BatchResult, error)
    ImportEntries(ctx context.Context, input dictionary.ImportInput) (*dictionary.ImportResult, error)
    ExportEntries(ctx context.Context) (*dictionary.ExportResult, error)
}

// contentService определяет что resolver-у нужно от Content.
type contentService interface {
    AddSense(ctx context.Context, input content.AddSenseInput) (*domain.Sense, error)
    UpdateSense(ctx context.Context, input content.UpdateSenseInput) (*domain.Sense, error)
    DeleteSense(ctx context.Context, senseID uuid.UUID) error
    ReorderSenses(ctx context.Context, input content.ReorderSensesInput) error
    AddTranslation(ctx context.Context, input content.AddTranslationInput) (*domain.Translation, error)
    UpdateTranslation(ctx context.Context, input content.UpdateTranslationInput) (*domain.Translation, error)
    DeleteTranslation(ctx context.Context, translationID uuid.UUID) error
    ReorderTranslations(ctx context.Context, input content.ReorderTranslationsInput) error
    AddExample(ctx context.Context, input content.AddExampleInput) (*domain.Example, error)
    UpdateExample(ctx context.Context, input content.UpdateExampleInput) (*domain.Example, error)
    DeleteExample(ctx context.Context, exampleID uuid.UUID) error
    ReorderExamples(ctx context.Context, input content.ReorderExamplesInput) error
    AddUserImage(ctx context.Context, input content.AddUserImageInput) (*domain.UserImage, error)
    DeleteUserImage(ctx context.Context, imageID uuid.UUID) error
}

// studyService определяет что resolver-у нужно от Study.
type studyService interface {
    GetStudyQueue(ctx context.Context, input study.GetQueueInput) ([]*domain.Entry, error)
    ReviewCard(ctx context.Context, input study.ReviewCardInput) (*domain.Card, *domain.ReviewLog, error)
    UndoReview(ctx context.Context, input study.UndoReviewInput) (*domain.Card, error)
    StartSession(ctx context.Context) (*domain.StudySession, error)
    FinishSession(ctx context.Context, input study.FinishSessionInput) (*domain.StudySession, error)
    AbandonSession(ctx context.Context) (*domain.StudySession, error)
    CreateCard(ctx context.Context, input study.CreateCardInput) (*domain.Card, error)
    DeleteCard(ctx context.Context, input study.DeleteCardInput) error
    BatchCreateCards(ctx context.Context, input study.BatchCreateCardsInput) (*study.BatchCreateResult, error)
    GetDashboard(ctx context.Context) (*domain.Dashboard, error)
    GetCardHistory(ctx context.Context, input study.GetCardHistoryInput) ([]*domain.ReviewLog, error)
    GetCardStats(ctx context.Context, cardID uuid.UUID) (*domain.CardStats, error)
}

// topicService определяет что resolver-у нужно от Topic.
type topicService interface {
    CreateTopic(ctx context.Context, input topic.CreateTopicInput) (*domain.Topic, error)
    UpdateTopic(ctx context.Context, input topic.UpdateTopicInput) (*domain.Topic, error)
    DeleteTopic(ctx context.Context, input topic.DeleteTopicInput) error
    ListTopics(ctx context.Context) ([]*domain.Topic, error)
    LinkEntry(ctx context.Context, input topic.LinkEntryInput) error
    UnlinkEntry(ctx context.Context, input topic.UnlinkEntryInput) error
    BatchLinkEntries(ctx context.Context, input topic.BatchLinkEntriesInput) (*topic.BatchLinkResult, error)
}

// inboxService определяет что resolver-у нужно от Inbox.
type inboxService interface {
    CreateItem(ctx context.Context, input inbox.CreateItemInput) (*domain.InboxItem, error)
    ListItems(ctx context.Context, input inbox.ListItemsInput) ([]*domain.InboxItem, int, error)
    GetItem(ctx context.Context, itemID uuid.UUID) (*domain.InboxItem, error)
    DeleteItem(ctx context.Context, input inbox.DeleteItemInput) error
    DeleteAll(ctx context.Context) (int, error)
}

// userService определяет что resolver-у нужно от User (Фаза 4).
type userService interface {
    GetProfile(ctx context.Context) (*domain.User, error)
    GetSettings(ctx context.Context) (*domain.UserSettings, error)
    UpdateSettings(ctx context.Context, input interface{}) (*domain.UserSettings, error)
    // Сигнатура UpdateSettings будет уточнена после реализации Фазы 4.
    // Вероятно: UpdateSettings(ctx, user.UpdateSettingsInput) (*domain.UserSettings, error)
}

// Resolver — корневой resolver, содержит все сервисные зависимости.
type Resolver struct {
    dictionary dictionaryService
    content    contentService
    study      studyService
    topic      topicService
    inbox      inboxService
    user       userService
    log        *slog.Logger
}

func NewResolver(
    log *slog.Logger,
    dictionary dictionaryService,
    content contentService,
    study studyService,
    topic topicService,
    inbox inboxService,
    user userService,
) *Resolver {
    return &Resolver{
        dictionary: dictionary,
        content:    content,
        study:      study,
        topic:      topic,
        inbox:      inbox,
        user:       user,
        log:        log.With("component", "graphql"),
    }
}

// Query returns the generated QueryResolver implementation.
func (r *Resolver) Query() generated.QueryResolver {
    return &queryResolver{r}
}

// Mutation returns the generated MutationResolver implementation.
func (r *Resolver) Mutation() generated.MutationResolver {
    return &mutationResolver{r}
}

// DictionaryEntry returns the field resolver for DictionaryEntry.
func (r *Resolver) DictionaryEntry() generated.DictionaryEntryResolver {
    return &dictionaryEntryResolver{r}
}

// Sense returns the field resolver for Sense.
func (r *Resolver) Sense() generated.SenseResolver {
    return &senseResolver{r}
}

// User returns the field resolver for User.
func (r *Resolver) User() generated.UserResolver {
    return &userResolver{r}
}

type queryResolver struct{ *Resolver }
type mutationResolver struct{ *Resolver }
type dictionaryEntryResolver struct{ *Resolver }
type senseResolver struct{ *Resolver }
type userResolver struct{ *Resolver }
```

> **Примечание:** Точные сигнатуры `userService` зависят от Фазы 4 (User Service). Указанные сигнатуры — предварительные. При реализации Фазы 4 интерфейс будет уточнён.

---

#### `errpresenter.go` — Error Presentation

```go
package graphql

import (
    "context"
    "errors"
    "log/slog"

    "github.com/99designs/gqlgen/graphql"
    "github.com/vektah/gqlparser/v2/gqlerror"
    "github.com/heartmarshall/myenglish-backend/internal/domain"
    "github.com/heartmarshall/myenglish-backend/pkg/ctxutil"
)

// NewErrorPresenter возвращает gqlgen error presenter,
// который маппит domain-ошибки в GraphQL error codes.
func NewErrorPresenter(log *slog.Logger) graphql.ErrorPresenterFunc {
    return func(ctx context.Context, err error) *gqlerror.Error {
        // Получаем оригинальную ошибку (gqlgen оборачивает)
        gqlErr := graphql.DefaultErrorPresenter(ctx, err)

        // Разворачиваем до domain-ошибки
        var origErr error
        if unwrapped := errors.Unwrap(err); unwrapped != nil {
            origErr = unwrapped
        } else {
            origErr = err
        }

        switch {
        case errors.Is(origErr, domain.ErrNotFound):
            gqlErr.Extensions = map[string]interface{}{"code": "NOT_FOUND"}

        case errors.Is(origErr, domain.ErrAlreadyExists):
            gqlErr.Extensions = map[string]interface{}{"code": "ALREADY_EXISTS"}

        case errors.Is(origErr, domain.ErrValidation):
            gqlErr.Extensions = map[string]interface{}{"code": "VALIDATION"}
            var ve *domain.ValidationError
            if errors.As(origErr, &ve) {
                gqlErr.Extensions["fields"] = ve.Errors
            }

        case errors.Is(origErr, domain.ErrUnauthorized):
            gqlErr.Extensions = map[string]interface{}{"code": "UNAUTHENTICATED"}

        case errors.Is(origErr, domain.ErrForbidden):
            gqlErr.Extensions = map[string]interface{}{"code": "FORBIDDEN"}

        case errors.Is(origErr, domain.ErrConflict):
            gqlErr.Extensions = map[string]interface{}{"code": "CONFLICT"}

        default:
            // Неожиданная ошибка — логируем, клиенту возвращаем generic
            requestID := ctxutil.RequestIDFromCtx(ctx)
            log.ErrorContext(ctx, "unexpected GraphQL error",
                slog.String("error", origErr.Error()),
                slog.String("request_id", requestID),
            )
            gqlErr.Message = "internal error"
            gqlErr.Extensions = map[string]interface{}{"code": "INTERNAL"}
        }

        return gqlErr
    }
}
```

**Формат ответа клиенту:**

```json
{
  "errors": [{
    "message": "entry abc-123: not found",
    "path": ["dictionaryEntry"],
    "extensions": {
      "code": "NOT_FOUND"
    }
  }]
}
```

Для validation errors:

```json
{
  "errors": [{
    "message": "validation: 2 errors",
    "path": ["createEntryCustom"],
    "extensions": {
      "code": "VALIDATION",
      "fields": [
        {"field": "text", "message": "required"},
        {"field": "senses", "message": "at least one sense required"}
      ]
    }
  }]
}
```

---

#### `resolver/fieldresolvers.go` — Field Resolvers (DataLoaders)

```go
package resolver

import (
    "context"
    "fmt"

    "github.com/heartmarshall/myenglish-backend/internal/domain"
    "github.com/heartmarshall/myenglish-backend/internal/transport/graphql/dataloader"
)

// ---- DictionaryEntry field resolvers ----

func (r *dictionaryEntryResolver) Senses(ctx context.Context, obj *domain.Entry) ([]*domain.Sense, error) {
    if len(obj.Senses) > 0 {
        return obj.Senses, nil
    }
    loaders := dataloader.FromContext(ctx)
    if loaders == nil {
        return nil, fmt.Errorf("dataloaders not in context")
    }
    return loaders.SensesByEntryID.Load(ctx, obj.ID)()
}

func (r *dictionaryEntryResolver) Pronunciations(ctx context.Context, obj *domain.Entry) ([]*domain.RefPronunciation, error) {
    if len(obj.Pronunciations) > 0 {
        return obj.Pronunciations, nil
    }
    loaders := dataloader.FromContext(ctx)
    if loaders == nil {
        return nil, fmt.Errorf("dataloaders not in context")
    }
    return loaders.PronunciationsByEntryID.Load(ctx, obj.ID)()
}

func (r *dictionaryEntryResolver) CatalogImages(ctx context.Context, obj *domain.Entry) ([]*domain.RefImage, error) {
    if len(obj.CatalogImages) > 0 {
        return obj.CatalogImages, nil
    }
    loaders := dataloader.FromContext(ctx)
    if loaders == nil {
        return nil, fmt.Errorf("dataloaders not in context")
    }
    return loaders.CatalogImagesByEntryID.Load(ctx, obj.ID)()
}

func (r *dictionaryEntryResolver) UserImages(ctx context.Context, obj *domain.Entry) ([]*domain.UserImage, error) {
    if len(obj.UserImages) > 0 {
        return obj.UserImages, nil
    }
    loaders := dataloader.FromContext(ctx)
    if loaders == nil {
        return nil, fmt.Errorf("dataloaders not in context")
    }
    return loaders.UserImagesByEntryID.Load(ctx, obj.ID)()
}

func (r *dictionaryEntryResolver) Card(ctx context.Context, obj *domain.Entry) (*domain.Card, error) {
    if obj.Card != nil {
        return obj.Card, nil
    }
    loaders := dataloader.FromContext(ctx)
    if loaders == nil {
        return nil, fmt.Errorf("dataloaders not in context")
    }
    return loaders.CardByEntryID.Load(ctx, obj.ID)()
}

func (r *dictionaryEntryResolver) Topics(ctx context.Context, obj *domain.Entry) ([]*domain.Topic, error) {
    if len(obj.Topics) > 0 {
        return obj.Topics, nil
    }
    loaders := dataloader.FromContext(ctx)
    if loaders == nil {
        return nil, fmt.Errorf("dataloaders not in context")
    }
    return loaders.TopicsByEntryID.Load(ctx, obj.ID)()
}

// ---- Sense field resolvers ----

func (r *senseResolver) Translations(ctx context.Context, obj *domain.Sense) ([]*domain.Translation, error) {
    if len(obj.Translations) > 0 {
        return obj.Translations, nil
    }
    loaders := dataloader.FromContext(ctx)
    if loaders == nil {
        return nil, fmt.Errorf("dataloaders not in context")
    }
    return loaders.TranslationsBySenseID.Load(ctx, obj.ID)()
}

func (r *senseResolver) Examples(ctx context.Context, obj *domain.Sense) ([]*domain.Example, error) {
    if len(obj.Examples) > 0 {
        return obj.Examples, nil
    }
    loaders := dataloader.FromContext(ctx)
    if loaders == nil {
        return nil, fmt.Errorf("dataloaders not in context")
    }
    return loaders.ExamplesBySenseID.Load(ctx, obj.ID)()
}

// ---- User field resolvers ----

func (r *userResolver) Settings(ctx context.Context, obj *domain.User) (*domain.UserSettings, error) {
    return r.user.GetSettings(ctx)
}
```

**Паттерн field resolvers:**

1. Проверить struct field (не nil / не пустой) → вернуть напрямую
2. Получить DataLoaders из контекста
3. Вызвать `loader.Load(ctx, parentID)()` — batched загрузка

Данные загружены напрямую (struct field != nil) когда:
- `CreateEntryFromCatalog` / `CreateEntryCustom` возвращают полное дерево
- `GetEntry` предзагружает данные

Данные НЕ загружены (field == nil) когда:
- `FindEntries` возвращает только entries (дочерние — через DataLoaders)
- `GetStudyQueue` возвращает entries для очереди

---

#### Unit-тесты Error Presenter

| # | Тест | Assert |
|---|------|--------|
| 1 | `TestErrorPresenter_NotFound` | `domain.ErrNotFound` → code `NOT_FOUND` |
| 2 | `TestErrorPresenter_AlreadyExists` | `domain.ErrAlreadyExists` → code `ALREADY_EXISTS` |
| 3 | `TestErrorPresenter_Validation` | `*domain.ValidationError` → code `VALIDATION` + `fields` в extensions |
| 4 | `TestErrorPresenter_ValidationSingleField` | 1 field error → `fields` содержит 1 элемент |
| 5 | `TestErrorPresenter_Unauthorized` | `domain.ErrUnauthorized` → code `UNAUTHENTICATED` |
| 6 | `TestErrorPresenter_Forbidden` | `domain.ErrForbidden` → code `FORBIDDEN` |
| 7 | `TestErrorPresenter_Conflict` | `domain.ErrConflict` → code `CONFLICT` |
| 8 | `TestErrorPresenter_WrappedError` | `fmt.Errorf("op: %w", domain.ErrNotFound)` → code `NOT_FOUND` (unwrap работает) |
| 9 | `TestErrorPresenter_UnexpectedError` | Произвольная ошибка → code `INTERNAL`, message `"internal error"` |
| 10 | `TestErrorPresenter_UnexpectedError_NoLeakDetails` | Оригинальное сообщение ошибки НЕ попадает клиенту |

**Всего: 10 тест-кейсов**

---

**Acceptance criteria TASK-9.2:**
- [ ] `resolver/resolver.go` создан с 6 приватными интерфейсами (dictionary, content, study, topic, inbox, user)
- [ ] Конструктор `NewResolver` с 7 параметрами (log + 6 сервисов)
- [ ] Query(), Mutation(), DictionaryEntry(), Sense(), User() возвращают правильные resolver interfaces
- [ ] `errpresenter.go` маппит все 6 domain errors + `INTERNAL` fallback
- [ ] Validation errors содержат `fields` в extensions
- [ ] Unexpected errors логируются с `request_id`, клиенту — `"internal error"`
- [ ] `fieldresolvers.go` содержит 9 field resolvers (6 Entry + 2 Sense + 1 User)
- [ ] Field resolvers: struct field check → DataLoader fallback
- [ ] 10 unit-тестов error presenter проходят
- [ ] `go build ./...` компилируется

---

### TASK-9.2.1: Коррекция интерфейсов (после ревью сервисов)

> **Важно:** Ниже перечислены расхождения между интерфейсами из TASK-9.2 и реальными сигнатурами сервисов (Фазы 5–8). При реализации TASK-9.2 использовать **скорректированные** версии.

#### Коррекции `dictionaryService`

```go
type dictionaryService interface {
    SearchCatalog(ctx context.Context, query string, limit int) ([]domain.RefEntry, error)          // []T, не []*T
    PreviewRefEntry(ctx context.Context, text string) (*domain.RefEntry, error)
    CreateEntryFromCatalog(ctx context.Context, input dictionary.CreateFromCatalogInput) (*domain.Entry, error) // CreateFromCatalogInput
    CreateEntryCustom(ctx context.Context, input dictionary.CreateCustomInput) (*domain.Entry, error)          // CreateCustomInput
    FindEntries(ctx context.Context, input dictionary.FindInput) (*dictionary.FindResult, error)
    GetEntry(ctx context.Context, entryID uuid.UUID) (*domain.Entry, error)
    UpdateNotes(ctx context.Context, input dictionary.UpdateNotesInput) (*domain.Entry, error)
    DeleteEntry(ctx context.Context, entryID uuid.UUID) error
    FindDeletedEntries(ctx context.Context, limit, offset int) ([]domain.Entry, int, error) // []T + count
    RestoreEntry(ctx context.Context, entryID uuid.UUID) (*domain.Entry, error)
    BatchDeleteEntries(ctx context.Context, entryIDs []uuid.UUID) (*dictionary.BatchResult, error)
    ImportEntries(ctx context.Context, input dictionary.ImportInput) (*dictionary.ImportResult, error)
    ExportEntries(ctx context.Context) (*dictionary.ExportResult, error)
}
```

**Изменения:**
- `SearchCatalog` возвращает `[]domain.RefEntry` (value slice), не `[]*domain.RefEntry`
- `FindDeletedEntries` возвращает `([]domain.Entry, int, error)` — три значения, int = totalCount
- Имена input типов: `CreateFromCatalogInput`, `CreateCustomInput` (не `CreateEntryFromCatalogInput`)

#### Коррекции `studyService`

```go
type studyService interface {
    GetStudyQueue(ctx context.Context, input study.GetQueueInput) ([]*domain.Card, error)             // Cards, не Entries
    ReviewCard(ctx context.Context, input study.ReviewCardInput) (*domain.Card, error)                 // 1 return, не 2
    UndoReview(ctx context.Context, input study.UndoReviewInput) (*domain.Card, error)
    StartSession(ctx context.Context) (*domain.StudySession, error)
    FinishSession(ctx context.Context, input study.FinishSessionInput) (*domain.StudySession, error)
    AbandonSession(ctx context.Context) error                                                          // error only
    CreateCard(ctx context.Context, input study.CreateCardInput) (*domain.Card, error)
    DeleteCard(ctx context.Context, input study.DeleteCardInput) error
    BatchCreateCards(ctx context.Context, input study.BatchCreateCardsInput) (study.BatchCreateResult, error)  // value, не pointer
    GetDashboard(ctx context.Context) (domain.Dashboard, error)                                        // value, не pointer
    GetCardHistory(ctx context.Context, input study.GetCardHistoryInput) ([]*domain.ReviewLog, int, error) // + count
    GetCardStats(ctx context.Context, input study.GetCardHistoryInput) (domain.CardStats, error)       // GetCardHistoryInput, не cardID
}
```

**Изменения:**
- `GetStudyQueue` возвращает `[]*domain.Card`, НЕ `[]*domain.Entry` — resolver должен загрузить entries по `card.EntryID`
- `ReviewCard` возвращает `(*domain.Card, error)` — НЕ `(*domain.Card, *domain.ReviewLog, error)`
- `AbandonSession` возвращает `error`, не `(*domain.StudySession, error)`
- `BatchCreateCards` возвращает value `BatchCreateResult`, не pointer
- `GetDashboard` возвращает value `domain.Dashboard`, не pointer
- `GetCardHistory` возвращает 3 значения (+ count)
- `GetCardStats` принимает `GetCardHistoryInput`, не `cardID uuid.UUID`

#### Коррекция `contentService` — `AddSenseInput`

Реальный `content.AddSenseInput` не содержит `Examples` — только `EntryID`, `Definition`, `PartOfSpeech`, `CEFRLevel`, `Translations`. Примеры добавляются отдельно через `AddExample`.

#### Коррекции GraphQL-схемы (TASK-9.1)

1. **`CreateEntryFromCatalogInput`** — убрать `translationIds`, `exampleIds`, `pronunciationIds`, `imageIds`, `topicId`. Реальный сервис принимает только `refEntryId`, `senseIds`, `createCard`, `notes`.

Скорректированная версия:
```graphql
input CreateEntryFromCatalogInput {
  refEntryId: UUID!
  senseIds: [UUID!]!
  notes: String
  createCard: Boolean
}
```

2. **`ImportItemInput`** — привести к реальному формату сервиса (без вложенных senses):
```graphql
input ImportItemInput {
  text: String!
  translations: [String!]
  notes: String
  topicName: String
}
```
Убрать `ImportSenseInput` и `ImportExampleInput` — сервис не поддерживает вложенную структуру для импорта.

3. **`AddSenseInput`** — убрать `examples`:
```graphql
input AddSenseInput {
  entryId: UUID!
  definition: String
  partOfSpeech: PartOfSpeech
  cefrLevel: String
  translations: [String!]
}
```

4. **`studyQueue` query** — возвращает cards, но resolver загружает entries:
```graphql
extend type Query {
  """Очередь повторения: карточки → entries (с лимитом)."""
  studyQueue(limit: Int): [DictionaryEntry!]!
}
```
> Resolver получает `[]*domain.Card` от сервиса, затем загружает entries через DataLoader по `card.EntryID`.

5. **`ReviewCardPayload`** — убрать `reviewLog` (сервис не возвращает его):
```graphql
type ReviewCardPayload {
  card: Card!
}
```

6. **`AbandonSessionPayload`** — resolver получает только error, нужен отдельный запрос:
```graphql
type AbandonSessionPayload {
  success: Boolean!
}
```

7. **`ExportItem`** — добавить недостающие поля из реального `ExportResult`:
```graphql
type ExportItem {
  text: String!
  notes: String
  senses: [ExportSense!]!
  cardStatus: LearningStatus
  createdAt: DateTime!
}
```
Убрать `topicNames` из GraphQL (нет в сервисе). Добавить `cardStatus` и `createdAt`.

---

### TASK-9.3: Dictionary Resolvers

**Зависит от:** TASK-9.1, TASK-9.2

**Контекст:**
- `services/dictionary_service_spec_v4.md` — 13 операций
- `internal/service/dictionary/input.go` — input типы
- `internal/service/dictionary/result.go` — result типы
- `internal/service/dictionary/service.go` — метод-сигнатуры

**Что сделать:**

Реализовать все Dictionary resolvers (6 queries + 7 mutations) в `dictionary.resolvers.go`. Маппинг GraphQL inputs → service inputs. Конвертация результатов в payload types.

---

#### Таблица операций Dictionary

| # | GraphQL Operation | Resolver метод | Service метод | Auth | Ключевые маппинги |
|---|-------------------|----------------|---------------|------|-------------------|
| 1 | `searchCatalog(query, limit)` | Query | `SearchCatalog(ctx, query, limit)` | Нет | Прямая передача аргументов; default limit=10 |
| 2 | `previewRefEntry(text)` | Query | `PreviewRefEntry(ctx, text)` | Нет | Прямая передача |
| 3 | `dictionary(input)` | Query | `FindEntries(ctx, FindInput)` | Да | GraphQL DictionaryFilterInput → FindInput; результат → DictionaryConnection |
| 4 | `dictionaryEntry(id)` | Query | `GetEntry(ctx, id)` | Да | Прямой вызов; nil → graphql null |
| 5 | `deletedEntries(limit, offset)` | Query | `FindDeletedEntries(ctx, limit, offset)` | Да | Результат → DeletedEntriesList |
| 6 | `exportEntries` | Query | `ExportEntries(ctx)` | Да | Прямой результат |
| 7 | `createEntryFromCatalog(input)` | Mutation | `CreateEntryFromCatalog(ctx, input)` | Да | GraphQL input → CreateFromCatalogInput |
| 8 | `createEntryCustom(input)` | Mutation | `CreateEntryCustom(ctx, input)` | Да | GraphQL input → CreateCustomInput |
| 9 | `updateEntryNotes(input)` | Mutation | `UpdateNotes(ctx, input)` | Да | GraphQL input → UpdateNotesInput |
| 10 | `deleteEntry(id)` | Mutation | `DeleteEntry(ctx, id)` | Да | Возвращает DeleteEntryPayload с id |
| 11 | `restoreEntry(id)` | Mutation | `RestoreEntry(ctx, id)` | Да | Возвращает RestoreEntryPayload |
| 12 | `batchDeleteEntries(ids)` | Mutation | `BatchDeleteEntries(ctx, ids)` | Да | BatchResult → BatchDeletePayload |
| 13 | `importEntries(input)` | Mutation | `ImportEntries(ctx, input)` | Да | ImportInput → ImportPayload |

---

#### Примеры реализации (паттерны)

**Паттерн 1: Простой query без auth**

```go
// SearchCatalog — searchCatalog(query: String!, limit: Int): [RefEntry!]!
func (r *queryResolver) SearchCatalog(ctx context.Context, query string, limit *int) ([]*generated.RefEntry, error) {
    l := 10 // default
    if limit != nil {
        l = *limit
    }
    entries, err := r.dictionary.SearchCatalog(ctx, query, l)
    if err != nil {
        return nil, err
    }
    // []domain.RefEntry → []*domain.RefEntry для GraphQL
    result := make([]*domain.RefEntry, len(entries))
    for i := range entries {
        result[i] = &entries[i]
    }
    return result, nil
}
```

> **Примечание:** `SearchCatalog` возвращает `[]domain.RefEntry` (value slice). Resolver конвертирует в pointer slice для gqlgen. Автоматический autobind может обрабатывать это — проверить при генерации.

**Паттерн 2: Query с auth и маппингом результата в Connection**

```go
// Dictionary — dictionary(input: DictionaryFilterInput!): DictionaryConnection!
func (r *queryResolver) Dictionary(ctx context.Context, input generated.DictionaryFilterInput) (*generated.DictionaryConnection, error) {
    svcInput := dictionary.FindInput{
        Limit: 20, // default
    }

    if input.Search != nil {
        svcInput.Search = input.Search
    }
    if input.HasCard != nil {
        svcInput.HasCard = input.HasCard
    }
    if input.PartOfSpeech != nil {
        svcInput.PartOfSpeech = input.PartOfSpeech
    }
    if input.TopicID != nil {
        svcInput.TopicID = input.TopicID
    }
    if input.Status != nil {
        svcInput.Status = input.Status
    }
    if input.SortField != nil {
        svcInput.SortBy = string(*input.SortField)
    }
    if input.SortDirection != nil {
        svcInput.SortOrder = string(*input.SortDirection)
    }
    if input.First != nil {
        svcInput.Limit = *input.First
    }
    if input.After != nil {
        svcInput.Cursor = input.After
    }
    if input.Limit != nil {
        svcInput.Limit = *input.Limit
    }
    if input.Offset != nil {
        svcInput.Offset = input.Offset
    }

    result, err := r.dictionary.FindEntries(ctx, svcInput)
    if err != nil {
        return nil, err
    }

    edges := make([]*generated.DictionaryEdge, len(result.Entries))
    for i, entry := range result.Entries {
        e := entry // capture loop var
        edges[i] = &generated.DictionaryEdge{
            Node:   &e,
            Cursor: encodeCursor(entry.ID), // helper
        }
    }

    pageInfo := &generated.PageInfo{
        HasNextPage: result.HasNextPage,
    }
    if result.PageInfo != nil {
        pageInfo.StartCursor = result.PageInfo.StartCursor
        pageInfo.EndCursor = result.PageInfo.EndCursor
    }

    return &generated.DictionaryConnection{
        Edges:      edges,
        PageInfo:   pageInfo,
        TotalCount: result.TotalCount,
    }, nil
}
```

**Паттерн 3: Mutation с input mapping**

```go
// CreateEntryFromCatalog — createEntryFromCatalog(input): CreateEntryPayload!
func (r *mutationResolver) CreateEntryFromCatalog(ctx context.Context, input generated.CreateEntryFromCatalogInput) (*generated.CreateEntryPayload, error) {
    svcInput := dictionary.CreateFromCatalogInput{
        RefEntryID: input.RefEntryID,
        SenseIDs:   input.SenseIds,
        Notes:      input.Notes,
    }
    if input.CreateCard != nil {
        svcInput.CreateCard = *input.CreateCard
    }

    entry, err := r.dictionary.CreateEntryFromCatalog(ctx, svcInput)
    if err != nil {
        return nil, err
    }
    return &generated.CreateEntryPayload{Entry: entry}, nil
}
```

**Паттерн 4: Delete mutation (возврат ID)**

```go
// DeleteEntry — deleteEntry(id: UUID!): DeleteEntryPayload!
func (r *mutationResolver) DeleteEntry(ctx context.Context, id uuid.UUID) (*generated.DeleteEntryPayload, error) {
    if err := r.dictionary.DeleteEntry(ctx, id); err != nil {
        return nil, err
    }
    return &generated.DeleteEntryPayload{EntryID: id}, nil
}
```

**Паттерн 5: Batch operation с маппингом ошибок**

```go
// BatchDeleteEntries — batchDeleteEntries(ids: [UUID!]!): BatchDeletePayload!
func (r *mutationResolver) BatchDeleteEntries(ctx context.Context, ids []uuid.UUID) (*generated.BatchDeletePayload, error) {
    result, err := r.dictionary.BatchDeleteEntries(ctx, ids)
    if err != nil {
        return nil, err
    }

    batchErrors := make([]*generated.BatchError, len(result.Errors))
    for i, e := range result.Errors {
        batchErrors[i] = &generated.BatchError{
            ID:      e.EntryID,
            Message: e.Error,
        }
    }

    return &generated.BatchDeletePayload{
        DeletedCount: result.Deleted,
        Errors:       batchErrors,
    }, nil
}
```

---

#### Вспомогательные функции

```go
// resolver/helpers.go

package resolver

import (
    "encoding/base64"

    "github.com/google/uuid"
)

// encodeCursor кодирует UUID entry в cursor для pagination.
func encodeCursor(id uuid.UUID) string {
    return base64.StdEncoding.EncodeToString([]byte(id.String()))
}

// decodeCursor декодирует cursor обратно в UUID string.
func decodeCursor(cursor string) (string, error) {
    b, err := base64.StdEncoding.DecodeString(cursor)
    if err != nil {
        return "", err
    }
    return string(b), nil
}
```

> **Примечание:** Cursor encoding/decoding может быть реализован по-другому в зависимости от формата курсора в `dictionary.FindResult`. Если `FindResult.PageInfo` уже содержит закодированные курсоры — использовать их напрямую.

---

#### Unit-тесты Dictionary Resolvers

| # | Тест | Scenario | Assert |
|---|------|----------|--------|
| 1 | `TestDictionary_SearchCatalog_Success` | Запрос с query и limit | Возвращает `[]*RefEntry`, правильный маппинг |
| 2 | `TestDictionary_SearchCatalog_DefaultLimit` | Запрос без limit | Service вызван с limit=10 |
| 3 | `TestDictionary_SearchCatalog_Error` | Service возвращает ошибку | Ошибка проброшена |
| 4 | `TestDictionary_PreviewRefEntry_Success` | Запрос с text | Возвращает `*RefEntry` |
| 5 | `TestDictionary_PreviewRefEntry_NotFound` | Service → nil | Возвращает nil (no error) |
| 6 | `TestDictionary_Dictionary_Success` | Полный фильтр | FindInput маппится правильно, Connection построен |
| 7 | `TestDictionary_Dictionary_DefaultValues` | Пустой фильтр | Default limit=20, остальное nil |
| 8 | `TestDictionary_Dictionary_CursorPagination` | first + after | svcInput.Limit = first, svcInput.Cursor = after |
| 9 | `TestDictionary_Dictionary_OffsetPagination` | limit + offset | svcInput.Limit = limit, svcInput.Offset = offset |
| 10 | `TestDictionary_DictionaryEntry_Success` | ID существует | Возвращает entry |
| 11 | `TestDictionary_DictionaryEntry_NotFound` | Service → ErrNotFound | Error с кодом NOT_FOUND |
| 12 | `TestDictionary_DeletedEntries_Success` | limit=10, offset=0 | Возвращает DeletedEntriesList с totalCount |
| 13 | `TestDictionary_ExportEntries_Success` | Экспорт | Возвращает ExportResult |
| 14 | `TestDictionary_CreateEntryFromCatalog_Success` | Полный input | Input маппится → CreateFromCatalogInput, payload содержит entry |
| 15 | `TestDictionary_CreateEntryFromCatalog_ValidationError` | Service → ErrValidation | Error с кодом VALIDATION |
| 16 | `TestDictionary_CreateEntryCustom_Success` | Полный input | Input маппится → CreateCustomInput |
| 17 | `TestDictionary_CreateEntryCustom_DuplicateText` | Service → ErrAlreadyExists | Error с кодом ALREADY_EXISTS |
| 18 | `TestDictionary_UpdateEntryNotes_Success` | entryId + notes | Input → UpdateNotesInput |
| 19 | `TestDictionary_DeleteEntry_Success` | ID | Service.DeleteEntry вызван, payload.EntryID = input ID |
| 20 | `TestDictionary_RestoreEntry_Success` | ID | Возвращает RestoreEntryPayload с entry |
| 21 | `TestDictionary_BatchDeleteEntries_Success` | [id1, id2] | Маппинг BatchResult → BatchDeletePayload |
| 22 | `TestDictionary_BatchDeleteEntries_PartialErrors` | 1 success + 1 error | DeletedCount=1, Errors=[1] |
| 23 | `TestDictionary_ImportEntries_Success` | items | ImportResult → ImportPayload |
| 24 | `TestDictionary_ImportEntries_PartialErrors` | 1 imported + 1 skipped | Все поля маппятся |

**Всего: 24 тест-кейса**

**Паттерн тестирования:**

```go
func TestDictionary_CreateEntryFromCatalog_Success(t *testing.T) {
    // Arrange: создать мок dictionaryService с ожидаемым вызовом
    dictMock := &dictionaryServiceMock{
        CreateEntryFromCatalogFunc: func(ctx context.Context, input dictionary.CreateFromCatalogInput) (*domain.Entry, error) {
            // Assert: проверить что input маппится правильно
            assert.Equal(t, refEntryID, input.RefEntryID)
            assert.Equal(t, senseIDs, input.SenseIDs)
            return &domain.Entry{ID: entryID, Text: "test"}, nil
        },
    }
    r := NewResolver(slog.Default(), dictMock, nil, nil, nil, nil, nil)

    // Act: вызвать resolver
    payload, err := r.Mutation().CreateEntryFromCatalog(ctx, gqlInput)

    // Assert: проверить payload
    require.NoError(t, err)
    assert.Equal(t, entryID, payload.Entry.ID)
}
```

> **Примечание:** Моки генерируются через `moq` из приватных интерфейсов в `resolver_test.go`:
> ```
> //go:generate moq -out dict_service_mock_test.go -pkg resolver dictionaryService
> //go:generate moq -out content_service_mock_test.go -pkg resolver contentService
> //go:generate moq -out study_service_mock_test.go -pkg resolver studyService
> //go:generate moq -out topic_service_mock_test.go -pkg resolver topicService
> //go:generate moq -out inbox_service_mock_test.go -pkg resolver inboxService
> //go:generate moq -out user_service_mock_test.go -pkg resolver userService
> ```

---

**Acceptance criteria TASK-9.3:**
- [ ] `dictionary.resolvers.go` содержит 13 resolver-методов (6 queries + 7 mutations)
- [ ] `resolver/helpers.go` с cursor encoding
- [ ] `searchCatalog` и `previewRefEntry` не требуют auth
- [ ] Остальные 11 операций требуют auth (userID из ctx)
- [ ] `FindEntries` результат корректно маппится в `DictionaryConnection` с edges, pageInfo, totalCount
- [ ] `FindDeletedEntries` результат маппится в `DeletedEntriesList`
- [ ] BatchDelete/Import маппят partial errors в payload
- [ ] 24 unit-теста проходят
- [ ] Все моки сгенерированы через `moq`
- [ ] `go build ./...` компилируется

---

### TASK-9.4: Content Resolvers

**Зависит от:** TASK-9.1, TASK-9.2

**Контекст:**
- `services/content_service_spec_v4.md` — 14 операций
- `internal/service/content/input.go` — input типы
- `internal/service/content/` — sense.go, translation.go, userimage.go

**Что сделать:**

Реализовать все Content resolvers (14 mutations) в `content.resolvers.go`. Все операции требуют авторизации. Content не имеет собственных queries — все данные загружаются через field resolvers (DataLoaders) от DictionaryEntry/Sense.

---

#### Таблица операций Content

| # | GraphQL Mutation | Service метод | Input маппинг |
|---|------------------|---------------|---------------|
| 1 | `addSense(input)` | `AddSense(ctx, AddSenseInput)` | entryId→EntryID, definition, partOfSpeech, cefrLevel, translations |
| 2 | `updateSense(input)` | `UpdateSense(ctx, UpdateSenseInput)` | senseId→SenseID, definition, partOfSpeech, cefrLevel |
| 3 | `deleteSense(id)` | `DeleteSense(ctx, senseID)` | Прямой UUID |
| 4 | `reorderSenses(input)` | `ReorderSenses(ctx, ReorderSensesInput)` | entryId→EntryID, items→[]ReorderItem |
| 5 | `addTranslation(input)` | `AddTranslation(ctx, AddTranslationInput)` | senseId→SenseID, text→Text |
| 6 | `updateTranslation(input)` | `UpdateTranslation(ctx, UpdateTranslationInput)` | translationId→TranslationID, text→Text |
| 7 | `deleteTranslation(id)` | `DeleteTranslation(ctx, translationID)` | Прямой UUID |
| 8 | `reorderTranslations(input)` | `ReorderTranslations(ctx, ReorderTranslationsInput)` | senseId→SenseID, items→[]ReorderItem |
| 9 | `addExample(input)` | `AddExample(ctx, AddExampleInput)` | senseId→SenseID, sentence, translation |
| 10 | `updateExample(input)` | `UpdateExample(ctx, UpdateExampleInput)` | exampleId→ExampleID, sentence, translation |
| 11 | `deleteExample(id)` | `DeleteExample(ctx, exampleID)` | Прямой UUID |
| 12 | `reorderExamples(input)` | `ReorderExamples(ctx, ReorderExamplesInput)` | senseId→SenseID, items→[]ReorderItem |
| 13 | `addUserImage(input)` | `AddUserImage(ctx, AddUserImageInput)` | entryId→EntryID, url→URL, caption |
| 14 | `deleteUserImage(id)` | `DeleteUserImage(ctx, imageID)` | Прямой UUID |

---

#### Примеры реализации

**Паттерн: CRUD mutation с input mapping**

```go
// AddSense — addSense(input: AddSenseInput!): AddSensePayload!
func (r *mutationResolver) AddSense(ctx context.Context, input generated.AddSenseInput) (*generated.AddSensePayload, error) {
    svcInput := content.AddSenseInput{
        EntryID:     input.EntryID,
        Definition:  input.Definition,
        PartOfSpeech: input.PartOfSpeech,
        CEFRLevel:   input.CefrLevel,
    }
    if input.Translations != nil {
        svcInput.Translations = input.Translations
    }

    sense, err := r.content.AddSense(ctx, svcInput)
    if err != nil {
        return nil, err
    }
    return &generated.AddSensePayload{Sense: sense}, nil
}
```

**Паттерн: Reorder mutation с items mapping**

```go
// ReorderSenses — reorderSenses(input: ReorderSensesInput!): ReorderPayload!
func (r *mutationResolver) ReorderSenses(ctx context.Context, input generated.ReorderSensesInput) (*generated.ReorderPayload, error) {
    items := make([]content.ReorderItem, len(input.Items))
    for i, item := range input.Items {
        items[i] = content.ReorderItem{
            ID:       item.ID,
            Position: item.Position,
        }
    }

    err := r.content.ReorderSenses(ctx, content.ReorderSensesInput{
        EntryID: input.EntryID,
        Items:   items,
    })
    if err != nil {
        return nil, err
    }
    return &generated.ReorderPayload{Success: true}, nil
}
```

**Паттерн: Delete mutation (возврат ID)**

```go
// DeleteSense — deleteSense(id: UUID!): DeleteSensePayload!
func (r *mutationResolver) DeleteSense(ctx context.Context, id uuid.UUID) (*generated.DeleteSensePayload, error) {
    if err := r.content.DeleteSense(ctx, id); err != nil {
        return nil, err
    }
    return &generated.DeleteSensePayload{SenseID: id}, nil
}
```

---

#### Unit-тесты Content Resolvers

| # | Тест | Scenario | Assert |
|---|------|----------|--------|
| 1 | `TestContent_AddSense_Success` | Полный input | Input маппится, sense в payload |
| 2 | `TestContent_AddSense_ValidationError` | Нет entryId | ErrValidation propagated |
| 3 | `TestContent_AddSense_EntryNotFound` | Service → ErrNotFound | NOT_FOUND |
| 4 | `TestContent_UpdateSense_Success` | senseId + definition | Input маппится |
| 5 | `TestContent_DeleteSense_Success` | senseId | DeleteSense вызван, payload.SenseID = input |
| 6 | `TestContent_DeleteSense_NotFound` | Service → ErrNotFound | NOT_FOUND |
| 7 | `TestContent_ReorderSenses_Success` | 3 items | ReorderItem маппинг, success=true |
| 8 | `TestContent_AddTranslation_Success` | senseId + text | Input маппится |
| 9 | `TestContent_UpdateTranslation_Success` | translationId + text | Input маппится |
| 10 | `TestContent_DeleteTranslation_Success` | translationId | Payload.TranslationID = id |
| 11 | `TestContent_ReorderTranslations_Success` | 2 items | Items маппятся корректно |
| 12 | `TestContent_AddExample_Success` | senseId + sentence | Input маппится |
| 13 | `TestContent_AddExample_WithTranslation` | + translation | Translation в input не nil |
| 14 | `TestContent_UpdateExample_Success` | exampleId + sentence | Input маппится |
| 15 | `TestContent_DeleteExample_Success` | exampleId | Payload.ExampleID = id |
| 16 | `TestContent_ReorderExamples_Success` | 2 items | Items маппятся корректно |
| 17 | `TestContent_AddUserImage_Success` | entryId + url | Input маппится |
| 18 | `TestContent_AddUserImage_WithCaption` | + caption | Caption в input не nil |
| 19 | `TestContent_DeleteUserImage_Success` | imageId | Payload.ImageID = id |
| 20 | `TestContent_DeleteUserImage_NotFound` | Service → ErrNotFound | NOT_FOUND |

**Всего: 20 тест-кейсов**

---

**Acceptance criteria TASK-9.4:**
- [ ] `content.resolvers.go` содержит 14 mutation-методов
- [ ] Все 14 мутаций требуют auth
- [ ] CRUD маппинг: GraphQL input → content.XxxInput
- [ ] Reorder маппит `[]ReorderItemInput` → `[]content.ReorderItem`
- [ ] Delete mutations возвращают ID удалённого объекта
- [ ] 20 unit-тестов проходят
- [ ] `go build ./...` компилируется

---

### TASK-9.5: Study Resolvers

**Зависит от:** TASK-9.1, TASK-9.2

**Контекст:**
- `services/study_service_spec_v4_v1.1.md` — 12 операций
- `internal/service/study/input.go` — input типы
- `internal/service/study/result.go` — result типы
- `internal/service/study/service.go` — метод-сигнатуры

**Что сделать:**

Реализовать все Study resolvers (4 queries + 8 mutations) в `study.resolvers.go`. Все операции требуют авторизации.

**Ключевая особенность:** `GetStudyQueue` возвращает `[]*domain.Card`, а GraphQL query `studyQueue` должен вернуть `[DictionaryEntry!]!`. Resolver загружает entries по `card.EntryID` через DataLoader.

---

#### Таблица операций Study

| # | GraphQL Operation | Service метод | Ключевые особенности |
|---|-------------------|---------------|---------------------|
| 1 | `studyQueue(limit)` | `GetStudyQueue(ctx, GetQueueInput)` | **Cards → Entries**: загрузить entries через DataLoader |
| 2 | `dashboard` | `GetDashboard(ctx)` | Возвращает value `domain.Dashboard`, не pointer |
| 3 | `cardHistory(input)` | `GetCardHistory(ctx, GetCardHistoryInput)` | Возвращает `([]*ReviewLog, int, error)` — count не используется в текущей схеме |
| 4 | `cardStats(cardId)` | `GetCardStats(ctx, GetCardHistoryInput)` | Принимает `GetCardHistoryInput`, не `cardID` |
| 5 | `reviewCard(input)` | `ReviewCard(ctx, ReviewCardInput)` | Возвращает `(*Card, error)` — без ReviewLog |
| 6 | `undoReview(cardId)` | `UndoReview(ctx, UndoReviewInput)` | UndoReviewInput.CardID = cardId |
| 7 | `createCard(entryId)` | `CreateCard(ctx, CreateCardInput)` | CreateCardInput.EntryID = entryId |
| 8 | `deleteCard(id)` | `DeleteCard(ctx, DeleteCardInput)` | DeleteCardInput.CardID = id |
| 9 | `batchCreateCards(entryIds)` | `BatchCreateCards(ctx, BatchCreateCardsInput)` | Возвращает value `BatchCreateResult` |
| 10 | `startStudySession` | `StartSession(ctx)` | Нет input |
| 11 | `finishStudySession(input)` | `FinishSession(ctx, FinishSessionInput)` | SessionID из input |
| 12 | `abandonStudySession` | `AbandonSession(ctx)` | Возвращает `error` — payload `{success: true}` |

---

#### Примеры реализации

**Паттерн: studyQueue — Cards → Entries через DataLoader**

```go
// StudyQueue — studyQueue(limit: Int): [DictionaryEntry!]!
func (r *queryResolver) StudyQueue(ctx context.Context, limit *int) ([]*domain.Entry, error) {
    l := 20 // default
    if limit != nil {
        l = *limit
    }

    cards, err := r.study.GetStudyQueue(ctx, study.GetQueueInput{Limit: l})
    if err != nil {
        return nil, err
    }

    if len(cards) == 0 {
        return []*domain.Entry{}, nil
    }

    // Собираем entryIDs из карточек
    entryIDs := make([]uuid.UUID, len(cards))
    for i, card := range cards {
        entryIDs[i] = card.EntryID
    }

    // Загружаем entries через dictionaryService (batch)
    // Или через DataLoader если доступен entryByID loader
    entries := make([]*domain.Entry, 0, len(cards))
    for _, card := range cards {
        entry, err := r.dictionary.GetEntry(ctx, card.EntryID)
        if err != nil {
            continue // skip entries that fail to load
        }
        entries = append(entries, entry)
    }

    return entries, nil
}
```

> **Примечание:** В идеале использовать DataLoader для batch-загрузки entries по ID. Если `EntriesByIDs` loader есть в dataloader — использовать его. Если нет — вызов `GetEntry` в цикле допустим на начальном этапе (N запросов, но каждый фильтруется по userID и кэшируется в per-request cache DataLoader'а).

**Паттерн: Dashboard (value return)**

```go
// Dashboard — dashboard: Dashboard!
func (r *queryResolver) Dashboard(ctx context.Context) (*generated.Dashboard, error) {
    d, err := r.study.GetDashboard(ctx)
    if err != nil {
        return nil, err
    }
    return &generated.Dashboard{
        DueCount:      d.DueCount,
        NewCount:      d.NewCount,
        ReviewedToday: d.ReviewedToday,
        Streak:        d.Streak,
        StatusCounts:  &d.StatusCounts,
        OverdueCount:  d.OverdueCount,
        ActiveSession: d.ActiveSession,
    }, nil
}
```

> **Примечание:** Если gqlgen autobind корректно маппит `domain.Dashboard` → GraphQL `Dashboard`, можно вернуть `&d` напрямую. Проверить при генерации.

**Паттерн: cardStats (input wrapping)**

```go
// CardStats — cardStats(cardId: UUID!): CardStats!
func (r *queryResolver) CardStats(ctx context.Context, cardID uuid.UUID) (*generated.CardStats, error) {
    stats, err := r.study.GetCardStats(ctx, study.GetCardHistoryInput{
        CardID: cardID,
    })
    if err != nil {
        return nil, err
    }
    return &stats, nil
}
```

**Паттерн: AbandonSession (error-only return)**

```go
// AbandonStudySession — abandonStudySession: AbandonSessionPayload!
func (r *mutationResolver) AbandonStudySession(ctx context.Context) (*generated.AbandonSessionPayload, error) {
    if err := r.study.AbandonSession(ctx); err != nil {
        return nil, err
    }
    return &generated.AbandonSessionPayload{Success: true}, nil
}
```

**Паттерн: BatchCreateCards (value result mapping)**

```go
// BatchCreateCards — batchCreateCards(entryIds: [UUID!]!): BatchCreateCardsPayload!
func (r *mutationResolver) BatchCreateCards(ctx context.Context, entryIds []uuid.UUID) (*generated.BatchCreateCardsPayload, error) {
    result, err := r.study.BatchCreateCards(ctx, study.BatchCreateCardsInput{
        EntryIDs: entryIds,
    })
    if err != nil {
        return nil, err
    }

    batchErrors := make([]*generated.BatchCreateCardError, len(result.Errors))
    for i, e := range result.Errors {
        batchErrors[i] = &generated.BatchCreateCardError{
            EntryID: e.EntryID,
            Message: e.Reason,
        }
    }

    return &generated.BatchCreateCardsPayload{
        CreatedCount: result.Created,
        SkippedCount: result.SkippedExisting + result.SkippedNoSenses,
        Errors:       batchErrors,
    }, nil
}
```

---

#### Unit-тесты Study Resolvers

| # | Тест | Scenario | Assert |
|---|------|----------|--------|
| 1 | `TestStudy_StudyQueue_Success` | limit=10, 3 cards | Cards → Entries загружены, 3 entry в ответе |
| 2 | `TestStudy_StudyQueue_DefaultLimit` | limit=nil | GetQueueInput.Limit=20 |
| 3 | `TestStudy_StudyQueue_Empty` | 0 cards | Пустой slice, no error |
| 4 | `TestStudy_Dashboard_Success` | Полный dashboard | Все поля маппятся |
| 5 | `TestStudy_Dashboard_WithActiveSession` | + activeSession != nil | ActiveSession в ответе |
| 6 | `TestStudy_CardHistory_Success` | cardId + limit | GetCardHistoryInput маппится |
| 7 | `TestStudy_CardStats_Success` | cardId | GetCardHistoryInput{CardID: id} |
| 8 | `TestStudy_CardStats_NotFound` | Service → ErrNotFound | NOT_FOUND |
| 9 | `TestStudy_ReviewCard_Success` | cardId + grade | Input маппится, card в payload |
| 10 | `TestStudy_ReviewCard_WithDuration` | + durationMs | DurationMs в input |
| 11 | `TestStudy_UndoReview_Success` | cardId | UndoReviewInput.CardID = cardId |
| 12 | `TestStudy_UndoReview_NotFound` | Service → ErrNotFound | NOT_FOUND |
| 13 | `TestStudy_CreateCard_Success` | entryId | CreateCardInput.EntryID = entryId |
| 14 | `TestStudy_CreateCard_AlreadyExists` | Service → ErrAlreadyExists | ALREADY_EXISTS |
| 15 | `TestStudy_DeleteCard_Success` | cardId | DeleteCardInput.CardID = id |
| 16 | `TestStudy_BatchCreateCards_Success` | [id1, id2, id3] | Маппинг BatchCreateResult |
| 17 | `TestStudy_BatchCreateCards_PartialErrors` | 1 created + 1 skipped | SkippedCount, Errors |
| 18 | `TestStudy_StartStudySession_Success` | Нет input | Session в payload |
| 19 | `TestStudy_StartStudySession_Conflict` | Service → ErrConflict | CONFLICT (уже есть active session) |
| 20 | `TestStudy_FinishStudySession_Success` | sessionId | FinishSessionInput маппится |
| 21 | `TestStudy_AbandonStudySession_Success` | Нет input | Success=true |
| 22 | `TestStudy_AbandonStudySession_NotFound` | Service → ErrNotFound | NOT_FOUND (нет active session) |

**Всего: 22 тест-кейса**

---

**Acceptance criteria TASK-9.5:**
- [ ] `study.resolvers.go` содержит 12 resolver-методов (4 queries + 8 mutations)
- [ ] Все операции требуют auth
- [ ] `studyQueue`: Cards → Entries маппинг реализован
- [ ] `dashboard`: value result корректно конвертируется
- [ ] `cardStats`: cardID wrapping в `GetCardHistoryInput`
- [ ] `abandonStudySession`: error-only → `{success: true}`
- [ ] `batchCreateCards`: `SkippedExisting + SkippedNoSenses` → `skippedCount`
- [ ] 22 unit-теста проходят
- [ ] `go build ./...` компилируется

---

### TASK-9.6: Organization и User Resolvers

**Зависит от:** TASK-9.1, TASK-9.2

**Контекст:**
- `services/topic_service_spec_v4.md` — 7 операций
- `services/inbox_service_spec_v4.md` — 5 операций
- `internal/service/topic/input.go`, `internal/service/inbox/input.go` — input типы
- `internal/service/topic/service.go`, `internal/service/inbox/service.go` — метод-сигнатуры

**Что сделать:**

Реализовать Organization resolvers (3 queries + 9 mutations: Topic + Inbox) в `organization.resolvers.go` и User resolvers (1 query + 1 mutation) в `user.resolvers.go`. Все операции требуют авторизации.

---

#### Таблица операций Topic

| # | GraphQL Operation | Service метод | Input маппинг |
|---|-------------------|---------------|---------------|
| 1 | `topics` | `ListTopics(ctx)` | Нет input |
| 2 | `createTopic(input)` | `CreateTopic(ctx, CreateTopicInput)` | name→Name, description→Description |
| 3 | `updateTopic(input)` | `UpdateTopic(ctx, UpdateTopicInput)` | topicId→TopicID, name→Name, description→Description |
| 4 | `deleteTopic(id)` | `DeleteTopic(ctx, DeleteTopicInput)` | id → DeleteTopicInput.TopicID |
| 5 | `linkEntryToTopic(input)` | `LinkEntry(ctx, LinkEntryInput)` | topicId→TopicID, entryId→EntryID |
| 6 | `unlinkEntryFromTopic(input)` | `UnlinkEntry(ctx, UnlinkEntryInput)` | topicId→TopicID, entryId→EntryID |
| 7 | `batchLinkEntriesToTopic(input)` | `BatchLinkEntries(ctx, BatchLinkEntriesInput)` | topicId→TopicID, entryIds→EntryIDs |

#### Таблица операций Inbox

| # | GraphQL Operation | Service метод | Input маппинг |
|---|-------------------|---------------|---------------|
| 8 | `inboxItems(limit, offset)` | `ListItems(ctx, ListItemsInput)` | limit→Limit, offset→Offset; returns (items, count, err) |
| 9 | `inboxItem(id)` | `GetItem(ctx, itemID)` | Прямой UUID |
| 10 | `createInboxItem(input)` | `CreateItem(ctx, CreateItemInput)` | text→Text, context→Context |
| 11 | `deleteInboxItem(id)` | `DeleteItem(ctx, DeleteItemInput)` | id → DeleteItemInput.ItemID |
| 12 | `clearInbox` | `DeleteAll(ctx)` | Нет input; returns (count, err) |

#### Таблица операций User

| # | GraphQL Operation | Service метод | Input маппинг |
|---|-------------------|---------------|---------------|
| 13 | `me` | `GetProfile(ctx)` | Нет input |
| 14 | `updateSettings(input)` | `UpdateSettings(ctx, input)` | Маппинг зависит от Фазы 4 |

---

#### Примеры реализации

**Паттерн: Список без pagination**

```go
// Topics — topics: [Topic!]!
func (r *queryResolver) Topics(ctx context.Context) ([]*domain.Topic, error) {
    return r.topic.ListTopics(ctx)
}
```

**Паттерн: Delete через input struct**

```go
// DeleteTopic — deleteTopic(id: UUID!): DeleteTopicPayload!
func (r *mutationResolver) DeleteTopic(ctx context.Context, id uuid.UUID) (*generated.DeleteTopicPayload, error) {
    if err := r.topic.DeleteTopic(ctx, topic.DeleteTopicInput{TopicID: id}); err != nil {
        return nil, err
    }
    return &generated.DeleteTopicPayload{TopicID: id}, nil
}
```

**Паттерн: Inbox list с count**

```go
// InboxItems — inboxItems(limit: Int, offset: Int): InboxItemList!
func (r *queryResolver) InboxItems(ctx context.Context, limit *int, offset *int) (*generated.InboxItemList, error) {
    svcInput := inbox.ListItemsInput{
        Limit:  20, // default
        Offset: 0,
    }
    if limit != nil {
        svcInput.Limit = *limit
    }
    if offset != nil {
        svcInput.Offset = *offset
    }

    items, total, err := r.inbox.ListItems(ctx, svcInput)
    if err != nil {
        return nil, err
    }
    return &generated.InboxItemList{
        Items:      items,
        TotalCount: total,
    }, nil
}
```

**Паттерн: ClearInbox (count return)**

```go
// ClearInbox — clearInbox: ClearInboxPayload!
func (r *mutationResolver) ClearInbox(ctx context.Context) (*generated.ClearInboxPayload, error) {
    count, err := r.inbox.DeleteAll(ctx)
    if err != nil {
        return nil, err
    }
    return &generated.ClearInboxPayload{DeletedCount: count}, nil
}
```

**Паттерн: BatchLink с result mapping**

```go
// BatchLinkEntriesToTopic — batchLinkEntriesToTopic(input): BatchLinkPayload!
func (r *mutationResolver) BatchLinkEntriesToTopic(ctx context.Context, input generated.BatchLinkEntriesInput) (*generated.BatchLinkPayload, error) {
    result, err := r.topic.BatchLinkEntries(ctx, topic.BatchLinkEntriesInput{
        TopicID:  input.TopicID,
        EntryIDs: input.EntryIds,
    })
    if err != nil {
        return nil, err
    }
    return &generated.BatchLinkPayload{
        Linked:  result.Linked,
        Skipped: result.Skipped,
    }, nil
}
```

**Паттерн: User — me query**

```go
// Me — me: User!
func (r *queryResolver) Me(ctx context.Context) (*domain.User, error) {
    return r.user.GetProfile(ctx)
}
```

> **Примечание:** `User.settings` загружается через field resolver (TASK-9.2, `userResolver.Settings`), не inline.

**Паттерн: UpdateSettings (предварительный)**

```go
// UpdateSettings — updateSettings(input: UpdateSettingsInput!): UpdateSettingsPayload!
func (r *mutationResolver) UpdateSettings(ctx context.Context, input generated.UpdateSettingsInput) (*generated.UpdateSettingsPayload, error) {
    // TODO: точный маппинг зависит от user.UpdateSettingsInput (Фаза 4)
    settings, err := r.user.UpdateSettings(ctx, input)
    if err != nil {
        return nil, err
    }
    return &generated.UpdateSettingsPayload{Settings: settings}, nil
}
```

---

#### Unit-тесты Organization + User Resolvers

| # | Тест | Scenario | Assert |
|---|------|----------|--------|
| 1 | `TestOrg_Topics_Success` | 3 topics | ListTopics вызван, 3 topics в ответе |
| 2 | `TestOrg_Topics_Empty` | 0 topics | Пустой slice |
| 3 | `TestOrg_CreateTopic_Success` | name + description | Input маппится, topic в payload |
| 4 | `TestOrg_CreateTopic_DuplicateName` | Service → ErrAlreadyExists | ALREADY_EXISTS |
| 5 | `TestOrg_UpdateTopic_Success` | topicId + name | Input маппится |
| 6 | `TestOrg_UpdateTopic_NotFound` | Service → ErrNotFound | NOT_FOUND |
| 7 | `TestOrg_DeleteTopic_Success` | topicId | DeleteTopicInput.TopicID = id |
| 8 | `TestOrg_LinkEntryToTopic_Success` | topicId + entryId | Input маппится, success=true |
| 9 | `TestOrg_LinkEntryToTopic_AlreadyExists` | Service → ErrAlreadyExists | ALREADY_EXISTS |
| 10 | `TestOrg_UnlinkEntryFromTopic_Success` | topicId + entryId | Input маппится |
| 11 | `TestOrg_BatchLinkEntriesToTopic_Success` | topicId + [id1, id2] | Result маппится: linked, skipped |
| 12 | `TestOrg_InboxItems_Success` | limit=10, offset=0 | ListItemsInput маппится, InboxItemList |
| 13 | `TestOrg_InboxItems_DefaultValues` | limit=nil, offset=nil | Default limit=20, offset=0 |
| 14 | `TestOrg_InboxItem_Success` | itemId | GetItem вызван |
| 15 | `TestOrg_InboxItem_NotFound` | Service → ErrNotFound | NOT_FOUND |
| 16 | `TestOrg_CreateInboxItem_Success` | text + context | Input маппится |
| 17 | `TestOrg_DeleteInboxItem_Success` | itemId | DeleteItemInput.ItemID = id |
| 18 | `TestOrg_ClearInbox_Success` | 5 items deleted | DeletedCount=5 |
| 19 | `TestOrg_ClearInbox_Empty` | 0 items | DeletedCount=0 |
| 20 | `TestUser_Me_Success` | Профиль | GetProfile вызван |
| 21 | `TestUser_Me_Unauthorized` | Service → ErrUnauthorized | UNAUTHENTICATED |
| 22 | `TestUser_UpdateSettings_Success` | newCardsPerDay=20 | Input маппится, settings в payload |

**Всего: 22 тест-кейса**

---

**Acceptance criteria TASK-9.6:**
- [ ] `organization.resolvers.go` содержит 12 resolver-методов (3 queries + 9 mutations)
- [ ] `user.resolvers.go` содержит 2 resolver-метода (1 query + 1 mutation)
- [ ] Все операции требуют auth
- [ ] Topic: CRUD + link/unlink/batch маппинг
- [ ] Inbox: ListItems → InboxItemList с totalCount
- [ ] ClearInbox: count → DeletedCount
- [ ] User.settings: загружается через field resolver (не inline)
- [ ] 22 unit-теста проходят
- [ ] `go build ./...` компилируется

---

## Граф зависимостей задач

```
TASK-9.1 (gqlgen + schema)
    │
    └──→ TASK-9.2 (resolver foundation + error presenter + field resolvers)
              │
              ├──→ TASK-9.2.1 (коррекции интерфейсов) ──→ TASK-9.3, 9.4, 9.5, 9.6
              │
              ├──→ TASK-9.3 (Dictionary resolvers)     ─┐
              ├──→ TASK-9.4 (Content resolvers)         ├── Параллельно
              ├──→ TASK-9.5 (Study resolvers)           │
              └──→ TASK-9.6 (Organization + User)      ─┘
```

---

## Порядок выполнения (волны)

| Волна | Задачи | Параллельность | Описание |
|-------|--------|----------------|----------|
| 1 | TASK-9.1 | — | gqlgen setup, GraphQL-схема, генерация кода |
| 2 | TASK-9.2, TASK-9.2.1 | — | Resolver foundation, error presenter, field resolvers, коррекции |
| 3 | TASK-9.3, TASK-9.4, TASK-9.5, TASK-9.6 | **Параллельно** | Все resolver-файлы независимы, используют общий `resolver.go` |

**Волна 3** — 4 задачи можно выполнять параллельно, т.к. каждая пишет в свой `.resolvers.go` файл и тестирует свой набор сервисных операций.

---

## Сводка тестов фазы

| Задача | Файл | Тест-кейсов |
|--------|------|-------------|
| TASK-9.2 | `errpresenter_test.go` | 10 |
| TASK-9.3 | `resolver/resolver_test.go` (dictionary) | 24 |
| TASK-9.4 | `resolver/resolver_test.go` (content) | 20 |
| TASK-9.5 | `resolver/resolver_test.go` (study) | 22 |
| TASK-9.6 | `resolver/resolver_test.go` (org + user) | 22 |
| **Итого** | | **98** |

> Все resolver-тесты могут быть в одном `resolver_test.go` или разбиты по файлам (`dictionary_test.go`, `content_test.go` и т.д.) — по усмотрению.

---

## Файлы фазы (итого)

| Действие | Путь | Задача |
|----------|------|--------|
| Создать | `internal/transport/graphql/gqlgen.yml` | 9.1 |
| Создать | `internal/transport/graphql/generate.go` | 9.1 |
| Создать | `internal/transport/graphql/model/scalars.go` | 9.1 |
| Создать | `internal/transport/graphql/schema/schema.graphql` | 9.1 |
| Создать | `internal/transport/graphql/schema/enums.graphql` | 9.1 |
| Создать | `internal/transport/graphql/schema/pagination.graphql` | 9.1 |
| Создать | `internal/transport/graphql/schema/dictionary.graphql` | 9.1 |
| Создать | `internal/transport/graphql/schema/content.graphql` | 9.1 |
| Создать | `internal/transport/graphql/schema/study.graphql` | 9.1 |
| Создать | `internal/transport/graphql/schema/organization.graphql` | 9.1 |
| Создать | `internal/transport/graphql/schema/user.graphql` | 9.1 |
| Создать | `internal/transport/graphql/generated/generated.go` | 9.1 (сгенерирован) |
| Создать | `internal/transport/graphql/generated/models_gen.go` | 9.1 (сгенерирован) |
| Создать | `internal/transport/graphql/resolver/resolver.go` | 9.2 |
| Создать | `internal/transport/graphql/errpresenter.go` | 9.2 |
| Создать | `internal/transport/graphql/errpresenter_test.go` | 9.2 |
| Создать | `internal/transport/graphql/resolver/fieldresolvers.go` | 9.2 |
| Создать | `internal/transport/graphql/resolver/helpers.go` | 9.3 |
| Заменить | `internal/transport/graphql/resolver/dictionary.resolvers.go` | 9.3 (заменяет stub) |
| Заменить | `internal/transport/graphql/resolver/content.resolvers.go` | 9.4 (заменяет stub) |
| Заменить | `internal/transport/graphql/resolver/study.resolvers.go` | 9.5 (заменяет stub) |
| Заменить | `internal/transport/graphql/resolver/organization.resolvers.go` | 9.6 (заменяет stub) |
| Заменить | `internal/transport/graphql/resolver/user.resolvers.go` | 9.6 (заменяет stub) |
| Создать | `internal/transport/graphql/resolver/resolver_test.go` | 9.3–9.6 |
| Создать | `internal/transport/graphql/resolver/*_mock_test.go` | 9.3 (сгенерированы moq) |
| Изменить | `backend_v4/tools.go` | 9.1 (добавить gqlgen) |
| Изменить | `backend_v4/go.mod` | 9.1 (gqlgen dependency) |

**Итого:** ~14 новых файлов, ~5 замена stubs, ~6 сгенерированных, ~2 изменения существующих.

---

## Чеклист завершённости фазы

- [ ] **TASK-9.1:** gqlgen сконфигурирован, 8 schema-файлов, генерация проходит, stubs компилируются
- [ ] **TASK-9.2:** Resolver struct с 6 интерфейсами, error presenter с 10 тестами, 9 field resolvers
- [ ] **TASK-9.2.1:** Интерфейсы скорректированы под реальные сигнатуры сервисов
- [ ] **TASK-9.3:** 13 dictionary resolvers + 24 теста
- [ ] **TASK-9.4:** 14 content resolvers + 20 тестов
- [ ] **TASK-9.5:** 12 study resolvers + 22 теста
- [ ] **TASK-9.6:** 14 organization + user resolvers + 22 тестов
- [ ] `go generate ./internal/transport/graphql/...` — без ошибок
- [ ] `go build ./...` — компилируется
- [ ] `go test ./internal/transport/graphql/...` — 98 тестов проходят
- [ ] `go vet ./internal/transport/graphql/...` — без warnings
- [ ] Coverage ≥80% для resolver пакета
- [ ] Все 53 GraphQL-операции реализованы (14 queries + 39 mutations)
- [ ] Error presenter покрывает все domain errors
- [ ] Field resolvers используют DataLoaders с struct-field fallback