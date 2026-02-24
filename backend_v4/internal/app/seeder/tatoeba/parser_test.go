package tatoeba

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/google/uuid"
)

func testdataPath(t *testing.T, name string) string {
	t.Helper()
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("cannot determine test file path")
	}
	return filepath.Join(filepath.Dir(file), "testdata", name)
}

// writeFile is a test helper that creates a file with given content.
func writeFile(path, content string) error {
	return os.WriteFile(path, []byte(content), 0o644)
}

// --- Tokenizer tests ---

func TestTokenize(t *testing.T) {
	tests := []struct {
		name     string
		sentence string
		want     []string
	}{
		{
			name:     "simple sentence",
			sentence: "The cat sat on the mat.",
			want:     []string{"the", "cat", "sat", "on", "mat"},
		},
		{
			name:     "contraction don't",
			sentence: "I don't know.",
			want:     []string{"i", "don't", "know"},
		},
		{
			name:     "contraction can't",
			sentence: "I can't believe it.",
			want:     []string{"i", "can't", "believe", "it"},
		},
		{
			name:     "warehouse not matching house",
			sentence: "The warehouse is big.",
			want:     []string{"the", "warehouse", "is", "big"},
		},
		{
			name:     "punctuation removal",
			sentence: "Hello, world! How are you?",
			want:     []string{"hello", "world", "how", "are", "you"},
		},
		{
			name:     "mixed case",
			sentence: "BIG Cat SAT",
			want:     []string{"big", "cat", "sat"},
		},
		{
			name:     "extra spaces",
			sentence: "  the   cat  ",
			want:     []string{"the", "cat"},
		},
		{
			name:     "empty string",
			sentence: "",
			want:     nil,
		},
		{
			name:     "only punctuation",
			sentence: "!!! ???",
			want:     nil,
		},
		{
			name:     "hyphenated word",
			sentence: "well-known fact",
			want:     []string{"well", "known", "fact"},
		},
		{
			name:     "apostrophe at start",
			sentence: "'twas the night",
			want:     []string{"twas", "the", "night"},
		},
		{
			name:     "possessive",
			sentence: "The cat's toy.",
			want:     []string{"the", "cat's", "toy"},
		},
		{
			name:     "it's contraction",
			sentence: "It's raining.",
			want:     []string{"it's", "raining"},
		},
		{
			name:     "duplicate words in sentence",
			sentence: "the cat and the cat",
			want:     []string{"the", "cat", "and"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tokenize(tt.sentence)

			if len(got) != len(tt.want) {
				t.Fatalf("tokenize(%q) = %v (len %d), want %v (len %d)",
					tt.sentence, got, len(got), tt.want, len(tt.want))
			}

			// Convert to set for order-independent comparison.
			gotSet := make(map[string]bool, len(got))
			for _, w := range got {
				gotSet[w] = true
			}
			for _, w := range tt.want {
				if !gotSet[w] {
					t.Errorf("tokenize(%q): missing expected token %q; got %v",
						tt.sentence, w, got)
				}
			}
		})
	}
}

// --- Parse tests ---

