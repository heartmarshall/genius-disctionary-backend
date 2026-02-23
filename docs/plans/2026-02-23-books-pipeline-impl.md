# Books Pipeline Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Build a 3-stage EPUB word extraction pipeline for 72 books, mirroring the TV Shows pipeline architecture.

**Architecture:** YAML config → EPUB parser (ebooklib+BS4) → spaCy word processor → per-book CSV + cross-book pool. Three stages: parse_book → build_unique → build_pool, orchestrated by Makefile.

**Tech Stack:** Python 3, spaCy (en_core_web_sm), ebooklib, BeautifulSoup4, lxml, PyYAML

**Reference files:**
- TV Shows config: `datasets/TV_shows/parser/config.py`
- TV Shows parser: `datasets/TV_shows/parser/parse_show.py`
- TV Shows word proc: `datasets/TV_shows/parser/word_processor.py`
- TV Shows build_unique: `datasets/TV_shows/parser/build_unique.py`
- TV Shows build_pool: `datasets/TV_shows/parser/build_pool.py`
- TV Shows Makefile: `datasets/TV_shows/Makefile`
- Lyrics vocabulary: `datasets/lyrics/pipeline/build_vocabulary.py` (spaCy reference)
- Existing EPUB converter: `datasets/books/epub_to_markdown.py` (ebooklib reference)

---

### Task 1: Create books.yaml config with all 72 books

**Files:**
- Create: `datasets/books/books.yaml`

**Step 1: Generate books.yaml**

Create `datasets/books/books.yaml` with all 72 EPUB files mapped to snake_case directory names and short labels.

Format:
```yaml
books:
  brave_new_world:
    label: BraveNew
    epub: "Aldous Huxley - Brave New World.epub"
  the_color_purple:
    label: ColorPurple
    epub: "Alice Walker - The Color Purple.epub"
  # ... all 72 books
```

Rules for generating entries:
- `dir_name`: book title in snake_case, lowercase, no author name. Drop articles ("the_", "a_") only if they START the title. Keep them if mid-title.
- `label`: Short readable label, max 12 chars. Use abbreviated title. No spaces.
- `epub`: Exact filename from `all_boks/` directory.
- Sort entries alphabetically by dir_name.
- Skip non-EPUB files (`.rtf`, `.fb2`).

For series (Harry Potter, multi-book authors), use numbered suffixes:
```yaml
  harry_potter_01:
    label: HP1
    epub: "J.K. Rowling - Harry Potter 01 - Sorcerer's Stone.epub"
```

**Step 2: Commit**

```bash
git add datasets/books/books.yaml
git commit -m "feat(books): add books.yaml config for 72 EPUB books"
```

---

### Task 2: Create parser/config.py and parser/requirements.txt

**Files:**
- Create: `datasets/books/parser/config.py`
- Create: `datasets/books/parser/requirements.txt`

**Step 1: Create config.py**

Model after `datasets/TV_shows/parser/config.py`. The key difference: books.yaml has a nested structure (`label` + `epub` per entry) vs TV Shows' flat `label`-only structure.

```python
"""Shared configuration loader for books pipeline scripts."""

import sys
from pathlib import Path

import yaml

# books/ is one level up from parser/
ROOT = Path(__file__).resolve().parent.parent
EPUBS_DIR = ROOT / "all_boks"


def load_books() -> dict[str, dict]:
    """Load books.yaml and return {dir_name: {"label": str, "epub": str}}."""
    config_path = ROOT / "books.yaml"
    if not config_path.exists():
        print(f"Error: {config_path} not found", file=sys.stderr)
        sys.exit(1)
    with open(config_path) as f:
        data = yaml.safe_load(f)
    return {name: info for name, info in data["books"].items()}


def book_dir(book_name: str) -> Path:
    """Return absolute path to a book's output directory."""
    return ROOT / book_name


def epub_path(book_name: str, books: dict) -> Path:
    """Return absolute path to a book's EPUB file."""
    return EPUBS_DIR / books[book_name]["epub"]
```

