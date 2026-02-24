#!/usr/bin/env python3
"""Build deduplicated common word pool from all book datasets.

Usage:
    python build_pool.py

For each book: reads words.txt, subtracts exclude.csv.
Merges remaining words into a deduplicated pool.

Output:
    output/common_pool.csv — word, books, book_count
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
        words_file = dir_path / "words.txt"
        exclude_file = dir_path / "exclude.csv"

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

    out_path = ROOT / "output" / "common_pool.csv"
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
