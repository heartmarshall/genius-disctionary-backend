"""Word tokenization, lemmatization, and filtering using spaCy."""

import re
from collections import Counter

import spacy

RE_ONLY_ALPHA = re.compile(r"^[a-z][a-z']*[a-z]$|^[a-z]$")

# POS tags to keep: nouns, verbs, adjectives, adverbs.
KEEP_POS = {"NOUN", "VERB", "ADJ", "ADV"}

# NER labels to reject (proper nouns / named entities).
REJECT_NER = {"PERSON", "GPE", "ORG", "FAC", "NORP", "EVENT", "WORK_OF_ART", "LOC"}

# Singleton — loaded once, reused across calls.
_nlp = None


def _get_nlp():
    """Load spaCy model on first use."""
    global _nlp
    if _nlp is None:
        _nlp = spacy.load("en_core_web_sm")
        _nlp.max_length = 600_000  # Handle large chapters
    return _nlp


def extract_words(text: str) -> Counter:
    """Tokenize, lemmatize, filter, and count words from text.

    Uses spaCy en_core_web_sm for tokenization, lemmatization, POS tagging, NER.

    Keeps: NOUN, VERB, ADJ, ADV (non-proper, non-entity, non-stop).
    Rejects: punctuation, numbers, proper nouns, named entities, stopwords,
             words < 2 chars, non-alphabetic tokens.
    """
    nlp = _get_nlp()
    doc = nlp(text)

    # Collect entity spans for fast lookup.
    ent_tokens: set[int] = set()
    for ent in doc.ents:
        if ent.label_ in REJECT_NER:
            for token in ent:
                ent_tokens.add(token.i)

    counts: Counter = Counter()

    for token in doc:
        # Skip punctuation, spaces, numbers.
        if token.is_punct or token.is_space or token.like_num:
            continue

        # Skip proper nouns by POS.
        if token.pos_ == "PROPN":
            continue

        # Skip named entities.
        if token.i in ent_tokens:
            continue

        # Only keep content POS.
        if token.pos_ not in KEEP_POS:
            continue

        # Lemmatize and lowercase.
        lemma = token.lemma_.lower().replace("\u2019", "'")

        # Regex validation: only alphabetic + apostrophes.
        if not RE_ONLY_ALPHA.match(lemma):
            continue

        # Min length 2.
        if len(lemma) < 2:
            continue

        # Skip stopwords.
        if token.is_stop:
            continue

        counts[lemma] += 1

    return counts
