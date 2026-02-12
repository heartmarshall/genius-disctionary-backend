# MyEnglish Backend v4 — Repository Layer Specification


Этот документ описывает **требования, паттерны и corner cases** для database-слоя (`internal/adapter/postgres/`). Он не содержит готового кода — разработчик свободен в выборе конкретной реализации, если она соответствует описанным здесь правилам.

Полный DDL всех таблиц, индексов, triggers и ON DELETE поведение описаны в **data_model_v4.md**. Настоящий документ ссылается на него и не дублирует DDL.

---

## 1. Структура пакета

```
internal/adapter/postgres/
├── pool.go                  # Создание pgxpool.Pool из config
├── txmanager.go             # Транзакции через context
├── errors.go                # Маппинг pgx → domain errors
│
├── ref_entry_repo.go        # Reference catalog: ref_entries + всё дочернее
├── entry_repo.go            # User entries
├── sense_repo.go            # User senses (COALESCE-паттерн)
├── translation_repo.go      # User translations (COALESCE-паттерн)
├── example_repo.go          # User examples (COALESCE-паттерн)
├── pronunciation_repo.go    # M2M: entry ↔ ref_pronunciations
├── image_repo.go            # M2M + user_images
├── card_repo.go             # Cards + SRS
├── review_log_repo.go       # Review logs
├── user_repo.go             # Users + user_settings
├── token_repo.go            # Refresh tokens
├── topic_repo.go            # Topics + entry_topics M2M
├── inbox_repo.go            # Inbox items
├── audit_repo.go            # Audit log
│
├── query/                   # SQL-файлы для sqlc (статические запросы)
├── sqlc/                    # Сгенерированный код
└── sqlc.yaml                # Конфигурация sqlc
```

Один репозиторий = одна основная таблица (или группа тесно связанных таблиц). Группировка дочерних таблиц в отдельные repo (sense_repo, translation_repo) обусловлена тем, что каждая из них имеет собственный COALESCE-паттерн и может использоваться независимо через DataLoaders.

---

## 2. Миграции

### 2.1. Формат и порядок

Миграции в формате goose. Каждая содержит `-- +goose Up` и `-- +goose Down`. Down-миграция обязательна.

Порядок определяется зависимостями между таблицами:

| # | Миграция | Содержимое |
|---|----------|-----------|
| 001 | Enums | Все ENUM types (должны существовать до таблиц, которые их используют) |
| 002 | Reference Catalog | ref_entries, ref_senses, ref_translations, ref_examples, ref_pronunciations, ref_images |
| 003 | Users | users, user_settings, refresh_tokens |
| 004 | User Dictionary | entries, senses, translations, examples, entry_pronunciations, entry_images, user_images |
| 005 | Cards & SRS | cards, review_logs |
| 006 | Organization | topics, entry_topics, inbox_items, audit_log |
| 007 | Triggers | fn_preserve_*_on_ref_delete (защита данных при удалении ref-записей) |
| 008 | Extensions & Search | pg_trgm extension, GIN-индексы для fuzzy search |

Полный DDL каждой таблицы — в data_model_v4.md, секции 2–7.

### 2.2. Правила миграций

- Имя файла: `YYYYMMDDHHMMSS_краткое_описание.sql`
- `IF NOT EXISTS` / `IF EXISTS` где применимо для идемпотентности
- Миграции не должны содержать DML (INSERT/UPDATE/DELETE) — начальные данные загружаются через seed-скрипты, не через миграции
- При изменении ENUM (добавление значения) использовать `ALTER TYPE ... ADD VALUE` — PostgreSQL не поддерживает удаление значений из ENUM, поэтому down-миграция для ENUM-изменений невозможна. Документировать это в комментарии

---

## 3. Нормализация текста

### 3.1. Правила нормализации

Нормализация применяется при создании и обновлении `entries.text_normalized` и `ref_entries.text_normalized`. Правила:

1. `TRIM()` — убрать пробелы по краям
2. `LOWER()` — привести к нижнему регистру
3. Сжать множественные пробелы в один — для multi-word phrases ("ice cream")
4. **Не** убирать диакритические знаки — "café" и "cafe" считаются разными словами
5. **Не** убирать дефисы и апострофы — "well-known", "don't" сохраняются как есть

Нормализация выполняется **на application level** (в service layer), не в триггере БД. Это позволяет переиспользовать логику в разных контекстах (поиск, импорт, API).

