#!/usr/bin/env python3
"""Build per-show unique word lists and cross-show common words.

Usage:
    python build_unique.py

Reads {show}/{show}.csv for each show in shows.yaml.

Output:
    {show}/{show}_unique.csv  — words exclusive to one show (word, total_count)
    common_words.csv          — words in ALL shows (word, total, per-show counts)
    common_words.txt          — same, plain word list
"""

import csv
import sys
from pathlib import Path

from config import ROOT, load_shows, show_dir


def load_show_words(show_name: str) -> dict[str, int]:
    """Load {show}.csv and return {word: total_count}."""
    csv_path = show_dir(show_name) / f"{show_name}.csv"
    if not csv_path.exists():
        print(f"Error: {csv_path} not found. Run parse first.", file=sys.stderr)
        sys.exit(1)
    words: dict[str, int] = {}
    with open(csv_path, encoding="utf-8") as f:
        for row in csv.DictReader(f):
            words[row["word"]] = int(row["total_count"])
    return words


def write_unique(show_name: str, unique: dict[str, int]) -> None:
    """Write {show}_unique.csv sorted by count DESC."""
    rows = sorted(unique.items(), key=lambda r: (-r[1], r[0]))
    output_path = show_dir(show_name) / f"{show_name}_unique.csv"
    with open(output_path, "w", newline="", encoding="utf-8") as f:
        writer = csv.writer(f)
        writer.writerow(["word", "total_count"])
        writer.writerows(rows)
    print(f"  {show_name}: {len(rows)} unique words -> {output_path.name}")
    if rows:
        top = ", ".join(f"{w}({c})" for w, c in rows[:5])
        print(f"    top 5: {top}")


def write_common(
    show_words: dict[str, dict[str, int]], shows: dict[str, str]
) -> None:
    """Write common_words.csv and common_words.txt."""
    common = set.intersection(*(set(w.keys()) for w in show_words.values()))

    show_names = sorted(show_words.keys())
    rows = []
    for word in common:
        counts = {s: show_words[s][word] for s in show_names}
        total = sum(counts.values())
        rows.append((word, total, counts))
    rows.sort(key=lambda r: (-r[1], r[0]))

    # Write CSV
    csv_path = ROOT / "common_words.csv"
    with open(csv_path, "w", newline="", encoding="utf-8") as f:
        writer = csv.writer(f)
        writer.writerow(["word", "total"] + show_names)
        for word, total, counts in rows:
            writer.writerow([word, total] + [counts[s] for s in show_names])

    # Write TXT
    txt_path = ROOT / "common_words.txt"
    txt_path.write_text("\n".join(r[0] for r in rows) + "\n", encoding="utf-8")

    print(f"\n{len(common)} common words across {len(show_words)} shows")
    print(f"  -> {csv_path.name}")
    print(f"  -> {txt_path.name}")


def main() -> None:
    shows = load_shows()

    if len(shows) < 2:
        print("Error: need at least 2 shows in shows.yaml", file=sys.stderr)
        sys.exit(1)

    # Load all show data
    show_words: dict[str, dict[str, int]] = {}
    for name in shows:
        show_words[name] = load_show_words(name)
        print(f"  Loaded {name}: {len(show_words[name])} words")

    # Build unique words per show
    print("\nUnique words:")
    for name in shows:
        other_words: set[str] = set()
        for other, words in show_words.items():
            if other != name:
                other_words.update(words.keys())
        unique = {w: c for w, c in show_words[name].items() if w not in other_words}
        write_unique(name, unique)

    # Build common words
    write_common(show_words, shows)

    print("\nDone.")


if __name__ == "__main__":
    main()
