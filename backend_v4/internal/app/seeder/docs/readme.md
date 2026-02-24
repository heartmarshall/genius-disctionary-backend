# Package `seeder`

> Офлайн-пайплайн для наполнения справочного каталога (`ref_entries`) данными из внешних лингвистических датасетов. Запускается отдельной CLI-командой, не является частью основного сервера.

## Быстрый старт

### 1. Подготовьте датасеты

Скачайте файлы и положите в любую удобную директорию:

| Датасет | Формат | Где взять | Примерный размер |
|---------|--------|-----------|------------------|
| **Wiktionary (Kaikki)** | JSONL | [kaikki.org/dictionary/English](https://kaikki.org/dictionary/English/) — файл `kaikki.org-dictionary-English.jsonl` | ~6 GB |
| **NGSL** | CSV | New General Service List (CSV с заголовком, слово в первом столбце) | ~100 KB |
| **NAWL** | CSV | New Academic Word List (CSV с заголовком, слово в первом столбце) | ~30 KB |
| **CMU Pronouncing Dictionary** | TXT | [github.com/cmusphinx/cmudict](https://github.com/cmusphinx/cmudict) — файл `cmudict-0.7b` | ~4 MB |
| **Open English WordNet** | JSON (директория) | [github.com/globalwordnet/english-wordnet/releases](https://github.com/globalwordnet/english-wordnet/releases) — скачать JSON ZIP, распаковать в директорию | ~80 MB |
| **Tatoeba** | TSV | [tatoeba.org/downloads](https://tatoeba.org/downloads) — EN-RU пары (4 колонки через TAB) | ~50 MB |

### 2. Настройте конфигурацию

Есть два способа: переменные окружения или YAML-файл.

**Вариант A — переменные окружения:**

```bash
export SEEDER_WIKTIONARY_PATH=/data/kaikki.org-dictionary-English.jsonl
export SEEDER_NGSL_PATH=/data/ngsl.csv
export SEEDER_NAWL_PATH=/data/nawl.csv
export SEEDER_CMU_PATH=/data/cmudict-0.7b
export SEEDER_WORDNET_PATH=/data/english-wordnet-2024.json
export SEEDER_TATOEBA_PATH=/data/en-ru-sentences.tsv
export SEEDER_TOP_N=20000        # сколько топ-слов взять из Wiktionary (по умолчанию 20000)
export SEEDER_BATCH_SIZE=500     # размер батча для bulk insert (по умолчанию 500)
export SEEDER_MAX_EXAMPLES=5     # макс. примеров Tatoeba на слово (по умолчанию 5)
export SEEDER_DRY_RUN=false      # true = только парсинг, без записи в БД
```

**Вариант B — YAML-файл** (например `seeder.yaml`):

```yaml
wiktionary_path: /data/kaikki.org-dictionary-English.jsonl
ngsl_path: /data/ngsl.csv
nawl_path: /data/nawl.csv
cmu_path: /data/cmudict-0.7b
wordnet_path: /data/english-wordnet-2024.json
tatoeba_path: /data/en-ru-sentences.tsv
top_n: 20000
batch_size: 500
max_examples_per_word: 5
dry_run: false
```

Приоритет: **ENV > YAML > defaults** (значения по умолчанию из `env-default` тегов).

### 3. Убедитесь, что БД доступна

Сидер использует ту же конфигурацию БД, что и основной сервер (через `config.Load()`). Убедитесь, что PostgreSQL запущен и миграции применены:

```bash
cd backend_v4
make docker-up      # поднять PostgreSQL
make migrate-up     # применить миграции
```

### 4. Запустите пайплайн

```bash
cd backend_v4

# Запустить все фазы
make seed

# Или напрямую с флагами:
go run ./cmd/seeder/

# С YAML-конфигом:
go run ./cmd/seeder/ --seeder-config=seeder.yaml

# Только определённые фазы:
go run ./cmd/seeder/ --phase=wiktionary,ngsl

# Dry-run (парсинг без записи в БД):
go run ./cmd/seeder/ --dry-run
```

## Фазы пайплайна

Пайплайн выполняет 5 фаз строго последовательно. Каждая фаза пропускается, если путь к её датасету не указан.

### Фаза 1: `wiktionary` — основные данные

**Что делает:** Парсит Kaikki JSONL дамп Wiktionary и создаёт записи в ref-каталоге.

**Алгоритм (два прохода по файлу):**
1. **Scoring pass** — читает весь файл, оценивает каждое английское слово по качеству контента
2. **Selection** — выбирает топ-N слов (по умолчанию 20000); слова из NGSL/NAWL получают бонус +1000 к скору и гарантированно попадают в выборку
3. **Parsing pass** — повторно читает файл, полностью парсит только отобранные слова
4. **Insert** — батчами вставляет в БД в порядке parent→child: entries → senses → translations → examples → pronunciations

**Критерии скоринга:**
| Критерий | Баллы |
|----------|-------|
| Каждое значение (sense) с gloss | +1.0 |
| Каждый русский перевод | +0.5 |
| Каждый пример | +0.3 |
| Наличие IPA произношения | +2.0 (однократно) |
| Слово из одного слова (без пробелов) | +1.0 |
| Слово из NGSL/NAWL | +1000.0 |

**Что вставляется:** `ref_entries`, `ref_senses`, `ref_translations`, `ref_examples`, `ref_pronunciations`, `ref_entry_source_coverage`.

### Фаза 2: `ngsl` — частотные ранги и уровни CEFR

**Что делает:** Парсит NGSL (New General Service List) и NAWL (New Academic Word List), обновляет метаданные у уже вставленных ref_entries.

**Маппинг CEFR по рангу NGSL:**
| Ранг | CEFR |
|------|------|
| 1–500 | A1 |
| 501–1200 | A2 |
| 1201–2000 | B1 |
| 2001+ | B2 |

Все слова NAWL получают уровень **C1**.

**Что обновляется:** `frequency_rank`, `cefr_level`, `is_core_lexicon` (через `COALESCE` — не перезаписывает существующие значения).

### Фаза 3: `cmu` — произношения (IPA)

**Что делает:** Парсит CMU Pronouncing Dictionary, конвертирует ARPAbet → IPA, вставляет произношения для слов, которые уже есть в ref-каталоге.

- Все произношения помечаются регионом **US** (CMU — словарь американского английского)
- Поддерживаются варианты произношения (например, `HOUSE` и `HOUSE(2)`)

**Что вставляется:** `ref_pronunciations`, `ref_entry_source_coverage`.

### Фаза 4: `wordnet` — семантические связи

**Что делает:** Парсит Open English WordNet (OEWN 2025 JSON, директория с файлами), извлекает связи между словами.

> `SEEDER_WORDNET_PATH` должен указывать на **распакованную директорию** с файлами `entries-*.json` и `noun.*.json`/`verb.*.json`/`adj.*.json`/`adv.*.json`.

**Типы связей:**
| Тип | Описание | Направленность |
|-----|----------|----------------|
| `synonym` | Синонимы (слова в одном synset) | Алфавитный порядок |
| `antonym` | Антонимы | Алфавитный порядок |
| `derived` | Производные формы | Алфавитный порядок |
| `hypernym` | Гиперонимы (specific → general) | Сохраняется как есть |

- Связи создаются только между словами, уже существующими в ref-каталоге
- Самореферентные связи и дубликаты фильтруются

**Что вставляется:** `ref_word_relations`.

### Фаза 5: `tatoeba` — примеры предложений (EN-RU)

**Что делает:** Парсит EN-RU пары из Tatoeba, привязывает к словам из ref-каталога.

- Предложения длиннее 500 символов пропускаются
- Для каждого слова берутся самые короткие предложения (сортировка по длине)
- Максимум примеров на слово: `SEEDER_MAX_EXAMPLES` (по умолчанию 5)
- Примеры привязываются к первому sense слова

**Что вставляется:** `ref_examples`.

## Флаги CLI

| Флаг | Описание | Пример |
|------|----------|--------|
| `--phase` | Запустить только указанные фазы (через запятую) | `--phase=wiktionary,cmu` |
| `--dry-run` | Парсить файлы без записи в БД | `--dry-run` |
| `--seeder-config` | Путь к YAML-конфигу | `--seeder-config=seeder.yaml` |

## Типичные сценарии использования

### Первоначальное наполнение каталога

```bash
# Все фазы, все данные
export SEEDER_WIKTIONARY_PATH=...  # (все пути)
make seed
```

### Обновить только произношения из CMU

```bash
export SEEDER_CMU_PATH=/data/cmudict-0.7b
go run ./cmd/seeder/ --phase=cmu
```

### Проверить парсинг без записи в БД

```bash
go run ./cmd/seeder/ --dry-run
```

### Добавить примеры из Tatoeba к существующим словам

```bash
export SEEDER_TATOEBA_PATH=/data/en-ru-sentences.tsv
go run ./cmd/seeder/ --phase=tatoeba
```

### Уменьшить объём данных для dev-окружения

```bash
export SEEDER_TOP_N=5000           # только 5000 слов
export SEEDER_MAX_EXAMPLES=2       # 2 примера на слово
make seed
```

## Конфигурация

### Параметры

| Параметр | ENV-переменная | По умолчанию | Описание |
|----------|---------------|--------------|----------|
| `WiktionaryPath` | `SEEDER_WIKTIONARY_PATH` | — | Путь к Kaikki JSONL |
| `NGSLPath` | `SEEDER_NGSL_PATH` | — | Путь к NGSL CSV |
| `NAWLPath` | `SEEDER_NAWL_PATH` | — | Путь к NAWL CSV |
| `CMUPath` | `SEEDER_CMU_PATH` | — | Путь к CMU dict |
| `WordNetPath` | `SEEDER_WORDNET_PATH` | — | Путь к директории с OEWN 2025 JSON |
| `TatoebaPath` | `SEEDER_TATOEBA_PATH` | — | Путь к Tatoeba TSV |
| `TopN` | `SEEDER_TOP_N` | `20000` | Макс. слов из Wiktionary |
| `BatchSize` | `SEEDER_BATCH_SIZE` | `500` | Размер батча для bulk-операций |
| `MaxExamplesPerWord` | `SEEDER_MAX_EXAMPLES` | `5` | Макс. примеров Tatoeba на слово |
| `DryRun` | `SEEDER_DRY_RUN` | `false` | Только парсинг, без записи в БД |

### Захардкоженные значения

| Значение | Файл | Текущее значение | Смысл |
|----------|------|-----------------|-------|
| `coreWordBonus` | `wiktionary/parser.go:16` | `1000.0` | Бонус к скору для слов из NGSL/NAWL |
| `maxLineSize` | `wiktionary/parser.go:19` | 1 MB | Размер буфера для чтения JSONL |
| `maxDefinitionLen` | `wiktionary/parser.go:22` | `5000` | Макс. длина определения |
| `maxSentenceLen` | `tatoeba/parser.go:18` | `500` | Макс. длина предложения Tatoeba |
| `positionOffset` | `tatoeba/parser.go:19` | `1000` | Сдвиг позиции для примеров Tatoeba (чтобы не конфликтовать с Wiktionary) |
| Таймаут контекста | `cmd/seeder/main.go:69` | 30 минут | Общий таймаут выполнения пайплайна |

## Источники данных (Data Sources)

Перед запуском фаз пайплайн автоматически регистрирует 8 источников данных в таблице `ref_data_sources`:

| Slug | Название | Тип |
|------|----------|-----|
| `freedict` | Free Dictionary API | definitions |
| `translate` | Google Translate | translations |
| `wiktionary` | Wiktionary (Kaikki) | definitions |
| `ngsl` | New General Service List | metadata |
| `nawl` | New Academic Word List | metadata |
| `cmu` | CMU Pronouncing Dictionary | pronunciations |
| `wordnet` | Open English WordNet | relations |
| `tatoeba` | Tatoeba | examples |

`freedict` и `translate` — онлайн-источники, используемые сервисом `refcatalog` в runtime. Остальные 6 — офлайн-датасеты, подгружаемые этим пайплайном.

## Обработка ошибок

- Каждая фаза работает независимо: ошибка в одной фазе **не останавливает** остальные
- Если хотя бы одна фаза завершилась с ошибкой, CLI завершается с кодом `1`
- Все bulk-операции используют `ON CONFLICT DO NOTHING` — повторный запуск безопасен (идемпотентен)
- `BulkUpdateEntryMetadata` использует `COALESCE` — не перезаписывает существующие значения

## Порядок зависимостей

Фазы должны выполняться в определённом порядке, так как поздние фазы зависят от данных ранних:

```
wiktionary ──→ ngsl ──→ cmu ──→ wordnet ──→ tatoeba
   │              │         │         │          │
   │ создаёт      │ обновл. │ доб.    │ доб.     │ доб.
   │ entries      │ метадан. │ произн. │ связи    │ примеры
   │ senses       │         │         │          │
   │ translations │         │         │          │
   │ examples     │         │         │          │
   │ pronunciat.  │         │         │          │
```

- **`ngsl`** обновляет `frequency_rank`/`cefr_level` у entries, созданных `wiktionary`
- **`cmu`**, **`wordnet`**, **`tatoeba`** ищут слова по `normalized_text` в ref_entries — без `wiktionary` им нечего обогащать
- Можно запускать отдельные фазы повторно (идемпотентно), но `wiktionary` должна быть выполнена первой хотя бы один раз