### 3.2. Unique constraint

`ux_entries_user_text ON entries(user_id, text_normalized) WHERE deleted_at IS NULL` — гарантирует уникальность слова в рамках одного пользователя среди не-удалённых записей.

Это означает: пользователь может soft-delete "abandon" и создать новый "abandon". Но два активных "abandon" одновременно невозможны.

---

## 4. Инструменты: sqlc vs Squirrel

### 4.1. Правило выбора

**sqlc** — когда все условия запроса известны при написании кода. Это подавляющее большинство запросов: CRUD, получение по ID, batch-загрузка по массиву ID, агрегаты.

**Squirrel** — когда набор WHERE-условий формируется динамически в runtime. На практике это **только**:

- `entry_repo.Find()` — поиск с комбинацией опциональных фильтров (search, hasCard, partOfSpeech, topicID, status)
- `ref_entry_repo.Search()` — fuzzy search по каталогу

### 4.2. Требования к sqlc-запросам

Для каждой таблицы нужны запросы следующих категорий:

**Единичные операции:** GetByID, Create, Update, Delete (или SoftDelete для entries).

**Batch-загрузка:** `Get*ByIDs` (`WHERE id = ANY($1::uuid[])`) и `Get*By<Parent>IDs` (`WHERE parent_id = ANY($1::uuid[])`). Эти запросы используются DataLoaders для предотвращения N+1.

**Особенности:**
- Все запросы к user-данным **обязательно** фильтруют по `user_id`
- Все запросы к entries и дочерним таблицам фильтруют soft-deleted записи (подробнее в секции 8)
- Запросы к senses, translations, examples используют COALESCE-паттерн (подробнее в секции 5)

### 4.3. Требования к Squirrel-запросам

Метод `Find` для entries должен поддерживать следующие **опциональные** фильтры (любая комбинация):

| Фильтр | Условие |
|---------|---------|
| `Search` (string) | `text_normalized ILIKE '%...%'` — используется GIN trgm index |
| `HasCard` (bool) | `EXISTS / NOT EXISTS` подзапрос к cards |
| `PartOfSpeech` (enum) | `EXISTS` подзапрос к senses с `COALESCE(s.part_of_speech, rs.part_of_speech)` |
| `TopicID` (uuid) | `EXISTS` подзапрос к entry_topics |
| `Status` (LearningStatus) | `EXISTS` подзапрос к cards с фильтром по status |

Метод также должен возвращать `totalCount` (через отдельный count-запрос с теми же фильтрами, но без LIMIT/OFFSET) и поддерживать сортировку по text, created_at, updated_at.

---

## 5. COALESCE-паттерн

### 5.1. Суть

User-таблицы senses, translations, examples содержат nullable поля, которые **наследуют** значения из reference catalog через LEFT JOIN + COALESCE.

Каждый SELECT из этих таблиц, который возвращает данные клиенту, должен включать LEFT JOIN на соответствующую ref-таблицу и COALESCE для каждого наследуемого поля.

### 5.2. Какие поля наследуются

| Таблица | Наследуемые поля | Ref-таблица |
|---------|-----------------|-------------|
| senses | definition, part_of_speech, cefr_level | ref_senses |
| translations | text | ref_translations |
| examples | sentence, translation | ref_examples |

### 5.3. Семантика NULL

`NULL` в наследуемом поле означает **"наследую из каталога"**, а не "значение отсутствует". Пользователь не может установить поле в NULL напрямую — если нужно убрать значение, удаляется вся строка.

### 5.4. Partial customization

Пользователь может кастомизировать **часть** полей. Например, изменить definition, но оставить part_of_speech из каталога. Каждое поле наследуется независимо. Это корректное и ожидаемое поведение.

### 5.5. Когда COALESCE не нужен

Внутренние запросы (проверка существования, подсчёт, удаление) не требуют COALESCE. Паттерн нужен только при **выдаче данных наружу** — в сервис или через DataLoader.

### 5.6. Маппинг в domain

sqlc-модели с COALESCE-полями маппятся в domain-модели, где поля уже resolved. Domain-модели (Sense, Translation, Example) хранят resolved-значения, а также `RefSenseID`/`RefTranslationID`/`RefExampleID` для origin tracking.

---

## 6. Репозитории: принципы реализации

### 6.1. Получение querier

