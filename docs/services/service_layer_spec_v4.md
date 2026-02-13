# MyEnglish Backend v4 — Service Layer Specification

> **Статус:** Draft v1.0
> **Дата:** 2026-02-12
> **Зависимости:** code_conventions_v4.md, data_model_v4.md, repo_layer_spec_v4.md

Этот документ описывает **архитектуру, паттерны и правила** сервисного слоя. Детальные спецификации каждого сервиса (операции, corner cases, валидация) — в отдельных документах.

---

## 1. Роль сервисного слоя

Сервис — единственное место, где живёт бизнес-логика. Ответственность:

- Валидация бизнес-правил (лимиты, уникальность, допустимые переходы состояний)
- Оркестрация вызовов к репозиториям и внешним провайдерам
- Управление транзакциями (через TxManager)
- Аудит мутирующих операций
- Логирование значимых бизнес-событий

Сервис **не** отвечает за: парсинг HTTP/GraphQL, аутентификацию (middleware), маппинг SQL → domain (repo).

---

## 2. Структура пакетов

```
internal/service/
├── auth/              # OAuth, JWT, refresh, logout
├── dictionary/        # CRUD entries, поиск, фильтрация
├── content/           # CRUD senses, translations, examples, user images
├── study/             # Study queue, review, SRS algorithm, dashboard
├── inbox/             # Быстрый захват слов
├── topic/             # Категоризация entries
├── refcatalog/        # Reference Catalog: поиск + заполнение из внешних API
└── user/              # Профиль, настройки
```

Каждый пакет содержит:

| Файл | Содержимое |
|------|-----------|
| `service.go` | Struct сервиса, конструктор, приватные интерфейсы зависимостей |
| `input.go` | Input-структуры с методами `Validate() error` |
| `service_test.go` | Unit-тесты с моками |

Дополнительные файлы по необходимости: `mapper.go`, и т.д.

---

## 3. Паттерны

### 3.1. Интерфейсы зависимостей

Каждый сервис определяет **приватные** интерфейсы в `service.go`. Минимальные — только то, что сервису реально нужно. Связывание с реализациями — в `main.go` через duck typing.

```
// service/dictionary/service.go

type entryRepo interface {
    GetByID(ctx, userID, id) → *Entry, error
    Create(ctx, userID, ...) → *Entry, error
    // только нужные методы
}

type Service struct {
    entries entryRepo
    audit   auditLogger
    tx      txManager
}
```

### 3.2. Input и валидация

Каждая мутирующая операция принимает input-структуру. Input определяется в `input.go` и имеет метод `Validate() error`.

Правила валидации:

- `Validate()` собирает **все** ошибки (не возвращает при первой)
- Возвращает `*domain.ValidationError` (unwraps → `domain.ErrValidation`)
- Transport-слой валидирует формат (парсинг, типы). Service валидирует бизнес-правила (лимиты, уникальность).
- `user_id` **никогда** не валидируется в сервисе — приходит из middleware через context, считается доверенным

### 3.3. User ID

Каждый public метод сервиса начинается с `UserIDFromCtx(ctx)`. Если userID отсутствует — `domain.ErrUnauthorized`. Исключения: методы, не требующие аутентификации (Auth.Login, Auth.ValidateToken, RefCatalog.Search).

### 3.4. Транзакции

Мутации, затрагивающие несколько таблиц, оборачиваются в `TxManager.RunInTx`. Правила:

- Audit record пишется **внутри** той же транзакции
- **Никаких внешних вызовов** (HTTP, gRPC) внутри транзакции — данные подготавливаются до начала tx
- Если ошибка внутри RunInTx — fn возвращает error, TxManager откатывает
- Операции чтения **не** оборачиваются в транзакции

### 3.5. Error handling

Сервис оборачивает ошибки с бизнес-контекстом, но **не** маскирует domain-ошибки:

- Repo вернул `ErrNotFound` → сервис прокидывает как есть (или оборачивает: `fmt.Errorf("entry: %w", err)`)
- Repo вернул `ErrAlreadyExists` → сервис может вернуть как есть или преобразовать в `ValidationError` с понятным сообщением
- Неожиданная ошибка → логировать ERROR, вернуть как есть

Ошибка логируется **один раз** — там, где она впервые обнаружена. Если прокидывается выше — не дублировать.

### 3.6. Логирование

| Уровень | Что |
|---------|-----|
| INFO | Создание/удаление сущностей, значимые бизнес-события |
| WARN | Нет карточек в очереди, лимит близок, нетипичные ситуации |
| ERROR | Ошибки от внешних API, неожиданные ошибки инфраструктуры |

Каждый лог включает `user_id` и `request_id` из контекста.

---

## 4. Аудит

### 4.1. Что аудитируем

Все **мутирующие** операции над основными сущностями:

| EntityType | Аудитируемые actions |
|------------|---------------------|
| ENTRY | create, update (notes, text), delete |
| SENSE | create, update, delete |
| CARD | create, update (SRS review) |
| USER (settings) | update |

Translations, examples — аудитируются как UPDATE на parent SENSE (чтобы не раздувать audit log).

### 4.2. Где и как

Audit record создаётся **внутри транзакции** с основной операцией. Каждый сервис определяет интерфейс `auditLogger` с единственным методом `Log(ctx, record)`.

### 4.3. Формат changes

CREATE: `{"field": {"new": value}}` для значимых полей.

