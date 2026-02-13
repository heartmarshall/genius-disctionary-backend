# MyEnglish Backend v4 — Inbox Service Specification

> **Статус:** Draft v1.0
> **Дата:** 2026-02-13
> **Зависимости:** code_conventions_v4.md, data_model_v4.md, repo_layer_spec_v4.md, service_layer_spec_v4.md
> **Покрываемые сценарии:** I1–I5 из business_scenarios_v4.md

---

## 1. Ответственность

Inbox Service — сервис быстрого захвата слов и фраз «на потом». Inbox — это промежуточный буфер между моментом, когда пользователь встретил незнакомое слово, и моментом, когда он готов осознанно добавить его в словарь.

Отвечает за:

- Быстрое сохранение текста (слово/фраза) с опциональным контекстом
- Просмотр inbox items (paginated, newest first)
- Удаление отдельного item
- Очистка inbox целиком (delete all)
- Контроль лимита items на пользователя

Inbox Service **не** отвечает за: конвертацию item в entry (это отдельный вызов Dictionary.CreateEntry на стороне клиента), поиск по каталогу (RefCatalog Service), автоматический парсинг текста в senses/translations.

**Ключевой принцип:** Inbox — это «заметки на салфетке». Минимум friction при записи, минимум структуры. Пользователь читает статью, встречает слово, одним тапом сохраняет его с контекстом (предложение, в котором встретилось). Потом, когда есть время, открывает inbox и решает: добавить в словарь, отбросить или оставить на потом.

---

## 2. Структура пакета

```
internal/service/inbox/
├── service.go          # Struct, конструктор, приватные интерфейсы
├── input.go            # Input-структуры с Validate()
└── service_test.go     # Unit-тесты с моками
```

Сервис компактный — не требует дополнительных файлов.

---

## 3. Зависимости (приватные интерфейсы)

```go
// service/inbox/service.go
package inbox

import (
    "context"
    "log/slog"

    "github.com/google/uuid"
    "myenglish/internal/domain"
)

type inboxRepo interface {
    Create(ctx context.Context, userID uuid.UUID, item *domain.InboxItem) (*domain.InboxItem, error)
    GetByID(ctx context.Context, userID, itemID uuid.UUID) (*domain.InboxItem, error)
    List(ctx context.Context, userID uuid.UUID, limit, offset int) ([]*domain.InboxItem, int, error)
    Delete(ctx context.Context, userID, itemID uuid.UUID) error
    DeleteAll(ctx context.Context, userID uuid.UUID) (int, error)
    Count(ctx context.Context, userID uuid.UUID) (int, error)
}

type Service struct {
    inbox inboxRepo
    log   *slog.Logger
}

func NewService(
    log *slog.Logger,
    inbox inboxRepo,
) *Service {
    return &Service{
        inbox: inbox,
        log:   log.With("service", "inbox"),
    }
}
```

### 3.1. Почему нет зависимостей

Inbox — standalone сервис без межсервисных зависимостей:

- **Нет auditLogger** — inbox items не аудитируются. Это легковесные заметки, не бизнес-критичные данные. Audit на каждый захват слова раздул бы audit_log без пользы.
- **Нет txManager** — каждая операция затрагивает ровно одну таблицу. Транзакции не нужны.
- **Нет entryRepo / refCatalogService** — конвертация item → entry выполняется клиентом как два отдельных вызова (просмотр item → вызов Dictionary.CreateEntry → удаление item). Inbox не знает про словарь.

---

## 4. Domain-модель

```go
// domain/inbox.go

type InboxItem struct {
    ID        uuid.UUID
    UserID    uuid.UUID
    Text      string      // Слово или фраза — то, что пользователь хочет запомнить
    Context   *string     // Опционально: предложение, где встретилось слово
    CreatedAt time.Time
}
```

Модель намеренно минимальна. Нет статусов, тегов, приоритетов — всё это усложнение для «заметки на салфетке». Если пользователю нужна структура — он добавляет слово в словарь.

