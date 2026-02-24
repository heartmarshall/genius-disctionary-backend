# Datasets

Коллекция датасетов для извлечения словарного запаса из разных источников: сериалы, книги, тексты песен, академические и частотные списки, словари.

## Обзор датасетов

| Датасет | Размер | Тип | Назначение |
|---------|--------|-----|------------|
| **TV_shows/** | ~81 MB | Субтитры сериалов (.srt) | Разговорная лексика из диалогов |
| **books/** | ~470 MB | EPUB -> Markdown | Литературная лексика |
| **lyrics/** | ~8 MB | Тексты песен (Genius API) | Словарь из песен |
| **NGLS/** | ~430 KB | Академические списки | Частотные списки слов по доменам |
| **cmudict.dict** | 3.5 MB | CMU Pronouncing Dictionary | Фонетическая транскрипция |
| **kaikki.org-dictionary-English.jsonl** | 2.7 GB | Wiktionary JSONL | Полный толковый словарь |
| **english-wordnet-2025-plus-json/** | ~79 MB | WordNet JSON | Семантические связи между словами |
| **Sentence pairs in English-Russian** | ~85 MB | TSV | Параллельные переводы EN-RU |
| **NAWL_1.2_*.csv** | несколько KB | CSV | New Academic Word List |

---

## TV_shows: Пайплайн обработки сериалов

### Текущие сериалы

| Сериал | Сезонов | Уникальных слов | Файлы |
|--------|---------|-----------------|-------|
| Big Bang Theory | 12 | ~4,988 | big_bang_theory.csv |
| Breaking Bad | 5 | ~1,147 | breaking_bad.csv |
| Game of Thrones | 8 | ~1,425 | game_of_thrones.csv |
| The Office | 9 | ~3,623 | office.csv |
| South Park | 9+ | ~1,135 | south_park.csv |

### Структура директории

```
TV_shows/
├── parser/                     # Python-скрипты обработки
│   ├── main.py                 # Основной парсер: SRT -> CSV с частотами
│   ├── srt_parser.py           # Парсинг .srt файлов
│   ├── word_processor.py       # Токенизация, лемматизация, фильтрация (NLTK)
│   ├── unique_words.py         # Слова уникальные для каждого сериала
│   ├── common_words.py         # Слова общие для всех сериалов
│   ├── wordlist.py             # Простые списки слов (txt)
│   └── requirements.txt        # Зависимости: nltk>=3.8
├── <show_name>/                # Директория сериала
│   ├── s01/ ... sNN/           # Сезоны с .srt файлами
│   ├── <show_name>.csv         # Основной выходной файл (частоты)
│   ├── <show_name>_unique.csv  # Слова уникальные для этого сериала
│   ├── <show_name>_words.txt   # Список слов (одно на строку)
│   ├── exclusion_list.md       # Описание исключённых слов
│   └── exclusion_words.txt     # Список слов для исключения
├── common_words.csv            # Слова общие для всех сериалов
└── common_words.txt            # То же, в виде списка
```

### Как добавить новый сериал

#### 1. Подготовить субтитры

Скачать .srt файлы для всех сезонов. Создать директорию с именем в snake_case:

```
TV_shows/
└── friends/
    ├── s01/
    │   ├── Friends - 1x01 - The One Where Monica Gets a Roommate.srt
    │   ├── Friends - 1x02 - The One with the Sonogram at the End.srt
    │   └── ...
    ├── s02/
    │   └── ...
    └── s10/
        └── ...
```

**Требования к именам файлов:**
- Должны содержать паттерн `NxNN` (сезон x серия): `1x01`, `3x15`, `10x22`
- Парсер использует regex `(\d+)x(\d+)` для извлечения номера
- Допускается несколько .srt файлов на серию (разные источники) — парсер берёт первый по алфавиту
- Пример: `Show Name - 1x01 - Episode Title.720p.source.en.srt`

**Требования к директориям сезонов:**
- Имя директории может быть любым (`s01`, `season1`, `S01E01` и т.д.) — парсер рекурсивно ищет .srt файлы во всех поддиректориях
- Главное чтобы внутри директории сериала были поддиректории с .srt файлами

#### 2. Запустить основной парсер

```bash
cd datasets/TV_shows/parser

# Установить зависимости (один раз)
python -m venv .venv
source .venv/bin/activate
pip install -r requirements.txt

# Запустить парсер
python main.py --input .. --output ..
```

Результат — файл `TV_shows/friends/friends.csv` (или `TV_shows/friends.csv` если указан `--output`):

```csv
word,total_count,episode_count,episodes
get,5012,236,"1x01:15,1x02:22,1x03:18,..."
know,4944,235,"1x01:12,1x02:18,..."
```

**Пайплайн обработки каждого слова:**
1. Токенизация (NLTK `word_tokenize`)
2. POS-тегирование (Penn Treebank)
3. Лемматизация (WordNet Lemmatizer)
4. Фильтрация: удаляются стоп-слова, имена собственные (NNP/NNPS), части контракций (`n't`, `'s`, `'re`), неалфавитные токены, одиночные символы
5. Оставляются только контентные части речи: существительные, глаголы, прилагательные, наречия, модальные

#### 3. Составить exclusion list

Проверить CSV на артефакты и создать два файла:

**`friends/exclusion_words.txt`** — по одному слову на строку:
```
ross
rachel
monica
chandler
joey
phoebe
gunther
janice
```

**`friends/exclusion_list.md`** — документация с категориями:
```markdown
# Friends — Exclusion List

## Character Names (~50 words)
ross, rachel, monica, chandler, joey, phoebe, gunther, janice, ...

## Parsing Artifacts (~20 words)
...

## Non-English Words
...
```

Типичные категории для исключения:
- **Имена персонажей** (основные и второстепенные)
- **Артефакты парсинга** (обрезанные слова, склеенные слова)
- **Мусор/бессмыслица** (звукоподражания, междометия)
- **Не-английские слова** (если персонажи говорят на других языках)
- **Вымышленные слова** / неологизмы
- **Бренды и интернет** (если мешают)

#### 4. Сгенерировать дополнительные файлы

```bash
# Список уникальных слов (одно на строку)
python wordlist.py --input .. --output ..
# -> friends/friends_words.txt

# Слова уникальные для каждого сериала (относительно остальных)
python unique_words.py --input ..
# -> friends/friends_unique.csv

# Обновить общие слова для всех сериалов
python common_words.py --input ..
# -> common_words.txt, common_words.csv
```

#### 5. Итоговая структура нового сериала

```
TV_shows/friends/
├── s01/ ... s10/               # .srt файлы по сезонам
├── friends.csv                 # word, total_count, episode_count, episodes
├── friends_unique.csv          # word, total_count (только для этого шоу)
├── friends_words.txt           # отсортированный список лемм
├── exclusion_list.md           # документация по исключениям
└── exclusion_words.txt         # слова для исключения
```

### Формат выходных файлов

**`<show>.csv`** — основной датасет:
| Колонка | Описание |
|---------|----------|
| word | Лемматизированное слово |
| total_count | Общее количество вхождений во всех эпизодах |
| episode_count | В скольких эпизодах встречается |
| episodes | Детализация: `1x01:15,1x02:22,...` (сезонxсерия:количество) |

**`<show>_unique.csv`** — слова которых нет в других сериалах:
| Колонка | Описание |
|---------|----------|
| word | Лемматизированное слово |
| total_count | Общее количество вхождений |

**`common_words.csv`** — пересечение всех сериалов:
| Колонка | Описание |
|---------|----------|
| word | Слово |
| total | Суммарное количество |
| <show1>, <show2>, ... | Количество по каждому сериалу |

---

## Books: Литературные датасеты

**Скрипты:**
- `epub_to_markdown.py` — конвертирует EPUB в Markdown с маркерами `[ch:p]` (глава:параграф)
- `extract_words.py` — извлекает уникальные слова из EPUB (spaCy NLP)

**Содержимое:** 40+ книг в `all_boks_markdown/` — Children of Time, Brave New World, The Martian, Fight Club, Blood Meridian, For Whom the Bell Tolls и др.

---

## Lyrics: Пайплайн обработки текстов песен

### Текущие исполнители

| Исполнитель | Песен | Назначение |
|-------------|-------|------------|
| Adele | 50 | Поп-баллады, эмоциональная лексика |
| David Bowie | 50 | Арт-рок, разнообразный словарь |
| Eminem | 100 | Хип-хоп, сленг, сторителлинг |
| Linkin Park | 100 | Альтернатива, nu-metal |
| Queen | 30 | Классический рок, театральность |
| The Beatles | 50 | Британский рок, поэтичная лексика |

### Структура директории исполнителя

```
lyrics/<artist_name>/
├── _dataset.json         # Сырые данные с Genius API (промежуточный)
├── _index.md             # Markdown-индекс всех песен с ссылками
├── dataset.json          # Очищенный JSON-датасет
├── dataset.csv           # CSV-версия (для анализа в Excel/pandas)
├── vocabulary.csv        # NLP-анализ: все слова с частотами, POS, леммами
├── words.txt             # Уникальные леммы (без стоп-слов, по алфавиту)
├── md_lyrics/            # Markdown-файлы с метаданными по каждой песне
│   ├── 001_song_title.md
│   ├── 002_another_song.md
│   └── ...
└── clean_lyrics/         # Чистый текст без секционных тегов
    ├── 001_song_title.txt
    └── ...
```

### Скрипты

| Скрипт | Вход | Выход | Зависимости |
|--------|------|-------|-------------|
| `download_lyrics.py` | Имя исполнителя + Genius API token | `_dataset.json`, `_index.md`, `md_lyrics/*.md` | `lyricsgenius` |
| `build_dataset.py` | `_dataset.json` | `dataset.json`, `dataset.csv`, `clean_lyrics/` | stdlib |
| `build_vocabulary.py` | `dataset.json` | `vocabulary.csv`, `words.txt` | `spacy`, `pandas` |

### Как добавить нового исполнителя

#### Предварительные требования

```bash
# Python 3.10+
pip install lyricsgenius spacy pandas

# Языковая модель spaCy (один раз)
python -m spacy download en_core_web_sm
```

Genius API токен: зарегистрировать приложение на https://genius.com/api-clients и скопировать "Client Access Token".

#### Шаг 1. Скачать тексты с Genius

```bash
cd datasets/lyrics

python download_lyrics.py "Artist Name" \
  --token YOUR_GENIUS_TOKEN \
  --max-songs 50
```

| Флаг | По умолчанию | Описание |
|------|-------------|----------|
| `artist` (positional) | — | Имя исполнителя как на Genius |
| `--token` | — | Genius API access token (обязателен) |
| `--output` | `.` | Базовая директория для сохранения |
| `--max-songs` | `50` | Количество топ-песен по популярности |

**Что происходит:**
1. Поиск исполнителя на Genius по имени
2. Получение списка песен, отсортированных по pageviews (популярность)
3. Скачивание текста и метаданных каждой песни
4. Очистка артефактов Genius (`NNNEmbed`, `You might also like`, заголовок)
5. Инкрементальное сохранение — если скрипт упадёт, уже скачанные песни не потеряются

**Результат:** директория `lyrics/<artist_name>/` с `_dataset.json`, `_index.md` и `md_lyrics/*.md`.

> **Имя директории** создаётся автоматически из имени исполнителя: lowercase + пробелы → `_` (например, `Linkin Park` → `linkin_park`).

#### Шаг 2. Собрать чистый датасет

```bash
python build_dataset.py <artist_name>/
```

**Что происходит:**
1. Читает сырой `_dataset.json`
2. Удаляет секционные теги (`[Verse 1]`, `[Chorus]`, `[Bridge]`, `[Intro: Name]` и т.д.)
3. Удаляет артефакты Genius (`You might also like`, trailing `Embed`)
4. Нормализует пробелы и пустые строки
5. Считает `word_count` и `line_count` для каждой песни
6. Извлекает и нормализует метаданные альбомов

**Результат:** `dataset.json`, `dataset.csv`, `clean_lyrics/`.

#### Шаг 3. Построить словарь

```bash
# Конкретный исполнитель
python build_vocabulary.py <artist_name>

# Все исполнители сразу
python build_vocabulary.py

# Только слова с 2+ вхождениями
python build_vocabulary.py <artist_name> --min-count 2
```

**Что происходит:**
1. Загружает spaCy модель `en_core_web_sm`
2. Токенизирует все тексты (фильтрует пунктуацию, числа, одиночные символы)
3. Для каждого слова определяет лемму, POS-тег, принадлежность к стоп-словам
4. Считает частоту по песням и в целом
5. Ранжирует по убыванию частоты

**Результат:** `vocabulary.csv` и `words.txt`.

#### Шаг 4 (опционально). Ручная проверка

Проверить `vocabulary.csv` на артефакты:
- Обрезанные слова, мусор из Genius
- Слова на других языках
- Бессмысленные звукоподражания

### Форматы выходных файлов

**`dataset.json`** — массив объектов:
```json
{
  "number": 1,
  "title": "Bohemian Rhapsody",
  "artist": "Queen",
  "album": "A Night at the Opera",
  "album_release_date": "November 21, 1975",
  "release_date": "October 31, 1975",
  "genius_url": "https://genius.com/Queen-bohemian-rhapsody-lyrics",
  "genius_id": 1063,
  "pageviews": 10987236,
  "featured_artists": null,
  "word_count": 391,
  "line_count": 49,
  "lyrics": "Is this the real life?..."
}
```

**`vocabulary.csv`** — таблица слов:
| Колонка | Описание |
|---------|----------|
| `word` | Слово в lowercase (как в тексте) |
| `lemma` | Словарная форма (spaCy) |
| `pos` | Часть речи: NOUN, VERB, ADJ, ADV, PRON и т.д. |
| `total_count` | Общее количество вхождений |
| `song_count` | В скольких песнях встречается |
| `songs` | Список песен через `;` |
| `avg_per_song` | Среднее вхождений на песню |
| `is_stopword` | True/False — стоп-слово |
| `frequency_rank` | Ранг по частоте (1 = самое частое) |

**`words.txt`** — одна лемма на строку, только контентные слова (без стоп-слов), отсортированы по алфавиту.

**`md_lyrics/*.md`** — markdown с метаданными:
```markdown
# Song Title

**Artist:** Queen
**Album:** A Night at the Opera
**Released:** October 31, 1975
**Genius pageviews:** 10,987,236

---

[Verse 1]
Is this the real life?
Is this just fantasy?
...
```

### Быстрый старт: добавить исполнителя за 3 команды

```bash
cd datasets/lyrics
python download_lyrics.py "Radiohead" --token $GENIUS_TOKEN --max-songs 50
python build_dataset.py radiohead/
python build_vocabulary.py radiohead
```

---

## NGLS: Частотные списки

9 списков слов объединённых в `NGSL_combined.csv`:
- **NGSL** — New General Service List (2,800 наиболее частотных слов)
- **NAWL** — New Academic Word List (960 академических слов)
- **BSL** — Business Service List
- **FEL** — Financial English List
- **NDL** — Nursing Discipline List
- **TSL** — TOEIC Service List
- **NGSL-Spoken** — Spoken English variant
- **NGSL-GR** — Greek ranked list
- **Medical** — Medical terminology

**Скрипт:** `merge_lists.py` — парсит все форматы и объединяет в единый CSV с тегами.

---

## Справочные словари

- **cmudict.dict** — CMU Pronouncing Dictionary. Формат: `word PHONEME1 PHONEME2...` с маркерами ударения (0/1/2)
- **kaikki.org-dictionary-English.jsonl** — полный Wiktionary дамп в JSONL. Этимология, определения, примеры, формы слова
- **english-wordnet-2025-plus-json/** — WordNet в JSON. Файлы по алфавиту (entries-a.json...entries-z.json) и по семантическим категориям (noun.person.json, verb.motion.json и т.д.)
