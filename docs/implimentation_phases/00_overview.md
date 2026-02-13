# MyEnglish Backend v4 — План реализации

> **Дата:** 2026-02-13
> **Назначение:** Общий обзор фаз реализации backend v4. Каждая фаза описана в отдельном файле в этой директории.

---

## Принципы плана

- **TDD-порядок** для каждого модуля: определить интерфейсы зависимостей → написать тесты → реализовать → интегрировать
- **Описания задач не содержат кода реализации** — только требования, ограничения, acceptance criteria
- Каждая задача ссылается на конкретные секции документации для контекста
- Acceptance criteria — чеклист завершённости задачи
- Фазы выполняются последовательно; задачи внутри фазы могут выполняться параллельно (если зависимости позволяют)

---

## Технологический стек (сводка)

| Компонент | Технология |
|-----------|-----------|
| Язык | Go 1.23+ |
| Module path | `github.com/heartmarshall/myenglish-backend` |
| БД | PostgreSQL 17, pgx/v5, pgxpool |
| SQL | sqlc (статические) + Squirrel (динамические) |
| Миграции | goose (sequential numbering) |
| GraphQL | gqlgen |
| Auth | OAuth (Google/Apple), JWT (HS256), golang-jwt/jwt/v5 |
| Config | cleanenv (YAML + ENV) |
| Logging | log/slog (stdlib) |
| HTTP | net/http, http.ServeMux (Go 1.22+) |
| Тестирование | testcontainers-go, moq, table-driven tests |
| Прочее | scany/v2, google/uuid |

---

## Архитектура (сводка)

```
transport → (свои интерфейсы) ← service → (свои интерфейсы) ← adapter
                                    ↓
                                  domain
```

