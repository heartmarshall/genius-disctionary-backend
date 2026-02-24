# BA Analysis: Reference Catalog Dataset Seeding

**Date**: 2026-02-19
**Status**: Draft
**Roadmap item**: #1 — Transition to pre-loaded datasets

---

## 1. Feature Summary

**Core problem**: The RefCatalog relies on lazy loading via FreeDictionary API (no SLA) and Google Translate API (paid). Every new user faces an empty catalog where each word lookup requires two synchronous HTTP calls to external services. No metadata exists for CEFR difficulty, word frequency, or semantic relations — limiting guided learning capabilities.

**Solution**: An offline CLI tool (`cmd/seeder/`) implements a 5-phase ETL pipeline that pre-loads ~20,000 reference entries from open linguistic datasets into the existing `ref_*` tables. Runtime code (`GetOrFetchEntry`) remains completely unchanged — seeded data is transparent. Words outside the 20k seed set continue using lazy loading via external APIs.

**Business value**:
- Eliminates cold start for common vocabulary (~92% coverage of general English text via NGSL)
- Removes runtime dependency on unreliable external APIs for core vocabulary
- Enables future "Discover Words" feature via CEFR/frequency metadata
- Enriches learning with semantic relations (synonyms, hypernyms) and more examples

**Relationship to existing system (per ADR-003)**: Purely additive. Populates the same immutable `ref_*` tables. No user-facing runtime code changes needed for the core seeding capability.

---

## 2. User Stories

### Primary (MUST)

**US-1** [MUST] As a **new user**, I want the catalog to already contain common English words with definitions, translations, and pronunciations, so that I can immediately add words to my dictionary without waiting for external API calls.
> Extends: RefCatalog `GetOrFetchEntry` flow. Business scenario: first-time word lookup.

**US-2** [MUST] As a **learner**, I want catalog entries to include Russian translations sourced from Wiktionary, so that basic vocabulary doesn't depend on the Google Translate API (reducing cost and latency).
> Extends: RefCatalog translation attachment. Business scenario: dictionary entry creation.

**US-3** [MUST] As an **operator**, I want the seeder to be idempotent (safe re-run) and resumable (can restart after failure), so that dataset updates and failure recovery don't require manual database cleanup.
> New admin workflow. Business scenario: deployment / data management.

**US-4** [MUST] As an **operator**, I want the seeder to process the 2.7 GB Wiktionary file in a streaming fashion (not loading it all into memory), so that it can run on a standard server without excessive RAM requirements.
> Technical constraint from dataset size. Target: <512 MB RAM during execution.

### Secondary (SHOULD)

**US-5** [SHOULD] As a **learner**, I want catalog entries from NGSL/NAWL to show CEFR difficulty level (A1-C1) and frequency rank, so that I can gauge word difficulty before adding it to my dictionary.
> Extends: RefEntry domain model. Enables future "Discover Words" feature (US-11).

**US-6** [SHOULD] As a **learner**, I want to see synonym and hypernym relations for catalog words (from WordNet), so that I can explore vocabulary in semantic clusters.
> New capability. Requires `ref_word_relations` table and API exposure.

**US-7** [SHOULD] As a **learner**, I want catalog entries to have rich example sentences with Russian translations (from Tatoeba), supplementing the often-sparse Wiktionary examples, so that I understand words in context.
> Extends: RefExample data. Business scenario: vocabulary study.

**US-8** [SHOULD] As an **operator**, I want to run individual seeder phases independently (e.g., `--phase=ngsl` to only enrich metadata), so that I can incrementally update specific data sources.
> Admin ergonomics. Business scenario: partial dataset update.

**US-9** [SHOULD] As an **operator**, I want progress logging with entry counts, phase durations, and error summaries, so that I can monitor seeder execution and verify completeness.
> Observability. Business scenario: post-deployment verification.

### Tertiary (COULD)

**US-10** [COULD] As an **operator**, I want a dry-run mode that parses and validates dataset files without writing to the database, so that I can verify dataset changes before applying them.
> Safety net. Business scenario: pre-production validation.

**US-11** [COULD] As a **learner**, I want to browse catalog words filtered by CEFR level ("Discover Words"), so that I can systematically learn vocabulary by difficulty tier.
> **DEFERRED** — separate feature. Enabled by US-5 metadata but requires its own UI, pagination, and GraphQL query. Not in scope for this task.