**Step 2: Create requirements.txt**

```
spacy>=3.7,<4.0
ebooklib>=0.18
beautifulsoup4>=4.12
lxml>=5.0
pyyaml>=6.0
```

**Step 3: Commit**

```bash
git add datasets/books/parser/config.py datasets/books/parser/requirements.txt
git commit -m "feat(books): add config loader and requirements"
```

---

### Task 3: Create parser/epub_parser.py

**Files:**
- Create: `datasets/books/parser/epub_parser.py`

**Step 1: Write epub_parser.py**

Extract clean text from EPUB files, grouped by chapter. Reuse ebooklib patterns from `datasets/books/epub_to_markdown.py` but output plain text instead of markdown.

```python
"""EPUB parser — extract clean text per chapter from EPUB files."""

import re
from pathlib import Path

import ebooklib
from ebooklib import epub
from bs4 import BeautifulSoup

HEADING_TAGS = {"h1", "h2", "h3"}
MIN_TEXT_LEN = 40  # Skip tiny fragments


def extract_chapters(epub_path: Path) -> list[tuple[int, str]]:
    """Parse EPUB and return [(chapter_num, text), ...].

    Chapter boundaries detected by <h1>, <h2>, <h3> tags.
    Returns plain text with HTML stripped.
    """
    book = epub.read_epub(str(epub_path), options={"ignore_ncx": True})

    chapter_num = 0
    chapters: dict[int, list[str]] = {}

    for item in book.get_items_of_type(ebooklib.ITEM_DOCUMENT):
        html = item.get_body_content().decode("utf-8", errors="replace")
        if not html.strip():
            continue

        soup = BeautifulSoup(html, "lxml")
        has_headings = bool(soup.find(HEADING_TAGS))

        # If no headings and substantial text, treat as new chapter.
        if not has_headings:
            plain = soup.get_text(strip=True)
            if len(plain) > 200:
                chapter_num += 1
                chapters.setdefault(chapter_num, [])

        for el in soup.find_all(["h1", "h2", "h3", "p", "blockquote"]):
            tag = el.name.lower()

            if tag in HEADING_TAGS:
                heading_text = el.get_text(strip=True)
                if heading_text:
                    chapter_num += 1
                    chapters.setdefault(chapter_num, [])
                continue

            text = el.get_text(separator=" ", strip=True)
            if not text or len(text) < MIN_TEXT_LEN:
                continue

            chapters.setdefault(chapter_num, []).append(text)

    # Build result: join paragraphs per chapter
    result = []
    for ch_num in sorted(chapters):
        text = "\n".join(chapters[ch_num])
        if text.strip():
            result.append((ch_num, text))

    return result
```

**Step 2: Smoke test**

```bash
cd datasets/books
parser/.venv/bin/python3 -c "
from parser.epub_parser import extract_chapters
from pathlib import Path
chapters = extract_chapters(Path('all_boks/George Orwell - Animal Farm.epub'))
print(f'{len(chapters)} chapters')
for ch_num, text in chapters[:3]:
    print(f'  ch {ch_num}: {len(text)} chars, starts: {text[:80]}...')
"
```

Expected: Multiple chapters extracted with substantial text.

**Step 3: Commit**

```bash
git add datasets/books/parser/epub_parser.py
git commit -m "feat(books): add EPUB chapter parser"
```

---

### Task 4: Create parser/word_processor.py

**Files:**
- Create: `datasets/books/parser/word_processor.py`

**Step 1: Write word_processor.py**

spaCy-based word extraction. Combines patterns from lyrics `build_vocabulary.py` (spaCy usage) with the filtering approach from TV Shows `word_processor.py` (strict POS + regex filtering).