- **domain/** — чистые Go-структуры, sentinel errors, value objects. Без внешних зависимостей.
- **service/** — бизнес-логика. Каждый сервис определяет интерфейсы своих зависимостей в своём пакете (consumer-defined interfaces).
- **adapter/** — реализации: PostgreSQL repos, внешние API. Реализует интерфейсы из service/ через duck typing.
- **transport/** — HTTP/GraphQL. Определяет свои интерфейсы сервисов.
- Связывание — только в `cmd/server/main.go`.

---

## Фазы реализации

| # | Фаза | Файл | Описание |
|---|------|------|----------|
| 1 | Скелет проекта и доменный слой | `phase_01_project_skeleton.md` | Структура проекта, go.mod, конфиг, логгер, context helpers, domain models/errors |
| 2 | Инфраструктура базы данных | `phase_02_database_infra.md` | Docker, Makefile, миграции, pgxpool, TxManager, Querier, error mapping, sqlc template, test helpers |
| 3 | Слой репозиториев | `phase_03_repository_layer.md` | Все репозитории (14 пакетов) + DataLoaders. Детальная разбивка задач — в `docs/repo/repo_layer_tasks_v4.md` |
| 4 | Система аутентификации и User сервис | `phase_04_auth_system.md` | Auth service, JWT, OAuth providers, REST endpoints, auth middleware, User service |
| 5 | RefCatalog и Dictionary сервисы | `phase_05_dictionary.md` | RefCatalog service, внешние провайдеры (FreeDictionary), Dictionary service (CRUD, поиск, фильтрация, импорт/экспорт) |
| 6 | Content сервис | `phase_06_content.md` | CRUD для senses, translations, examples, user images; reorder; ownership chain verification |
| 7 | Study сервис | `phase_07_study.md` | SRS алгоритм (CalculateSRS), карточки, review flow, undo, study sessions, очередь, dashboard, статистика |
| 8 | Сервисы организации | `phase_08_organization.md` | Topic service (CRUD, link/unlink, batch), Inbox service (CRUD, clear) |
| 9 | Транспортный слой (GraphQL + REST) | `phase_09_transport.md` | gqlgen schema, resolvers, DataLoaders middleware, error presentation, input validation |
| 10 | Сборка сервера и интеграция | `phase_10_integration.md` | HTTP server, middleware stack, health checks, graceful shutdown, bootstrap wiring, E2E tests, Docker optimization |

---

## Граф зависимостей фаз

```
Фаза 1 → Фаза 2 → Фаза 3 ──→ Фаза 4  (Auth + User)  ──────┐
                              ├→ Фаза 5  (RefCatalog + Dict) ─┤
                              ├→ Фаза 6  (Content)            ├──→ Фаза 9  (Transport)
                              ├→ Фаза 7  (Study)              │         │
                              └→ Фаза 8  (Organization) ──────┘         │
                                                                        └──→ Фаза 10 (Integration)
```

**Рекомендуемый порядок:** Фаза 4 (Auth) первой, затем Фазы 5–8 параллельно.

Фазы 5–8 технически не зависят от Фазы 4 — unit-тесты сервисов используют моки, а `userID` в тестах создаётся напрямую (`uuid.New()`). Однако Фаза 4 рекомендуется первой, т.к. Auth middleware и User service формируют основу для интеграционного тестирования остальных сервисов.

Фаза 9 требует завершения всех сервисных фаз (4–8).
Фаза 10 требует завершения Фазы 9.

### Background jobs

Фоновые задачи из `business_scenarios_v4.md` распределены по фазам:

| Задача | Описание | Фаза |
|--------|----------|------|
| BG1 | Hard delete entries (deleted_at > 30 days) | Фаза 5 (Dictionary service) |
| BG2 | Cleanup expired/revoked refresh tokens | Фаза 4 (Auth service) |
| BG3 | Cleanup audit_log старше 1 года | Фаза 10 (Integration) |

---

## Соответствие рекомендуемому порядку из code_conventions

| code_conventions §14 | Фаза плана |
|----------------------|-----------|
| 1. Skeleton | Фаза 1 |
| 2. Database | Фаза 2 |
| 3. Auth | Фаза 4 |
| 4. Dictionary | Фаза 5 |
| 5. Content | Фаза 6 |
| 6. Cards & SRS | Фаза 7 |
| 7. Study | Фаза 7 |
| 8. Inbox | Фаза 8 |
| 9. Topics | Фаза 8 |
| 10. Import/Export | Фаза 5 (часть Dictionary service) |
| 11. Suggestions | Фаза 5 (RefCatalog external providers) |

Фаза 3 (Repository Layer) добавлена как отдельный этап, т.к. для неё существует детальная разбивка задач в `repo_layer_tasks_v4.md`.

---

## Документы-источники

| Документ | Содержание |
|----------|-----------|
| `docs/code_conventions_v4.md` | Архитектура, стиль кода, паттерны, правила |
| `docs/data_model_v4.md` | Полная схема БД (22 таблицы), ER-диаграмма |
| `docs/repo/repo_layer_spec_v4.md` | Спецификация repository layer |
| `docs/repo/repo_layer_tasks_v4.md` | Детальные задачи repository layer (19 задач, 6 фаз) |
| `docs/infra/infra_spec_v4.md` | Инфраструктура: config, logging, Docker, Makefile |
| `docs/services/service_layer_spec_v4.md` | Архитектура service layer, паттерны |
| `docs/services/auth_service_spec_v4.md` | Спецификация Auth service |
| `docs/services/dictionary_service_spec_v4.md` | Спецификация Dictionary service |
| `docs/services/content_service_spec_v4.md` | Спецификация Content service |
| `docs/services/study_service_spec_v4_v1.1.md` | Спецификация Study service (v1.1 с learning steps) |
| `docs/services/inbox_service_spec_v4.md` | Спецификация Inbox service |
| `docs/services/topic_service_spec_v4.md` | Спецификация Topic service |
| `docs/services/business_scenarios_v4.md` | Карта бизнес-сценариев |

---

## Общие правила для всех фаз

### Стиль кода
- Consumer-defined interfaces (определяются потребителем, не централизованно)
- Ошибки оборачиваются через `fmt.Errorf("operation: %w", err)`, sentinel errors из `domain/`
- Логирование через `slog` из контекста, ошибки логируются один раз
- Все запросы к БД фильтруются по `user_id`
- Тесты: `Test<Target>_<Method>_<Scenario>`, table-driven для чистых функций
- Моки: `moq`, генерируются из локальных интерфейсов в `_test.go` файлы
- Commits: `<type>(<scope>): <description>`

### Валидация
- **Transport:** формат данных (парсинг, типы, required fields)
- **Service:** бизнес-правила (дубликаты, лимиты, состояния)
- **Repository:** не валидирует (доверяет сервису)

### Тестирование
- Unit-тесты сервисов — основной фокус (≥80% coverage)
- Integration-тесты репозиториев — testcontainers с реальным PostgreSQL
- E2E-тесты транспорта — GraphQL запросы к полному стеку