### Negative Stories

**US-N1** [MUST] As a **user who previously searched a word via FreeDictionary**, I should NOT lose that existing catalog entry when the seeder runs — existing entries with `source_slug="freedict"` must be preserved.
> Data integrity. Skip-on-conflict strategy.

**US-N2** [MUST] As a **user searching during a seeder run**, I should NOT experience errors or inconsistent data — partially-seeded entries (senses loaded but NGSL enrichment not yet applied) should return valid results with nullable metadata fields.
> Concurrent access safety.

---

## 3. Acceptance Criteria

### US-1: Pre-populated catalog

```
AC-1.1
GIVEN a freshly seeded database
WHEN a user calls GetOrFetchEntry("abandon")
THEN the entry is returned from DB (step 2 of the existing flow)
  AND the entry has >=1 RefSense with non-empty definition
  AND the entry has >=1 RefPronunciation with IPA transcription
  AND source_slug = "wiktionary" on senses and pronunciations
  AND no HTTP call is made to FreeDictionary or Google Translate

AC-1.2
GIVEN a seeded database with ~20,000 entries
WHEN checking entry count
THEN ref_entries contains between 19,500 and 21,000 rows
  AND all ~3,800 NGSL+NAWL words are present (is_core_lexicon = true)

AC-1.3
GIVEN the seeder CLI
WHEN invoked with valid Wiktionary JSONL file and DB connection
THEN the Wiktionary phase completes without errors
  AND execution time is < 15 minutes on standard hardware
  AND batch inserts use groups of ~500 entries
```

### US-2: Russian translations from Wiktionary

```
AC-2.1
GIVEN a seeded entry for a common word (e.g., "water", "house", "run")
WHEN the full tree is loaded
THEN at least one RefTranslation exists with source_slug="wiktionary"
  AND the translation text contains Cyrillic characters

AC-2.2
GIVEN a seeded entry where Wiktionary has no Russian translations
WHEN the full tree is loaded
THEN the entry exists with senses and pronunciations
  AND the translations list may be empty
  AND no error is raised
```

### US-3: Idempotent seeder

```
AC-3.1
GIVEN a database with ~20,000 seeded entries from a previous run
WHEN the seeder is run again with the same Wiktionary dataset
THEN no duplicate ref_entries are created (upsert by text_normalized)
  AND final entry count matches the first run (+/- 0)
  AND execution completes without errors

AC-3.2
GIVEN a seeder run interrupted after phase 2 (NGSL enrichment)
WHEN the seeder is restarted from phase 1
THEN phases 1-2 complete via upsert (no duplicates, idempotent)
  AND phases 3-5 execute normally on top of existing data

AC-3.3
GIVEN a seeder run with a newer version of Wiktionary data
WHEN an existing word has additional senses in the new data
THEN the seeder strategy is skip-on-conflict at the ref_entry level
  AND existing entries are NOT modified (preserving data stability)
```

### US-5: CEFR levels and frequency

```
AC-5.1
GIVEN a seeded entry for "the" (NGSL rank 1)
WHEN the entry is queried
THEN frequency_rank = 1, cefr_level = "A1", is_core_lexicon = true

AC-5.2
GIVEN a seeded entry for an NAWL word (e.g., "abstract")
WHEN the entry is queried
THEN cefr_level = "C1", is_core_lexicon = true

AC-5.3
GIVEN a seeded entry NOT in NGSL or NAWL
WHEN the entry is queried
THEN frequency_rank IS NULL, cefr_level IS NULL, is_core_lexicon = false

AC-5.4
GIVEN CEFR heuristic mapping
THEN ranks 1-500 → A1, 501-1200 → A2, 1201-2000 → B1, 2001-2809 → B2, NAWL → C1
```

### US-6: Semantic relations

```
AC-6.1
GIVEN seeded entries for "happy" and "glad" (both in 20k set, synonyms in WordNet)
WHEN relations for "happy" are queried
THEN a row exists in ref_word_relations with relation_type = "synonym"
  AND target_entry_id points to the "glad" ref_entry

AC-6.2
GIVEN a WordNet synonym pair where one word is NOT in the 20k seed set
WHEN the WordNet phase processes this pair
THEN the relation is skipped (both ref_entries must exist)
  AND no orphaned foreign keys are created

AC-6.3
GIVEN the ref_word_relations table after seeding
WHEN checking relation types
THEN only these values exist: "synonym", "hypernym", "derived", "antonym"
```

