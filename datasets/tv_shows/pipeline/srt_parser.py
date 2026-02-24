"""SRT subtitle file parser.

Reads .srt files, strips formatting/metadata, returns clean text lines.
"""

import re
from pathlib import Path

RE_TIMESTAMP = re.compile(r"^\d{2}:\d{2}:\d{2},\d{3}\s*-->")
RE_SEQUENCE = re.compile(r"^\d+$")
RE_HTML_TAG = re.compile(r"<[^>]+>")
RE_BRACKETS = re.compile(r"\[[^\]]*\]")
RE_PARENS = re.compile(r"\([^)]*\)")
RE_EPISODE = re.compile(r"(\d+)x(\d+)")


def parse_episode_from_filename(filename: str) -> tuple[int, int] | None:
    """Extract (season, episode) from SRT filename like 'Show - 1x01 - Title.srt'."""
    m = RE_EPISODE.search(filename)
    if not m:
        return None
    return int(m.group(1)), int(m.group(2))


def clean_line(line: str) -> str:
    """Strip HTML tags, sound effects, parentheticals, and speaker dashes."""
    line = RE_HTML_TAG.sub("", line)
    line = RE_BRACKETS.sub("", line)
    line = RE_PARENS.sub("", line)
    line = line.strip()
    # Strip leading speaker dash.
    if line.startswith("- "):
        line = line[2:]
    elif line.startswith("-") and len(line) > 1 and line[1] != "-":
        line = line[1:]
    return line.strip()


def parse_srt(filepath: Path) -> list[str]:
    """Parse an SRT file and return cleaned text lines."""
    lines = []
    # Try UTF-8 first (handles BOM), fall back to latin-1.
    for encoding in ("utf-8-sig", "latin-1"):
        try:
            text = filepath.read_text(encoding=encoding)
            break
        except UnicodeDecodeError:
            continue
    else:
        return []

    for raw_line in text.splitlines():
        line = raw_line.strip()
        if not line:
            continue
        if RE_SEQUENCE.match(line):
            continue
        if RE_TIMESTAMP.match(line):
            continue
        cleaned = clean_line(line)
        if cleaned:
            lines.append(cleaned)
    return lines


def discover_episodes(show_dir: Path) -> dict[tuple[int, int], Path]:
    """Walk a show directory and return one SRT path per (season, episode).

    For duplicate SRTs (same episode, different sources), picks the first alphabetically.
    """
    episodes: dict[tuple[int, int], Path] = {}

    for season_dir in sorted(show_dir.iterdir()):
        if not season_dir.is_dir():
            continue
        for srt_file in sorted(season_dir.glob("*.srt")):
            ep = parse_episode_from_filename(srt_file.name)
            if ep is None:
                continue
            if ep not in episodes:
                episodes[ep] = srt_file

    return episodes