---

## 5. Операции

### 5.1. CreateItem (I1)

**Описание:** Быстрое сохранение слова/фразы. Главное требование — минимум friction: текст + опциональный контекст, ничего больше.

**Input:**
```go
type CreateItemInput struct {
    Text    string
    Context *string
}

func (i CreateItemInput) Validate() error {
    var errs []domain.FieldError

    text := strings.TrimSpace(i.Text)
    if text == "" {
        errs = append(errs, domain.FieldError{Field: "text", Message: "required"})
    }
    if len(text) > 500 {
        errs = append(errs, domain.FieldError{Field: "text", Message: "max 500 characters"})
    }

    if i.Context != nil {
        ctx := strings.TrimSpace(*i.Context)
        if len(ctx) > 2000 {
            errs = append(errs, domain.FieldError{Field: "context", Message: "max 2000 characters"})
        }
    }

    if len(errs) > 0 {
        return &domain.ValidationError{Errors: errs}
    }
    return nil
}
```

**Flow:**

1. `userID := UserIDFromCtx(ctx)` → если нет → `ErrUnauthorized`
2. `input.Validate()` → если ошибка → `ValidationError`
3. Trim input: `text = strings.TrimSpace(input.Text)`, `context = trimOrNil(input.Context)`
4. Проверить лимит: `count := inboxRepo.Count(ctx, userID)` → если ≥ 500 → `ValidationError("inbox", "inbox is full (max 500 items)")`
5. Создать item: `item := inboxRepo.Create(ctx, userID, &domain.InboxItem{Text: text, Context: context})`
6. Логировать INFO: `user_id`, `item_id`, `text` (первые 50 символов)
7. Вернуть `item`

**Corner cases:**

| Ситуация | Поведение |
|----------|-----------|
| Текст из одних пробелов | `ValidationError("text", "required")` после trim |
| Текст > 500 символов | `ValidationError` |
| Context > 2000 символов | `ValidationError` |
| Context пустая строка | Трактуется как nil (нет контекста) |
| Inbox полон (500 items) | `ValidationError("inbox", "inbox is full")` |
| Дублирующий текст | **Допускается** — пользователь может сохранить одно слово дважды с разным контекстом |
| Unicode, emoji, спецсимволы | Допускаются — text хранится как есть (без нормализации) |

**Почему дубли допускаются:** Inbox — это не словарь. Пользователь может встретить «set» в контексте «set the table» и в контексте «chess set» — это два разных значения одного слова. Дедупликация здесь вредна.

**Почему нет нормализации:** В отличие от `entries.text_normalized`, inbox хранит текст as-is. Пользователь может записать фразу «I couldn't figure it out», и нормализация её испортит. Inbox — свободная форма, словарь — структурированная.

---

### 5.2. ListItems (I2)

**Описание:** Пагинированный список inbox items, newest first.

**Input:**
```go
type ListItemsInput struct {
    Limit  int  // default 50, max 200
    Offset int
}

func (i ListItemsInput) Validate() error {
    var errs []domain.FieldError
    if i.Limit < 0 {
        errs = append(errs, domain.FieldError{Field: "limit", Message: "must be non-negative"})
    }
    if i.Limit > 200 {
        errs = append(errs, domain.FieldError{Field: "limit", Message: "max 200"})
    }
    if i.Offset < 0 {
        errs = append(errs, domain.FieldError{Field: "offset", Message: "must be non-negative"})
    }
    if len(errs) > 0 {
        return &domain.ValidationError{Errors: errs}
    }
    return nil
}
```

**Flow:**

1. `userID := UserIDFromCtx(ctx)` → если нет → `ErrUnauthorized`
2. `input.Validate()`. Если limit = 0 → default 50.
3. `items, totalCount := inboxRepo.List(ctx, userID, limit, offset)`
4. Вернуть `items, totalCount`

**Порядок:** `ORDER BY created_at DESC` — newest first. Это единственный разумный порядок для inbox: последние сохранённые слова наверху, как в мессенджере.

