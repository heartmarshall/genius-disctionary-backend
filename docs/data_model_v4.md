# MyEnglish Backend v4 — Data Model

## 1. Архитектурное решение: Reference Catalog + User Dictionary

### Проблема

В multi-user приложении каждый пользователь добавляет слова в свой словарь. Большинство данных (определения, переводы, примеры, произношения) приходят из одних и тех же внешних источников (FreeDictionary, Google Translate, OpenAI). Полная дупликация этих данных для каждого пользователя — расточительна.

### Решение

База данных разделена на два слоя:

**Reference Catalog** (префикс `ref_`) — shared, immutable каталог эталонных данных. Заполняется при первом запросе слова из внешних API. Данные не принадлежат ни одному пользователю. Используются как шаблон при добавлении слова.

**User Dictionary** — per-user данные. Каждая запись принадлежит конкретному пользователю (`user_id`). Содержит **nullable ссылки** (`ref_*_id`) на каталог. Если пользователь не кастомизировал данные — поля NULL, значения берутся из каталога через `COALESCE`. Если кастомизировал — хранятся в самой записи.

### Принцип "Origin Link"

`ref_*_id` — это **origin link**, а не "источник данных". Ссылка **никогда не обнуляется** при редактировании пользователем. Это позволяет:

- Находить всех пользователей, добавивших одно и то же значение слова
- Строить social-фичи: "примеры других пользователей", "популярные переводы"
- Агрегировать статистику по каталогу

### Чтение данных: COALESCE

```sql
-- Пример: загрузка senses
SELECT
    s.id,
    s.entry_id,
    COALESCE(s.definition, rs.definition) AS definition,
    COALESCE(s.part_of_speech, rs.part_of_speech) AS part_of_speech,
    COALESCE(s.cefr_level, rs.cefr_level) AS cefr_level,
    s.source_slug,
    s.position,
    s.ref_sense_id IS NOT NULL AS is_reference
FROM senses s
LEFT JOIN ref_senses rs ON s.ref_sense_id = rs.id
WHERE s.entry_id = $1
ORDER BY s.position;
```

**Правило NULL:** `NULL` в user-поле означает "наследую из каталога". Пользователь не может осознанно установить поле в NULL (очистить). Если нужно удалить — удаляется вся строка.

### Copy-on-Write

При редактировании пользователем данных, ссылающихся на каталог:

```sql
-- Пользователь меняет definition у sense, который ссылается на каталог
UPDATE senses
SET definition = 'my custom definition',
    part_of_speech = 'VERB'
-- ref_sense_id НЕ обнуляется (origin link сохраняется)
WHERE id = $1;
```

### Процесс добавления слова

1. Пользователь ищет "abandon"
2. Бекенд проверяет `ref_entries` — если нет, вызывает FreeDictionary API, сохраняет в `ref_*` таблицы
3. Пользователь выбирает нужные senses, translations, examples в конструкторе
4. Создаётся `entries` + для каждого выбранного sense — запись в `senses` с `ref_sense_id` (16 bytes вместо ~300B текста)
5. Аналогично для translations, examples — ссылки
6. Pronunciations — M2M записи в `entry_pronunciations`
7. Свои заметки, примеры, переводы — записи с `ref_*_id = NULL`, данные прямо в полях

### Экономия

| Сценарий (1000 юзеров × 2000 слов) | Полная дупликация | Reference + ссылки |
|-------------------------------------|-------------------|--------------------|
| Entries | ~100 MB | ~100 MB (text дублируется) |
| Senses, translations, examples | ~9 GB | ~0.5 GB (UUID ссылки) |
| Pronunciations | ~0.8 GB | ~64 MB (M2M) |
| Reference catalog | — | ~5 MB |
| **Итого** | **~10 GB** | **~0.7 GB** |

---

## 2. Reference Catalog (shared, immutable)

Таблицы с префиксом `ref_`. Не принадлежат пользователям. Заполняются из внешних API, могут курироваться вручную.

### ref_entries

Эталонные слова. Одна запись = одно уникальное слово в каталоге.