UPDATE: `{"field": {"old": oldValue, "new": newValue}}` — только изменённые поля.

DELETE: `{"text": {"old": "abandon"}}` — идентифицирующие поля.

Каждый сервис строит changes map самостоятельно.

---

## 5. Application-Level Limits

Лимиты проверяются **сервисом** перед мутацией. Repo предоставляет count-методы.

| Ресурс | Лимит | Проверяет |
|--------|-------|-----------|
| Entries на пользователя | 10 000 | DictionaryService |
| Senses на entry | 20 | ContentService |
| Translations на sense | 20 | ContentService |
| Examples на sense | 50 | ContentService |
| Topics на пользователя | 100 | TopicService |
| Inbox items на пользователя | 500 | InboxService |
| New cards в день | user_settings.new_cards_per_day | StudyService |
| Reviews в день | user_settings.reviews_per_day | StudyService |

При превышении — `domain.ValidationError` с сообщением "limit reached".

---

## 6. Карта сервисов

### 6.1. Ответственности

| Сервис | Ответственность |
|--------|----------------|
| **AuthService** | OAuth login (Google/Apple), JWT access+refresh пара с token rotation, logout, ValidateToken для middleware |
| **DictionaryService** | CRUD entries, поиск с фильтрами (Squirrel), добавление слов из Reference Catalog, soft delete/restore |
| **ContentService** | CRUD senses/translations/examples/user images, ownership chain (sense→entry→user), reorder, лимиты на дочерние сущности |
| **StudyService** | Study queue с daily limits, ReviewCard с SRS algorithm, dashboard (due count, streak, stats) |
| **InboxService** | Быстрый захват слов (text + context), list/delete. Конвертация в entry — через DictionaryService отдельным вызовом |
| **TopicService** | CRUD topics, link/unlink entries ↔ topics |
| **RefCatalogService** | GetOrFetchEntry (check DB → fetch from external API → upsert), fuzzy search по каталогу |
| **UserService** | Профиль (name, avatar), настройки (cards per day, timezone и т.д.) |

### 6.2. Зависимости между сервисами

```
auth ──────────────────────────── (standalone)
user ──────────────────────────── (standalone)
inbox ─────────────────────────── (standalone)
topic ─────────────────────────── (standalone)
content ───────────────────────── (standalone, ownership check через entryRepo)
study ─────────────────────────── (standalone, reads settings через settingsRepo)
refcatalog ──→ externalProvider ─ (external dependency)
dictionary ──→ refcatalog ─────── (единственная inter-service dependency)
```

Только **одна** межсервисная зависимость: `DictionaryService → RefCatalogService`. DictionaryService определяет интерфейс `refCatalogService` с методом `GetOrFetchEntry`.

Все остальные сервисы получают данные через свои repo-интерфейсы.

### 6.3. External Providers

RefCatalogService зависит от `externalProvider` — интерфейс для FreeDictionary, Google Translate и других внешних API. Реализации живут в `internal/adapter/provider/`.

Требования к провайдерам:
- Timeout: 10 секунд на HTTP-вызов
- Retry: 1 retry с backoff (500ms) при 5xx или timeout
- Rate limiting: уважать лимиты провайдеров
- Тесты: mock HTTP server, не реальные API

---

## 7. Тестирование

### 7.1. Подход

**Unit-тесты** с моками зависимостей. Все интерфейсы (repos, txManager, providers) мокаются через `testify/mock`. Сервис тестируется изолированно, без БД.

### 7.2. Mock TxManager

Для unit-тестов TxManager мокается так: `RunInTx(ctx, fn)` просто вызывает `fn(ctx)` без реальной транзакции. Это позволяет тестировать бизнес-логику и проверять, что правильные repo-методы вызваны в правильном порядке.

### 7.3. Категории тестов

Для **каждого** сервиса:

| Категория | Что проверяем |
|-----------|--------------|
| Happy path | Все операции с корректным input |
| Validation | Невалидный input → ValidationError с корректными FieldErrors |
| Not found | Несуществующий ID → ErrNotFound |
| Already exists | Дубликат → ErrAlreadyExists |
| Unauthorized | Нет userID в ctx → ErrUnauthorized |
| Limits | Превышение лимита → ValidationError |
| Audit | Audit record создаётся с правильными полями и action |
| Transaction | При ошибке в середине — fn возвращает error (TxManager откатит) |

### 7.4. SRS Algorithm

SRS algorithm (в study service) — **чистая функция**. Тестируется отдельно, **table-driven tests** с минимум 30 кейсами, покрывающими все переходы состояний и boundary conditions.

### 7.5. Правила

- Каждый тест — один сценарий
- Мок repo настраивается per-test через `.On().Return()`
- Тесты не обращаются к БД
- Тесты не зависят друг от друга
- Ошибка логируется в тесте через `t.Errorf`, не через mock assertions на логгер

---

## 8. Hard Delete Job

Периодическая задача очистки данных. Не является сервисом — отдельный компонент (scheduler), запускаемый при старте приложения.

| Задача | Что чистит | Периодичность |
|--------|-----------|---------------|
| Hard delete entries | entries с deleted_at > 30 days (CASCADE удаляет дочерние) | Ежедневно |
| Cleanup tokens | Expired и revoked refresh tokens | Ежедневно |
| Cleanup audit | audit_log старше 1 года | Еженедельно |

Реализация: горутина с `time.Ticker` или external cron. Вызывает repo-методы напрямую (без service layer).