Каждый репозиторий хранит `*pgxpool.Pool`. В начале каждого метода получает querier через `QuerierFromCtx(ctx, pool)` — это либо tx из контекста, либо pool. Это единственный механизм получения соединения; репозиторий никогда не вызывает `pool.Begin()` напрямую.

### 6.2. Error mapping

Каждый репозиторий преобразует pgx/pgconn ошибки в domain-ошибки:

| pgx/pgconn ошибка | domain ошибка |
|---|---|
| `pgx.ErrNoRows` | `domain.ErrNotFound` |
| PgError `23505` (unique_violation) | `domain.ErrAlreadyExists` |
| PgError `23503` (foreign_key_violation) | `domain.ErrNotFound` (referenced entity not found) |
| PgError `23514` (check_violation) | `domain.ErrValidation` |
| Всё остальное | Возвращается как есть, обёрнутое с контекстом операции |

Ошибки оборачиваются с указанием entity и ID для трассировки.

### 6.3. Маппинг sqlc → domain

Каждый репозиторий содержит приватные функции маппинга `sqlcModel → domain.Model`. Маппинг обрабатывает nullable поля (pointer → value, NULL → zero value или nil). Функции маппинга не содержат бизнес-логики.

### 6.4. User ID — обязательный параметр

Все методы user-репозиториев (entries, senses, cards, topics, inbox и т.д.) принимают `userID` как параметр и **всегда** используют его в WHERE. Исключения:

- Ref catalog репозиторий — shared данные, нет user_id
- Методы, работающие по ID дочерней сущности (sense, translation), где ownership проверяется через JOIN с entries

---

## 7. Транзакции

### 7.1. TxManager

Единственный способ запустить транзакцию — через `TxManager.RunInTx(ctx, fn)`. TxManager:

- Начинает tx через `pool.Begin(ctx)`
- Помещает tx в context
- Вызывает `fn(txCtx)`
- При ошибке или panic — Rollback
- При успехе — Commit
- Не поддерживает nested transactions (вызов RunInTx внутри RunInTx создаст вторую tx — это баг, не фича)

### 7.2. Какие операции требуют транзакции

| Операция | Почему |
|----------|--------|
| Добавление слова из каталога | entry + senses + translations + examples + pronunciations + images + card + audit — всё или ничего |
| Добавление пользовательского слова | entry + custom senses + card + audit |
| Удаление слова (soft delete) | entry update + audit |
| Повторение карточки | review_log + card SRS update + audit |
| Заполнение Reference Catalog из API | ref_entry + ref_senses + ref_translations + ref_examples + ref_pronunciations + ref_images |
| Регистрация пользователя | user + user_settings |
| Обновление sense/translation/example | update + audit |


### 7.3. Размер транзакций

Транзакция должна быть минимальной. Правило: **никаких внешних вызовов внутри транзакции** (HTTP к FreeDictionary, OAuth validation и т.д.). Данные от внешних API подготавливаются до начала транзакции, внутри tx — только запись в БД.

---

## 8. Soft Delete

### 8.1. Какие таблицы

Только `entries` имеет `deleted_at`. Дочерние записи (senses, translations, examples, cards и т.д.) **не** имеют своего `deleted_at` — они каскадно "скрываются" через JOIN с entries.

### 8.2. Правило фильтрации

**Все запросы**, которые возвращают user-данные, связанные с entries, должны фильтровать soft-deleted:

- Прямые запросы к entries: `WHERE deleted_at IS NULL`
- Запросы к cards, senses и другим дочерним таблицам: через `JOIN entries ... WHERE entries.deleted_at IS NULL`
- DataLoader batch-запросы: аналогично, JOIN с entries

**Исключение:** административные запросы — hard delete старых записей, статистика, дебаг.

### 8.3. Hard delete

Отдельный процесс (cron job или scheduled task) физически удаляет записи с `deleted_at < now() - 30 days`. Каскадные FK автоматически удаляют все дочерние записи. Этот процесс не требует транзакции — каждый DELETE выполняется независимо.

### 8.4. Corner case: re-create after soft delete

Пользователь soft-deletes "abandon", потом добавляет "abandon" снова. Unique constraint `(user_id, text_normalized) WHERE deleted_at IS NULL` это позволяет — новая запись создаётся со своим ID. Старая остаётся soft-deleted с другим ID. Это два разных entry.

---

## 9. Конкурентность и идемпотентность

### 9.1. Конкурентное заполнение каталога