```sql
CREATE TABLE ref_entries (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    text            TEXT NOT NULL,
    text_normalized TEXT NOT NULL,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE UNIQUE INDEX ux_ref_entries_text_norm ON ref_entries(text_normalized);
```

### ref_senses

Эталонные значения слова из внешних источников.

```sql
CREATE TABLE ref_senses (
    id            UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    ref_entry_id  UUID NOT NULL REFERENCES ref_entries(id) ON DELETE CASCADE,
    definition    TEXT,
    part_of_speech part_of_speech,  -- ENUM
    cefr_level    TEXT,
    source_slug   TEXT NOT NULL,    -- "freedict", "openai"
    position      INT NOT NULL DEFAULT 0,
    created_at    TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX ix_ref_senses_entry ON ref_senses(ref_entry_id);
```

### ref_translations

Эталонные переводы.

```sql
CREATE TABLE ref_translations (
    id            UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    ref_sense_id  UUID NOT NULL REFERENCES ref_senses(id) ON DELETE CASCADE,
    text          TEXT NOT NULL,
    source_slug   TEXT NOT NULL,
    position      INT NOT NULL DEFAULT 0
);

CREATE INDEX ix_ref_translations_sense ON ref_translations(ref_sense_id);
```

### ref_examples

Эталонные примеры.

```sql
CREATE TABLE ref_examples (
    id            UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    ref_sense_id  UUID NOT NULL REFERENCES ref_senses(id) ON DELETE CASCADE,
    sentence      TEXT NOT NULL,
    translation   TEXT,
    source_slug   TEXT NOT NULL,
    position      INT NOT NULL DEFAULT 0
);

CREATE INDEX ix_ref_examples_sense ON ref_examples(ref_sense_id);
```

### ref_pronunciations

Эталонные произношения (транскрипции + аудио).

```sql
CREATE TABLE ref_pronunciations (
    id            UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    ref_entry_id  UUID NOT NULL REFERENCES ref_entries(id) ON DELETE CASCADE,
    transcription TEXT NOT NULL,
    audio_url     TEXT,
    region        TEXT,              -- "US", "UK"
    source_slug   TEXT NOT NULL
);

CREATE INDEX ix_ref_pronunciations_entry ON ref_pronunciations(ref_entry_id);
```

### ref_images

Эталонные изображения.

```sql
CREATE TABLE ref_images (
    id            UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    ref_entry_id  UUID NOT NULL REFERENCES ref_entries(id) ON DELETE CASCADE,
    url           TEXT NOT NULL,
    caption       TEXT,
    source_slug   TEXT NOT NULL
);

CREATE INDEX ix_ref_images_entry ON ref_images(ref_entry_id);
```

---

## 3. User Management

### users

```sql
CREATE TABLE users (
    id             UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    email          TEXT NOT NULL,
    name           TEXT,
    avatar_url     TEXT,
    oauth_provider TEXT NOT NULL,    -- "google", "apple"
    oauth_id       TEXT NOT NULL,    -- ID в провайдере
    created_at     TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at     TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE UNIQUE INDEX ux_users_email ON users(email);
CREATE UNIQUE INDEX ux_users_oauth ON users(oauth_provider, oauth_id);
```

### user_settings

Пользовательские настройки. Один к одному с users.

```sql
CREATE TABLE user_settings (
    user_id           UUID PRIMARY KEY REFERENCES users(id) ON DELETE CASCADE,
    new_cards_per_day INT NOT NULL DEFAULT 20,
    reviews_per_day   INT NOT NULL DEFAULT 200,
    max_interval_days INT NOT NULL DEFAULT 365,
    timezone          TEXT NOT NULL DEFAULT 'UTC',
    updated_at        TIMESTAMPTZ NOT NULL DEFAULT now()
);
```

### refresh_tokens

Хранение refresh-токенов для OAuth.

