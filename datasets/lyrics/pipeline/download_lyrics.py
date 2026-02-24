#!/usr/bin/env python3
"""Download top lyrics for a given artist from Genius with metadata.

Files are saved incrementally as they're downloaded.
Songs sorted by popularity (pageviews on Genius).
Each song saved as markdown + a combined JSON dataset.
"""

import argparse
import json
import os
import re
import time

import lyricsgenius


def sanitize_filename(name: str) -> str:
    """Convert a song title to a safe filename."""
    name = re.sub(r"[^\w\s\-]", "", name)
    name = re.sub(r"\s+", "_", name.strip())
    return name.lower()


def clean_lyrics(lyrics: str) -> str:
    """Clean up Genius-specific artifacts from lyrics."""
    lyrics = re.sub(r"\d*Embed$", "", lyrics).strip()
    lyrics = lyrics.replace("You might also like", "")
    lines = lyrics.split("\n")
    if lines and lines[0].endswith("Lyrics"):
        lines = lines[1:]
    return "\n".join(lines).strip()


def download_artist_lyrics(
    token: str,
    artist_name: str,
    output_dir: str,
    sources_dir: str,
    max_songs: int = 50,
):
    """Download top lyrics for an artist by popularity."""
    genius = lyricsgenius.Genius(token, timeout=15, retries=3)
    genius.verbose = False
    genius.remove_section_headers = False

    # --- Find artist ---
    print(f"Searching for: {artist_name}...")
    search = genius.search_songs(artist_name)
    hits = search.get("hits", [])
    artist_id = None
    real_name = artist_name
    for hit in hits:
        r = hit.get("result", {})
        pa = r.get("primary_artist", {})
        if pa.get("name", "").lower() == artist_name.lower():
            artist_id = pa["id"]
            real_name = pa["name"]
            break

    if not artist_id:
        if hits:
            pa = hits[0]["result"]["primary_artist"]
            artist_id = pa["id"]
            real_name = pa["name"]
        else:
            print("Artist not found.")
            return

    artist_dir = os.path.join(output_dir, sanitize_filename(real_name))
    os.makedirs(artist_dir, exist_ok=True)

    artist_sources = os.path.join(sources_dir, sanitize_filename(real_name))
    os.makedirs(artist_sources, exist_ok=True)

    print(f"Found: {real_name} (id={artist_id})")
    print(f"Output: {artist_dir}/")
    print(f"Fetching top {max_songs} songs by popularity...\n")

    # --- Collect songs sorted by popularity ---
    page = 1
    all_songs = []
    while page is not None:
        res = genius.artist_songs(artist_id, sort="popularity", per_page=50, page=page)
        songs = res.get("songs", [])
        next_page = res.get("next_page")

        for s in songs:
            # Only songs where this is the primary artist
            if s.get("primary_artist", {}).get("id") != artist_id:
                continue
            all_songs.append(s)
            if len(all_songs) >= max_songs:
                break

        if len(all_songs) >= max_songs:
            break
        page = next_page

    print(f"Found {len(all_songs)} songs. Downloading lyrics...\n")

    # --- Download each song ---
    dataset = []
    saved = 0

    for i, song_data in enumerate(all_songs, 1):
        title = song_data.get("title", "")
        song_url = song_data.get("url", "")

        # Fetch full song info (for album) and lyrics by song ID
        song_id = song_data.get("id")
        try:
            song_info = genius.song(song_id)["song"]
            lyrics_text = genius.lyrics(song_id)
        except Exception as e:
            print(f"  [{i}] ERROR: {title} — {e}")
            continue

        if not lyrics_text:
            print(f"  [{i}] SKIP (no lyrics): {title}")
            continue

        saved += 1
        lyrics = clean_lyrics(lyrics_text)

        # Extract album name from song info
        raw_album = song_info.get("album")
        album_name = raw_album.get("name") if isinstance(raw_album, dict) else None

        # Metadata
        meta = {
            "number": saved,
            "title": title,
            "artist": real_name,
            "album": album_name,
            "genius_url": song_url,
            "genius_id": song_id,
            "featured_artists": [
                fa.get("name") for fa in song_data.get("featured_artists", [])
            ],
            "release_date": song_data.get("release_date_for_display"),
            "pageviews": song_data.get("stats", {}).get("pageviews"),
        }

        # Save markdown immediately
        md_dir = os.path.join(artist_sources, "md_lyrics")
        os.makedirs(md_dir, exist_ok=True)
        filename = f"{saved:03d}_{sanitize_filename(title)}.md"
        filepath = os.path.join(md_dir, filename)

        md_content = f"# {title}\n\n"
        md_content += f"**Artist:** {real_name}\n"
        if meta["album"]:
            md_content += f"**Album:** {meta['album']}\n"
        if meta["release_date"]:
            md_content += f"**Released:** {meta['release_date']}\n"
        if meta["featured_artists"]:
            md_content += f"**Featuring:** {', '.join(meta['featured_artists'])}\n"
        if meta["pageviews"]:
            md_content += f"**Genius pageviews:** {meta['pageviews']:,}\n"
        md_content += f"\n---\n\n{lyrics}\n"

        with open(filepath, "w", encoding="utf-8") as f:
            f.write(md_content)

        # Add to dataset
        meta["lyrics"] = lyrics
        meta["lyrics_length"] = len(lyrics)
        dataset.append(meta)

        # Save dataset JSON incrementally (overwrite each time)
        dataset_path = os.path.join(artist_dir, "_dataset.json")
        with open(dataset_path, "w", encoding="utf-8") as f:
            json.dump(dataset, f, indent=2, ensure_ascii=False)

        pv = f" ({meta['pageviews']:,} views)" if meta["pageviews"] else ""
        album_str = f" [{album_name}]" if album_name else ""
        print(f"  [{saved}] {title}{album_str}{pv} → {filename}")

    # --- Save index ---
    index_path = os.path.join(artist_dir, "_index.md")
    with open(index_path, "w", encoding="utf-8") as f:
        f.write(f"# {real_name} — Top {saved} Songs (by popularity)\n\n")
        for entry in dataset:
            n = entry["number"]
            t = entry["title"]
            safe = f"{n:03d}_{sanitize_filename(t)}"
            album = f" — *{entry['album']}*" if entry["album"] else ""
            rel = f" ({entry['release_date']})" if entry.get("release_date") else ""
            pv = f" — {entry['pageviews']:,} views" if entry["pageviews"] else ""
            f.write(f"{n}. [{t}]({safe}.md){album}{rel}{pv}\n")

    print(f"\nDone! Saved {saved} songs to {artist_dir}/")
    print(f"  Markdown files: {saved} .md files")
    print(f"  Dataset: _dataset.json ({len(dataset)} entries)")
    print(f"  Index: _index.md")


def main():
    parser = argparse.ArgumentParser(description="Download artist lyrics from Genius.")
    parser.add_argument("artist", help="Artist name (e.g. 'Linkin Park')")
    parser.add_argument("--token", required=True, help="Genius API access token")
    parser.add_argument("--output", default=".", help="Output base directory")
    parser.add_argument("--sources", default=".", help="Sources base directory")
    parser.add_argument("--max-songs", type=int, default=50, help="Number of top songs (default: 50)")
    args = parser.parse_args()

    download_artist_lyrics(args.token, args.artist, args.output, args.sources, args.max_songs)


if __name__ == "__main__":
    main()
