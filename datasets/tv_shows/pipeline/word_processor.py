"""Word tokenization, lemmatization, and filtering using NLTK."""

import re
import nltk
from nltk.stem import WordNetLemmatizer
from nltk.corpus import wordnet, stopwords
from nltk.tokenize import word_tokenize
from collections import Counter

# Ensure required NLTK data is available.
_RESOURCES = {
    "tokenizers/punkt_tab": "punkt_tab",
    "corpora/wordnet": "wordnet",
    "corpora/stopwords": "stopwords",
    "taggers/averaged_perceptron_tagger_eng": "averaged_perceptron_tagger_eng",
}
for path, name in _RESOURCES.items():
    try:
        nltk.data.find(path)
    except LookupError:
        nltk.download(name, quiet=True)

_lemmatizer = WordNetLemmatizer()
_stopwords = set(stopwords.words("english"))

# Map Penn Treebank POS tags to WordNet POS for better lemmatization.
_TAG_MAP = {
    "J": wordnet.ADJ,
    "V": wordnet.VERB,
    "N": wordnet.NOUN,
    "R": wordnet.ADV,
}

# POS tags to keep: common nouns, verbs, adjectives, adverbs, modals.
_KEEP_POS = {
    "NN", "NNS",                                    # common nouns
    "VB", "VBD", "VBG", "VBN", "VBP", "VBZ",       # verbs
    "JJ", "JJR", "JJS",                             # adjectives
    "RB", "RBR", "RBS",                             # adverbs
    "MD",                                            # modals
}

# POS tags for proper nouns — always reject.
_PROPER_NOUN_POS = {"NNP", "NNPS"}

RE_ONLY_ALPHA = re.compile(r"^[a-z][a-z']*[a-z]$|^[a-z]$")

# Contraction fragments produced by NLTK tokenizer.
_CONTRACTION_PARTS = {"n't", "'s", "'re", "'ve", "'ll", "'d", "'m", "na", "gon",
                       "N'T", "'S", "'RE", "'VE", "'LL", "'D", "'M",
                       "ca", "wo", "sha",   # can't→ca, won't→wo, shan't→sha
                       "Ca", "Wo", "Sha"}


def _get_wordnet_pos(treebank_tag: str) -> str:
    return _TAG_MAP.get(treebank_tag[0], wordnet.NOUN)


def extract_words(lines: list[str]) -> Counter:
    """Tokenize, lemmatize, filter, and count words from subtitle text lines.

    POS tagging runs on original-case text so NLTK can detect proper nouns.
    """
    counts: Counter = Counter()

    for line in lines:
        # Tokenize and POS-tag with original case for proper noun detection.
        tokens_original = word_tokenize(line)
        tagged = nltk.pos_tag(tokens_original)

        for token, tag in tagged:
            # Skip contraction fragments.
            if token in _CONTRACTION_PARTS:
                continue

            # Lowercase for further processing.
            lower = token.lower().replace("\u2019", "'")

            # Skip non-alphabetic tokens.
            if not RE_ONLY_ALPHA.match(lower):
                continue

            # Skip single characters.
            if len(lower) == 1:
                continue

            # Reject proper nouns.
            if tag in _PROPER_NOUN_POS:
                continue

            # Filter by POS: only keep content words.
            if tag not in _KEEP_POS:
                continue

            # Lemmatize with POS context.
            lemma = _lemmatizer.lemmatize(lower, _get_wordnet_pos(tag))

            # Skip stopwords.
            if lemma in _stopwords or lower in _stopwords:
                continue

            if len(lemma) < 2:
                continue

            counts[lemma] += 1

    return counts