```python
"""Word tokenization, lemmatization, and filtering using spaCy."""

import re
from collections import Counter

import spacy

RE_ONLY_ALPHA = re.compile(r"^[a-z][a-z']*[a-z]$|^[a-z]$")

# POS tags to keep: nouns, verbs, adjectives, adverbs.
KEEP_POS = {"NOUN", "VERB", "ADJ", "ADV"}

# NER labels to reject (proper nouns / named entities).
REJECT_NER = {"PERSON", "GPE", "ORG", "FAC", "NORP", "EVENT", "WORK_OF_ART", "LOC"}

# Singleton — loaded once, reused across calls.
_nlp = None


def _get_nlp():
    """Load spaCy model on first use."""
    global _nlp
    if _nlp is None:
        _nlp = spacy.load("en_core_web_sm")
        _nlp.max_length = 600_000  # Handle large chapters
    return _nlp


def extract_words(text: str) -> Counter:
    """Tokenize, lemmatize, filter, and count words from text.

    Uses spaCy en_core_web_sm for tokenization, lemmatization, POS tagging, NER.

    Keeps: NOUN, VERB, ADJ, ADV (non-proper, non-entity, non-stop).
    Rejects: punctuation, numbers, proper nouns, named entities, stopwords,
             words < 2 chars, non-alphabetic tokens.
    """
    nlp = _get_nlp()
    doc = nlp(text)

    # Collect entity spans for fast lookup.
    ent_tokens: set[int] = set()
    for ent in doc.ents:
        if ent.label_ in REJECT_NER:
            for token in ent:
                ent_tokens.add(token.i)

    counts: Counter = Counter()

    for token in doc:
        # Skip punctuation, spaces, numbers.
        if token.is_punct or token.is_space or token.like_num:
            continue

        # Skip proper nouns by POS.
        if token.pos_ == "PROPN":
            continue

        # Skip named entities.
        if token.i in ent_tokens:
            continue

        # Only keep content POS.
        if token.pos_ not in KEEP_POS:
            continue

        # Lemmatize and lowercase.
        lemma = token.lemma_.lower().replace("\u2019", "'")

        # Regex validation: only alphabetic + apostrophes.
        if not RE_ONLY_ALPHA.match(lemma):
            continue

        # Min length 2.
        if len(lemma) < 2:
            continue

        # Skip stopwords.
        if token.is_stop:
            continue

        counts[lemma] += 1

    return counts
```

**Step 2: Smoke test**

```bash
cd datasets/books
parser/.venv/bin/python3 -c "
from parser.word_processor import extract_words
counts = extract_words('The old man walked slowly to the bright harbor, thinking about life and death.')
print(dict(counts.most_common(20)))
"
```

Expected: Words like `walk`, `slowly`, `bright`, `harbor`, `think`, `life`, `death` present. `The`, `old`, `man`, `to` (stopwords/articles) absent. No proper nouns.

**Step 3: Commit**

```bash
git add datasets/books/parser/word_processor.py
git commit -m "feat(books): add spaCy-based word processor"
```

---

### Task 5: Create parser/parse_book.py (Stage 1)

**Files:**
- Create: `datasets/books/parser/parse_book.py`

**Step 1: Write parse_book.py**

Model directly after `datasets/TV_shows/parser/parse_show.py`. Replaces episodes with chapters.

