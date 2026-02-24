#!/usr/bin/env python3
"""Parse EPUB books into word frequency datasets.

Usage:
    python parse_book.py --book the_great_gatsby   # One book
    python parse_book.py --all                      # All books

Output per book:
    output/{book}/dataset.csv    — word, total_count, chapter_count, chapters
    output/{book}/words.txt      — sorted unique words, one per line
"""

import argparse
import csv
import sys
from collections import defaultdict
from pathlib import Path

from config import load_books, book_dir, epub_path
from epub_parser import extract_chapters
from word_processor import extract_words


def process_book(epub_file: Path) -> dict[str, dict]:
    """Process all chapters of a book.

    Returns {word: {"total": int, "chapters": {ch_num: count, ...}}}.
    """
    chapters = extract_chapters(epub_file)
    words: dict[str, dict] = defaultdict(lambda: {"total": 0, "chapters": {}})

    for ch_num, text in chapters:
        word_counts = extract_words(text)
        for word, count in word_counts.items():
            words[word]["total"] += count
            words[word]["chapters"][ch_num] = (
                words[word]["chapters"].get(ch_num, 0) + count
            )

    return dict(words)


def write_csv(words: dict[str, dict], output_path: Path) -> None:
    """Write word frequency CSV: word, total_count, chapter_count, chapters."""
    rows = []
    for word, data in words.items():
        ch_list = sorted(data["chapters"].keys())
        ch_str = ",".join(f"{ch}:{data['chapters'][ch]}" for ch in ch_list)
        rows.append((word, data["total"], len(ch_list), ch_str))

    rows.sort(key=lambda r: (-r[1], r[0]))

    with open(output_path, "w", newline="", encoding="utf-8") as f:
        writer = csv.writer(f)
        writer.writerow(["word", "total_count", "chapter_count", "chapters"])
        for word, total, ch_count, chapters in rows:
            writer.writerow([word, total, ch_count, chapters])


def write_wordlist(words: dict[str, dict], output_path: Path) -> None:
    """Write sorted unique word list, one per line."""
    sorted_words = sorted(words.keys())
    output_path.write_text("\n".join(sorted_words) + "\n", encoding="utf-8")


def run_book(book_name: str, books: dict) -> None:
    """Parse EPUB for one book and write both outputs."""
    epub_file = epub_path(book_name, books)
    if not epub_file.exists():
        print(f"Error: EPUB not found: {epub_file}", file=sys.stderr)
        sys.exit(1)

    out_dir = book_dir(book_name)
    out_dir.mkdir(exist_ok=True)

    print(f"Parsing {book_name}...")
    words = process_book(epub_file)

    if not words:
        print(f"  No data extracted, skipping.")
        return

    all_chapters = set()
    for data in words.values():
        all_chapters.update(data["chapters"].keys())

    csv_path = out_dir / "dataset.csv"
    write_csv(words, csv_path)

    txt_path = out_dir / "words.txt"
    write_wordlist(words, txt_path)

    print(f"  {len(all_chapters)} chapters, {len(words)} unique words")
    print(f"  -> {csv_path.name}")
    print(f"  -> {txt_path.name}")


def main() -> None:
    parser = argparse.ArgumentParser(
        description="Parse EPUB books into word frequency datasets."
    )
    group = parser.add_mutually_exclusive_group(required=True)
    group.add_argument("--book", type=str, help="Book directory name to parse")
    group.add_argument("--all", action="store_true", help="Parse all books")
    args = parser.parse_args()

    books = load_books()

    if args.all:
        for name in books:
            run_book(name, books)
    else:
        if args.book not in books:
            print(
                f"Error: '{args.book}' not in config.yaml. "
                f"Available: {', '.join(sorted(books))}",
                file=sys.stderr,
            )
            sys.exit(1)
        run_book(args.book, books)

    print("Done.")


if __name__ == "__main__":
    main()