Два пользователя одновременно ищут "abandon" → оба вызывают внешний API → оба пытаются создать `ref_entries` с `text_normalized = 'abandon'`.

Стратегия: **INSERT ... ON CONFLICT (text_normalized) DO NOTHING**, затем **SELECT** по text_normalized. Первый writer создаёт запись, второй получает существующую. Аналогично для ref_senses и других ref-таблиц в рамках одного ref_entry.

Альтернатива: advisory lock на уровне text_normalized перед заполнением каталога. Решение за разработчиком, но результат должен быть: никаких дубликатов, никаких ошибок unique violation для пользователя.

### 9.2. Конкурентное добавление слова

Два запроса от одного пользователя пытаются добавить "abandon". Unique constraint `ux_entries_user_text` гарантирует, что один из них получит `domain.ErrAlreadyExists`. Сервис должен обработать эту ошибку и вернуть существующую запись.

### 9.3. Конкурентный review карточки

Два запроса пытаются сделать review одной карточки. Оба читают текущее состояние, вычисляют SRS, обновляют. Второй UPDATE перезапишет первый. Это допустимо — SRS-результат идемпотентен для одного grade, а для разных grades один из review_logs будет "лишним", но не критично.

Если нужна строгая защита — использовать `SELECT ... FOR UPDATE` при чтении карточки внутри транзакции review.

### 9.4. Идемпотентность мутаций

| Операция | Идемпотентна? | Как обеспечить |
|----------|--------------|----------------|
| Создание entry | Нет → при повторе вернуть ErrAlreadyExists | Unique constraint |
| Soft delete entry | Да | `WHERE deleted_at IS NULL` — повторный вызов ничего не меняет |
| Link pronunciation | Да | `ON CONFLICT DO NOTHING` |
| Link topic | Да | `ON CONFLICT DO NOTHING` |
| Create review log | Нет → каждый вызов создаёт новую запись | Сервис должен защищать от дублей (rate limit или dedup по card_id + timestamp window) |

---

## 10. Управление Reference Catalog

### 10.1. Immutability

Каталог считается **фактически immutable** после создания. Данные, однажды записанные, не обновляются в штатном режиме. Это упрощает кеширование и исключает проблемы с concurrent reads.

### 10.2. Стратегия обновления

Если внешний источник обновил данные (новое определение, исправленная ошибка):

1. **Не обновлять** существующие ref-записи — это сломает expectation у пользователей, которые видели старые данные
2. Если нужно обновить — создать **новый** ref_entry с обновлёнными данными, пометить старый как deprecated (через отдельное поле или таблицу, если потребуется в будущем)
3. На MVP: не реализовывать обновление каталога. Каталог write-once.

### 10.3. Удаление ref-записей

При удалении ref-записи BEFORE DELETE trigger (описан в data_model_v4.md, секция 9) копирует данные в user-поля. Это обязательная защита: без неё `COALESCE(NULL, NULL) = NULL` потеряет данные.

### 10.4. Orphaned ref-записи

Ref_entries, на которые не ссылается ни один user entry (или все ссылающиеся user entries удалены). Это нормальная ситуация — каталог не очищается автоматически. Потенциально: periodic cleanup job, который удаляет ref_entries без ссылок старше N дней. Реализация — по необходимости, не на MVP.

---

## 11. Application-Level Limits

Следующие лимиты проверяются на уровне **сервиса** (не в БД constraints), но репозитории должны поддерживать count-запросы для проверки:

| Ресурс | Лимит | Обоснование |
|--------|-------|-------------|
| Entries на пользователя | 10 000 | Разумный объём словаря |
| Senses на entry | 20 | Одно слово не имеет больше значений |
| Translations на sense | 20 | |
| Examples на sense | 50 | |
| Topics на пользователя | 100 | |
| Inbox items на пользователя | 500 | |
| Новые карточки в день | user_settings.new_cards_per_day (default 20) | |
| Повторений в день | user_settings.reviews_per_day (default 200) | |

Репозитории должны предоставлять count-методы: `CountEntriesByUser`, `CountSensesByEntry`, `CountInboxByUser`, `CountReviewsToday` и т.д.

---

## 12. Position Management

Senses, translations, examples имеют поле `position INT` для управления порядком отображения.

### 12.1. Стратегия