func TestParse(t *testing.T) {
	path := testdataPath(t, "sample.tsv")

	knownWords := map[string]bool{
		"cat":   true,
		"house": true,
		"don't": true,
		"can't": true,
		"hello": true,
	}

	result, err := Parse(path, knownWords, DefaultMaxPerWord)
	if err != nil {
		t.Fatalf("Parse returned error: %v", err)
	}

	// Stats checks.
	// 14 total lines in sample.tsv.
	if result.Stats.TotalLines != 14 {
		t.Errorf("TotalLines: got %d, want 14", result.Stats.TotalLines)
	}
	// 1 line is >500 chars.
	if result.Stats.SkippedLong != 1 {
		t.Errorf("SkippedLong: got %d, want 1", result.Stats.SkippedLong)
	}

	// "cat" should be matched (appears in multiple sentences).
	// "house" should be matched (appears in "This is a house.").
	// "don't" should be matched.
	// "can't" should be matched.
	// "hello" should be matched.
	// But NOT "warehouse" — "house" does NOT match "warehouse" thanks to tokenization.
	if result.Stats.MatchedWords < 5 {
		t.Errorf("MatchedWords: got %d, want >= 5", result.Stats.MatchedWords)
	}

	// Check cat sentences.
	catPairs, ok := result.Sentences["cat"]
	if !ok {
		t.Fatal("expected 'cat' in Sentences")
	}
	// cat appears in lines: 1 (sat on mat), 4 (sleeping), 5 (big cat appeared),
	// 6 (ran quickly...), 7 (likes fish), 8 (old), 9 (old cat sleeps) = 7 valid lines
	// (line 14 is >500 chars so excluded)
	// maxPerWord=5, so should be capped at 5.
	if len(catPairs) != 5 {
		t.Errorf("cat pairs: got %d, want 5 (maxPerWord)", len(catPairs))
	}
	// Sorted by English sentence length (shortest first).
	for i := 1; i < len(catPairs); i++ {
		if len(catPairs[i].English) < len(catPairs[i-1].English) {
			t.Errorf("cat pairs not sorted by length: [%d]=%q (len %d) before [%d]=%q (len %d)",
				i-1, catPairs[i-1].English, len(catPairs[i-1].English),
				i, catPairs[i].English, len(catPairs[i].English))
		}
	}

	// Check house — only "This is a house." should match, NOT "The warehouse is big."
	housePairs, ok := result.Sentences["house"]
	if !ok {
		t.Fatal("expected 'house' in Sentences")
	}
	if len(housePairs) != 1 {
		t.Errorf("house pairs: got %d, want 1", len(housePairs))
	}
	if housePairs[0].English != "This is a house." {
		t.Errorf("house[0].English: got %q, want %q", housePairs[0].English, "This is a house.")
	}
	if housePairs[0].Russian != "Это дом." {
		t.Errorf("house[0].Russian: got %q, want %q", housePairs[0].Russian, "Это дом.")
	}

	// Check don't.
	dontPairs, ok := result.Sentences["don't"]
	if !ok {
		t.Fatal("expected \"don't\" in Sentences")
	}
	if len(dontPairs) != 1 {
		t.Errorf("don't pairs: got %d, want 1", len(dontPairs))
	}
	if dontPairs[0].English != "I don't know." {
		t.Errorf("don't[0].English: got %q, want %q", dontPairs[0].English, "I don't know.")
	}

	// Check can't.
	cantPairs, ok := result.Sentences["can't"]
	if !ok {
		t.Fatal("expected \"can't\" in Sentences")
	}
	if len(cantPairs) != 1 {
		t.Errorf("can't pairs: got %d, want 1", len(cantPairs))
	}

	// Check hello.
	helloPairs, ok := result.Sentences["hello"]
	if !ok {
		t.Fatal("expected 'hello' in Sentences")
	}
	if len(helloPairs) != 1 {
		t.Errorf("hello pairs: got %d, want 1", len(helloPairs))
	}
}

func TestParse_SubstringFalseMatch(t *testing.T) {
	// This is the critical test: "house" must NOT match "warehouse".
	dir := t.TempDir()
	path := filepath.Join(dir, "test.tsv")
	content := "1\tThe warehouse is big.\t2\tСклад большой.\n"
	if err := writeFile(path, content); err != nil {
		t.Fatal(err)
	}

	knownWords := map[string]bool{"house": true}
	result, err := Parse(path, knownWords, DefaultMaxPerWord)
	if err != nil {
		t.Fatalf("Parse returned error: %v", err)
	}

	if _, ok := result.Sentences["house"]; ok {
		t.Error("'house' should NOT match 'warehouse' — tokenization prevents substring matching")
	}
}

func TestParse_FileNotFound(t *testing.T) {
	_, err := Parse("/nonexistent/file.tsv", map[string]bool{"cat": true}, DefaultMaxPerWord)
	if err == nil {
		t.Error("Parse should return error for missing file")
	}
}

func TestParse_EmptyFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "empty.tsv")
	if err := writeFile(path, ""); err != nil {
		t.Fatal(err)
	}

	result, err := Parse(path, map[string]bool{"cat": true}, DefaultMaxPerWord)
	if err != nil {
		t.Fatalf("Parse should not error on empty file: %v", err)
	}
	if len(result.Sentences) != 0 {
		t.Errorf("expected 0 sentences, got %d", len(result.Sentences))
	}
	if result.Stats.TotalLines != 0 {
		t.Errorf("TotalLines: got %d, want 0", result.Stats.TotalLines)
	}
}