```sql
CREATE TABLE refresh_tokens (
    id         UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id    UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    token_hash TEXT NOT NULL,       -- SHA-256 хеш токена
    expires_at TIMESTAMPTZ NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    revoked_at TIMESTAMPTZ          -- NULL = активен
);

CREATE INDEX ix_refresh_tokens_user ON refresh_tokens(user_id);
CREATE INDEX ix_refresh_tokens_hash ON refresh_tokens(token_hash) WHERE revoked_at IS NULL;
```

---

## 4. User Dictionary

### entries

Слова пользователя. Центральная таблица пользовательского словаря.

```sql
CREATE TABLE entries (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id         UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    ref_entry_id    UUID REFERENCES ref_entries(id) ON DELETE SET NULL,
    text            TEXT NOT NULL,
    text_normalized TEXT NOT NULL,
    notes           TEXT,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
    deleted_at      TIMESTAMPTZ          -- Soft delete
);

-- Одно слово на пользователя (среди не-удалённых)
CREATE UNIQUE INDEX ux_entries_user_text ON entries(user_id, text_normalized) WHERE deleted_at IS NULL;

-- Быстрая фильтрация по пользователю (только живые записи)
CREATE INDEX ix_entries_user_alive ON entries(user_id) WHERE deleted_at IS NULL;

-- Social-фичи: "кто ещё добавил это слово"
CREATE INDEX ix_entries_user_ref ON entries(user_id, ref_entry_id) WHERE ref_entry_id IS NOT NULL;
```

`text` дублируется из каталога намеренно (~50 байт) — не нужен JOIN для отображения списка слов.

### senses

Значения слова. Nullable `ref_sense_id` — origin link на каталог.

```sql
CREATE TABLE senses (
    id             UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    entry_id       UUID NOT NULL REFERENCES entries(id) ON DELETE CASCADE,
    ref_sense_id   UUID REFERENCES ref_senses(id) ON DELETE SET NULL,
    definition     TEXT,             -- NULL = наследуется из ref
    part_of_speech part_of_speech,   -- NULL = наследуется из ref
    cefr_level     TEXT,             -- NULL = наследуется из ref
    source_slug    TEXT NOT NULL,    -- "freedict", "user"
    position       INT NOT NULL DEFAULT 0,
    created_at     TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX ix_senses_entry ON senses(entry_id);
CREATE INDEX ix_senses_ref ON senses(ref_sense_id) WHERE ref_sense_id IS NOT NULL;
```

### translations

Переводы. Nullable `ref_translation_id` — origin link.

```sql
CREATE TABLE translations (
    id                 UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    sense_id           UUID NOT NULL REFERENCES senses(id) ON DELETE CASCADE,
    ref_translation_id UUID REFERENCES ref_translations(id) ON DELETE SET NULL,
    text               TEXT,          -- NULL = наследуется из ref
    source_slug        TEXT NOT NULL,
    position           INT NOT NULL DEFAULT 0
);

CREATE INDEX ix_translations_sense ON translations(sense_id);
```

### examples

Примеры использования. Nullable `ref_example_id` — origin link.

```sql
CREATE TABLE examples (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    sense_id        UUID NOT NULL REFERENCES senses(id) ON DELETE CASCADE,
    ref_example_id  UUID REFERENCES ref_examples(id) ON DELETE SET NULL,
    sentence        TEXT,             -- NULL = наследуется из ref
    translation     TEXT,             -- NULL = наследуется из ref
    source_slug     TEXT NOT NULL,
    position        INT NOT NULL DEFAULT 0,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX ix_examples_sense ON examples(sense_id);
```

### entry_pronunciations

M2M: произношения из каталога, привязанные к записи пользователя. Всегда ссылка — пользователь не редактирует каталожные произношения.

```sql
CREATE TABLE entry_pronunciations (
    entry_id            UUID NOT NULL REFERENCES entries(id) ON DELETE CASCADE,
    ref_pronunciation_id UUID NOT NULL REFERENCES ref_pronunciations(id) ON DELETE CASCADE,
    PRIMARY KEY (entry_id, ref_pronunciation_id)
);
```

### entry_images

M2M: изображения из каталога, привязанные к записи пользователя.