- Позиции начинаются с 0
- При создании: новый элемент получает `MAX(position) + 1` среди siblings
- При удалении: **gap допустим** — оставшиеся элементы не renumber-ятся. Позиции используются только для ORDER BY, абсолютные значения не важны
- При явном reorder: клиент отправляет массив `[{id, newPosition}]`, репозиторий обновляет батчем в одной транзакции

### 12.2. Обоснование gap-подхода

Renumbering при удалении требует UPDATE всех siblings — O(N) операция, потенциально внутри транзакции. Gap-подход — O(1). Для UI-целей `ORDER BY position` даёт корректный порядок независимо от gaps.

---

## 13. Pagination

### 13.1. Два режима

Репозиторий entries.Find поддерживает **оба** режима пагинации:

**Offset-based** (для простых случаев и internal usage):
- Параметры: `limit`, `offset`
- Возвращает: `[]Entry`, `totalCount`

**Cursor-based** (для GraphQL, основной режим для клиента):
- Параметры: `limit`, `cursor` (encoded значение sort field + ID)
- Возвращает: `[]Entry`, `hasNextPage`, `hasPreviousPage`
- Cursor = base64(sort_value + "|" + entry_id) — keyset pagination
- WHERE clause: `(sort_field, id) > ($cursor_value, $cursor_id)` — для стабильной пагинации без дубликатов

### 13.2. Ограничения

- Максимальный `limit`: 200 (жёстко, даже если клиент запросит больше)
- Default `limit`: 50
- Cursor-based pagination работает только с полями, по которым есть index

---

## 14. Timezone Handling

### 14.1. Проблема

"Новые карточки за сегодня" и "повторений за сегодня" — "сегодня" определяется по timezone пользователя (`user_settings.timezone`). Все timestamps в БД хранятся в UTC.

### 14.2. Правило

Репозитории **не** занимаются timezone conversion. Они принимают `dayStart time.Time` (начало "сегодня" по timezone пользователя, уже сконвертированное в UTC) от сервиса. Сервис отвечает за конвертацию.

Это означает: `GetDueCards` принимает `now time.Time` (UTC), `CountReviewsToday` принимает `dayStart time.Time` (UTC) — оба уже рассчитаны с учётом timezone.

---

## 15. Timeouts и обработка context

### 15.1. Кто устанавливает timeout

Timeout на запрос устанавливается **middleware** (transport layer) через `context.WithTimeout`. Репозитории **не** устанавливают свои timeouts — они используют context, полученный от сервиса.

Default timeout: 5 секунд (из `config.DatabaseConfig.QueryTimeout`). Для тяжёлых операций (импорт, batch delete) transport может установить увеличенный timeout.

### 15.2. Обработка context.DeadlineExceeded

`context.DeadlineExceeded` и `context.Canceled` не маппятся в domain-ошибки. Они прокидываются как есть, обёрнутые в контекст операции. Transport-слой обрабатывает их отдельно (408 Request Timeout или 499 Client Closed).

---

## 16. Batch Operations

### 16.1. Импорт словаря

Массовое добавление слов (импорт из файла) не должно выполняться в одной гигантской транзакции. Стратегия:

- Разбить на chunks по 50 слов
- Каждый chunk — отдельная транзакция
- При ошибке в chunk — откатить только этот chunk, продолжить со следующего
- Вернуть отчёт: `{imported: N, skipped: M, errors: [{text, reason}]}`

### 16.2. Массовое присвоение топика

Массовый link для N entries — один multi-row INSERT с `ON CONFLICT DO NOTHING`. Не требует транзакции — каждый INSERT идемпотентен.

### 16.3. Hard delete cleanup

Периодическая очистка soft-deleted entries. Обрабатывать батчами по 100, каждый DELETE — отдельная операция (CASCADE удаляет дочерние).

---

## 17. Data Retention

| Данные | Retention | Механизм |
|--------|-----------|----------|
| Soft-deleted entries | 30 дней | Hard delete job |
| Revoked/expired refresh tokens | 0 дней (удалять сразу) | Periodic cleanup query |
| Audit log | 1 год | Periodic cleanup или partitioning |
| Review logs | Бессрочно | Нужны для статистики и аналитики |
| Orphaned ref_entries | Бессрочно (MVP) | Опциональный cleanup в будущем |

---

## 18. DataLoaders

### 18.1. Назначение

DataLoaders решают N+1 проблему в GraphQL. Каждый DataLoader собирает ID из нескольких resolver-ов в один batch-запрос.

