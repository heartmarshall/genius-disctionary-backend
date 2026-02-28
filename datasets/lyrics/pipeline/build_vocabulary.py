#!/usr/bin/env python3
"""Build word frequency dataset.csv for each artist from lyrics.

For every artist directory that contains songs.json, produces dataset.csv with:
  - word (lowercase form as it appears in lyrics)
  - lemma (dictionary form via spaCy)
  - pos (part of speech: NOUN, VERB, ADJ, etc.)
  - total_count (how many times this word appears across all songs)
  - song_count (in how many songs this word appears)
  - songs (semicolon-separated list of song titles)
  - avg_per_song (average occurrences per song where it appears)
  - is_stopword (True/False — common English stop words)
  - frequency_rank (1 = most frequent)

Also produces words.txt — sorted unique lemmas (no stopwords).

Usage:
    python build_vocabulary.py                     # all artists
    python build_vocabulary.py adele eminem        # specific artists
    python build_vocabulary.py --min-count 2       # only words appearing 2+ times
"""

import argparse
import json
import re
from collections import Counter, defaultdict
from pathlib import Path

import pandas as pd
import spacy

# Section tags that may remain in lyrics: [Verse 1], [Chorus], etc.
SECTION_TAG_RE = re.compile(r"\[.*?\]")
# Parenthetical annotations like (Ooh), (Other side)
PAREN_RE = re.compile(r"\(.*?\)")
# Contractions we want to keep as single tokens (spaCy handles most)
# But we strip possessive 's separately to avoid "it's" → "it" + "'s" noise
NON_ALPHA_RE = re.compile(r"[^a-z'\-]")
# Multiple hyphens / apostrophes
MULTI_PUNCT_RE = re.compile(r"['\-]{2,}")

SCRIPT_DIR = Path(__file__).parent
OUTPUT_DIR = SCRIPT_DIR.parent / "output"


def load_artist_data(artist_dir: Path) -> list[dict] | None:
    """Load songs.json for an artist. Returns None if not found."""
    dataset_file = artist_dir / "songs.json"
    if not dataset_file.exists():
        return None
    with open(dataset_file, "r", encoding="utf-8") as f:
        return json.load(f)


def preprocess_lyrics(text: str) -> str:
    """Light preprocessing before spaCy tokenization."""
    text = SECTION_TAG_RE.sub(" ", text)
    text = PAREN_RE.sub(" ", text)
    # Normalize common lyric artifacts
    text = text.replace("\u2019", "'")  # curly apostrophe → straight
    text = text.replace("\u2018", "'")
    text = text.replace("\u201c", "")
    text = text.replace("\u201d", "")
    return text


def build_vocabulary(nlp, songs: list[dict], min_count: int = 1) -> pd.DataFrame:
    """Analyze all songs and build the vocabulary DataFrame."""
    # word_lower → { total_count, song_set, lemma, pos }
    vocab: dict[str, dict] = defaultdict(lambda: {
        "total_count": 0,
        "songs": set(),
        "lemma": "",
        "pos": "",
        "is_stopword": False,
    })

    for song in songs:
        lyrics = preprocess_lyrics(song.get("lyrics", ""))
        title = song.get("title", "Unknown")

        doc = nlp(lyrics)
        song_words = Counter()

        for token in doc:
            # Skip punctuation, spaces, numbers, single characters
            if token.is_punct or token.is_space or token.like_num:
                continue
            if len(token.text.strip()) <= 1 and token.text.lower() not in ("i", "a"):
                continue

            word = token.text.lower().strip("'\".,!?;:-")
            if not word or not any(c.isalpha() for c in word):
                continue

            song_words[word] += 1

            # Keep the most common POS/lemma (first occurrence wins)
            entry = vocab[word]
            if not entry["lemma"]:
                entry["lemma"] = token.lemma_.lower()
                entry["pos"] = token.pos_
                entry["is_stopword"] = token.is_stop

        # Merge song counts into global vocab
        for word, count in song_words.items():
            vocab[word]["total_count"] += count
            vocab[word]["songs"].add(title)

    # Build DataFrame
    rows = []
    for word, info in vocab.items():
        if info["total_count"] < min_count:
            continue
        rows.append({
            "word": word,
            "lemma": info["lemma"],
            "pos": info["pos"],
            "total_count": info["total_count"],
            "song_count": len(info["songs"]),
            "songs": "; ".join(sorted(info["songs"])),
            "avg_per_song": round(info["total_count"] / len(info["songs"]), 2),
            "is_stopword": info["is_stopword"],
        })

    df = pd.DataFrame(rows)
    if df.empty:
        return df

    df = df.sort_values("total_count", ascending=False).reset_index(drop=True)
    df["frequency_rank"] = df.index + 1
    return df


def process_artist(nlp, artist_dir: Path, min_count: int = 1):
    """Process one artist directory."""
    songs = load_artist_data(artist_dir)
    if songs is None:
        print(f"  Skipping {artist_dir.name}: no songs.json")
        return

    artist_name = songs[0].get("artist", artist_dir.name) if songs else artist_dir.name
    print(f"\n{'='*60}")
    print(f"  {artist_name} ({len(songs)} songs)")
    print(f"{'='*60}")

    df = build_vocabulary(nlp, songs, min_count=min_count)

    if df.empty:
        print("  No words found.")
        return

    # Save CSV
    csv_path = artist_dir / "dataset.csv"
    df.to_csv(csv_path, index=False, encoding="utf-8")

    # Summary stats
    total_words = df["total_count"].sum()
    unique_words = len(df)
    non_stop = df[~df["is_stopword"]]
    unique_content = len(non_stop)

    # Save words-only text file (unique lemmas, sorted alphabetically, no stopwords)
    words_path = artist_dir / "words.txt"
    lemmas_sorted = sorted(non_stop["lemma"].unique().tolist())
    with open(words_path, "w", encoding="utf-8") as f:
        f.write("\n".join(lemmas_sorted) + "\n")

    print(f"  Total word occurrences: {total_words:,}")
    print(f"  Unique words:           {unique_words:,}")
    print(f"  Unique content words:   {unique_content:,} (excl. stopwords)")
    print(f"  Top 15 content words:   {', '.join(non_stop.head(15)['word'].tolist())}")
    print(f"  Saved: {csv_path}")
    print(f"  Saved: {words_path} ({len(lemmas_sorted)} lemmas)")


def main():
    parser = argparse.ArgumentParser(description="Build vocabulary CSV for each artist.")
    parser.add_argument("artists", nargs="*", help="Artist directory names (default: all)")
    parser.add_argument("--min-count", type=int, default=1,
                        help="Minimum total count to include a word (default: 1)")
    args = parser.parse_args()

    print("Loading spaCy model...")
    nlp = spacy.load("en_core_web_sm", disable=["ner"])  # NER not needed
    nlp.max_length = 500_000

    if args.artists:
        dirs = [OUTPUT_DIR / name for name in args.artists]
    else:
        dirs = sorted([d for d in OUTPUT_DIR.iterdir() if d.is_dir()])

    for artist_dir in dirs:
        if not artist_dir.is_dir():
            print(f"  Warning: {artist_dir} is not a directory, skipping")
            continue
        process_artist(nlp, artist_dir, min_count=args.min_count)

    print(f"\nDone! Vocabulary CSVs saved to each artist directory.")


if __name__ == "__main__":
    main()