**Corner cases:**

| Ситуация | Поведение |
|----------|-----------|
| Пустой inbox | Пустой slice, totalCount = 0 |
| offset > totalCount | Пустой slice, totalCount = N (корректный count, пустые данные) |

---

### 5.3. GetItem (дополнительная операция)

**Описание:** Получение одного item по ID. Нужно для просмотра контекста перед обработкой.

**Flow:**

1. `userID := UserIDFromCtx(ctx)`
2. `item := inboxRepo.GetByID(ctx, userID, itemID)` → если нет → `ErrNotFound`
3. Вернуть `item`

---

### 5.4. DeleteItem (I4)

**Описание:** Удаление одного inbox item.

**Input:**
```go
type DeleteItemInput struct {
    ItemID uuid.UUID
}

func (i DeleteItemInput) Validate() error {
    if i.ItemID == uuid.Nil {
        return domain.NewValidationError("item_id", "required")
    }
    return nil
}
```

**Flow:**

1. `userID := UserIDFromCtx(ctx)` → если нет → `ErrUnauthorized`
2. `input.Validate()`
3. `inboxRepo.Delete(ctx, userID, input.ItemID)` → если не найден → `ErrNotFound`
4. Логировать INFO: `user_id`, `item_id`
5. Вернуть `nil`

**Почему не GetByID перед Delete:** Repo.Delete сам проверяет существование (по affected rows). Если rows affected = 0 → `ErrNotFound`. Это один запрос вместо двух.

**Corner cases:**

| Ситуация | Поведение |
|----------|-----------|
| Item не существует | `ErrNotFound` |
| Чужой item | `ErrNotFound` (repo фильтрует по user_id) |

---

### 5.5. DeleteAll (I5)

**Описание:** Очистка inbox — удаление всех items пользователя.

**Flow:**

1. `userID := UserIDFromCtx(ctx)` → если нет → `ErrUnauthorized`
2. `deletedCount := inboxRepo.DeleteAll(ctx, userID)`
3. Логировать INFO: `user_id`, `deleted_count`
4. Вернуть `deletedCount`

**Corner cases:**

| Ситуация | Поведение |
|----------|-----------|
| Пустой inbox | deletedCount = 0, не ошибка |

**Без подтверждения на стороне сервиса.** Confirmation dialog — ответственность клиента. Сервис выполняет операцию безусловно. Данные удаляются необратимо (inbox items не имеют soft delete).

---

## 6. Обработка inbox item (I3) — пользовательский сценарий

Обработка inbox item — это **не одна атомарная операция**, а последовательность отдельных действий пользователя. Сервер не оркестрирует этот flow, клиент делает отдельные вызовы:

```
1. Пользователь открывает inbox item       → Inbox.GetItem(itemID)
2. Видит текст и контекст
3. Решает добавить в словарь               → RefCatalog.Search(text)  // найти в каталоге
                                           → RefCatalog.GetTree(refEntryID)  // загрузить senses
                                           → Dictionary.CreateEntry(...)  // добавить в словарь
4. Удаляет item из inbox                   → Inbox.DeleteItem(itemID)
```

**Почему не атомарно:** Между шагами 2 и 4 пользователь принимает решения: какие senses выбрать, добавлять ли карточку, какие переводы оставить. Это интерактивный процесс, который не сводится к одному API-вызову. Если создание entry упадёт — item остаётся в inbox, пользователь может попробовать снова. Если пользователь не удалит item после добавления в словарь — ничего страшного, item просто останется.

**Альтернатива (не реализуем на MVP):** QuickAdd — одна операция «добавить первый sense из каталога + создать entry + удалить item». Это удобно для простых слов, но требует принятия решений за пользователя (какой sense выбрать?) и межсервисной оркестрации. Post-MVP.

---

## 7. Error Scenarios (сводная таблица)

