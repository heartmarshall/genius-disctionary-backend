#!/usr/bin/env python3
"""Build per-book unique word lists and cross-book common words.

Usage:
    python build_unique.py

Reads output/{book}/dataset.csv for each book in config.yaml.

Output:
    output/{book}/unique.csv      — words exclusive to one book (word, total_count)
    output/common_words.csv       — words in ALL books (word, total, per-book counts)
    output/common_words.txt       — same, plain word list
"""

import csv
import sys
from pathlib import Path

from config import ROOT, load_books, book_dir


def load_book_words(book_name: str) -> dict[str, int]:
    """Load dataset.csv and return {word: total_count}."""
    csv_path = book_dir(book_name) / "dataset.csv"
    if not csv_path.exists():
        print(f"Error: {csv_path} not found. Run parse first.", file=sys.stderr)
        sys.exit(1)
    words: dict[str, int] = {}
    with open(csv_path, encoding="utf-8") as f:
        for row in csv.DictReader(f):
            words[row["word"]] = int(row["total_count"])
    return words


def write_unique(book_name: str, unique: dict[str, int]) -> None:
    """Write unique.csv sorted by count DESC."""
    rows = sorted(unique.items(), key=lambda r: (-r[1], r[0]))
    output_path = book_dir(book_name) / "unique.csv"
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

    csv_path = ROOT / "output" / "common_words.csv"
    with open(csv_path, "w", newline="", encoding="utf-8") as f:
        writer = csv.writer(f)
        writer.writerow(["word", "total"] + book_names)
        for word, total, counts in rows:
            writer.writerow([word, total] + [counts[b] for b in book_names])

    txt_path = ROOT / "output" / "common_words.txt"
    txt_path.write_text("\n".join(r[0] for r in rows) + "\n", encoding="utf-8")

    print(f"\n{len(common)} common words across {len(book_words)} books")
    print(f"  -> {csv_path.name}")
    print(f"  -> {txt_path.name}")


def main() -> None:
    books = load_books()

    if len(books) < 2:
        print("Error: need at least 2 books in config.yaml", file=sys.stderr)
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
