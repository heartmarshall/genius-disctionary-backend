"""EPUB parser — extract clean text per chapter from EPUB files."""

import re
from pathlib import Path

import ebooklib
from ebooklib import epub
from bs4 import BeautifulSoup

HEADING_TAGS = {"h1", "h2", "h3"}
MIN_TEXT_LEN = 40  # Skip tiny fragments


def extract_chapters(epub_path: Path) -> list[tuple[int, str]]:
    """Parse EPUB and return [(chapter_num, text), ...].

    Chapter boundaries detected by <h1>, <h2>, <h3> tags.
    Returns plain text with HTML stripped.
    """
    book = epub.read_epub(str(epub_path), options={"ignore_ncx": True})

    chapter_num = 0
    chapters: dict[int, list[str]] = {}

    for item in book.get_items_of_type(ebooklib.ITEM_DOCUMENT):
        html = item.get_body_content().decode("utf-8", errors="replace")
        if not html.strip():
            continue

        soup = BeautifulSoup(html, "lxml")
        has_headings = bool(soup.find(HEADING_TAGS))

        # If no headings and substantial text, treat as new chapter.
        if not has_headings:
            plain = soup.get_text(strip=True)
            if len(plain) > 200:
                chapter_num += 1
                chapters.setdefault(chapter_num, [])

        for el in soup.find_all(["h1", "h2", "h3", "p", "blockquote"]):
            tag = el.name.lower()

            if tag in HEADING_TAGS:
                heading_text = el.get_text(strip=True)
                if heading_text:
                    chapter_num += 1
                    chapters.setdefault(chapter_num, [])
                continue

            text = el.get_text(separator=" ", strip=True)
            if not text or len(text) < MIN_TEXT_LEN:
                continue

            chapters.setdefault(chapter_num, []).append(text)

    # Build result: join paragraphs per chapter
    result = []
    for ch_num in sorted(chapters):
        text = "\n".join(chapters[ch_num])
        if text.strip():
            result.append((ch_num, text))

    return result
