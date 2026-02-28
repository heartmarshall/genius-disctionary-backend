"""Shared configuration loader for lyrics pipeline scripts."""

import sys
from pathlib import Path

# lyrics/ is one level up from pipeline/
ROOT = Path(__file__).resolve().parent.parent


def load_artists() -> dict[str, str]:
    """Load config.yaml and return {dir_name: label} dict.

    Scans output/ for artist directories that have dataset.csv,
    using the directory name as both key and label.
    """
    output = ROOT / "output"
    if not output.exists():
        print(f"Error: {output} not found", file=sys.stderr)
        sys.exit(1)
    artists = {}
    for d in sorted(output.iterdir()):
        if d.is_dir() and (d / "dataset.csv").exists():
            artists[d.name] = d.name
    return artists


def artist_dir(artist_name: str) -> Path:
    """Return absolute path to an artist's output directory."""
    return ROOT / "output" / artist_name
