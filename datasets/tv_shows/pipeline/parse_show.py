#!/usr/bin/env python3
"""Parse TV show SRT subtitles into word frequency datasets.

Usage:
    python parse_show.py --show breaking_bad   # One show
    python parse_show.py --all                  # All shows

Output per show:
    output/{show}/dataset.csv    — word, total_count, episode_count, episodes
    output/{show}/words.txt      — sorted unique words, one per line
"""

import argparse
import csv
import sys
from collections import defaultdict
from pathlib import Path

from config import load_shows, show_dir, show_sources_dir
from srt_parser import discover_episodes, parse_srt
from word_processor import extract_words


def process_show(sources_path: Path) -> dict[str, dict]:
    """Process all episodes for a show.

    Returns {word: {"total": int, "episodes": {(s,e): count, ...}}}.
    """
    episodes = discover_episodes(sources_path)
    words: dict[str, dict] = defaultdict(lambda: {"total": 0, "episodes": {}})

    for (season, episode), srt_path in sorted(episodes.items()):
        lines = parse_srt(srt_path)
        if not lines:
            continue
        word_counts = extract_words(lines)
        for word, count in word_counts.items():
            words[word]["total"] += count
            words[word]["episodes"][(season, episode)] = count

    return dict(words)


def write_csv(words: dict[str, dict], output_path: Path) -> None:
    """Write word frequency CSV: word, total_count, episode_count, episodes."""
    rows = []
    for word, data in words.items():
        ep_list = sorted(data["episodes"].keys())
        ep_str = ",".join(
            f"{s}x{e:02d}:{data['episodes'][(s, e)]}" for s, e in ep_list
        )
        rows.append((word, data["total"], len(ep_list), ep_str))

    rows.sort(key=lambda r: (-r[1], r[0]))

    with open(output_path, "w", newline="", encoding="utf-8") as f:
        writer = csv.writer(f)
        writer.writerow(["word", "total_count", "episode_count", "episodes"])
        for word, total, ep_count, episodes in rows:
            writer.writerow([word, total, ep_count, episodes])


def write_wordlist(words: dict[str, dict], output_path: Path) -> None:
    """Write sorted unique word list, one per line."""
    sorted_words = sorted(words.keys())
    output_path.write_text("\n".join(sorted_words) + "\n", encoding="utf-8")


def run_show(show_name: str) -> None:
    """Parse SRTs for one show and write both outputs."""
    sources_path = show_sources_dir(show_name)
    out_dir = show_dir(show_name)

    if not sources_path.is_dir():
        print(f"Error: directory {sources_path} not found", file=sys.stderr)
        sys.exit(1)

    out_dir.mkdir(parents=True, exist_ok=True)

    print(f"Parsing {show_name}...")
    words = process_show(sources_path)

    if not words:
        print(f"  No data extracted, skipping.")
        return

    all_episodes = set()
    for data in words.values():
        all_episodes.update(data["episodes"].keys())

    csv_path = out_dir / "dataset.csv"
    write_csv(words, csv_path)

    txt_path = out_dir / "words.txt"
    write_wordlist(words, txt_path)

    print(f"  {len(all_episodes)} episodes, {len(words)} unique words")
    print(f"  -> {csv_path.name}")
    print(f"  -> {txt_path.name}")


def main() -> None:
    parser = argparse.ArgumentParser(
        description="Parse TV show subtitles into word frequency datasets."
    )
    group = parser.add_mutually_exclusive_group(required=True)
    group.add_argument("--show", type=str, help="Show directory name to parse")
    group.add_argument("--all", action="store_true", help="Parse all shows")
    args = parser.parse_args()

    shows = load_shows()

    if args.all:
        for name in shows:
            run_show(name)
    else:
        if args.show not in shows:
            print(
                f"Error: '{args.show}' not in config.yaml. "
                f"Available: {', '.join(shows)}",
                file=sys.stderr,
            )
            sys.exit(1)
        run_show(args.show)

    print("Done.")


if __name__ == "__main__":
    main()