### US-8: Individual phase execution

```
AC-8.1
GIVEN the seeder CLI
WHEN invoked with --phase=wiktionary
THEN only the Wiktionary loading phase executes
  AND phases 2-5 are skipped

AC-8.2
GIVEN a database with Wiktionary entries already loaded
WHEN invoked with --phase=ngsl
THEN NGSL enrichment runs: updates frequency_rank, cefr_level, is_core_lexicon
  AND only ref_entries matching NGSL/NAWL words are updated
```

### US-N1: Existing data preservation

```
AC-N1.1
GIVEN a ref_entry for "run" with source_slug="freedict" (from a previous user search)
WHEN the seeder runs with Wiktionary data containing "run"
THEN the existing "run" entry is preserved unchanged
  AND text_normalized unique constraint prevents duplicate creation
  AND the user's linked dictionary entry (via ref_entry_id) continues to work
```

---

## 4. Edge Cases & Failure Scenarios

### 4.1 Data Overlap & Merge Strategy

| Scenario | Behavior | Rationale |
|---|---|---|
| Seeder word already exists from FreeDictionary | **Skip** (upsert DO NOTHING on text_normalized) | Existing user-triggered data is valid. Stability over freshness. |
| Seeder word exists from a previous seed run | **Skip** (same upsert) | Idempotent. No accidental overwrites. |
| Same word with different capitalization ("Run" vs "run") | Both normalize to "run" → skip duplicate | text_normalized unique index handles this. |
| Wiktionary has newer/better data than an old seed | Not updated. Operator must delete old entry first to force re-seed. | Trade-off: data stability wins. Document in SOP. |

**Decision needed**: Should we ever allow seeder to UPDATE existing entries (merge new senses, update translations)? **Recommendation**: Not for MVP. The complexity of merge logic (which senses to keep, position reassignment) is high and the benefit is marginal. Skip-on-conflict is safest.

### 4.2 Wiktionary Parsing

| Edge Case | Handling |
|---|---|
| Multiple POS entries for one word ("run" has noun, verb, adj in Kaikki) | Merge into one RefEntry with multiple RefSenses grouped by POS. Group by `word` field. |
| Missing IPA in `sounds[]` | Create entry without pronunciation. CMU phase (phase 3) fills this gap. |
| Missing Russian translations | Create entry without translations. Translations are optional per RefCatalog business rules (translation graceful degradation). |
| Missing definition (empty `glosses`) | Skip this specific sense. A RefSense without a definition has no value. |
| Unusual POS values ("phrase", "idiom", "particle", "determiner", "numeral") | Map known values to existing `PartOfSpeech` enum. Unknown → `OTHER`. The enum already includes PHRASE, IDIOM. New values: "particle" → OTHER, "determiner" → OTHER, "numeral" → OTHER, "article" → OTHER. |
| Extremely long definition (>10,000 chars) | Truncate to a reasonable max (e.g., 5,000 chars). Log warning. |
| Non-English word entries in English Wiktionary dump | Filter: only process entries where `lang_code = "en"`. Kaikki includes borrowed words but marks the language. |
| HTML/wiki markup in definitions or examples | Strip markup during parsing. Definitions should be plain text. |

### 4.3 Pipeline Failures

| Scenario | Behavior | Recovery |
|---|---|---|
| Wiktionary file not found or corrupted | Seeder exits with clear error. No partial data. | Fix file path, re-run. |
| Database connection lost mid-batch | Current batch rolls back. Previous committed batches are durable. | Re-run seeder (idempotent upsert). |
| Disk space insufficient for 2.7 GB file | Seeder validates file exists and is readable at startup. | Operator allocates space. Not a seeder concern. |
| Out of memory during Wiktionary parsing | Stream JSONL line-by-line. Never load full file. | If still OOM, reduce batch size. |
| Partial phase completion (e.g., NGSL CSV malformed at row 1500) | Rows 1-1499 committed (batch boundary). Error logged with row number. | Fix CSV, re-run phase (idempotent UPDATE). |

### 4.4 Concurrent Access During Seeding

