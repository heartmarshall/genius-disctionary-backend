# Classifier Prompt Template

Use this exact prompt when launching Task tool batches. Fill in the variables marked
with `{curly_braces}`.

---

## Prompt for each Task

```
You are classifying words extracted from song lyrics by {artist_name} for a language
learning application. Your job: identify words that are NOT useful for an English learner.

This is batch {batch_number} of {total_batches}.

## Categories

Only classify a word if you are CONFIDENT it is non-learnable. When in doubt, DO NOT
classify it -- leave it out of the results (it will be kept as a learnable word).

### ARTIST_NAME
The artist being analyzed, their alter egos, featured artists, or other musician names
referenced in lyrics.
- The artist's own name and aliases (e.g., "eminem", "slim", "shady", "marshall" for Eminem)
- Featured artist names (e.g., "rihanna", "dido", "skylar" when appearing in Eminem's songs)
- Rapper aliases and stage names (e.g., "2pac", "biggie", "dre", "50cent")
- BUT: if a name is ALSO a common English word, DO NOT classify it.
  Examples to KEEP: "drake" (a male duck), "sting" (to sting), "madonna" (artistic term)

### PROPER_NOUN
Real-world named entities that are not learnable vocabulary:
- People's names: first names, surnames, historical figures
  Examples: zimmerman, abernathy, jonbenet
- Place names that are not common vocabulary: detroit, compton, bronx
  BUT KEEP: common place-derived words used as regular English (champagne, cologne)
- Brand names with no general meaning: toyota, gucci, hennessy
  BUT KEEP: brands that have become common words (google as a verb, uber as "very")

### FOREIGN
Non-English words from other languages:
- Spanish: callate, espanol, maricon, adios
- French (not adopted into English): grace (French sense), murcie
- Any other language
- BUT DO NOT classify English borrowings that appear in English dictionaries:
  cliche, fiance, deja vu, karate, sushi, etc.

### SOUND_EFFECT
Pure vocal sounds and ad-libs with NO dictionary meaning:
- Vocal sounds: brr, skrrt, grrah, pew
- Extended sounds: la la la (as "la"), ba da (as "ba", "da")
- BUT KEEP: sounds that ARE also real words: bang, crash, boom, pop, snap, buzz, hiss, click

### SLANG
ONLY classify as SLANG if the word:
1. Has no entry in any English dictionary (including slang dictionaries)
2. Is completely opaque to a language learner (they cannot guess the meaning)
3. Is hyper-specific to a subculture with zero general use

DO NOT exclude informal -in' endings (actin, flexin, zonin). These are standard
informal English that learners encounter constantly. KEEP THEM.

Be EXTREMELY conservative with SLANG. Most "slang" is actually useful for learners.
Only exclude truly impenetrable coinages.

### MUSIC_JUNK
Song-specific or music-metadata artifacts:
- Made-up words specific to one song with no general meaning
- This category should be VERY rarely used

Almost all music-related words (remix, beat, verse, hook, chorus, intro, outro)
are real English words. DO NOT classify them.

## Rules

1. When in doubt, DO NOT classify the word. Leaving a junk word in is cheap.
   Excluding a useful word is expensive.
2. Words that are BOTH a name AND a real English word -> DO NOT classify.
   grace, hunter, rose, frank, bill, mason, angel, chase, dawn, faith, joy, mark,
   ray, will, harmony, melody, summer, violet, crystal, diamond, pearl, ruby, amber,
   brook, cliff, dale, glen, heath, lane, wade, chance, hope, miles, august, may, june
3. Informal spellings of real words -> DO NOT classify.
   gonna, wanna, gotta, tryna, finna, bouta, ima, fam, lit, vibe, lowkey, highkey,
   flex, cap, slay, extra, basic, salty, shook, fire, dope, sick, tight, cold, hard
4. Profanity and vulgar words -> DO NOT classify. They are real English words.
5. Musical terms -> DO NOT classify: beat, bass, drop, hook, bridge, verse, flow, bar,
   rhythm, groove, jam, track, tune, record, album, single, gig, riff, solo

## Output Format

Return ONLY a JSON array. No markdown fences. No explanation. No commentary.

Each element: {"word": "the_word", "category": "CATEGORY_NAME"}

Only include words you are classifying as non-learnable. Omit all words you consider
learnable (they are kept by default).

Example:
[
  {"word": "2pac", "category": "ARTIST_NAME"},
  {"word": "zimmerman", "category": "PROPER_NOUN"},
  {"word": "callate", "category": "FOREIGN"}
]

If NO words in this batch should be excluded, return an empty array: []

## Words to Classify

{batch_words}
```
