#!/usr/bin/env python3
"""Find words common to all (or N) artists.

Reads words.txt from each artist directory and produces:
  - common_words.csv       full stats: word, artist_count, artists list
  - common_words.txt       words shared by ALL artists
  - common_words_90.txt    words shared by ≥90% of artists

Usage:
    python build_common_words.py                # all artists
    python build_common_words.py --min-pct 80   # words in ≥80% of artists
"""

import argparse
import csv
from pathlib import Path

SCRIPT_DIR = Path(__file__).parent
OUTPUT_DIR = SCRIPT_DIR.parent / "output"


def load_words(words_path: Path) -> set[str]:
    """Load words.txt into a set."""
    with open(words_path, "r", encoding="utf-8") as f:
        return {line.strip() for line in f if line.strip()}


def main():
    parser = argparse.ArgumentParser(description="Find words common across artists.")
    parser.add_argument("--min-pct", type=int, default=100,
                        help="Minimum %% of artists a word must appear in (default: 100 = all)")
    args = parser.parse_args()

    # Collect all artist word sets
    artist_words: dict[str, set[str]] = {}
    for words_path in sorted(OUTPUT_DIR.glob("*/words.txt")):
        artist = words_path.parent.name
        words = load_words(words_path)
        if words:
            artist_words[artist] = words

    if not artist_words:
        print("No artist words.txt files found.")
        return

    total = len(artist_words)
    print(f"Found {total} artists with vocabulary data.\n")

    # Count how many artists use each word
    word_artists: dict[str, list[str]] = {}
    for artist, words in artist_words.items():
        for word in words:
            word_artists.setdefault(word, []).append(artist)

    # Sort by artist count (desc), then alphabetically
    all_words = sorted(word_artists.items(), key=lambda x: (-len(x[1]), x[0]))

    # Save full stats CSV
    csv_path = OUTPUT_DIR / "common_words.csv"
    with open(csv_path, "w", encoding="utf-8", newline="") as f:
        writer = csv.writer(f)
        writer.writerow(["word", "artist_count", "artist_pct", "artists"])
        for word, artists in all_words:
            pct = round(len(artists) / total * 100, 1)
            writer.writerow([word, len(artists), pct, "; ".join(sorted(artists))])
    print(f"Full stats: {csv_path} ({len(all_words)} unique words)")

    # Save common words at different thresholds
    for threshold_pct in [100, 90, 75, 50]:
        min_artists = max(1, int(total * threshold_pct / 100))
        common = sorted(w for w, a in all_words if len(a) >= min_artists)

        if threshold_pct == 100:
            out_path = OUTPUT_DIR / "common_words.txt"
        else:
            out_path = OUTPUT_DIR / f"common_words_{threshold_pct}.txt"

        with open(out_path, "w", encoding="utf-8") as f:
            f.write("\n".join(common) + "\n")
        print(f"  {threshold_pct}% ({min_artists}+ artists): {len(common):,} words → {out_path.name}")

    # Print summary
    universal = [w for w, a in all_words if len(a) == total]
    print(f"\nWords shared by ALL {total} artists: {len(universal)}")
    print(f"  Sample: {', '.join(universal[:30])}")


if __name__ == "__main__":
    main()