```sql
CREATE TABLE entry_images (
    entry_id     UUID NOT NULL REFERENCES entries(id) ON DELETE CASCADE,
    ref_image_id UUID NOT NULL REFERENCES ref_images(id) ON DELETE CASCADE,
    PRIMARY KEY (entry_id, ref_image_id)
);
```

### user_images

Собственные изображения пользователя (не из каталога).

```sql
CREATE TABLE user_images (
    id         UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    entry_id   UUID NOT NULL REFERENCES entries(id) ON DELETE CASCADE,
    url        TEXT NOT NULL,
    caption    TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX ix_user_images_entry ON user_images(entry_id);
```

---

## 5. Study & SRS

### cards

Карточки интервального повторения. 1:1 с entries.

```sql
CREATE TABLE cards (
    id             UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id        UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    entry_id       UUID NOT NULL REFERENCES entries(id) ON DELETE CASCADE,
    status         learning_status NOT NULL DEFAULT 'NEW',
    next_review_at TIMESTAMPTZ,
    interval_days  INT NOT NULL DEFAULT 0,
    ease_factor    FLOAT NOT NULL DEFAULT 2.5,
    created_at     TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at     TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- Одна карточка на entry
CREATE UNIQUE INDEX ux_cards_entry ON cards(entry_id);

-- Горячий запрос: очередь на повторение
CREATE INDEX ix_cards_user_due ON cards(user_id, status, next_review_at) WHERE status != 'MASTERED';

-- Статистика по пользователю
CREATE INDEX ix_cards_user ON cards(user_id);
```

`user_id` — денормализация (entry уже содержит user_id), но необходима для эффективных запросов очереди повторения без JOIN на entries.

### review_logs

Полная история повторений.

```sql
CREATE TABLE review_logs (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    card_id     UUID NOT NULL REFERENCES cards(id) ON DELETE CASCADE,
    grade       review_grade NOT NULL,
    duration_ms INT,
    reviewed_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX ix_review_logs_card ON review_logs(card_id, reviewed_at DESC);
```

---

## 6. Organization

### topics

Пользовательские топики для категоризации слов.

```sql
CREATE TABLE topics (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id     UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    name        TEXT NOT NULL,
    description TEXT,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- Уникальное имя топика для пользователя
CREATE UNIQUE INDEX ux_topics_user_name ON topics(user_id, name);
```

### entry_topics

M2M: связь слов и топиков.

```sql
CREATE TABLE entry_topics (
    entry_id UUID NOT NULL REFERENCES entries(id) ON DELETE CASCADE,
    topic_id UUID NOT NULL REFERENCES topics(id) ON DELETE CASCADE,
    PRIMARY KEY (entry_id, topic_id)
);

CREATE INDEX ix_entry_topics_topic ON entry_topics(topic_id);
```

### inbox_items

Входящие — быстрые заметки для последующей обработки.

```sql
CREATE TABLE inbox_items (
    id         UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id    UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    text       TEXT NOT NULL,
    context    TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX ix_inbox_items_user ON inbox_items(user_id, created_at DESC);
```

---

## 7. Audit

### audit_log

```sql
CREATE TABLE audit_log (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id     UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    entity_type entity_type NOT NULL,
    entity_id   UUID,
    action      audit_action NOT NULL,
    changes     JSONB NOT NULL DEFAULT '{}',
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX ix_audit_log_user ON audit_log(user_id, created_at DESC);
CREATE INDEX ix_audit_log_entity ON audit_log(entity_type, entity_id) WHERE entity_id IS NOT NULL;
```

---

## 8. Enums

```sql
CREATE TYPE learning_status AS ENUM ('NEW', 'LEARNING', 'REVIEW', 'MASTERED');
CREATE TYPE review_grade    AS ENUM ('AGAIN', 'HARD', 'GOOD', 'EASY');
CREATE TYPE part_of_speech  AS ENUM (
    'NOUN', 'VERB', 'ADJECTIVE', 'ADVERB', 'PRONOUN',
    'PREPOSITION', 'CONJUNCTION', 'INTERJECTION',
    'PHRASE', 'IDIOM', 'OTHER'
);
CREATE TYPE entity_type     AS ENUM ('ENTRY', 'SENSE', 'EXAMPLE', 'IMAGE', 'PRONUNCIATION', 'CARD');
CREATE TYPE audit_action    AS ENUM ('CREATE', 'UPDATE', 'DELETE');
```