### 18.2. Требуемые DataLoaders

| DataLoader | Ключ | Результат | Примечание |
|------------|------|-----------|------------|
| SensesByEntryID | entry_id | []Sense | С COALESCE |
| TranslationsBySenseID | sense_id | []Translation | С COALESCE |
| ExamplesBySenseID | sense_id | []Example | С COALESCE |
| PronunciationsByEntryID | entry_id | []RefPronunciation | Через M2M JOIN |
| CatalogImagesByEntryID | entry_id | []RefImage | Через M2M JOIN |
| UserImagesByEntryID | entry_id | []UserImage | Прямой SELECT |
| CardByEntryID | entry_id | *Card (nullable) | |
| TopicsByEntryID | entry_id | []Topic | Через M2M JOIN |
| ReviewLogsByCardID | card_id | []ReviewLog | |

### 18.3. Требования к batch-запросам

- Все batch-запросы именуются `Get*By<Parent>IDs`
- Результат должен быть группируем по ключу (запрос возвращает parent_id в каждой строке)
- Запросы к user-данным фильтруют soft-deleted entries через JOIN
- COALESCE применяется в batch-запросах для senses, translations, examples
- Batch size: до 100 ключей

### 18.4. Создание и lifecycle

DataLoaders создаются **per-request** в middleware. Кеш живёт в рамках одного HTTP-запроса. Параметры: `maxBatch = 100`, `wait = 2ms`.

---

## 19. Набор операций по группам

Здесь описаны **какие операции** нужны для каждой группы таблиц. Конкретные сигнатуры определяются сервисами-потребителями через их локальные интерфейсы.

### 19.1. Reference Catalog (ref_entry_repo)

**Чтение:**
- Получить ref_entry по ID
- Получить ref_entry по normalized text
- Поиск ref_entries по substring (fuzzy, pg_trgm)
- Получить полное дерево ref_entry: senses → translations, examples; pronunciations; images

**Запись:**
- Создать ref_entry со всем дочерним контентом (в транзакции)
- Upsert-семантика: создать или вернуть существующий по normalized text (для конкурентного доступа)

**Batch:** Получить ref_senses/ref_translations/ref_examples/ref_pronunciations/ref_images по массиву ID (для DataLoaders, если нужно показать origin info).

### 19.2. Entries (entry_repo)

**Чтение:**
- Получить по ID (с фильтром user_id + deleted_at)
- Получить по normalized text (для проверки дубликатов)
- Найти с динамическими фильтрами (Squirrel) — search, hasCard, partOfSpeech, topicID, status
- Получить по массиву ID (batch)
- Count по пользователю (для лимитов)

**Запись:**
- Создать entry
- Обновить notes
- Обновить text + text_normalized
- Soft delete
- Restore (отменить soft delete)

**Batch:** Hard delete старых soft-deleted записей.

### 19.3. Content (sense_repo, translation_repo, example_repo)

**Чтение:**
- Получить по parent ID — с COALESCE
- Получить по массиву parent IDs (batch для DataLoaders) — с COALESCE
- Получить по ID (единичный) — с COALESCE
- Count по parent (для лимитов)

**Запись:**
- Создать из ref (с ref_*_id, без локальных данных)
- Создать custom (без ref_*_id, с локальными данными)
- Обновить (set local fields; ref_*_id НЕ трогается — origin link)
- Удалить
- Обновить position (reorder)

### 19.4. Pronunciations и Images (pronunciation_repo, image_repo)

**Pronunciations (только M2M ссылки):**
- Получить ref_pronunciations по entry_id (через JOIN)
- Получить по массиву entry_ids (batch)
- Link / Unlink (ON CONFLICT DO NOTHING)

**Catalog Images (M2M ссылки):**
- Аналогично pronunciations

**User Images (полные записи):**
- CRUD + batch-получение по массиву entry_ids

### 19.5. Cards (card_repo)

**Чтение:**
- Получить по ID (с user_id)
- Получить по entry_id (с user_id)
- Получить по массиву entry_ids (batch для DataLoader)
- Очередь due cards (фильтр: soft-deleted entries, status != MASTERED, overdue first, limit)
- Count due, count new, count по статусам (для dashboard)

**Запись:**
- Создать
- Обновить SRS-поля (после review)
- Удалить

### 19.6. Review Logs (review_log_repo)

