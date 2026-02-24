#!/usr/bin/env python3
"""Parse downloaded lyrics and build a clean dataset (CSV + JSON).

Reads _dataset.json from an artist directory, cleans lyrics and metadata,
outputs a flat CSV and a clean JSON ready for analysis/ML.
"""

import argparse
import csv
import json
import re
from pathlib import Path


# Section tags inserted by Genius: [Verse 1], [Chorus], [Bridge], [Intro: Name], etc.
SECTION_TAG_RE = re.compile(r"\[.*?\]")

# "You might also like" artifact
YOU_MIGHT_RE = re.compile(r"You might also like", re.IGNORECASE)

# Trailing "NNNEmbed"
EMBED_RE = re.compile(r"\d*Embed\s*$")

# Multiple blank lines
MULTI_BLANK_RE = re.compile(r"\n{3,}")


def clean_lyrics(raw: str) -> str:
    """Remove section tags, Genius artifacts, and normalize whitespace."""
    text = raw
    # Remove section tags [Verse 1], [Chorus], [Intro: Chester Bennington], etc.
    text = SECTION_TAG_RE.sub("", text)
    # Remove "You might also like"
    text = YOU_MIGHT_RE.sub("", text)
    # Remove trailing Embed
    text = EMBED_RE.sub("", text)
    # Collapse multiple blank lines into one
    text = MULTI_BLANK_RE.sub("\n\n", text)
    # Strip leading/trailing whitespace per line, then overall
    lines = [line.strip() for line in text.split("\n")]
    text = "\n".join(lines).strip()
    # Remove empty lines at the start
    text = text.lstrip("\n")
    return text


def extract_album_name(album_field) -> str | None:
    """Extract clean album name from raw album field (could be str, dict, or None)."""
    if album_field is None:
        return None
    if isinstance(album_field, str):
        return album_field
    if isinstance(album_field, dict):
        return album_field.get("name")
    return str(album_field)


def extract_album_release(album_field) -> str | None:
    """Extract album release date from raw album field."""
    if isinstance(album_field, dict):
        return album_field.get("release_date_for_display")
    return None


def count_words(text: str) -> int:
    """Count words in text."""
    return len(text.split())


def count_lines(text: str) -> int:
    """Count non-empty lines."""
    return len([l for l in text.split("\n") if l.strip()])


def build_dataset(artist_dir: str, output_dir: str | None = None, sources_dir: str | None = None):
    """Read _dataset.json and produce clean CSV + JSON."""
    artist_path = Path(artist_dir)
    dataset_file = artist_path / "_dataset.json"

    if not dataset_file.exists():
        print(f"Error: {dataset_file} not found")
        return

    with open(dataset_file, "r", encoding="utf-8") as f:
        raw_data = json.load(f)

    out_path = Path(output_dir) if output_dir else artist_path
    out_path.mkdir(parents=True, exist_ok=True)

    artist_name = raw_data[0]["artist"] if raw_data else "Unknown"
    print(f"Building dataset for: {artist_name}")
    print(f"Songs: {len(raw_data)}")

    clean_data = []

    for entry in raw_data:
        raw_lyrics = entry.get("lyrics", "")
        lyrics = clean_lyrics(raw_lyrics)

        record = {
            "number": entry.get("number"),
            "title": entry.get("title"),
            "artist": entry.get("artist"),
            "album": extract_album_name(entry.get("album")),
            "album_release_date": extract_album_release(entry.get("album")),
            "release_date": entry.get("release_date"),
            "genius_url": entry.get("genius_url"),
            "genius_id": entry.get("genius_id"),
            "pageviews": entry.get("pageviews"),
            "featured_artists": ", ".join(entry.get("featured_artists", [])) or None,
            "word_count": count_words(lyrics),
            "line_count": count_lines(lyrics),
            "lyrics": lyrics,
        }
        clean_data.append(record)

    # --- Save clean JSON ---
    json_path = out_path / "dataset.json"
    with open(json_path, "w", encoding="utf-8") as f:
        json.dump(clean_data, f, indent=2, ensure_ascii=False)
    print(f"  JSON: {json_path}")

    # --- Save CSV ---
    csv_path = out_path / "dataset.csv"
    fieldnames = [
        "number", "title", "artist", "album", "album_release_date",
        "release_date", "pageviews", "word_count", "line_count",
        "featured_artists", "genius_url", "genius_id", "lyrics",
    ]
    with open(csv_path, "w", encoding="utf-8", newline="") as f:
        writer = csv.DictWriter(f, fieldnames=fieldnames, quoting=csv.QUOTE_ALL)
        writer.writeheader()
        writer.writerows(clean_data)
    print(f"  CSV:  {csv_path}")

    # --- Save lyrics-only plain text (one file per song, clean) ---
    lyrics_dir = Path(sources_dir) / "clean_lyrics" if sources_dir else out_path / "clean_lyrics"
    lyrics_dir.mkdir(parents=True, exist_ok=True)
    for rec in clean_data:
        fname = f"{rec['number']:03d}_{re.sub(r'[^a-z0-9]+', '_', rec['title'].lower()).strip('_')}.txt"
        with open(lyrics_dir / fname, "w", encoding="utf-8") as f:
            f.write(rec["lyrics"])
    print(f"  Clean lyrics: {lyrics_dir}/ ({len(clean_data)} files)")

    # --- Summary ---
    print(f"\nDataset summary:")
    print(f"  Songs:           {len(clean_data)}")
    total_words = sum(r["word_count"] for r in clean_data)
    print(f"  Total words:     {total_words:,}")
    avg_words = total_words // len(clean_data) if clean_data else 0
    print(f"  Avg words/song:  {avg_words}")
    albums = set(r["album"] for r in clean_data if r["album"])
    print(f"  Albums:          {len(albums)} ({', '.join(sorted(albums))})")


def main():
    parser = argparse.ArgumentParser(description="Build clean lyrics dataset from downloaded data.")
    parser.add_argument("artist_dir", help="Path to artist directory with _dataset.json")
    parser.add_argument("--output", help="Output directory (default: same as artist_dir)")
    parser.add_argument("--sources", help="Sources directory for clean lyrics")
    args = parser.parse_args()

    build_dataset(args.artist_dir, args.output, sources_dir=args.sources)


if __name__ == "__main__":
    main()
