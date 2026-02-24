# Design: Переход с RefCatalog на предзагруженные датасеты

**Дата:** 2026-02-24
**Roadmap:** пункт 1

## Контекст

Текущий RefCatalog получает данные из FreeDictionary API по запросу. Нужно перейти на предзагруженный каталог из ~20-25k слов, собранных из книг, сериалов и песен, обогащённых открытыми датасетами (Wiktionary, CMU, WordNet, Tatoeba, NGSL). Качество данных из датасетов недостаточно — реализуем lazy enrichment через LLM: слова, с которыми взаимодействуют пользователи, попадают в очередь на обогащение.

## Решения (по итогам обсуждения)

| Вопрос | Решение |
|--------|---------|
| Фильмы | Позже, отдельный этап, не блокирует |
| Триггер очереди | Добавление в словарь + просмотр/поиск ref-слова |
| Мерж LLM-данных | Полная замена существующих senses/examples |
| Админ-скоуп | Роль + базовые мутации (trigger, queue view, import) |
| Запуск enrichment | Внешний cron/scheduler → CLI или HTTP endpoint |
| Импорт результатов | CLI + HTTP endpoint (оба варианта) |

## Фазы реализации

### Фаза 0 — Подготовка

- Закоммитить перенос директорий в `internal/app/` (roadmap 1.1)
- Фильтрация `merged_pool.csv` (60k слов) через Wiktionary 2.7GB датасет:
  - Совпавшие → `seed_wordlist.txt`
  - Не найденные → `unmatched_words.txt` (артефакт)

### Фаза 1 — Первичный посев ref-каталога

- Прогон seeder с отфильтрованным `seed_wordlist.txt`
- Результат: ref_entries заполнены из Wiktionary + NGSL + CMU + WordNet + Tatoeba

### Фаза 2 — Админ-роль

- Role enum в domain (`user`, `admin`)
- Миграция: `role` в таблицу `users`
- Middleware проверки роли
- GraphQL: resolver-level проверка для админ-мутаций

### Фаза 3 — Enrichment Queue

- Миграция: таблица `enrichment_queue` (ref_entry_id, status, priority, requested_at, processed_at)
- Триггер: `createEntryFromCatalog` + просмотр/поиск ref-слова → upsert в очередь
- Статусы: `pending` → `processing` → `done` / `failed`
- EnrichmentService: добавление, выборка pending, обновление статусов

### Фаза 4 — Enrichment Pipeline (доработка)

- CLI/HTTP endpoint: слова из очереди → batch prompt файлы
- Admin-only endpoint `triggerEnrichment`: выбрать N pending, запустить enricher, обновить статусы
- Внешний cron вызывает endpoint или CLI

### Фаза 5 — Import с заменой

- Доработка LLM Importer: полная замена senses/examples/translations (вместо ON CONFLICT DO NOTHING)
- CLI команда `cmd/llm-import` с логикой замены
- Admin-only endpoint `importEnrichedWords`: указать директорию, запустить импорт
- Обновление статуса в `enrichment_queue` → `done`

### Фаза 6 — Админ-мутации

- `triggerEnrichment(batchSize: Int)` — запуск обработки очереди
- `enrichmentQueueStats` — статистика (pending/processing/done/failed)
- `enrichmentQueue(status, limit, offset)` — просмотр очереди
- `importEnrichedWords(directory: String)` — ручной импорт

### Фаза 7 — Верификация пропагации

- Проверить, что замена ref_senses корректно отражается в пользовательских entries
- Проверить каскадные удаления/обновления FK
- E2E тест: seed → user adds → enrich → import → user sees updated data
