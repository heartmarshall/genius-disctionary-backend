#!/usr/bin/env python3
"""
Merge all NGSL word lists into a single deduplicated CSV.
Output: word,tags (where tags = comma-separated list names)
"""

import csv
import os
import re
import xml.etree.ElementTree as ET
import zipfile
from collections import defaultdict

BASE = os.path.dirname(os.path.abspath(__file__))
OUTPUT = os.path.join(BASE, "NGSL_combined.csv")


def normalize(word: str) -> str:
    """Lowercase, strip whitespace and non-breaking spaces."""
    return word.replace("\xa0", " ").strip().lower()


def parse_description_txt(path: str, skip_until_blank_after_references=True) -> list[str]:
    """Parse .txt files that have a description header followed by word list."""
    words = []
    with open(path, encoding="utf-8") as f:
        lines = f.readlines()

    # Find where word list starts: after the last blank line following "References"
    start = 0
    if skip_until_blank_after_references:
        found_ref = False
        for i, line in enumerate(lines):
            if "references" in line.lower().strip().lower():
                found_ref = True
            if found_ref and line.strip() == "":
                start = i + 1
                # Don't break yet - keep going to find the actual word start
            if found_ref and start > 0 and line.strip() and not line.startswith("Browne") and not line.startswith("http"):
                # Check if it looks like a word (not a citation)
                if not line.strip().startswith("Browne") and len(line.strip()) < 100:
                    start = i
                    break

    for line in lines[start:]:
        line = line.strip()
        if not line:
            continue
        # Skip comment lines
        if line.startswith("##") or line.startswith("#"):
            continue
        words.append(line)
    return words


def parse_ngsl_csv(path: str) -> list[str]:
    """Parse NGSL_1.2_stats.csv - column 'Lemma'."""
    words = []
    with open(path, encoding="utf-8") as f:
        reader = csv.DictReader(f)
        for row in reader:
            w = row.get("Lemma", "").strip()
            if w:
                words.append(w)
    return words


def parse_tsl_csv(path: str) -> list[str]:
    """Parse TSL_1.2_stats.csv - column 'Word'."""
    words = []
    with open(path, encoding="latin-1") as f:
        reader = csv.DictReader(f)
        for row in reader:
            w = row.get("Word", "").strip()
            if w:
                words.append(w)
    return words


def parse_ngsl_gr_csv(path: str) -> list[str]:
    """Parse NGSL-GR_rank(1).csv - column 'Word'."""
    words = []
    with open(path, encoding="utf-8-sig") as f:
        reader = csv.DictReader(f)
        for row in reader:
            w = row.get("Word", "").strip()
            if w:
                words.append(w)
    return words


def parse_bsl(path: str) -> list[str]:
    """Parse BSL - words start after references section."""
    return parse_description_txt(path)


def parse_fel(path: str) -> list[str]:
    """Parse FEL - words after references, some have '/ alt' format."""
    raw = parse_description_txt(path)
    words = []
    for entry in raw:
        # Handle "abdominal / abs" -> two separate words
        if " / " in entry:
            parts = entry.split(" / ")
            for p in parts:
                p = p.strip()
                if p:
                    words.append(p)
        else:
            words.append(entry)
    return words


def parse_nawl(path: str) -> list[str]:
    """Parse NAWL - words after references."""
    return parse_description_txt(path)


def parse_ndl(path: str) -> list[str]:
    """Parse NDL - 'Rank\\tWord' format after references."""
    words = []
    with open(path, encoding="utf-8") as f:
        lines = f.readlines()

    in_words = False
    for line in lines:
        line = line.strip()
        if line.startswith("Rank") and "Word" in line:
            in_words = True
            continue
        if in_words and line:
            # Format: "4.	a" or "504.	able"
            parts = line.split("\t")
            if len(parts) >= 2:
                words.append(parts[-1].strip())
            elif "." in line:
                # Try splitting by dot and whitespace
                m = re.match(r"\d+\.\s+(.*)", line)
                if m:
                    words.append(m.group(1).strip())
    return words


