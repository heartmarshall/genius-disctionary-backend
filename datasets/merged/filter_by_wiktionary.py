#!/usr/bin/env python3
"""Filter merged_pool.csv: keep only words found in Wiktionary Kaikki dump."""

import argparse
import csv
import json
import sys


def load_wiktionary_words(path: str) -> set[str]:
    """Stream Kaikki JSONL, collect normalized English words."""
    words = set()
    with open(path, "r", encoding="utf-8") as f:
        for i, line in enumerate(f):
            if i % 500_000 == 0 and i > 0:
                print(f"  ...scanned {i:,} lines, {len(words):,} unique words", file=sys.stderr)
            try:
                entry = json.loads(line)
            except json.JSONDecodeError:
                continue
            if entry.get("lang") != "English":
                continue
            word = entry.get("word", "").strip().lower()
            if word:
                words.add(word)
    return words


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--wiktionary", required=True, help="Path to kaikki.org JSONL dump")
    parser.add_argument("--pool", default="merged_pool.csv", help="Path to merged_pool.csv")
    parser.add_argument("--matched", default="seed_wordlist.txt", help="Output: matched words")
    parser.add_argument("--unmatched", default="unmatched_words.txt", help="Output: unmatched words")
    args = parser.parse_args()

    print("Loading Wiktionary words...", file=sys.stderr)
    wikt_words = load_wiktionary_words(args.wiktionary)
    print(f"Wiktionary contains {len(wikt_words):,} unique English words", file=sys.stderr)

    matched = []
    unmatched = []
    with open(args.pool, "r", encoding="utf-8") as f:
        reader = csv.DictReader(f)
        for row in reader:
            word = row["word"].strip().lower()
            if word in wikt_words:
                matched.append(word)
            else:
                unmatched.append(word)

    matched.sort()
    unmatched.sort()

    with open(args.matched, "w", encoding="utf-8") as f:
        f.write("\n".join(matched) + "\n")

    with open(args.unmatched, "w", encoding="utf-8") as f:
        f.write("\n".join(unmatched) + "\n")

    print(f"Matched: {len(matched):,} | Unmatched: {len(unmatched):,}", file=sys.stderr)


if __name__ == "__main__":
    main()