```python
#!/usr/bin/env python3
"""Parse EPUB books into word frequency datasets.

Usage:
    python parse_book.py --book the_great_gatsby   # One book
    python parse_book.py --all                      # All books

Output per book:
    {book}/{book}.csv        — word, total_count, chapter_count, chapters
    {book}/{book}_words.txt  — sorted unique words, one per line
"""

import argparse
import csv
import sys
from collections import defaultdict
from pathlib import Path

from config import load_books, book_dir, epub_path
from epub_parser import extract_chapters
from word_processor import extract_words


def process_book(epub_file: Path) -> dict[str, dict]:
    """Process all chapters of a book.

    Returns {word: {"total": int, "chapters": {ch_num: count, ...}}}.
    """
    chapters = extract_chapters(epub_file)
    words: dict[str, dict] = defaultdict(lambda: {"total": 0, "chapters": {}})

    for ch_num, text in chapters:
        word_counts = extract_words(text)
        for word, count in word_counts.items():
            words[word]["total"] += count
            words[word]["chapters"][ch_num] = (
                words[word]["chapters"].get(ch_num, 0) + count
            )

    return dict(words)


def write_csv(words: dict[str, dict], output_path: Path) -> None:
    """Write word frequency CSV: word, total_count, chapter_count, chapters."""
    rows = []
    for word, data in words.items():
        ch_list = sorted(data["chapters"].keys())
        ch_str = ",".join(f"{ch}:{data['chapters'][ch]}" for ch in ch_list)
        rows.append((word, data["total"], len(ch_list), ch_str))

    rows.sort(key=lambda r: (-r[1], r[0]))

    with open(output_path, "w", newline="", encoding="utf-8") as f:
        writer = csv.writer(f)
        writer.writerow(["word", "total_count", "chapter_count", "chapters"])
        for word, total, ch_count, chapters in rows:
            writer.writerow([word, total, ch_count, chapters])


def write_wordlist(words: dict[str, dict], output_path: Path) -> None:
    """Write sorted unique word list, one per line."""
    sorted_words = sorted(words.keys())
    output_path.write_text("\n".join(sorted_words) + "\n", encoding="utf-8")


def run_book(book_name: str, books: dict) -> None:
    """Parse EPUB for one book and write both outputs."""
    epub_file = epub_path(book_name, books)
    if not epub_file.exists():
        print(f"Error: EPUB not found: {epub_file}", file=sys.stderr)
        sys.exit(1)

    out_dir = book_dir(book_name)
    out_dir.mkdir(exist_ok=True)

    print(f"Parsing {book_name}...")
    words = process_book(epub_file)

    if not words:
        print(f"  No data extracted, skipping.")
        return

    all_chapters = set()
    for data in words.values():
        all_chapters.update(data["chapters"].keys())

    csv_path = out_dir / f"{book_name}.csv"
    write_csv(words, csv_path)

    txt_path = out_dir / f"{book_name}_words.txt"
    write_wordlist(words, txt_path)

    print(f"  {len(all_chapters)} chapters, {len(words)} unique words")
    print(f"  -> {csv_path.name}")
    print(f"  -> {txt_path.name}")


def main() -> None:
    parser = argparse.ArgumentParser(
        description="Parse EPUB books into word frequency datasets."
    )
    group = parser.add_mutually_exclusive_group(required=True)
    group.add_argument("--book", type=str, help="Book directory name to parse")
    group.add_argument("--all", action="store_true", help="Parse all books")
    args = parser.parse_args()

    books = load_books()

    if args.all:
        for name in books:
            run_book(name, books)
    else:
        if args.book not in books:
            print(
                f"Error: '{args.book}' not in books.yaml. "
                f"Available: {', '.join(sorted(books))}",
                file=sys.stderr,
            )
            sys.exit(1)
        run_book(args.book, books)

    print("Done.")


if __name__ == "__main__":
    main()
```

**Step 2: Test with one book**

```bash
cd datasets/books
parser/.venv/bin/python3 parser/parse_book.py --book animal_farm
```

Expected output:
```
Parsing animal_farm...
  N chapters, NNNN unique words
  -> animal_farm.csv
  -> animal_farm_words.txt
Done.
```

Verify files:
```bash
head -5 animal_farm/animal_farm.csv
wc -l animal_farm/animal_farm_words.txt
```

**Step 3: Commit**

```bash
git add datasets/books/parser/parse_book.py
git commit -m "feat(books): add parse_book.py (stage 1 - EPUB to word CSV)"
```

---

### Task 6: Create parser/build_unique.py (Stage 2)

