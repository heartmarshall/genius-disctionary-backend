# Datasets

Word frequency datasets extracted from books, TV shows, song lyrics, and academic lists for the MyEnglish language learning app.

## Structure

All dataset categories follow a unified layout:

```
datasets/
├── books/          56 books (EPUB -> word frequencies)
├── tv_shows/       8 shows (SRT subtitles -> word frequencies)
├── lyrics/         63 artists (Genius API -> word frequencies)
├── NGLS/           Academic frequency lists (NGSL, NAWL, BSL, etc.)
├── english-wordnet-2025/   WordNet JSON (gitignored)
├── cmudict.dict            CMU pronunciation dictionary (gitignored)
└── kaikki.org-*.jsonl      Wiktionary dump, 2.7 GB (gitignored)
```

Each category (books, tv_shows, lyrics) has the same internal structure:

```
{category}/
├── config.yaml        # List of items to process
├── Makefile           # make parse-all, make unique, make pool, etc.
├── pipeline/          # Python scripts + .venv + requirements.txt
├── sources/           # Raw data: EPUBs, SRTs, lyrics (gitignored)
├── output/            # Processed data per item
│   ├── common_pool.csv, common_words.csv, common_words.txt
│   └── {item}/
│       ├── dataset.csv    # word, total_count, chapter/episode/song_count, details
│       ├── unique.csv     # Words exclusive to this item
│       ├── words.txt      # Sorted unique lemmas (no stopwords)
│       └── exclude.csv    # Words to exclude (names, artifacts)
└── references/        # Supplementary materials (lyrics only)
```

## Quick start

```bash
cd datasets/{category}

# First time: install dependencies
make setup

# Run full pipeline
make all            # books, tv_shows
make pipeline       # lyrics (requires GENIUS_TOKEN)

# Single item
make parse BOOK=animal_farm
make parse SHOW=breaking_bad

# Rebuild derived files
make unique         # Per-item unique words + cross-item common words
make pool           # Deduplicated word pool (words - excludes)
```

## Pipeline overview

| Step | books | tv_shows | lyrics |
|------|-------|----------|--------|
| Source | EPUB files | SRT subtitles | Genius API |
| NLP | spaCy (lemma, POS, NER) | NLTK (lemma, POS) | spaCy (lemma, POS) |
| Filter | NOUN, VERB, ADJ, ADV; no proper nouns, stopwords, entities | Same POS set; no proper nouns, stopwords | All tokens with frequency stats |
| Output | `dataset.csv`, `words.txt`, `unique.csv` | Same | Same + `songs.csv`, `songs.json` |

## Output file formats

**dataset.csv** -- main word frequency table:
- `word` -- lemmatized word
- `total_count` -- total occurrences
- `{unit}_count` -- in how many chapters/episodes/songs
- `{units}` -- breakdown: `1:15,2:22,...` or `1x01:15,1x02:22,...`

**unique.csv** -- words found only in this item (not in any other within the category).

**common_words.csv** -- words shared across all items, with per-item counts.

**common_pool.csv** -- all words from all items minus excludes, deduplicated.

## NGSL

Academic frequency lists merged into `NGSL_combined.csv`: NGSL (2800 general), NAWL (960 academic), BSL (business), TSL (TOEIC), FEL (finance), NDL (nursing), NGSL-Spoken, Medical.