**Чтение:**
- По card_id (ordered DESC)
- По массиву card_ids (batch)
- Count за "сегодня" (принимает dayStart в UTC)
- Streak: количество по дням за последние N дней

**Запись:**
- Создать лог

### 19.7. Users (user_repo)

**Чтение:** По ID, по OAuth (provider + oauth_id), по email, settings по user_id.

**Запись:** Создать user, обновить user, создать settings, обновить settings.

### 19.8. Tokens (token_repo)

**Чтение:** Активный refresh token по hash.

**Запись:** Создать, revoke по ID, revoke все токены пользователя, удалить expired/revoked.

### 19.9. Topics (topic_repo)

**Чтение:** По ID, список по пользователю, topics по entry_id (M2M), topics по массиву entry_ids (batch), entry_ids по topic_id.

**Запись:** CRUD topic, link/unlink entry ↔ topic.

### 19.10. Inbox (inbox_repo)

**Чтение:** По ID, список по пользователю (с пагинацией), count по пользователю.

**Запись:** Создать, удалить, удалить все.

### 19.11. Audit (audit_repo)

**Чтение:** По entity (type + id), по пользователю (с пагинацией).

**Запись:** Создать запись.

---

## 20. Тестирование

### 20.1. Подход

Integration-тесты с реальной PostgreSQL через testcontainers-go. Каждый тест работает с чистыми данными. Контейнер переиспользуется в рамках пакета через `TestMain`.

### 20.2. Test Helpers

Необходимы seed-функции для создания тестовых данных:

- `SeedUser` → user + user_settings
- `SeedRefEntry(text)` → ref_entry + ref_senses + ref_translations + ref_examples + ref_pronunciations
- `SeedEntry(userID, refEntryID)` → entry + senses (linked to ref) + card
- `SeedEntryCustom(userID)` → entry + custom senses (without ref links)

Каждая seed-функция возвращает созданные domain-модели для использования в assertions.

### 20.3. Категории тестов

**Для каждого репозитория:**

| Категория | Что проверяется |
|-----------|----------------|
| CRUD happy path | Создание, чтение, обновление, удаление работают корректно |
| Not found | Несуществующий ID возвращает `domain.ErrNotFound` |
| Wrong user | Чужой user_id возвращает `domain.ErrNotFound` (не Forbidden — не раскрываем существование) |
| Duplicate | Нарушение unique constraint возвращает `domain.ErrAlreadyExists` |
| Cascade delete | Удаление parent удаляет children |
| Transaction rollback | При ошибке в транзакции все изменения откатываются |

**Для COALESCE-запросов:**

| Тест | Проверяет |
|------|-----------|
| Ref values | Когда user-поля NULL — возвращаются ref-значения |
| Custom values | Когда user-поля заполнены — возвращаются user-значения |
| Partial custom | Один field кастомизирован, другой наследуется |
| Ref deleted | Trigger скопировал данные, sense/translation/example сохраняется |

**Для soft delete:**

| Тест | Проверяет |
|------|-----------|
| Filtered from queries | Soft-deleted entries не попадают в результаты |
| Cards filtered | Due cards не включают soft-deleted entries |
| Re-create after delete | Можно создать entry с тем же text после soft delete |
| Restore | Soft-deleted entry можно восстановить |

**Для Find (Squirrel):**

| Тест | Проверяет |
|------|-----------|
| No filters | Все entries пользователя (не чужие) |
| Each filter individually | Search, HasCard, PartOfSpeech, TopicID, Status |
| Combined filters | Несколько фильтров одновременно |
| Pagination | Limit + offset + total count |
| Cursor-based pagination | Keyset pagination, hasNextPage/hasPreviousPage |
| Sorting | По text, created_at, updated_at |
| Empty result | Фильтры, которым ничего не соответствует → пустой slice, totalCount = 0 |

**Для конкурентности:**

| Тест | Проверяет |
|------|-----------|
| Concurrent ref_entry creation | Два goroutine создают одно слово → один создал, другой получил existing |
| Concurrent entry creation | Два goroutine создают одно слово для одного user → один ErrAlreadyExists |

### 20.4. Правила

- Тесты используют `t.Parallel()` с разными seed данными
- Assertions через `errors.Is()`, не по строке ошибки
- Каждый тест создаёт свои данные, не зависит от других тестов
- Контейнер PostgreSQL: `postgres:17-alpine` с применёнными миграциями
