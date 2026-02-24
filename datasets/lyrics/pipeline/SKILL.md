---
name: lyrics-word-excluder
description: >
  Classifies and excludes non-learnable words from song lyrics datasets for a language
  learning app. Processes artist word lists using parallel Task tool batches, identifying
  artist/celebrity names, parsing errors, Genius annotation artifacts, gibberish, foreign words,
  and other junk. Produces a standardized exclude CSV.
  Use when user says "exclude words", "build exclude list", "classify words for {artist}",
  "find junk words in lyrics", "create exclude csv for lyrics", or wants to filter
  non-learnable words from a lyrics artist dataset.
---

# Lyrics Word Excluder

Analyze a lyrics-extracted word list and separate learnable English words from noise.
Keep every word a language learner would benefit from studying, exclude everything else —
names, parsing junk, gibberish, foreign words.

**Core principle: when in doubt, keep the word.** A false negative (junk word stays) is
cheap — it gets filtered later or the learner skips it. A false positive (useful word wrongly
excluded) means a learner never sees a word they should have learned.

## File Layout

```
datasets/lyrics/
├── pipeline/
│   ├── pipeline.yaml                      # Registry of all artists
│   └── references/                        # Classifier prompts and rules
├── output/
│   ├── artists/
│   │   ├── {artist}/
│   │   │   ├── words.txt                  # Input: one lemma per line, sorted, stopwords excluded
│   │   │   ├── vocabulary.csv             # Full vocabulary with POS, frequency, song list
│   │   │   └── {artist}_exclude.csv       # Output: word,exclusion_category
```

**Important:** `words.txt` contains lemmatized forms with stopwords already removed by spaCy.
This means word lists are cleaner than raw subtitle extractions but still contain noise:
artist/celebrity names, encoding artifacts, Genius annotation junk, foreign words, etc.

## Prerequisites

The artist must already be processed by the lyrics pipeline:
- `datasets/lyrics/output/artists/{artist}/words.txt` must exist
- Artist should be listed in `datasets/lyrics/pipeline/pipeline.yaml`

If the word list doesn't exist, tell the user to run the lyrics pipeline first.

## Workflow

### Step 1: Load and Validate

1. If the user didn't specify an artist, ask them.
2. Convert artist name to directory slug (lowercase, spaces→underscores, strip special chars).
3. Read `pipeline.yaml`, validate the artist exists or check directory directly.
4. Read `words.txt`. Count total words.
5. Print status:
   ```
   Artist: {artist_name}
   Directory: {artist_slug}
   Total words: {N}
   ```

### Step 2: Pre-filter with Deterministic Rules

Before sending words to Task agents, apply rule-based filters to catch obvious cases.
This reduces batch count, saves tokens, and is more reliable than LLM for clear-cut patterns.

Read `references/prefilter-rules.md` for the full rule set. In summary:

**Auto-exclude (write directly to exclude list):**
- Strings containing zero-width characters (U+200B, U+200C, U+200D, U+FEFF) → `PARSE_ERROR`
- Strings with Genius artifacts: `*scratch*`, `!"—`, `,"—`, `'–` etc. → `PARSE_ERROR`
- Strings starting with `-` (truncated fragments: `-inem`, `-se`, `-tober`) → `PARSE_ERROR`
- Pure numeric strings and alphanumeric codes (`3s`, `49er`, `a1`, `1x01`) → `PARSE_ERROR`
- Strings with encoding artifacts (`\x` sequences, HTML entities) → `PARSE_ERROR`
- Strings longer than 25 characters → `PARSE_ERROR`
- Known contraction fragments when isolated: `ll`, `ve`, `re`, `em` → `PARSE_ERROR`
- Words with 3+ consecutive identical characters: `aaahhh`, `wooooo` → `GIBBERISH`
- Strings containing punctuation mixed with words (`gay!"—that`, `pop,"—'cause`) → `PARSE_ERROR`
- Single non-word characters (keep only: a, i, o) → `PARSE_ERROR`