| Операция | Ошибка | Тип | HTTP-аналог |
|----------|--------|-----|-------------|
| Все | Нет userID в ctx | `ErrUnauthorized` | 401 |
| CreateItem | text пустой | `ValidationError` | 400 |
| CreateItem | text > 500 символов | `ValidationError` | 400 |
| CreateItem | context > 2000 символов | `ValidationError` | 400 |
| CreateItem | Inbox полон (≥ 500) | `ValidationError` | 400 |
| GetItem | Item не найден | `ErrNotFound` | 404 |
| GetItem | Чужой item | `ErrNotFound` | 404 |
| ListItems | Невалидный limit | `ValidationError` | 400 |
| ListItems | Невалидный offset | `ValidationError` | 400 |
| DeleteItem | item_id = nil | `ValidationError` | 400 |
| DeleteItem | Item не найден | `ErrNotFound` | 404 |
| DeleteItem | Чужой item | `ErrNotFound` | 404 |
| DeleteAll | — | Всегда OK (0 удалённых — не ошибка) | — |

---

## 8. Валидация (сводная)

| Input | Поле | Правило |
|-------|------|---------|
| CreateItemInput | text | required, trimmed, ≤ 500 символов |
| CreateItemInput | context | optional, если present: trimmed, ≤ 2000 символов, пустая строка → nil |
| ListItemsInput | limit | ≥ 0, ≤ 200, default 50 |
| ListItemsInput | offset | ≥ 0 |
| DeleteItemInput | item_id | required, не Nil UUID |

---

## 9. Аудит

**Inbox items не аудитируются.** Обоснование:

- Inbox — временные заметки, а не бизнес-критичные данные
- Items создаются и удаляются часто (десятки в день у активного пользователя)
- Аудит каждого создания/удаления раздувает audit_log без аналитической ценности
- Нет update-операций (item immutable после создания)

Если потребуется аналитика по inbox (сколько items в среднем до обработки, conversion rate inbox → entry), это лучше решать через отдельную метрику/событие, а не через audit_log.

---

## 10. Лимиты

| Ресурс | Лимит | Где проверяется | Обоснование |
|--------|-------|----------------|-------------|
| Items на пользователя | 500 | CreateItem (перед созданием) | Inbox — буфер, не хранилище. 500 необработанных слов — сигнал, что пользователь не пользуется inbox по назначению |
| Text length | 500 символов | Validate() | Слово/короткая фраза. Длинные тексты — не для inbox |
| Context length | 2000 символов | Validate() | Контекстное предложение + пара соседних предложений |
| List limit | 200 per request | Validate() | Стандартное ограничение пагинации |

---

## 11. Тестирование

### 11.1. Service Unit Tests (service_test.go)

