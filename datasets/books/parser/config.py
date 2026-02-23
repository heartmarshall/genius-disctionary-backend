"""Shared configuration loader for books pipeline scripts."""

import sys
from pathlib import Path

import yaml

# books/ is one level up from parser/
ROOT = Path(__file__).resolve().parent.parent
EPUBS_DIR = ROOT / "all_boks"


def load_books() -> dict[str, dict]:
    """Load books.yaml and return {dir_name: {"label": str, "epub": str}}."""
    config_path = ROOT / "books.yaml"
    if not config_path.exists():
        print(f"Error: {config_path} not found", file=sys.stderr)
        sys.exit(1)
    with open(config_path) as f:
        data = yaml.safe_load(f)
    return {name: info for name, info in data["books"].items()}


def book_dir(book_name: str) -> Path:
    """Return absolute path to a book's output directory."""
    return ROOT / book_name


def epub_path(book_name: str, books: dict) -> Path:
    """Return absolute path to a book's EPUB file."""
    return EPUBS_DIR / books[book_name]["epub"]