func TestParse_AllMalformedLines(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "malformed.tsv")
	content := "only one field\nonly\ttwo\tfields\n"
	if err := writeFile(path, content); err != nil {
		t.Fatal(err)
	}

	result, err := Parse(path, map[string]bool{"cat": true}, DefaultMaxPerWord)
	if err != nil {
		t.Fatalf("Parse should not error on malformed lines: %v", err)
	}
	if len(result.Sentences) != 0 {
		t.Errorf("expected 0 sentences, got %d", len(result.Sentences))
	}
	if result.Stats.TotalLines != 2 {
		t.Errorf("TotalLines: got %d, want 2", result.Stats.TotalLines)
	}
}

func TestParse_MaxPerWordLimit(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "many.tsv")

	// Create 10 lines all containing "cat".
	var content string
	for i := 0; i < 10; i++ {
		content += "1\tThe cat sat.\t2\tКот сидел.\n"
	}
	if err := writeFile(path, content); err != nil {
		t.Fatal(err)
	}

	knownWords := map[string]bool{"cat": true}
	result, err := Parse(path, knownWords, 3)
	if err != nil {
		t.Fatalf("Parse returned error: %v", err)
	}

	catPairs := result.Sentences["cat"]
	if len(catPairs) != 3 {
		t.Errorf("cat pairs: got %d, want 3 (maxPerWord=3)", len(catPairs))
	}
}

func TestParse_ShortestFirst(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "sort.tsv")
	content := "1\tThe big cat ran quickly.\t2\tБольшой кот бежал быстро.\n" +
		"3\tA cat.\t4\tКот.\n" +
		"5\tMy cat is here now.\t6\tМой кот здесь сейчас.\n"
	if err := writeFile(path, content); err != nil {
		t.Fatal(err)
	}

	knownWords := map[string]bool{"cat": true}
	result, err := Parse(path, knownWords, 10)
	if err != nil {
		t.Fatalf("Parse returned error: %v", err)
	}

	catPairs := result.Sentences["cat"]
	if len(catPairs) != 3 {
		t.Fatalf("cat pairs: got %d, want 3", len(catPairs))
	}

	// Verify sorted by English length ascending.
	if catPairs[0].English != "A cat." {
		t.Errorf("catPairs[0].English: got %q, want %q", catPairs[0].English, "A cat.")
	}
	if catPairs[1].English != "My cat is here now." {
		t.Errorf("catPairs[1].English: got %q, want %q", catPairs[1].English, "My cat is here now.")
	}
	if catPairs[2].English != "The big cat ran quickly." {
		t.Errorf("catPairs[2].English: got %q, want %q", catPairs[2].English, "The big cat ran quickly.")
	}
}

func TestParse_EmptyKnownWords(t *testing.T) {
	path := testdataPath(t, "sample.tsv")

	result, err := Parse(path, map[string]bool{}, DefaultMaxPerWord)
	if err != nil {
		t.Fatalf("Parse returned error: %v", err)
	}

	if len(result.Sentences) != 0 {
		t.Errorf("expected 0 matched words with empty knownWords, got %d", len(result.Sentences))
	}
	if result.Stats.MatchedWords != 0 {
		t.Errorf("MatchedWords: got %d, want 0", result.Stats.MatchedWords)
	}
}

func TestParse_SkipLongSentences(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "long.tsv")
	// 501+ char English sentence.
	longSentence := ""
	for i := 0; i < 100; i++ {
		longSentence += "word "
	}
	longSentence += "cat." // total > 500 chars
	content := "1\t" + longSentence + "\t2\tДлинное предложение.\n"
	if err := writeFile(path, content); err != nil {
		t.Fatal(err)
	}

	knownWords := map[string]bool{"cat": true}
	result, err := Parse(path, knownWords, DefaultMaxPerWord)
	if err != nil {
		t.Fatalf("Parse returned error: %v", err)
	}

	if result.Stats.SkippedLong != 1 {
		t.Errorf("SkippedLong: got %d, want 1", result.Stats.SkippedLong)
	}
	if _, ok := result.Sentences["cat"]; ok {
		t.Error("cat should not match in a sentence that exceeds 500 chars")
	}
}

// --- ToDomainExamples tests ---

