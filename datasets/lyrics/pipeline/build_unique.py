#!/usr/bin/env python3
"""Build per-artist unique word lists and cross-artist common words.

Usage:
    python build_unique.py

Reads output/{artist}/dataset.csv for each artist.

Output:
    output/{artist}/unique.csv  — words exclusive to one artist (word, total_count)
    output/common_words.csv     — words in ALL artists (word, total, per-artist counts)
    output/common_words.txt     — same, plain word list
"""

import csv
import sys
from pathlib import Path

from config import ROOT, load_artists, artist_dir


def load_artist_words(name: str) -> dict[str, int]:
    """Load dataset.csv and return {word: total_count}."""
    csv_path = artist_dir(name) / "dataset.csv"
    if not csv_path.exists():
        print(f"Error: {csv_path} not found. Run vocabulary step first.", file=sys.stderr)
        sys.exit(1)
    words: dict[str, int] = {}
    with open(csv_path, encoding="utf-8") as f:
        for row in csv.DictReader(f):
            words[row["word"]] = int(row["total_count"])
    return words


def write_unique(name: str, unique: dict[str, int]) -> None:
    """Write unique.csv sorted by count DESC."""
    rows = sorted(unique.items(), key=lambda r: (-r[1], r[0]))
    output_path = artist_dir(name) / "unique.csv"
    with open(output_path, "w", newline="", encoding="utf-8") as f:
        writer = csv.writer(f)
        writer.writerow(["word", "total_count"])
        writer.writerows(rows)
    print(f"  {name}: {len(rows)} unique words -> {output_path.name}")
    if rows:
        top = ", ".join(f"{w}({c})" for w, c in rows[:5])
        print(f"    top 5: {top}")


def write_common(all_words: dict[str, dict[str, int]]) -> None:
    """Write common_words.csv and common_words.txt."""
    common = set.intersection(*(set(w.keys()) for w in all_words.values()))

    names = sorted(all_words.keys())
    rows = []
    for word in common:
        counts = {n: all_words[n][word] for n in names}
        total = sum(counts.values())
        rows.append((word, total, counts))
    rows.sort(key=lambda r: (-r[1], r[0]))

    csv_path = ROOT / "output" / "common_words.csv"
    with open(csv_path, "w", newline="", encoding="utf-8") as f:
        writer = csv.writer(f)
        writer.writerow(["word", "total"] + names)
        for word, total, counts in rows:
            writer.writerow([word, total] + [counts[n] for n in names])

    txt_path = ROOT / "output" / "common_words.txt"
    txt_path.write_text("\n".join(r[0] for r in rows) + "\n", encoding="utf-8")

    print(f"\n{len(common)} common words across {len(all_words)} artists")
    print(f"  -> {csv_path.name}")
    print(f"  -> {txt_path.name}")


def main() -> None:
    artists = load_artists()

    if len(artists) < 2:
        print("Error: need at least 2 artists with dataset.csv", file=sys.stderr)
        sys.exit(1)

    all_words: dict[str, dict[str, int]] = {}
    for name in artists:
        all_words[name] = load_artist_words(name)
        print(f"  Loaded {name}: {len(all_words[name])} words")

    print("\nUnique words:")
    for name in artists:
        other_words: set[str] = set()
        for other, words in all_words.items():
            if other != name:
                other_words.update(words.keys())
        unique = {w: c for w, c in all_words[name].items() if w not in other_words}
        write_unique(name, unique)

    write_common(all_words)

    print("\nDone.")


if __name__ == "__main__":
    main()
