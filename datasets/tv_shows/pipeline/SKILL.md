---
name: tv-show-word-excluder
description: >
  Classifies and excludes non-learnable words from TV show subtitle datasets for a language
  learning app. Processes word lists using parallel Task tool batches, identifying character names,
  parsing errors, gibberish, foreign words, and other junk. Produces a standardized exclude CSV.
  Use when user says "exclude words", "build exclude list", "classify words for {show}",
  "find junk words", "create exclude csv", or wants to filter non-learnable words from a
  TV show dataset.
---

# TV Show Word Excluder

Analyze a TV show's extracted word list and separate learnable English words from noise.
Keep every word a language learner would benefit from studying, exclude everything else —
names, parsing junk, gibberish, foreign words.

**Core principle: when in doubt, keep the word.** A false negative (junk word stays) is
cheap — it gets filtered later or the learner skips it. A false positive (useful word wrongly
excluded) means a learner never sees a word they should have learned.

## File Layout

```
datasets/TV_shows/
├── shows.yaml                          # Registry of all parsed shows
├── {show}/
│   ├── {show}_words.txt                # Input: one word per line, sorted
│   └── {show}_exclude.csv              # Output: word,exclusion_category
```

## Prerequisites

The show must already be parsed:
- `datasets/TV_shows/{show}/{show}_words.txt` must exist
- Show must be listed in `datasets/TV_shows/shows.yaml`

If the word list doesn't exist, tell the user to run `make parse SHOW={show}` first.

## Workflow

### Step 1: Load and Validate

1. If the user didn't specify a show, ask them.
2. Read `shows.yaml`, validate the show exists. Get the show's `label` (display name).
3. Read `{show}_words.txt`. Count total words.
4. Print status:
   ```
   Show: {show} ({label})
   Total words: {N}
   ```

### Step 2: Pre-filter with Deterministic Rules

Before sending words to Task agents, apply rule-based filters to catch obvious cases.
This reduces batch count, saves tokens, and is more reliable than LLM for clear-cut patterns.

Read `references/prefilter-rules.md` for the full rule set. In summary:

**Auto-exclude (write directly to exclude list):**
- Single non-word characters (keep only: a, i, o) → `PARSE_ERROR`
- Pure numeric strings and subtitle metadata patterns (720p, 1x01) → `PARSE_ERROR`
- Strings with encoding artifacts (\x sequences, HTML entities) → `PARSE_ERROR`
- Strings longer than 25 characters → `PARSE_ERROR`
- Known contraction fragments when isolated: `ll`, `ve`, `re`, `em` → `PARSE_ERROR`
- Words with 3+ consecutive identical characters: `aaargh`, `wooooo` → `GIBBERISH`

**Auto-keep (never send to classifier, never exclude):**
- Real English two-letter words: go, do, be, am, an, at, by, if, in, is, it, my, no, of,
  on, or, so, to, up, us, we, ah, hi, ok, ow, ox
- Common informal words with dictionary entries: okay, yeah, yep, nope, alright, gonna,
  wanna, gotta, cool, dude, damn, shit, hell, wow, whoa, ooh, oops, shh

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
(fill in the variables: show_name, show_label, batch_number, total_batches, batch_words).

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
   - Words that are both names AND English words: grace, hunter, rose, frank, bill, mason
   - Common informal words tagged as SLANG: gonna, wanna, yeah
   - Onomatopoeia that's also a real word: bang, crash, buzz, pop, snap, crack
   Remove any obvious false positives from the exclude list.

### Step 5: Write the Exclude CSV

Merge pre-filter exclusions with classifier exclusions. Sort alphabetically. Write to:

```
datasets/TV_shows/{show}/{show}_exclude.csv
```

Format:
```csv
word,exclusion_category
aaargh,GIBBERISH
couldn,PARSE_ERROR
skyler,CHARACTER_NAME
```

### Step 6: Print Summary

```
=== {show} Exclusion Summary ===
Total words analyzed: {N}
Words excluded: {N} ({percent}%)
  - by pre-filter: {N}
  - by classifier: {N}
Words kept: {N}

By category:
  CHARACTER_NAME:  {count}
  PROPER_NOUN:     {count}
  BRAND:           {count}
  FOREIGN:         {count}
  SOUND_EFFECT:    {count}
  INTERJECTION:    {count}
  PARSE_ERROR:     {count}
  GIBBERISH:       {count}
  SLANG:           {count}
  SHOW_SPECIFIC:   {count}

Output: datasets/TV_shows/{show}/{show}_exclude.csv
```

## Error Handling

- If a Task returns invalid JSON: retry once, then skip and report.
- If total exclusion rate > 30%: warn user — suspiciously aggressive classification.
- If total exclusion rate < 2%: warn user — may indicate under-classification.
- If any batch was skipped: list skipped batch numbers and word ranges so user can re-run.

## Running for All Shows

If the user says "exclude words for all shows":
1. Read `shows.yaml` for the full list
2. Process each show sequentially (parallel Tasks within each show)
3. Print per-show summary + combined totals at the end
4. Flag shows with anomalous exclusion rates (>30% or <2%)