| Scenario | Behavior |
|---|---|
| User searches catalog while seeder is running | Works. Seeder uses upsert. User may see partially-enriched entries (senses loaded, CEFR not yet applied). All new fields are nullable — partial data is valid. |
| User creates entry from catalog for a word being seeded | Works. If ref_entry already committed by seeder, user links to it. If not yet committed, user triggers normal FreeDictionary fetch → creates entry → seeder later skips it (upsert DO NOTHING). |
| Two seeder instances running simultaneously | **Not supported.** Document as operator requirement: one seeder at a time. No distributed lock needed — upsert handles data safety, but performance and ordering would be unpredictable. |

### 4.5 Data Quality

| Edge Case | Handling |
|---|---|
| Wiktionary entry with 50+ senses (e.g., "set", "run") | Load all senses. No artificial limit. Position ordering preserves Wiktionary's order. |
| Duplicate Russian translations within one sense | Deduplicate by text during parsing. |
| Tatoeba sentence too long (>500 chars) | Skip. Short, clear examples are better for language learning. |
| Tatoeba sentence doesn't actually contain the target word (false positive from tokenization) | Accept this risk. Tatoeba lookup is by exact word match; most false positives are inflected forms which are still valuable. |
| WordNet synset where all words map to the same RefEntry (homograph) | Skip self-referential relations. |
| CMU pronunciation with multiple variants (e.g., "read" has two) | Insert all variants as separate RefPronunciation rows. |

### 4.6 Boundary Values

| Boundary | Value | Handling |
|---|---|---|
| Total entries to seed | ~20,000 | Configurable via CLI flag (default 20,000). |
| NGSL words | 2,809 | All guaranteed in seed set. |
| NAWL words | ~963 | All guaranteed in seed set. |
| Combined NGSL+NAWL | ~3,800 (some overlap) | Deduplicate by text_normalized. |
| Remaining quota from Wiktionary | ~16,200 | Filled by quality score. |
| Quality score = 0 (no senses, no IPA, no translations) | Excluded. Minimum threshold: at least 1 sense with a non-empty definition. |
| Empty seed set (all files missing) | Seeder exits with error. No empty run. |

---

## 5. Impact Analysis

### 5.1 Services Affected

| Service | Change Type | Details |
|---|---|---|
| **refcatalog** | Domain model + repo | New fields on RefEntry: `FrequencyRank *int`, `CEFRLevel *string`, `IsCoreLexicon bool`. New repo methods for batch upsert if seeder bypasses service layer. No runtime service code changes. |
| **dictionary** | GraphQL schema (optional) | Expose new RefEntry fields (`frequencyRank`, `cefrLevel`, `isCoreLexicon`) in GraphQL. Optional — frontend can ignore initially. |
| **content** | None | Operates on user-owned data. Unaffected. |
| **study** | None | Cards reference entries; entries reference ref_entries. Indirect benefit: more entries available from day 1. |
| **topic** | None | Unaffected. |
| **inbox** | None | Unaffected. |
| **auth/user** | None | Unaffected. |

### 5.2 Data Model Changes

**New columns on `ref_entries`** (migration 00012):
```sql
ALTER TABLE ref_entries ADD COLUMN frequency_rank INT;
ALTER TABLE ref_entries ADD COLUMN cefr_level TEXT;
ALTER TABLE ref_entries ADD COLUMN is_core_lexicon BOOLEAN NOT NULL DEFAULT false;

CREATE INDEX ix_ref_entries_cefr_level ON ref_entries(cefr_level) WHERE cefr_level IS NOT NULL;
CREATE INDEX ix_ref_entries_frequency_rank ON ref_entries(frequency_rank) WHERE frequency_rank IS NOT NULL;
```

**New table `ref_word_relations`** (migration 00012 or 00013):
```sql
CREATE TABLE ref_word_relations (
    id               UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    source_entry_id  UUID NOT NULL REFERENCES ref_entries(id) ON DELETE CASCADE,
    target_entry_id  UUID NOT NULL REFERENCES ref_entries(id) ON DELETE CASCADE,
    relation_type    TEXT NOT NULL,
    source_slug      TEXT NOT NULL,
    UNIQUE(source_entry_id, target_entry_id, relation_type)
);

CREATE INDEX ix_ref_word_relations_source ON ref_word_relations(source_entry_id);
CREATE INDEX ix_ref_word_relations_target ON ref_word_relations(target_entry_id);
```