func TestToDomainExamples(t *testing.T) {
	path := testdataPath(t, "sample.tsv")

	knownWords := map[string]bool{
		"cat":   true,
		"house": true,
		"hello": true,
	}

	result, err := Parse(path, knownWords, 2)
	if err != nil {
		t.Fatalf("Parse returned error: %v", err)
	}

	catEntryID := uuid.New()
	houseEntryID := uuid.New()
	helloEntryID := uuid.New()

	catSenseID := uuid.New()
	houseSenseID := uuid.New()
	helloSenseID := uuid.New()

	entryIDMap := map[string]uuid.UUID{
		"cat":   catEntryID,
		"house": houseEntryID,
		"hello": helloEntryID,
	}

	senseIDMap := map[uuid.UUID]uuid.UUID{
		catEntryID:   catSenseID,
		houseEntryID: houseSenseID,
		helloEntryID: helloSenseID,
	}

	examples := result.ToDomainExamples(entryIDMap, senseIDMap)

	// cat: 2 (maxPerWord=2), house: 1, hello: 1 = 4 total.
	if len(examples) != 4 {
		t.Fatalf("expected 4 examples, got %d", len(examples))
	}

	// Verify all have non-zero IDs.
	for i, ex := range examples {
		if ex.ID == uuid.Nil {
			t.Errorf("Example[%d] ID should be non-zero", i)
		}
	}

	// Verify source_slug.
	for i, ex := range examples {
		if ex.SourceSlug != "tatoeba" {
			t.Errorf("Example[%d] SourceSlug: got %q, want %q", i, ex.SourceSlug, "tatoeba")
		}
	}

	// Verify position starts at 1000.
	for i, ex := range examples {
		if ex.Position < 1000 {
			t.Errorf("Example[%d] Position: got %d, want >= 1000", i, ex.Position)
		}
	}

	// Verify translations are non-nil.
	for i, ex := range examples {
		if ex.Translation == nil {
			t.Errorf("Example[%d] Translation should not be nil", i)
		}
	}

	// Verify sense ID mapping.
	for _, ex := range examples {
		switch ex.RefSenseID {
		case catSenseID, houseSenseID, helloSenseID:
			// ok
		default:
			t.Errorf("unexpected RefSenseID: %s", ex.RefSenseID)
		}
	}
}

func TestToDomainExamples_EmptyEntryIDMap(t *testing.T) {
	result := ParseResult{
		Sentences: map[string][]SentencePair{
			"cat": {{English: "A cat.", Russian: "Кот."}},
		},
	}

	examples := result.ToDomainExamples(map[string]uuid.UUID{}, map[uuid.UUID]uuid.UUID{})
	if len(examples) != 0 {
		t.Errorf("expected 0 examples for empty entryIDMap, got %d", len(examples))
	}
}

func TestToDomainExamples_NilMaps(t *testing.T) {
	result := ParseResult{
		Sentences: map[string][]SentencePair{
			"cat": {{English: "A cat.", Russian: "Кот."}},
		},
	}

	examples := result.ToDomainExamples(nil, nil)
	if len(examples) != 0 {
		t.Errorf("expected 0 examples for nil maps, got %d", len(examples))
	}
}

func TestToDomainExamples_MissingSenseMapping(t *testing.T) {
	result := ParseResult{
		Sentences: map[string][]SentencePair{
			"cat":   {{English: "A cat.", Russian: "Кот."}},
			"house": {{English: "A house.", Russian: "Дом."}},
		},
	}

	catEntryID := uuid.New()
	houseEntryID := uuid.New()
	catSenseID := uuid.New()

	entryIDMap := map[string]uuid.UUID{
		"cat":   catEntryID,
		"house": houseEntryID,
	}
	// house has no sense mapping — should be skipped.
	senseIDMap := map[uuid.UUID]uuid.UUID{
		catEntryID: catSenseID,
	}

	examples := result.ToDomainExamples(entryIDMap, senseIDMap)
	if len(examples) != 1 {
		t.Fatalf("expected 1 example (house skipped due to missing sense), got %d", len(examples))
	}
	if examples[0].RefSenseID != catSenseID {
		t.Errorf("expected RefSenseID=%s, got %s", catSenseID, examples[0].RefSenseID)
	}
}

func TestToDomainExamples_EmptyParseResult(t *testing.T) {
	result := ParseResult{
		Sentences: map[string][]SentencePair{},
	}

	examples := result.ToDomainExamples(
		map[string]uuid.UUID{"cat": uuid.New()},
		map[uuid.UUID]uuid.UUID{uuid.New(): uuid.New()},
	)
	if len(examples) != 0 {
		t.Errorf("expected 0 examples for empty parse result, got %d", len(examples))
	}
}
