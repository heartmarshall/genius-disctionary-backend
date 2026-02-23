#!/usr/bin/env python3
"""Build deduplicated common word pool from all TV show datasets.

Usage:
    python build_pool.py

For each show: reads {show}_words.txt, subtracts {show}_exclude.csv.
Merges remaining words into a deduplicated pool.

Output:
    common_pool.csv — word, shows, show_count
"""

import csv
from collections import defaultdict
from pathlib import Path

from config import ROOT, load_shows, show_dir


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
    shows = load_shows()
    word_shows: dict[str, list[str]] = defaultdict(list)

    for name, label in shows.items():
        dir_path = show_dir(name)
        words_file = dir_path / f"{name}_words.txt"
        exclude_file = dir_path / f"{name}_exclude.csv"

        if not words_file.exists():
            print(f"Warning: {words_file} not found, skipping {name}")
            continue

        all_words = read_words_txt(words_file)
        exclude = read_csv_words(exclude_file) if exclude_file.exists() else set()
        clean = all_words - exclude

        print(
            f"{label:>8}: {len(all_words):>6} total"
            f" - {len(exclude):>5} exclude"
            f" = {len(clean):>6} clean"
        )

        for w in clean:
            word_shows[w].append(label)

    rows = sorted(word_shows.items(), key=lambda x: (-len(x[1]), x[0]))

    out_path = ROOT / "common_pool.csv"
    with open(out_path, "w", newline="") as f:
        writer = csv.writer(f)
        writer.writerow(["word", "shows", "show_count"])
        for word, word_show_list in rows:
            writer.writerow(
                [word, "|".join(sorted(word_show_list)), len(word_show_list)]
            )

    print(f"\nTotal unique words: {len(rows)}")
    print(f"Written to: {out_path.name}")

    by_count: dict[int, int] = defaultdict(int)
    for _, word_show_list in rows:
        by_count[len(word_show_list)] += 1
    print("\nDistribution:")
    for n in sorted(by_count):
        print(f"  in {n} show(s): {by_count[n]} words")

    print("\nDone.")


if __name__ == "__main__":
    main()