| # | Тест | Категория | Что проверяем |
|---|------|-----------|---------------|
| 1 | `TestCreateItem_Success` | Happy path | Item создан с text и context |
| 2 | `TestCreateItem_WithoutContext` | Happy path | Item создан без context (nil) |
| 3 | `TestCreateItem_EmptyText` | Validation | Пустой text → ValidationError |
| 4 | `TestCreateItem_WhitespaceOnlyText` | Validation | Text из пробелов → ValidationError после trim |
| 5 | `TestCreateItem_TextTooLong` | Validation | > 500 символов → ValidationError |
| 6 | `TestCreateItem_ContextTooLong` | Validation | > 2000 символов → ValidationError |
| 7 | `TestCreateItem_EmptyContextToNil` | Normalization | Context = "" → сохраняется как nil |
| 8 | `TestCreateItem_TextTrimmed` | Normalization | "  hello  " → "hello" |
| 9 | `TestCreateItem_InboxFull` | Limits | 500 items → ValidationError |
| 10 | `TestCreateItem_DuplicateAllowed` | Business | Одинаковый text допускается |
| 11 | `TestCreateItem_Unauthorized` | Auth | Нет userID → ErrUnauthorized |
| 12 | `TestListItems_Success` | Happy path | Список items, newest first |
| 13 | `TestListItems_Empty` | Edge case | Пустой inbox → пустой slice, totalCount = 0 |
| 14 | `TestListItems_Pagination` | Pagination | offset + limit корректно работают |
| 15 | `TestListItems_DefaultLimit` | Defaults | limit = 0 → 50 |
| 16 | `TestListItems_InvalidLimit` | Validation | limit = -1 → ValidationError |
| 17 | `TestListItems_Unauthorized` | Auth | Нет userID → ErrUnauthorized |
| 18 | `TestGetItem_Success` | Happy path | Item загружен |
| 19 | `TestGetItem_NotFound` | Not found | Несуществующий ID → ErrNotFound |
| 20 | `TestGetItem_WrongUser` | Auth | Чужой item → ErrNotFound |
| 21 | `TestGetItem_Unauthorized` | Auth | Нет userID → ErrUnauthorized |
| 22 | `TestDeleteItem_Success` | Happy path | Item удалён |
| 23 | `TestDeleteItem_NotFound` | Not found | Несуществующий ID → ErrNotFound |
| 24 | `TestDeleteItem_WrongUser` | Auth | Чужой item → ErrNotFound |
| 25 | `TestDeleteItem_NilID` | Validation | ItemID = nil → ValidationError |
| 26 | `TestDeleteItem_Unauthorized` | Auth | Нет userID → ErrUnauthorized |
| 27 | `TestDeleteAll_Success` | Happy path | Все items удалены, returns count |
| 28 | `TestDeleteAll_EmptyInbox` | Edge case | Пустой inbox → deletedCount = 0, не ошибка |
| 29 | `TestDeleteAll_Unauthorized` | Auth | Нет userID → ErrUnauthorized |

---

## 12. Обоснование решений

### 12.1. Почему inbox items immutable?

После создания item нельзя редактировать (нет Update). Обоснование:

- Inbox — «заметка на салфетке». Если нужно исправить — удали и создай заново.
- Отсутствие update упрощает модель: нет diff, нет audit, нет conflict resolution.
- Реальный use case для edit inbox item крайне редок — пользователь скорее удалит и пересоздаст.

### 12.2. Почему нет soft delete?

Inbox items удаляются физически (hard delete). Обоснование:

- Inbox — буфер, а не хранилище. Удалённый item не нужен никому и никогда.
- Нет сценария «восстановить удалённый inbox item» — если удалил по ошибке, быстрее пересоздать.
- Soft delete добавляет фильтрацию `WHERE deleted_at IS NULL` ко всем запросам без пользы.

### 12.3. Почему нет поиска?

Inbox не поддерживает поиск (search/filter). Обоснование:

- 500 items — это максимум. При таком объёме scrolling + pagination достаточно.
- Inbox — не словарь. Если пользователь хочет найти слово — пусть ищет в словаре.
- Поиск добавляет ILIKE/trgm индексы на таблицу, где это не оправдано.

### 12.4. Почему нет сортировки?

Единственный порядок — `created_at DESC` (newest first). Обоснование:

- Inbox — стек (LIFO): последнее сохранённое — первое обработанное.
- Альтернативная сортировка (по тексту, по длине) не имеет пользовательского смысла.
- Один фиксированный порядок = один индекс `(user_id, created_at DESC)`, уже есть в DDL.

---

## 13. Развитие post-MVP

| Фича | Описание | Приоритет |
|------|----------|-----------|
| QuickAdd | Одна кнопка «добавить в словарь» — берёт первый sense из каталога, создаёт entry + карточку, удаляет item | Высокий |
| Source URL | Опциональное поле: откуда было слово (URL статьи, название книги) | Средний |
| Теги / цвета | Быстрая пометка items (red = urgent, blue = interesting) | Низкий |
| Batch delete | Удаление выбранных items (а не всех или одного) | Средний |
| Auto-suggest | При просмотре item показать результаты RefCatalog.Search — «возможно, вы имели в виду...» | Высокий |
| Share to inbox | Deep link / share extension для мобильного приложения — выделил слово в браузере, нажал Share → MyEnglish → попало в inbox | Высокий |
