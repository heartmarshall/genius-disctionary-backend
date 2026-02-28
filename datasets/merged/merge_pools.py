#!/usr/bin/env python3
"""Merge common_pool.csv from all dataset categories into a single file.

Reads:
    books/output/common_pool.csv
    tv_shows/output/common_pool.csv
    lyrics/output/common_pool.csv

Output:
    merged_pool.csv — word, datasets (e.g. "books|lyrics|tv_shows"), dataset_count

Usage:
    cd datasets/merged && python merge_pools.py
"""

import csv
from collections import defaultdict
from pathlib import Path

SCRIPT_DIR = Path(__file__).resolve().parent
ROOT = SCRIPT_DIR.parent  # datasets/

POOLS = {
    "books": ROOT / "books" / "output" / "common_pool.csv",
    "tv_shows": ROOT / "tv_shows" / "output" / "common_pool.csv",
    "lyrics": ROOT / "lyrics" / "output" / "common_pool.csv",
}


def read_pool(path: Path) -> set[str]:
    """Read the word column from a common_pool.csv."""
    words = set()
    with open(path, encoding="utf-8") as f:
        for row in csv.DictReader(f):
            w = row["word"].strip().lower()
            if w:
                words.add(w)
    return words


def main() -> None:
    word_datasets: dict[str, list[str]] = defaultdict(list)

    for name, path in sorted(POOLS.items()):
        if not path.exists():
            print(f"Warning: {path} not found, skipping {name}")
            continue
        words = read_pool(path)
        print(f"  {name}: {len(words):,} words")
        for w in words:
            word_datasets[w].append(name)

    rows = sorted(word_datasets.items(), key=lambda x: (-len(x[1]), x[0]))

    out_path = SCRIPT_DIR / "merged_pool.csv"

    with open(out_path, "w", newline="", encoding="utf-8") as f:
        writer = csv.writer(f)
        writer.writerow(["word", "datasets", "dataset_count"])
        for word, datasets in rows:
            writer.writerow([word, "|".join(sorted(datasets)), len(datasets)])

    print(f"\nTotal unique words: {len(rows):,}")
    print(f"Written to: {out_path}")

    by_count: dict[int, int] = defaultdict(int)
    for _, datasets in rows:
        by_count[len(datasets)] += 1
    print("\nDistribution:")
    for n in sorted(by_count, reverse=True):
        label = f"all {n}" if n == len(POOLS) else str(n)
        print(f"  in {label} dataset(s): {by_count[n]:,} words")

    print("\nDone.")


if __name__ == "__main__":
    main()