**Files:**
- Create: `datasets/books/parser/build_unique.py`

**Step 1: Write build_unique.py**

Nearly identical to `datasets/TV_shows/parser/build_unique.py`. Only difference: uses `load_books()` instead of `load_shows()`, and book names instead of show names.

```python
#!/usr/bin/env python3
"""Build per-book unique word lists and cross-book common words.

Usage:
    python build_unique.py

Reads {book}/{book}.csv for each book in books.yaml.

Output:
    {book}/{book}_unique.csv  — words exclusive to one book (word, total_count)
    common_words.csv          — words in ALL books (word, total, per-book counts)
    common_words.txt          — same, plain word list
"""

import csv
import sys
from pathlib import Path

from config import ROOT, load_books, book_dir


def load_book_words(book_name: str) -> dict[str, int]:
    """Load {book}.csv and return {word: total_count}."""
    csv_path = book_dir(book_name) / f"{book_name}.csv"
    if not csv_path.exists():
        print(f"Error: {csv_path} not found. Run parse first.", file=sys.stderr)
        sys.exit(1)
    words: dict[str, int] = {}
    with open(csv_path, encoding="utf-8") as f:
        for row in csv.DictReader(f):
            words[row["word"]] = int(row["total_count"])
    return words


def write_unique(book_name: str, unique: dict[str, int]) -> None:
    """Write {book}_unique.csv sorted by count DESC."""
    rows = sorted(unique.items(), key=lambda r: (-r[1], r[0]))
    output_path = book_dir(book_name) / f"{book_name}_unique.csv"
    with open(output_path, "w", newline="", encoding="utf-8") as f:
        writer = csv.writer(f)
        writer.writerow(["word", "total_count"])
        writer.writerows(rows)
    print(f"  {book_name}: {len(rows)} unique words -> {output_path.name}")
    if rows:
        top = ", ".join(f"{w}({c})" for w, c in rows[:5])
        print(f"    top 5: {top}")


def write_common(book_words: dict[str, dict[str, int]]) -> None:
    """Write common_words.csv and common_words.txt."""
    common = set.intersection(*(set(w.keys()) for w in book_words.values()))

    book_names = sorted(book_words.keys())
    rows = []
    for word in common:
        counts = {b: book_words[b][word] for b in book_names}
        total = sum(counts.values())
        rows.append((word, total, counts))
    rows.sort(key=lambda r: (-r[1], r[0]))

    csv_path = ROOT / "common_words.csv"
    with open(csv_path, "w", newline="", encoding="utf-8") as f:
        writer = csv.writer(f)
        writer.writerow(["word", "total"] + book_names)
        for word, total, counts in rows:
            writer.writerow([word, total] + [counts[b] for b in book_names])

    txt_path = ROOT / "common_words.txt"
    txt_path.write_text("\n".join(r[0] for r in rows) + "\n", encoding="utf-8")

    print(f"\n{len(common)} common words across {len(book_words)} books")
    print(f"  -> {csv_path.name}")
    print(f"  -> {txt_path.name}")


def main() -> None:
    books = load_books()

    if len(books) < 2:
        print("Error: need at least 2 books in books.yaml", file=sys.stderr)
        sys.exit(1)

    book_words: dict[str, dict[str, int]] = {}
    for name in books:
        book_words[name] = load_book_words(name)
        print(f"  Loaded {name}: {len(book_words[name])} words")

    print("\nUnique words:")
    for name in books:
        other_words: set[str] = set()
        for other, words in book_words.items():
            if other != name:
                other_words.update(words.keys())
        unique = {w: c for w, c in book_words[name].items() if w not in other_words}
        write_unique(name, unique)

    write_common(book_words)

    print("\nDone.")


if __name__ == "__main__":
    main()
```

**Step 2: Commit**

```bash
git add datasets/books/parser/build_unique.py
git commit -m "feat(books): add build_unique.py (stage 2 - cross-book analysis)"
```

