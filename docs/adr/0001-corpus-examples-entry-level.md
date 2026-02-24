# 0001. Corpus Examples Attach to ref_entry, Not ref_sense

**Date**: 2026-02-24
**Status**: Proposed (from feature brainstorm)

## Context

The project has a `ref_examples` table where example sentences are stored at the `ref_sense` level — each example is tied to a specific word meaning (part of speech + definition). This works for curated dictionary data from Tatoeba and FreeDictionary, where a human or structured dataset has already classified each sentence by word sense.

For corpus data (TV subtitles, book text, song lyrics), sense disambiguation is impractical without a dedicated NLP pipeline. A sentence from Breaking Bad containing the word "resilient" cannot be automatically assigned to the correct ref_sense without heavy NLP tooling (word sense disambiguation). Forcing this assignment would require either:
- A full NLP WSD pipeline (significant complexity, error-prone)
- An arbitrary "catch-all" sense assignment (misleads users about sense-specific examples)

## Decision

Corpus-sourced examples are stored in a separate table `ref_corpus_examples` that attaches to `ref_entry_id` directly, bypassing `ref_sense_id`. The existing `ref_examples` table remains sense-level and is not modified.

This creates a two-tier example structure:
- `ref_examples` (sense-level): curated dictionary examples from Tatoeba, FreeDictionary, LLM generation — attached to specific word meanings
- `ref_corpus_examples` (entry-level): contextual examples from books/TV/lyrics — attached to the headword, no sense classification

## Consequences

### Positive
- No NLP sense disambiguation required for corpus pipeline
- Existing `ref_examples` schema and code remain unchanged
- Clean separation: dictionary examples vs. contextual corpus examples
- Schema is forward-compatible: `ref_sense_id` can be added to `ref_corpus_examples` later when/if NLP pipeline is built
- Rollout is safe: new table, additive GraphQL field

### Negative
- Corpus examples cannot indicate which meaning of the word they illustrate
- UI must present corpus examples as "usage in context" without sense disambiguation
- Duplicate data if a word has both curated and corpus examples (acceptable — different value)

### Neutral
- GraphQL exposes corpus examples via separate field `corpusExamples` on `RefEntry`, not mixed with `examples` on `RefSense`
- DataLoader pattern required to avoid N+1 (same as existing examples)

## Alternatives Considered

**Make `ref_sense_id` nullable in `ref_examples`**: Rejected — modifies existing table semantics; requires updating all queries; confusing when some examples have sense and others don't.

**Add NLP sense disambiguation to pipeline**: Deferred — would require spaCy WSD or an LLM call per sentence; adds ~30+ days of complexity; can be added later as an enhancement.

**Assign all corpus examples to a synthetic "general" sense**: Rejected — misleads users and corrupts the sense-level data model.