**Auto-keep (never send to classifier, never exclude):**
- Real English two-letter words: go, do, be, am, an, at, by, if, in, is, it, my, no, of,
  on, or, so, to, up, us, we, ah, hi, ok, ow, ox
- Common informal words with dictionary entries: okay, yeah, yep, nope, alright, gonna,
  wanna, gotta, cool, dude, damn, shit, hell, wow, whoa, ooh, oops, shh
- Words with accented characters that are real English borrowings: cliché, fiancé, façade,
  touché, café, déjà (as in déjà vu), séance, naïve, résumé

After pre-filtering, print:
```
Pre-filter results:
  Auto-excluded: {N} words
  Auto-kept (protected): {N} words
  Remaining for classification: {N} words
```

### Step 3: Parallel Classification with Task Tool

Split the remaining words (after pre-filtering) into batches of **500 words** each.

**Launch Tasks in groups of 2.** Wait for both Tasks in a group to complete before launching
the next pair. This keeps resource usage predictable and makes it easier to catch errors
early. With 500 words per batch, each pair processes 1,000 words before moving on.

For each batch, use the Task tool with the prompt from `references/classifier-prompt.md`
(fill in the variables: artist_name, artist_slug, batch_number, total_batches, batch_words).

Read `references/classifier-prompt.md` before launching Tasks — it contains the exact
prompt template with category definitions and classification rules.

**Important:** Each Task must return ONLY raw JSON. No markdown, no explanation.

### Step 4: Collect and Merge Results

After all Tasks complete:

1. Parse JSON from each Task's output.
   - If a Task returned markdown fences around JSON, extract the array from between them.
   - If a Task returned invalid JSON, log a warning with the batch number and retry that
     single batch once. If retry fails, skip and report at the end.

2. Collect all `{word, category}` pairs. Deduplicate by word (keep first classification).

3. **Post-classification safety check:** Scan results for obvious false positives — real
   English words that were misclassified. Watch for:
   - Words that are both names AND English words: grace, hunter, rose, frank, bill, mason,
     angel, chase, dawn, faith, joy, mark, ray, will, harmony, melody, summer, violet
   - Common informal words tagged as SLANG: gonna, wanna, yeah, ain't, y'all
   - Onomatopoeia that's also a real word: bang, crash, buzz, pop, snap, crack, boom
   - Musical terms that are real words: beat, bass, drop, hook, bridge, verse, chorus
   Remove any obvious false positives from the exclude list.

### Step 5: Write the Exclude CSV

Merge pre-filter exclusions with classifier exclusions. Sort alphabetically. Write to:

```
datasets/lyrics/output/artists/{artist}/{artist}_exclude.csv
```

Format:
```csv
word,exclusion_category
2pac,ARTIST_NAME
aaahhh,GIBBERISH
beyoncé,ARTIST_NAME
cállate,FOREIGN
zimmerman,PROPER_NOUN
```

### Step 6: Print Summary

```
=== {artist_name} Exclusion Summary ===
Total words analyzed: {N}
Words excluded: {N} ({percent}%)
  - by pre-filter: {N}
  - by classifier: {N}
Words kept: {N}

By category:
  ARTIST_NAME:     {count}
  PROPER_NOUN:     {count}
  FOREIGN:         {count}
  SOUND_EFFECT:    {count}
  PARSE_ERROR:     {count}
  GIBBERISH:       {count}
  SLANG:           {count}
  MUSIC_JUNK:      {count}

Output: datasets/lyrics/output/artists/{artist}/{artist}_exclude.csv
```

## Error Handling

- If a Task returns invalid JSON: retry once, then skip and report.
- If total exclusion rate > 30%: warn user — suspiciously aggressive classification.
- If total exclusion rate < 2%: warn user — may indicate under-classification.
- If any batch was skipped: list skipped batch numbers and word ranges so user can re-run.

## Running for All Artists

If the user says "exclude words for all artists":
1. Scan `datasets/lyrics/output/artists/` for directories containing `words.txt`
2. Process each artist sequentially (parallel Tasks within each artist)
3. Print per-artist summary + combined totals at the end
4. Flag artists with anomalous exclusion rates (>30% or <2%)