---

## 9. Защита при удалении записей каталога

При удалении записи из `ref_senses` (или `ref_translations`, `ref_examples`) FK-constraint выполняет `SET NULL`. Но если пользователь не кастомизировал свои поля (они ещё `NULL`), данные потеряются — `COALESCE(NULL, NULL) = NULL`.

Для защиты используется trigger, который копирует ref-данные в user-поля перед обнулением ссылки:

```sql
-- Trigger: перед удалением ref_sense копируем данные пользователям
CREATE OR REPLACE FUNCTION fn_preserve_sense_on_ref_delete()
RETURNS TRIGGER AS $$
BEGIN
    UPDATE senses s
    SET definition     = COALESCE(s.definition, OLD.definition),
        part_of_speech = COALESCE(s.part_of_speech, OLD.part_of_speech),
        cefr_level     = COALESCE(s.cefr_level, OLD.cefr_level)
    WHERE s.ref_sense_id = OLD.id;
    RETURN OLD;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER trg_preserve_sense_on_ref_delete
    BEFORE DELETE ON ref_senses
    FOR EACH ROW
    EXECUTE FUNCTION fn_preserve_sense_on_ref_delete();

-- Аналогичный trigger для ref_translations
CREATE OR REPLACE FUNCTION fn_preserve_translation_on_ref_delete()
RETURNS TRIGGER AS $$
BEGIN
    UPDATE translations t
    SET text = COALESCE(t.text, OLD.text)
    WHERE t.ref_translation_id = OLD.id;
    RETURN OLD;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER trg_preserve_translation_on_ref_delete
    BEFORE DELETE ON ref_translations
    FOR EACH ROW
    EXECUTE FUNCTION fn_preserve_translation_on_ref_delete();

-- Аналогичный trigger для ref_examples
CREATE OR REPLACE FUNCTION fn_preserve_example_on_ref_delete()
RETURNS TRIGGER AS $$
BEGIN
    UPDATE examples e
    SET sentence    = COALESCE(e.sentence, OLD.sentence),
        translation = COALESCE(e.translation, OLD.translation)
    WHERE e.ref_example_id = OLD.id;
    RETURN OLD;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER trg_preserve_example_on_ref_delete
    BEFORE DELETE ON ref_examples
    FOR EACH ROW
    EXECUTE FUNCTION fn_preserve_example_on_ref_delete();
```

---

## 10. Soft Delete

`deleted_at` присутствует только на `entries` — центральной таблице пользовательского словаря. Дочерние записи (senses, translations, cards и т.д.) не имеют своего `deleted_at`.

### Правило

**Все запросы** к user-данным, связанным с entries, должны фильтровать soft-deleted записи. Два способа:

**Прямые запросы к entries:**
```sql
SELECT * FROM entries WHERE user_id = $1 AND deleted_at IS NULL;
```

**Запросы к дочерним таблицам (cards, senses и т.д.):**
```sql
-- ВСЕГДА через JOIN с entries
SELECT c.* FROM cards c
JOIN entries e ON c.entry_id = e.id
WHERE c.user_id = $1
  AND e.deleted_at IS NULL
  AND (c.status = 'NEW' OR c.next_review_at <= $2);
```

### Hard Delete

Периодическая очистка (cron/job): записи с `deleted_at < now() - INTERVAL '30 days'` удаляются физически. Каскадное удаление чистит все дочерние таблицы.

---

## 11. ON DELETE сводка

