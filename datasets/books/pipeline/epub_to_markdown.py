#!/usr/bin/env python3
"""Convert EPUB books to Markdown files with chapter:paragraph markers for LLM analysis."""

import os
import re
import unicodedata
from pathlib import Path

import ebooklib
from ebooklib import epub
from bs4 import BeautifulSoup, Tag
from markdownify import markdownify as md


# Heading tags that signal a new chapter/section.
HEADING_TAGS = {"h1", "h2", "h3"}

# Minimum paragraph length (chars) to assign a marker — skip tiny fragments.
MIN_PARA_LEN = 40


def _inline_to_markdown(element: Tag) -> str:
    """Convert an HTML element to Markdown, preserving inline formatting."""
    html_str = str(element)
    text = md(
        html_str,
        heading_style="ATX",
        strip=["img", "script", "style", "svg", "figure", "figcaption"],
    )
    text = unicodedata.normalize("NFKC", text)
    text = re.sub(r"[ \t]+", " ", text)
    text = re.sub(r"\n{2,}", "\n", text)
    return text.strip()


def _extract_heading_text(element: Tag) -> str:
    """Extract clean text from a heading element."""
    return element.get_text(separator=" ", strip=True)


def epub_to_markdown(epub_path: str) -> str:
    """Extract text from EPUB and convert to Markdown with [chN:pM] markers."""
    book = epub.read_epub(epub_path, options={"ignore_ncx": True})

    chapter_num = 0
    para_num = 0
    lines: list[str] = []

    for item in book.get_items_of_type(ebooklib.ITEM_DOCUMENT):
        html = item.get_body_content().decode("utf-8", errors="replace")
        if not html.strip():
            continue

        soup = BeautifulSoup(html, "lxml")

        # Check if this document item has any heading tags.
        has_headings = bool(soup.find(HEADING_TAGS))

        # If no headings in this item and it has substantial text,
        # treat the entire item as a new chapter.
        if not has_headings:
            plain_text = soup.get_text(strip=True)
            if len(plain_text) > 200:
                chapter_num += 1
                para_num = 0

        for el in soup.find_all(["h1", "h2", "h3", "h4", "h5", "h6", "p", "blockquote"]):
            tag = el.name.lower()

            # --- Heading → new chapter ---
            if tag in HEADING_TAGS:
                heading_text = _extract_heading_text(el)
                if not heading_text:
                    continue
                chapter_num += 1
                para_num = 0
                level = int(tag[1])
                prefix = "#" * level
                lines.append("")
                lines.append(f"{prefix} {heading_text}")
                lines.append("")
                continue

            # --- Sub-headings (h4-h6) → keep as markdown headings, no chapter bump ---
            if tag in ("h4", "h5", "h6"):
                heading_text = _extract_heading_text(el)
                if heading_text:
                    level = int(tag[1])
                    prefix = "#" * level
                    lines.append(f"{prefix} {heading_text}")
                    lines.append("")
                continue

            # --- Paragraph / blockquote ---
            text = _inline_to_markdown(el)
            if not text or len(text) < MIN_PARA_LEN:
                continue

            para_num += 1
            marker = f"[ch{chapter_num}:p{para_num}]"

            if tag == "blockquote":
                bq_lines = text.split("\n")
                first = f"{marker} > {bq_lines[0]}"
                rest = [f"> {l}" for l in bq_lines[1:]]
                lines.append(first)
                lines.extend(rest)
            else:
                lines.append(f"{marker} {text}")

            lines.append("")

    result = "\n".join(lines)
    result = re.sub(r"\n{3,}", "\n\n", result)
    return result.strip()


def epub_filename_to_md_name(epub_path: str) -> str:
    """Convert EPUB filename to a clean Markdown filename."""
    name = Path(epub_path).stem
    name = re.sub(r"^\d+_", "", name)
    name = name.strip().lower().replace(" ", "_")
    name = re.sub(r"_+", "_", name)
    name = re.sub(r"[^a-z0-9_]", "", name)
    name = name.strip("_")
    return name + ".md"


def find_epubs(base_dir: str) -> list[str]:
    """Recursively find all .epub files under base_dir."""
    epubs = []
    for root, _, files in os.walk(base_dir):
        for f in sorted(files):
            if f.endswith(".epub"):
                epubs.append(os.path.join(root, f))
    return epubs


def main():
    import argparse

    parser = argparse.ArgumentParser(description="Convert EPUB books to Markdown.")
    parser.add_argument("--file", help="Process a single EPUB file")
    parser.add_argument("--dir", default=".", help="Base directory to scan for EPUBs (default: current dir)")
    parser.add_argument("--output", default="markdown", help="Output directory for MD files (default: markdown/)")
    args = parser.parse_args()

    if args.file:
        epubs = [args.file]
    else:
        epubs = find_epubs(args.dir)

    if not epubs:
        print("No EPUB files found.")
        return

    os.makedirs(args.output, exist_ok=True)
    print(f"Found {len(epubs)} EPUB file(s)\n")

    for epub_path in epubs:
        book_name = Path(epub_path).name
        md_name = epub_filename_to_md_name(epub_path)
        output_path = os.path.join(args.output, md_name)

        print(f"Processing: {book_name}")
        try:
            markdown = epub_to_markdown(epub_path)
        except Exception as e:
            print(f"  ERROR: {e} — skipping")
            continue
        print(f"  Length: {len(markdown):,} chars")

        with open(output_path, "w", encoding="utf-8") as f:
            f.write(markdown)
        print(f"  Saved: {output_path}")

    print(f"\nDone! Converted {len(epubs)} book(s).")


if __name__ == "__main__":
    main()
