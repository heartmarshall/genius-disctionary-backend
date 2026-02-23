# LLM Enrichment Pipeline — Design

**Date:** 2026-02-24
**Status:** Approved

## Problem

The existing seeder populates `ref_entries` from Wiktionary, WordNet, CMU, NGSL, and Tatoeba.
However, Wiktionary's Russian translations and definitions are low quality. There is also no
"learning notes" field — usage tips, collocations, and context hints useful for language learners.

## Goal

Build a two-stage pipeline that:
1. Assembles enrichment context from existing datasets for each word in a target list
2. Passes that context to an LLM (manually or via API) to produce improved JSON documents
3. Imports the LLM output into the database via a dedicated CLI tool

## Architecture

```
word_list.txt  (path in config)
      │
      ▼
┌─────────────────────────────────────────────────┐
│  cmd/enrich/   (Go binary)                      │
│                                                 │
│  Sources: Wiktionary, WordNet, CMU              │
│  Reuses:  internal/seeder/**/parser.go          │
│  Writes:  enrich-output/<word>.json             │
│           enrich-output/batch_NNN_prompt.txt    │
│                                                 │
│  --mode=manual  →  generates context files only │
│  --mode=api     →  calls LLM API,               │
│                    saves to llm-output/          │
└─────────────────────────────────────────────────┘
      │
      │  [manual step: user feeds prompts to LLM,
      │   saves responses to llm-output/]
      ▼
┌─────────────────────────────────────────────────┐
│  llm-output/<word>.json                         │
└─────────────────────────────────────────────────┘
      │
      ▼
┌─────────────────────────────────────────────────┐
│  cmd/llm-import/  (Go binary)                   │
│                                                 │
│  Reads:   llm-output/*.json                     │
│  Reuses:  RefEntryBulkRepo                      │
│  Writes:  ref_entries, ref_senses (with notes), │
│           ref_translations, ref_examples        │
│  source_slug: "llm"                             │
└─────────────────────────────────────────────────┘
```

**Key principles:**
- `enrich-output/` is a checkpoint — pipeline can be interrupted and resumed at any word
- `llm-output/` is a second checkpoint — import only when satisfied with LLM quality
- Both tools are idempotent: re-running is safe (`ON CONFLICT DO NOTHING`)
- Parsers from the existing seeder are reused directly (no duplication)

## Database Schema Change

One migration, one new nullable column:

```sql
ALTER TABLE ref_senses ADD COLUMN notes TEXT;
```

Domain model change in `domain/ref.go`:

```go
type RefSense struct {
    // ... existing fields ...
    Notes *string  // nullable, populated only by LLM source
}
```

GraphQL schema gains `notes: String` on `RefSense` so the frontend can display learning notes.

A new data source is registered in the `ref_data_sources` registry:

| Slug  | Name              | Type        |
|-------|-------------------|-------------|
| `llm` | LLM Enrichment    | definitions |

## JSON Formats

### `enrich-output/<word>.json` — context assembled from datasets

```json
{
  "word": "abandon",
  "ipa": "/əˈbændən/",
  "wiktionary_senses": [
    {
      "pos": "verb",
      "definition": "To give up; to surrender control or possession of.",
      "translations_ru": ["бросать", "покидать"]
    }
  ],
  "relations": {
    "synonyms": ["desert", "forsake", "leave"],
    "antonyms": ["keep", "retain"],
    "hypernyms": ["leave"]
  }
}
```

Note: Wiktionary examples are intentionally excluded — the LLM generates its own examples.

### `llm-output/<word>.json` — LLM output, input for the importer

```json
{
  "word": "abandon",
  "source_slug": "llm",
  "senses": [
    {
      "pos": "verb",
      "definition": "To leave someone or something permanently without intending to return.",
      "cefr_level": "B1",
      "notes": "Часто используется с предлогом 'to': abandon oneself to despair. Имеет негативную коннотацию.",
      "translations": ["бросать", "покидать", "отказываться от"],
      "examples": [
        {
          "sentence": "She abandoned her car on the motorway.",
          "translation": "Она бросила машину на шоссе."
        }
      ]
    }
  ]
}
```

## `cmd/enrich/` Tool

**Directory structure:**
```
backend_v4/
  cmd/enrich/
    main.go
  internal/enricher/
    pipeline.go         # orchestration: reads word list, runs enrichment
    context_builder.go  # assembles enrich-output JSON for one word
    llm_client.go       # API mode: calls LLM, saves response
    config.go           # cleanenv config
```

**Config (`enrich.yaml`):**
```yaml
word_list_path: ../datasets/common_words.txt
wiktionary_path: ../datasets/kaikki.org-dictionary-English.jsonl
wordnet_path: ../datasets/english-wordnet-2025
cmu_path: ../datasets/cmudict.dict
enrich_output_dir: ./enrich-output
llm_output_dir: ./llm-output
mode: manual          # manual | api
batch_size: 50        # words per batch (manual: prompt size, api: concurrent calls)
llm_api_key: ""       # from ENV: ENRICH_LLM_API_KEY
llm_model: "claude-opus-4-6"
```

**Manual mode:**
1. Load datasets into memory (reuse seeder parsers)
2. For each word in list: build context, write `enrich-output/<word>.json`
3. Group into batches, write `enrich-output/batch_NNN_prompt.txt` — ready-to-paste prompts

**API mode:**
- Same as manual + automatically calls LLM API per batch
- Saves responses to `llm-output/<word>.json`
- Resume support: skips words where output file already exists

## `cmd/llm-import/` Tool

**Directory structure:**
```
backend_v4/
  cmd/llm-import/
    main.go
  internal/llm_importer/
    importer.go     # reads llm-output/, calls bulk repo
    validator.go    # validates JSON before insert
    mapper.go       # JSON → domain.RefEntry/RefSense/etc.
    config.go
```

**Config (`llm-import.yaml`):**
```yaml
llm_output_dir: ./llm-output
batch_size: 500
dry_run: false
source_slug: "llm"
```

**Behavior:**
- Scans `llm_output_dir/*.json`
- Validates each file (required fields, valid POS values, valid CEFR levels)
- Maps to domain types, inserts via `RefEntryBulkRepo`
- `ON CONFLICT DO NOTHING` — idempotent, safe to re-run
- Prints per-file stats: `inserted/skipped/errors`
- `--dry-run`: parses and validates without writing to DB

## What Is NOT in Scope

- Building the `common_pool` / `word_list` — this is a separate task
- Changing the existing seeder pipeline
- Frontend changes beyond adding `notes: String` to GraphQL schema
