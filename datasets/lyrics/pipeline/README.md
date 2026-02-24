# Lyrics Dataset Pipeline

Download song lyrics from Genius, build clean datasets, and extract vocabulary for any artist.

## Quick Start

```bash
cd datasets/lyrics

# Full pipeline (download + build + analyze)
make pipeline GENIUS_TOKEN=your-token-here

# Rebuild dataset + vocabulary without re-downloading
make pipeline-skip-download

# Process specific artists only
make pipeline-only ONLY="Adele,Queen"

# Build vocabulary / common words separately
make vocabulary
make common-words
```

Get your Genius API token at https://genius.com/api-clients (create app -> copy Client Access Token).

## Configuration

Edit `pipeline/pipeline.yaml` to add/remove artists:

```yaml
artists:
  - name: "Radiohead"
    max_songs: 50
    min_word_count: 1

  - name: "Eminem"
    max_songs: 100
    min_word_count: 2   # exclude words that appear only once
```

| Field | Default | Description |
|-------|---------|-------------|
| `name` | required | Artist name as it appears on Genius |
| `max_songs` | 50 | Number of top songs to download (sorted by popularity) |
| `min_word_count` | 1 | Minimum total occurrences for a word to be included in vocabulary |

## Directory Structure

```
datasets/lyrics/
  Makefile                    # Pipeline commands
  pipeline/                   # Scripts and config
    pipeline.py               # Orchestrator (download + build + vocabulary)
    pipeline.yaml             # Artist configuration
    download_lyrics.py        # Step 1: Genius API → md_lyrics/ + _dataset.json
    build_dataset.py          # Step 2: _dataset.json → dataset.csv + dataset.json
    build_vocabulary.py       # Step 3: dataset.json → vocabulary.csv + words.txt
    build_common_words.py     # Cross-artist common words analysis
  output/                     # Generated data
    artists/
      adele/
        md_lyrics/            # Song lyrics as markdown (with metadata + section tags)
        clean_lyrics/         # Plain text lyrics (cleaned, no tags)
        _dataset.json         # Raw dataset from Genius API
        _index.md             # Song index with links
        dataset.json          # Clean dataset (lyrics + metadata)
        dataset.csv           # Same as above in CSV format
        vocabulary.csv        # Full vocabulary analysis
        words.txt             # Unique lemmas only (no stopwords), one per line
    common_words.txt          # Words shared by ALL artists
    common_words_90.txt       # Words shared by >=90% of artists
    common_words_75.txt       # Words shared by >=75% of artists
    common_words_50.txt       # Words shared by >=50% of artists
    common_words_stats.csv    # Full cross-artist word stats
```

## Pipeline Steps

```
pipeline.py
  |
  |-- Step 1: download_lyrics.py    Genius API -> md_lyrics/ + _dataset.json
  |-- Step 2: build_dataset.py      _dataset.json -> dataset.csv + dataset.json + clean_lyrics/
  |-- Step 3: build_vocabulary.py   dataset.json -> vocabulary.csv + words.txt
```

### Step 1 — Download Lyrics

Fetches top songs by popularity from Genius API. Saves each song as a markdown file with metadata (artist, album, release date, pageviews) and raw lyrics with section tags.

### Step 2 — Build Dataset

Cleans raw lyrics (removes section tags, Genius artifacts, normalizes whitespace) and produces a structured dataset in CSV and JSON formats. Also generates plain text lyrics files.

### Step 3 — Build Vocabulary

Uses spaCy NLP to tokenize, lemmatize, and POS-tag every word across all songs. Produces a detailed vocabulary CSV and a clean word list.

## Datasets

### dataset.csv

One row per song:

| Column | Example |
|--------|---------|
| `number` | 1 |
| `title` | Hello |
| `artist` | Adele |
| `album` | 25 (Target Exclusive) |
| `release_date` | October 23, 2015 |
| `pageviews` | 5802963 |
| `word_count` | 365 |
| `line_count` | 46 |
| `lyrics` | Hello, it's me... |

### vocabulary.csv

One row per unique word form, sorted by frequency:

| Column | Description | Example |
|--------|-------------|---------|
| `word` | Lowercase word as it appears in lyrics | crashed |
| `lemma` | Dictionary form (spaCy lemmatization) | crash |
| `pos` | Part of speech | VERB |
| `total_count` | Total occurrences across all songs | 12 |
| `song_count` | Number of songs containing this word | 5 |
| `songs` | Song titles (semicolon-separated) | Hello; Skyfall; ... |
| `avg_per_song` | Average uses per song where it appears | 2.4 |
| `is_stopword` | Common English stop word (the, and, is...) | False |
| `frequency_rank` | Rank by total frequency (1 = most frequent) | 42 |

### words.txt

One lemma per line, alphabetically sorted, stopwords excluded. Word forms are merged: `crash`, `crashed`, `crashes` -> `crash`.

## Running Scripts Individually

```bash
cd datasets/lyrics/pipeline

# Download only
GENIUS_TOKEN=xxx python3 download_lyrics.py "Linkin Park" --token $GENIUS_TOKEN --max-songs 100

# Build dataset only
python3 build_dataset.py ../output/artists/adele/

# Build vocabulary only
python3 build_vocabulary.py adele eminem --min-count 2
```

## Dependencies

```bash
pip install lyricsgenius spacy pandas pyyaml
python3 -m spacy download en_core_web_sm
```
