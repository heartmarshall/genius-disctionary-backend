# Books Dataset Pipeline — Design

**Date:** 2026-02-23
**Status:** Approved

## Goal

Full 3-stage word extraction pipeline for EPUB books, analogous to the TV Shows pipeline. Converts EPUB files into per-book word frequency CSVs, cross-book analysis, and a deduplicated common pool with exclude-list support.

## Input

72 EPUB files in `datasets/books/all_boks/`. Format: `Author Name - Book Title.epub`.

## Architecture

Mirrors TV Shows pipeline structure with spaCy instead of NLTK.

### Directory Layout

```
datasets/books/
├── books.yaml                    # Book registry: {dir_name: label}
├── Makefile                      # Orchestration: parse / unique / pool / all
├── parser/
│   ├── config.py                 # load_books() + book_dir() + epub_path()
│   ├── epub_parser.py            # EPUB → clean text lines per chapter
│   ├── word_processor.py         # spaCy: tokenize → lemmatize → POS → filter
│   ├── parse_book.py             # Stage 1: EPUB → {book}.csv + {book}_words.txt
│   ├── build_unique.py           # Stage 2: per-book unique + common_words
│   ├── build_pool.py             # Stage 3: deduplicated common_pool.csv
│   └── requirements.txt          # spacy, ebooklib, beautifulsoup4, lxml, pyyaml
│
├── all_boks/                     # Source EPUBs (unchanged)
│
├── the_great_gatsby/             # Per-book output directory
│   ├── the_great_gatsby.csv      # word, total_count, chapter_count, chapters
│   ├── the_great_gatsby_words.txt
│   ├── the_great_gatsby_unique.csv
│   └── the_great_gatsby_exclude.csv  (manual, later)
│
├── common_pool.csv               # Deduplicated pool with book coverage
├── common_words.csv              # Words in ALL books with per-book counts
└── common_words.txt              # Plain list of common words
```

### books.yaml

Maps EPUB filenames to directory names and labels:

```yaml
books:
  the_great_gatsby:
    label: Gatsby
    epub: "F. Scott Fitzgerald - The Great Gatsby.epub"
  animal_farm:
    label: AnimalFarm
    epub: "George Orwell - Animal Farm.epub"
  east_of_eden:
    label: EastOfEden
    epub: "John Steinbeck - East of Eden.epub"
  # ... all 72 books
```

- `dir_name` — output directory name (snake_case, short)
- `label` — short label for common_pool.csv (max ~12 chars)
- `epub` — filename in `all_boks/` (exact match)

### config.py

```python
ROOT = Path(__file__).resolve().parent.parent  # datasets/books/
EPUBS_DIR = ROOT / "all_boks"

def load_books() -> dict[str, dict]:
    """Return {dir_name: {"label": str, "epub": str}}."""

def book_dir(book_name: str) -> Path:
    """Return absolute path to book's output directory."""

def epub_path(book_name: str, books: dict) -> Path:
    """Return absolute path to book's EPUB file."""
```

### Stage 1: parse_book.py

**Input:** EPUB file via `epub_parser.py`
**Process:**
1. `epub_parser.extract_chapters(epub_path)` → list of `(chapter_num, text)` tuples
2. For each chapter: `word_processor.extract_words(text)` → `Counter`
3. Accumulate `{word: {"total": int, "chapters": {ch_num: count}}}`
4. Create book output directory
5. Write `{book}.csv` and `{book}_words.txt`

**Output CSV format:**
```csv
word,total_count,chapter_count,chapters
time,487,24,"1:23,2:18,3:21,..."
```

**CLI:**
```bash
python parse_book.py --book the_great_gatsby   # One book
python parse_book.py --all                      # All books
```

### epub_parser.py

Extracts clean text from EPUB, grouped by chapter.

```python
def extract_chapters(epub_path: Path) -> list[tuple[int, str]]:
    """Parse EPUB and return [(chapter_num, text), ...].

    Uses ebooklib for EPUB parsing, BeautifulSoup for HTML stripping.
    Chapter boundaries: <h1>, <h2>, <h3> tags.
    Strips: HTML tags, footnotes, copyright notices, TOC.
    """
```

- Uses `ebooklib.ITEM_DOCUMENT` to iterate spine items
- `BeautifulSoup("lxml")` for HTML → text
- Heading tags (`h1`, `h2`, `h3`) mark chapter boundaries
- Filters fragments < 40 chars
- Returns plain text per chapter (no markdown markers needed)

### word_processor.py

spaCy-based word processing (like lyrics pipeline but adapted for books).

```python
def extract_words(text: str) -> Counter:
    """Tokenize, lemmatize, filter, and count words from text.

    Uses spaCy en_core_web_sm for:
    - Tokenization
    - Lemmatization
    - POS tagging (keep NOUN, VERB, ADJ, ADV)
    - NER (reject PERSON, GPE, ORG, FAC, NORP, EVENT, WORK_OF_ART)
    - Stopword detection

    Filters:
    - Punctuation, spaces, numbers
    - Proper nouns via POS (PROPN)
    - Named entities via NER
    - Stopwords (spaCy built-in)
    - Words < 2 chars
    - Non-alphabetic tokens (must match [a-z][a-z']*[a-z]|[a-z])
    """
```

Key differences from TV Shows `word_processor.py`:
- spaCy instead of NLTK
- NER-based proper noun filtering (more accurate than POS-only)
- Same regex filter: `^[a-z][a-z']*[a-z]$|^[a-z]$`
- Same min length: 2 chars

### Stage 2: build_unique.py

Identical logic to TV Shows version:
- Load `{book}.csv` for all books
- Per-book unique words: `words_in_book - union(words_in_other_books)`
- Common words: `intersection(all_book_words)`
- Output: `{book}_unique.csv`, `common_words.csv`, `common_words.txt`

### Stage 3: build_pool.py

Identical logic to TV Shows version:
- Load `{book}_words.txt` per book
- Subtract `{book}_exclude.csv` if exists
- Merge into `common_pool.csv` with book labels and coverage count

### Makefile

```makefile
PYTHON := parser/.venv/bin/python3
PARSER := parser

parse:        # make parse BOOK=the_great_gatsby
parse-all:    # All books
unique:       # Build unique + common_words
pool:         # Build common_pool
all:          # Full pipeline
clean:        # Remove generated files
stats:        # Print statistics
setup:        # Create venv, install deps, download spaCy model
```

## Dependencies

```
spacy>=3.7,<4.0
ebooklib>=0.18
beautifulsoup4>=4.12
lxml>=5.0
pyyaml>=6.0
```

Plus: `python -m spacy download en_core_web_sm`

## Processing Notes

- spaCy model loaded once, reused across chapters (no threading needed — sequential processing)
- Large books (e.g., The Stand ~500K words) processed chapter-by-chapter to avoid memory spikes
- EPUB encoding: ebooklib handles decoding; BS4 handles malformed HTML
- Non-EPUB files in `all_boks/` (1 .rtf, 1 .fb2) are skipped — only `.epub` supported