| FK | ON DELETE | Причина |
|----|-----------|---------|
| ref_senses → ref_entries | CASCADE | Удаление слова из каталога удаляет его senses |
| ref_translations → ref_senses | CASCADE | |
| ref_examples → ref_senses | CASCADE | |
| ref_pronunciations → ref_entries | CASCADE | |
| ref_images → ref_entries | CASCADE | |
| entries → users | CASCADE | Удаление пользователя удаляет все его данные |
| entries → ref_entries | SET NULL | Каталог удалён — entry остаётся, теряет ссылку |
| senses → entries | CASCADE | Удаление entry удаляет его senses |
| senses → ref_senses | SET NULL | Каталог удалён — trigger копирует данные, ссылка обнуляется |
| translations → senses | CASCADE | |
| translations → ref_translations | SET NULL | Trigger + SET NULL |
| examples → senses | CASCADE | |
| examples → ref_examples | SET NULL | Trigger + SET NULL |
| entry_pronunciations → entries | CASCADE | |
| entry_pronunciations → ref_pronunciations | CASCADE | Ref удалён — M2M ссылка удаляется |
| entry_images → entries | CASCADE | |
| entry_images → ref_images | CASCADE | Ref удалён — M2M ссылка удаляется |
| user_images → entries | CASCADE | |
| cards → users | CASCADE | |
| cards → entries | CASCADE | |
| review_logs → cards | CASCADE | |
| topics → users | CASCADE | |
| entry_topics → entries | CASCADE | |
| entry_topics → topics | CASCADE | |
| inbox_items → users | CASCADE | |
| audit_log → users | CASCADE | |
| user_settings → users | CASCADE | |
| refresh_tokens → users | CASCADE | |

---

## 12. Полная ER-диаграмма

```
┌─────────────────────────────────────────────────────────────────────┐
│                       REFERENCE CATALOG                             │
│                                                                     │
│  ref_entries ──┬── ref_senses ──┬── ref_translations                │
│                │                └── ref_examples                    │
│                ├── ref_pronunciations                               │
│                └── ref_images                                       │
└────────┬───────────────┬────────────────────────────────────────────┘
         │ SET NULL       │ CASCADE (M2M)
         ▼               ▼
┌─────────────────────────────────────────────────────────────────────┐
│                       USER DICTIONARY                               │
│                                                                     │
│  users ──┬── user_settings                                          │
│          ├── refresh_tokens                                         │
│          ├── entries ──┬── senses ──┬── translations                │
│          │             │            └── examples                    │
│          │             ├── entry_pronunciations (M2M → ref)         │
│          │             ├── entry_images (M2M → ref)                 │
│          │             ├── user_images                              │
│          │             ├── entry_topics (M2M → topics)              │
│          │             └── cards ── review_logs                     │
│          ├── topics                                                 │
│          ├── inbox_items                                            │
│          └── audit_log                                              │
└─────────────────────────────────────────────────────────────────────┘
```

---

## 13. Сводка таблиц

| # | Таблица | Слой | Записей (1k юзеров × 2k слов) |
|---|---------|------|-------------------------------|
| 1 | ref_entries | Catalog | ~50k |
| 2 | ref_senses | Catalog | ~150k |
| 3 | ref_translations | Catalog | ~300k |
| 4 | ref_examples | Catalog | ~300k |
| 5 | ref_pronunciations | Catalog | ~100k |
| 6 | ref_images | Catalog | ~50k |
| 7 | users | User | ~1k |
| 8 | user_settings | User | ~1k |
| 9 | refresh_tokens | User | ~2k |
| 10 | entries | User | ~2M |
| 11 | senses | User | ~6M (ссылки, ~16B каждая) |
| 12 | translations | User | ~12M (ссылки) |
| 13 | examples | User | ~12M (ссылки) |
| 14 | entry_pronunciations | User | ~4M (M2M, 32B каждая) |
| 15 | entry_images | User | ~2M (M2M, 32B каждая) |
| 16 | user_images | User | ~10k (только загруженные) |
| 17 | cards | User | ~2M |
| 18 | review_logs | User | ~20M (растёт со временем) |
| 19 | topics | User | ~10k |
| 20 | entry_topics | User | ~4M (M2M) |
| 21 | inbox_items | User | ~50k |
| 22 | audit_log | User | ~10M (растёт со временем) |

**Итого: 22 таблицы** (6 reference + 16 user).