---

### Task 7: Create parser/build_pool.py (Stage 3)

**Files:**
- Create: `datasets/books/parser/build_pool.py`

**Step 1: Write build_pool.py**

Nearly identical to `datasets/TV_shows/parser/build_pool.py`.

```python
#!/usr/bin/env python3
"""Build deduplicated common word pool from all book datasets.

Usage:
    python build_pool.py

For each book: reads {book}_words.txt, subtracts {book}_exclude.csv.
Merges remaining words into a deduplicated pool.

Output:
    common_pool.csv — word, books, book_count
"""

import csv
from collections import defaultdict
from pathlib import Path

from config import ROOT, load_books, book_dir


def read_words_txt(path: Path) -> set[str]:
    """Read a plain word list file."""
    words = set()
    with open(path) as f:
        for line in f:
            w = line.strip().lower()
            if w:
                words.add(w)
    return words


def read_csv_words(path: Path) -> set[str]:
    """Read the 'word' column from a CSV file."""
    words = set()
    with open(path) as f:
        for row in csv.DictReader(f):
            w = row["word"].strip().lower()
            if w:
                words.add(w)
    return words


def main() -> None:
    books = load_books()
    word_books: dict[str, list[str]] = defaultdict(list)

    for name, info in books.items():
        dir_path = book_dir(name)
        words_file = dir_path / f"{name}_words.txt"
        exclude_file = dir_path / f"{name}_exclude.csv"

        if not words_file.exists():
            print(f"Warning: {words_file} not found, skipping {name}")
            continue

        all_words = read_words_txt(words_file)
        exclude = read_csv_words(exclude_file) if exclude_file.exists() else set()
        clean = all_words - exclude

        label = info["label"]
        print(
            f"{label:>12}: {len(all_words):>6} total"
            f" - {len(exclude):>5} exclude"
            f" = {len(clean):>6} clean"
        )

        for w in clean:
            word_books[w].append(label)

    rows = sorted(word_books.items(), key=lambda x: (-len(x[1]), x[0]))

    out_path = ROOT / "common_pool.csv"
    with open(out_path, "w", newline="") as f:
        writer = csv.writer(f)
        writer.writerow(["word", "books", "book_count"])
        for word, word_book_list in rows:
            writer.writerow(
                [word, "|".join(sorted(word_book_list)), len(word_book_list)]
            )

    print(f"\nTotal unique words: {len(rows)}")
    print(f"Written to: {out_path.name}")

    by_count: dict[int, int] = defaultdict(int)
    for _, word_book_list in rows:
        by_count[len(word_book_list)] += 1
    print("\nDistribution:")
    for n in sorted(by_count):
        print(f"  in {n} book(s): {by_count[n]} words")

    print("\nDone.")


if __name__ == "__main__":
    main()
```

**Step 2: Commit**

```bash
git add datasets/books/parser/build_pool.py
git commit -m "feat(books): add build_pool.py (stage 3 - deduplicated common pool)"
```

---

### Task 8: Create Makefile

**Files:**
- Create: `datasets/books/Makefile`

**Step 1: Write Makefile**

Model after `datasets/TV_shows/Makefile`.

