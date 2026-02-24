#!/usr/bin/env python3
"""Full pipeline: download lyrics → build dataset → build vocabulary.

Configure artists in config.yaml, then run:
    python pipeline.py                          # process all artists (4 workers)
    python pipeline.py --workers 8              # more parallel workers
    python pipeline.py --only "Adele,Queen"     # specific artists
    python pipeline.py --skip-download          # only rebuild dataset + vocabulary

Requires GENIUS_TOKEN env variable (or --token flag).
"""

import argparse
import json
import os
import re
import sys
import threading
import time
from concurrent.futures import ThreadPoolExecutor, as_completed
from pathlib import Path

import yaml

SCRIPT_DIR = Path(__file__).parent

# Thread-local storage for per-thread spaCy model
_thread_local = threading.local()
# Lock for synchronized print output
_print_lock = threading.Lock()


def log(artist: str, msg: str):
    """Thread-safe logging with artist prefix."""
    with _print_lock:
        print(f"  [{artist}] {msg}")


def load_config(config_path: Path) -> dict:
    """Load config.yaml config."""
    with open(config_path, "r", encoding="utf-8") as f:
        return yaml.safe_load(f)


def run_download(token: str, artist_name: str, output_dir: Path, sources_dir: Path, max_songs: int):
    """Step 1: Download lyrics from Genius API."""
    from download_lyrics import download_artist_lyrics
    download_artist_lyrics(token, artist_name, str(output_dir), str(sources_dir), max_songs)


def run_build_dataset(artist_dir: Path, sources_dir: Path):
    """Step 2: Clean lyrics and build dataset.csv / dataset.json."""
    from build_dataset import build_dataset
    build_dataset(str(artist_dir), sources_dir=str(sources_dir / artist_dir.name))


def get_nlp():
    """Get thread-local spaCy model (loaded once per thread)."""
    if not hasattr(_thread_local, "nlp"):
        import spacy
        _thread_local.nlp = spacy.load("en_core_web_sm", disable=["ner"])
        _thread_local.nlp.max_length = 500_000
    return _thread_local.nlp


def run_build_vocabulary(artist_dir: Path, min_count: int):
    """Step 3: Analyze vocabulary and build unique.csv + words.txt."""
    from build_vocabulary import process_artist
    nlp = get_nlp()
    process_artist(nlp, artist_dir, min_count=min_count)


def find_artist_dir(output_dir: Path, artist_name: str) -> Path | None:
    """Find the artist directory by matching folder name to artist name."""
    safe = re.sub(r"[^\w\s\-]", "", artist_name)
    safe = re.sub(r"\s+", "_", safe.strip()).lower()
    candidate = output_dir / safe
    if candidate.is_dir():
        return candidate
    for d in output_dir.iterdir():
        if not d.is_dir():
            continue
        dataset = d / "_dataset.json"
        if dataset.exists():
            with open(dataset, "r") as f:
                data = json.load(f)
            if data and data[0].get("artist", "").lower() == artist_name.lower():
                return d
    return None


def process_artist(cfg: dict, token: str | None, output_dir: Path, sources_dir: Path, skip_download: bool) -> str:
    """Run the full pipeline for one artist. Returns status summary."""
    name = cfg["name"].strip()
    max_songs = cfg.get("max_songs", 50)
    min_count = cfg.get("min_word_count", 1)

    log(name, "Starting...")

    # Step 1: Download
    if not skip_download:
        if not token:
            return f"{name}: FAILED — no Genius API token"
        log(name, f"Step 1/3: Downloading lyrics ({max_songs} songs)")
        try:
            run_download(token, name, output_dir, sources_dir, max_songs)
        except Exception as e:
            return f"{name}: FAILED at download — {e}"

    artist_dir = find_artist_dir(output_dir, name)
    if not artist_dir:
        return f"{name}: FAILED — artist directory not found (run without --skip-download)"

    # Step 2: Build dataset
    dataset_json = artist_dir / "_dataset.json"
    if dataset_json.exists():
        log(name, "Step 2/3: Building clean dataset")
        try:
            run_build_dataset(artist_dir, sources_dir)
        except Exception as e:
            return f"{name}: FAILED at build_dataset — {e}"
    else:
        return f"{name}: FAILED — no _dataset.json"

    # Step 3: Vocabulary analysis
    if (artist_dir / "dataset.json").exists():
        log(name, f"Step 3/3: Building vocabulary (min_count={min_count})")
        try:
            run_build_vocabulary(artist_dir, min_count)
        except Exception as e:
            return f"{name}: FAILED at vocabulary — {e}"
    else:
        return f"{name}: FAILED — no dataset.json"

    log(name, "Done!")
    return f"{name}: OK"


def main():
    parser = argparse.ArgumentParser(description="Full lyrics pipeline: download → dataset → vocabulary.")
    parser.add_argument("--config", default=str(SCRIPT_DIR.parent / "config.yaml"),
                        help="Path to config YAML (default: config.yaml)")
    parser.add_argument("--token", help="Genius API token (or set GENIUS_TOKEN env var)")
    parser.add_argument("--only", help="Comma-separated artist names to process (default: all)")
    parser.add_argument("--skip-download", action="store_true",
                        help="Skip download step, only rebuild dataset + vocabulary")
    parser.add_argument("--workers", type=int, default=4,
                        help="Number of parallel workers (default: 4)")
    args = parser.parse_args()

    config_path = Path(args.config)
    if not config_path.exists():
        print(f"Config not found: {config_path}")
        sys.exit(1)

    config = load_config(config_path)
    token = args.token or os.environ.get("GENIUS_TOKEN")
    output_dir = SCRIPT_DIR / config.get("output_dir", ".")
    sources_dir = SCRIPT_DIR / config.get("sources_dir", "../sources")

    artists = config.get("artists", [])
    if args.only:
        selected = {n.strip().lower() for n in args.only.split(",")}
        artists = [a for a in artists if a["name"].strip().lower() in selected]

    if not artists:
        print("No artists to process. Check your config or --only filter.")
        sys.exit(1)

    workers = min(args.workers, len(artists))
    print(f"Pipeline: {len(artists)} artist(s), {workers} parallel worker(s)\n")
    for a in artists:
        print(f"  - {a['name'].strip()} ({a.get('max_songs', 50)} songs)")
    print()

    start = time.time()
    results = []

    with ThreadPoolExecutor(max_workers=workers) as pool:
        futures = {
            pool.submit(process_artist, cfg, token, output_dir, sources_dir, args.skip_download): cfg["name"].strip()
            for cfg in artists
        }
        for future in as_completed(futures):
            name = futures[future]
            try:
                result = future.result()
            except Exception as e:
                result = f"{name}: FAILED — {e}"
            results.append(result)

    elapsed = time.time() - start
    print(f"\n{'='*60}")
    print(f"Results ({elapsed:.0f}s):")
    for r in sorted(results):
        status = "OK" if r.endswith(": OK") else "FAIL"
        print(f"  {'[+]' if status == 'OK' else '[-]'} {r}")
    ok = sum(1 for r in results if r.endswith(": OK"))
    print(f"\n{ok}/{len(results)} artists completed successfully.")
    print(f"{'='*60}")


if __name__ == "__main__":
    main()