def parse_ngsl_spoken(path: str) -> list[str]:
    """Parse NGSL-Spoken - words after ## comment lines."""
    words = []
    with open(path, encoding="utf-8") as f:
        for line in f:
            line = line.strip()
            if not line or line.startswith("##"):
                continue
            words.append(line)
    return words


def parse_medical_xlsx(path: str) -> list[str]:
    """Parse Oral English Medical Corpus xlsx."""
    words = []
    ns = {"s": "http://schemas.openxmlformats.org/spreadsheetml/2006/main"}

    with zipfile.ZipFile(path) as z:
        # Read shared strings
        shared_strings = []
        if "xl/sharedStrings.xml" in z.namelist():
            tree = ET.parse(z.open("xl/sharedStrings.xml"))
            for si in tree.findall(".//s:si", ns):
                texts = si.findall(".//s:t", ns)
                shared_strings.append("".join(t.text or "" for t in texts))

        tree = ET.parse(z.open("xl/worksheets/sheet1.xml"))
        for row in tree.findall(".//s:row", ns):
            cells = row.findall("s:c", ns)
            for c in cells:
                v = c.find("s:v", ns)
                if v is not None and v.text:
                    if c.get("t") == "s":
                        val = shared_strings[int(v.text)]
                    else:
                        val = v.text
                    val = val.replace("\xa0", " ").strip()
                    if val:
                        words.append(val)
                break  # Only first column
    return words


def main():
    # word -> set of list names
    word_tags: dict[str, set[str]] = defaultdict(set)

    sources = [
        ("NGSL", parse_ngsl_csv, "NGSL_1.2_stats.csv"),
        ("BSL", parse_bsl, "BSL_1.20_alphabetized_description.txt"),
        ("FEL", parse_fel, "FEL_1.2_alphabetized_description.txt"),
        ("NAWL", parse_nawl, "NAWL_1.2_alphabetized_description.txt"),
        ("NDL", parse_ndl, "NDL_1.1_alphabetized_description.txt"),
        ("NGSL-Spoken", parse_ngsl_spoken, "NGSL-Spoken_1.2_alphabetized_description.txt"),
        ("TSL", parse_tsl_csv, "TSL_1.2_stats.csv"),
        ("NGSL-GR", parse_ngsl_gr_csv, "NGSL-GR_rank(1).csv"),
        ("Medical", parse_medical_xlsx, "Oral+English+Medical+Corpus.xlsx"),
    ]

    for tag, parser, filename in sources:
        filepath = os.path.join(BASE, filename)
        if not os.path.exists(filepath):
            print(f"WARNING: {filename} not found, skipping")
            continue

        raw_words = parser(filepath)
        count = 0
        for w in raw_words:
            nw = normalize(w)
            if nw:
                word_tags[nw].add(tag)
                count += 1
        print(f"{tag}: {count} words extracted from {filename}")

    # Sort alphabetically and write CSV
    sorted_words = sorted(word_tags.keys())

    with open(OUTPUT, "w", newline="", encoding="utf-8") as f:
        writer = csv.writer(f)
        writer.writerow(["word", "tags"])
        for word in sorted_words:
            tags = ",".join(sorted(word_tags[word]))
            writer.writerow([word, tags])

    print(f"\nTotal unique words: {len(sorted_words)}")
    print(f"Output written to: {OUTPUT}")

    # Stats
    tag_counts = defaultdict(int)
    for tags in word_tags.values():
        for t in tags:
            tag_counts[t] += 1
    print("\nWords per list:")
    for tag, count in sorted(tag_counts.items()):
        print(f"  {tag}: {count}")

    # Multi-list words
    multi = sum(1 for tags in word_tags.values() if len(tags) > 1)
    print(f"\nWords appearing in multiple lists: {multi}")


if __name__ == "__main__":
    main()