```makefile
# Books Dataset Pipeline
#
# Usage:
#   make parse BOOK=animal_farm    Parse one book's EPUB
#   make parse-all                  Parse all books
#   make unique                     Build unique + common_words
#   make pool                       Build common_pool
#   make all                        Full pipeline (parse-all → unique → pool)
#   make clean                      Remove generated files
#   make stats                      Print dataset statistics

PYTHON := parser/.venv/bin/python3
PARSER := parser

.PHONY: parse parse-all unique pool all clean stats setup

# Parse a single book (requires BOOK=<name>)
parse:
ifndef BOOK
	$(error Usage: make parse BOOK=<book_name>)
endif
	$(PYTHON) $(PARSER)/parse_book.py --book $(BOOK)

# Parse all books
parse-all:
	$(PYTHON) $(PARSER)/parse_book.py --all

# Build unique words per book + common words
unique:
	$(PYTHON) $(PARSER)/build_unique.py

# Build common pool (words minus excludes)
pool:
	$(PYTHON) $(PARSER)/build_pool.py

# Full pipeline
all: parse-all unique pool

# Remove generated files (keeps exclude CSVs and EPUBs)
clean:
	rm -f common_pool.csv common_words.csv common_words.txt
	@for dir in $$($(PYTHON) -c "import sys; sys.path.insert(0,'parser'); from config import load_books; [print(b) for b in load_books()]"); do \
		rm -f $$dir/$$dir.csv $$dir/$${dir}_words.txt $$dir/$${dir}_unique.csv; \
	done
	@echo "Cleaned generated files."

# Print statistics
stats:
	@echo "=== Dataset Statistics ==="
	@for dir in $$($(PYTHON) -c "import sys; sys.path.insert(0,'parser'); from config import load_books; [print(b) for b in load_books()]"); do \
		words=$$(wc -l < $$dir/$${dir}_words.txt 2>/dev/null || echo "N/A"); \
		unique=$$(tail -n +2 $$dir/$${dir}_unique.csv 2>/dev/null | wc -l || echo "N/A"); \
		exclude=$$(tail -n +2 $$dir/$${dir}_exclude.csv 2>/dev/null | wc -l || echo "N/A"); \
		printf "  %-30s words: %6s  unique: %5s  exclude: %5s\n" "$$dir" "$$words" "$$unique" "$$exclude"; \
	done
	@pool=$$(tail -n +2 common_pool.csv 2>/dev/null | wc -l || echo "N/A"); \
	common=$$(tail -n +2 common_words.csv 2>/dev/null | wc -l || echo "N/A"); \
	echo "  ---"; \
	printf "  %-30s %s\n" "common_pool.csv" "$$pool words"; \
	printf "  %-30s %s\n" "common_words.csv" "$$common words"

# Install dependencies
setup:
	cd $(PARSER) && python3 -m venv .venv && .venv/bin/pip install -r requirements.txt && .venv/bin/python3 -m spacy download en_core_web_sm
```

**Step 2: Commit**

```bash
git add datasets/books/Makefile
git commit -m "feat(books): add Makefile for pipeline orchestration"
```

---

### Task 9: Setup venv and run full pipeline test

**Step 1: Setup**

```bash
cd datasets/books
make setup
```

Expected: venv created, deps installed, spaCy model downloaded.

**Step 2: Parse 2-3 test books**

```bash
make parse BOOK=animal_farm
make parse BOOK=the_great_gatsby
make parse BOOK=fahrenheit_451
```

Verify each produces `{book}.csv` and `{book}_words.txt` with reasonable word counts.

**Step 3: Verify CSV format**

```bash
head -5 animal_farm/animal_farm.csv
```

Expected:
```
word,total_count,chapter_count,chapters
...
```

**Step 4: Commit generated output directories to .gitignore or just note them**

Do NOT commit generated CSV/txt files. Only commit code and config.

---

### Task 10: Run full pipeline on all 72 books

**Step 1: Parse all**

```bash
cd datasets/books
make parse-all
```

This will take a while (72 books × spaCy processing). Monitor progress via stdout.

**Step 2: Build unique + common words**

```bash
make unique
```

**Step 3: Build common pool**

```bash
make pool
```

**Step 4: Check stats**

```bash
make stats
```

**Step 5: Verify common_pool.csv**

```bash
head -20 common_pool.csv
wc -l common_pool.csv
```

**Step 6: Final commit (code only)**

```bash
git add datasets/books/parser/ datasets/books/books.yaml datasets/books/Makefile
git commit -m "feat(books): complete 3-stage EPUB word extraction pipeline"
```
