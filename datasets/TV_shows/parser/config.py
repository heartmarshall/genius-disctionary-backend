"""Shared configuration loader for TV show pipeline scripts."""

import sys
from pathlib import Path

import yaml

# TV_shows/ is one level up from parser/
ROOT = Path(__file__).resolve().parent.parent


def load_shows() -> dict[str, str]:
    """Load shows.yaml and return {dir_name: label} dict."""
    config_path = ROOT / "shows.yaml"
    if not config_path.exists():
        print(f"Error: {config_path} not found", file=sys.stderr)
        sys.exit(1)
    with open(config_path) as f:
        data = yaml.safe_load(f)
    return {name: info["label"] for name, info in data["shows"].items()}


def show_dir(show_name: str) -> Path:
    """Return absolute path to a show's directory."""
    return ROOT / show_name