**Domain model changes** (`domain/reference.go`):
```go
type RefEntry struct {
    // ... existing fields ...
    FrequencyRank  *int     // NEW: NGSL/NAWL rank (lower = more frequent)
    CEFRLevel      *string  // NEW: A1/A2/B1/B2/C1 (null if unknown)
    IsCoreLexicon  bool     // NEW: true if in NGSL or NAWL
}

// NEW type
type RefWordRelation struct {
    ID            uuid.UUID
    SourceEntryID uuid.UUID
    TargetEntryID uuid.UUID
    RelationType  string    // "synonym", "hypernym", "derived", "antonym"
    SourceSlug    string
}
```

### 5.3 API Contract Changes

**GraphQL schema additions** (non-breaking, all new fields):
```graphql
type RefEntry {
  # ... existing fields ...
  frequencyRank: Int        # NEW
  cefrLevel: String         # NEW
  isCoreLexicon: Boolean!   # NEW
  relations: [RefWordRelation!]!  # NEW (loaded via DataLoader)
}

type RefWordRelation {       # NEW type
  id: UUID!
  targetEntry: RefEntry!
  relationType: String!
}
```

**No breaking changes.** All additions are optional nullable fields or new types. Existing queries continue to work unchanged.

### 5.4 New Entry Point

**`cmd/seeder/main.go`** — standalone CLI binary. Following existing pattern from `cmd/cleanup/main.go`:
- Loads config (DB DSN, dataset file paths)
- Connects to database
- Runs ETL phases sequentially
- Exits with 0 (success) or 1 (error)

CLI flags:
- `--config` / `CONFIG_PATH` — config file path
- `--phase` — run specific phase (wiktionary, ngsl, cmu, wordnet, tatoeba)
- `--dry-run` — parse and validate without writing
- `--limit` — override entry count limit (default 20,000)
- `--batch-size` — override batch size (default 500)

### 5.5 Configuration Changes

New config section in `config.yaml`:
```yaml
seeder:
  wiktionary_path: ""      # path to kaikki JSONL file
  ngsl_path: ""            # path to NGSL CSV
  nawl_path: ""            # path to NAWL CSV
  cmu_path: ""             # path to CMU dict (IPA version)
  wordnet_path: ""         # path to WordNet JSON
  tatoeba_sentences_path: "" # path to Tatoeba sentences TSV
  tatoeba_links_path: ""    # path to Tatoeba links TSV
  entry_limit: 20000       # max entries to seed
  batch_size: 500          # DB batch insert size
```

**SPEC UPDATE REQUIRED**: config package — add SeederConfig struct.

### 5.6 Existing Behavior Changes

**None.** This is purely additive:
- `GetOrFetchEntry` logic unchanged — step 2 (DB lookup) now hits for ~20k words
- Search returns more results from richer catalog
- User entry creation flow unchanged
- Study sessions unchanged

---

## 6. Risk Assessment

| # | Risk | Level | Description | Mitigation |
|---|---|---|---|---|
| R1 | Scope creep into "Discover Words" UI | **HIGH** | CEFR/frequency metadata enables browsing, level progression, word-of-the-day — each a separate feature | Strict scope: seed data + expose metadata in API only. All browsing/discovery features are separate tasks. Flag US-11 as DEFERRED. |
| R2 | Wiktionary data quality variance | **MEDIUM** | Entries vary in completeness. Some words may have poor definitions, no translations, or no IPA. | Quality scoring filter during two-pass scan. Post-seed sampling of 100 random entries for manual review. Log statistics (entries with/without translations, IPA coverage %). |
| R3 | Merge conflict with existing FreeDictionary data | **MEDIUM** | Users who already fetched words have `source_slug="freedict"` entries. Seeder must not corrupt these. | Skip-on-conflict strategy (INSERT ON CONFLICT DO NOTHING). Existing data preserved. Seeded data never overwrites. |
| R4 | Seeder performance on 2.7 GB file | **MEDIUM** | Streaming 2.7 GB JSONL + filtering + inserting 20k entries with ~300k child rows. | Stream line-by-line (never load full file). Two-pass approach (scan then load) requires two file reads but keeps memory bounded. Batch inserts of 500. Target: <15 min total. |
| R5 | Wiktionary POS mapping completeness | **LOW** | Wiktionary uses POS strings different from FreeDictionary. Unmapped values silently become OTHER. | Build comprehensive mapping table. Log unmapped values during seeding for operator review. Existing enum already covers major categories. |
| R6 | License compliance | **LOW** | All datasets are CC/BSD. Must attribute. | Add `LICENSE-DATASETS.md` documenting each source, license, and attribution. |
| R7 | Dataset staleness | **LOW** | Kaikki updates biweekly, WordNet yearly. Seeded data could drift. | Seeder is re-runnable. Document recommended re-seed schedule (quarterly). |
| R8 | Migration rollback complexity | **LOW** | New columns and table are additive. | Migration down drops new columns/table. No data loss in other tables. Seeded data would be lost but is regeneratable. |

