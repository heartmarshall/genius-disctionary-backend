# Pre-filter Rules for Lyrics Word Lists

Deterministic rules applied BEFORE sending words to the LLM classifier.
These catch clear-cut cases reliably and reduce batch sizes.

## Auto-Exclude Rules

Each rule maps to an exclusion category written directly to the CSV.

### PARSE_ERROR

1. **Zero-width characters**: Any string containing U+200B (zero-width space), U+200C,
   U+200D (zero-width joiner), or U+FEFF (BOM). These are Genius annotation artifacts.
   Examples: `a.m.(zwsp)`, `inc.(zwsp)`, `(zwsp)-pectation`, `(zwsp)turne`

2. **Genius annotation junk**: Strings containing `*scratch*`, `!"--`, `,"--`, `'--`,
   or similar punctuation-heavy sequences that are clearly parsing artifacts.
   Examples: `gay!"--that`, `pop,"--'cause`, `somethin'--nigga`, `galf*`

3. **Leading hyphens**: Words starting with `-` are truncated fragments from line breaks
   or annotation splits.
   Examples: `-inem`, `-se`, `-tober`

4. **Pure numeric**: Strings that are entirely digits.
   Examples: `1`, `69`, `100`

5. **Alphanumeric codes**: Strings matching patterns like `digit+letter` or
   `letter+digit` that are not real words. BUT keep common valid forms:
   - Keep: `a.m.`, `p.m.`, `ok` (already in auto-keep)
   - Exclude: `a1`, `3d`, `3s`, `49er`, `5'9"s`, `69ed`, `1x01`, `720p`
   - Rule: exclude if string contains digits AND is not a recognized English word

6. **Encoding artifacts**: Strings containing `\x`, `&#`, `&amp;`, `&lt;`, `&gt;`,
   or mixed Cyrillic+Latin characters (e.g., Cyrillic `e` in otherwise Latin text).
   Examples: `bitches` (Cyrillic e), `children`, `eight`, `repent`, `temple`, `the`
   Detection: check if any character in the word has a Unicode category of Cyrillic
   while the rest is Latin.

7. **Strings longer than 25 characters**: Almost certainly parsing errors or concatenated
   words.

8. **Isolated contraction fragments**: These exact strings when they appear as standalone
   words: `ll`, `ve`, `re`
   BUT: do NOT exclude `em`, `er`, `ol`, `ya`, `da`, `ta`, `na` -- these could be
   informal words. Let the classifier decide.

9. **Single characters**: Exclude all single characters EXCEPT: `a`, `i`, `o`
   (these are real English words or common interjections).

10. **Punctuation-laden strings**: Words containing `!`, `"`, `(`, `)`, `{`, `}`, `*`
    or more than one period that is not `a.m.` or `p.m.`

### GIBBERISH

11. **Repeated characters**: Words with 3 or more consecutive identical characters.
    Examples: `aaahhh`, `wooooo`, `yeeeah`, `ooooh`
    BUT: do NOT flag words where the repetition is standard English spelling
    (there are very few -- `brrr` is about the only common one).

## Auto-Keep Rules (Protected Words)

These words are NEVER sent to the classifier and NEVER excluded.

### Two-letter English words
```
ah, am, an, as, at, ax, be, by, do, go, ha, he, hi, ho, if, in, is, it,
me, my, no, of, oh, ok, on, or, ow, ox, so, to, up, us, we
```

### Common informal/colloquial words
```
okay, yeah, yep, nope, nah, alright, gonna, wanna, gotta, lemme, gimme,
kinda, sorta, dunno, cool, dude, damn, shit, hell, ass, bitch, fuck,
wow, whoa, ooh, oops, shh, ugh, hmm, huh, hey, yo, aye, naw
```

### English borrowings with diacritics
Words that use accented characters but ARE standard English vocabulary:
```
cliche, fiance, fiancee, facade, touche, cafe, naive, resume, seance,
decor, debris, debut, protege, souffle, creme, entree, premiere
```
These often appear in lyrics and should be kept. The pre-filter should recognize
them (with or without accent marks) and protect them from being flagged as FOREIGN.

### Contractions with apostrophes
```
'bout, 'cause, 'til, 'em, 'nough, 'round
```
These are informal but standard English contractions commonly used in lyrics.
Keep them -- they are learnable vocabulary.

## Implementation Notes

- Apply auto-exclude rules first, collect excluded words with categories
- Then apply auto-keep rules to the remaining words
- Send only the remainder to the Task classifier
- Log counts for each rule to help with debugging
- When in doubt about a rule edge case, let the word through to the classifier