---

## 7. Success Metrics

| Metric | Type | Target | Measurement |
|---|---|---|---|
| Catalog hit rate | Technical | >95% of common vocabulary lookups resolved from DB (no external API) | Log analysis: GetOrFetchEntry calls hitting step 2 vs step 3 |
| Seeded entry count | Data quality | 19,500-21,000 entries | `SELECT COUNT(*) FROM ref_entries WHERE created_at >= seed_timestamp` |
| Translation coverage | Data quality | >60% of seeded entries have >=1 Russian translation | `SELECT COUNT(*) FROM ref_entries e JOIN ref_senses s ON ... JOIN ref_translations t ON ... WHERE t.source_slug = 'wiktionary'` |
| IPA coverage | Data quality | >80% of seeded entries have IPA transcription | Post-seed query on ref_pronunciations |
| CEFR coverage | Data quality | ~3,800 entries (all NGSL+NAWL words) with non-null cefr_level | `SELECT COUNT(*) FROM ref_entries WHERE cefr_level IS NOT NULL` |
| Seeder execution time | Operational | <15 minutes for full pipeline | CLI timing output |
| Peak memory usage | Operational | <512 MB | Monitoring during execution |
| External API call reduction | Cost | >80% reduction in FreeDictionary/Translate calls for common words | Before/after comparison over first month |

---

## 8. Open Questions

| # | Question | Recommended Answer | Status |
|---|---|---|---|
| Q1 | Should seeder ever UPDATE existing entries (merge new senses from updated Wiktionary)? | **No** for MVP. Skip-on-conflict only. Merge logic is complex and risky. | Recommended: defer to v2 |
| Q2 | Should seeder use the existing `refentry.Repo` or direct SQL for batch operations? | Direct SQL with `pgx.CopyFrom` or batch `INSERT ... ON CONFLICT`. The repo's `CreateWithTree` is designed for single-entry transactions. Batch operations need a separate code path per ADR-002 (seeder is admin tool, can bypass service layer). | Recommended: direct SQL |
| Q3 | Where should dataset files be stored? In-repo, external download, or Docker volume? | External download. Too large for git. Document download URLs in `README.md` or `SEEDER.md`. Config points to local paths. | Recommended: external + docs |
| Q4 | Should the Wiktionary two-pass scan persist intermediate results (quality scores)? | In-memory map `word → score` during pass 1 is sufficient for 1.35M entries (~100 MB). No need for intermediate file or DB table. | Recommended: in-memory |
| Q5 | Should relation_type in ref_word_relations be an ENUM or TEXT? | TEXT. Only 4 values now but WordNet has 30+ relation types. TEXT avoids migration for each new type. Validate in application layer. | Recommended: TEXT |

---

## 9. Implementation Scope Boundary

### In scope (this task)
- Migration: new columns on ref_entries, new ref_word_relations table
- Domain model: RefEntry new fields, RefWordRelation type
- `cmd/seeder/`: CLI binary with 5-phase ETL pipeline
- Seeder config: dataset paths, limits
- Parsers: Wiktionary JSONL, NGSL/NAWL CSV, CMU text, WordNet JSON, Tatoeba TSV
- GraphQL schema: expose new RefEntry fields and relations
- `LICENSE-DATASETS.md`

### Explicitly NOT in scope
- "Discover Words" browsing UI (US-11) — separate feature
- Seeder automated scheduling (cron) — manual run for now
- Updating/merging existing entries — skip-on-conflict only
- Frontend changes — new fields are optional, frontend can ignore
- Replacing FreeDictionary lazy loading — it continues as fallback
- Word-of-the-day, level progression, or any gamification
