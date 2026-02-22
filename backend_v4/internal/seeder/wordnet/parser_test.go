package wordnet

import (
	"os"
	"path/filepath"
	"runtime"
	"sort"
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

// --- Parse: file handling ---

func TestParse_FileNotFound(t *testing.T) {
	_, err := Parse("/nonexistent/file.json", map[string]bool{"word": true})
	if err == nil {
		t.Error("Parse should return error for missing file")
	}
}

func TestParse_InvalidJSON(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "bad.json")
	if err := writeFile(path, "not json at all"); err != nil {
		t.Fatal(err)
	}

	_, err := Parse(path, map[string]bool{"word": true})
	if err == nil {
		t.Error("Parse should return error for invalid JSON")
	}
}

func TestParse_EmptyGraph(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "empty.json")
	content := `{"@context":"...","@graph":[]}`
	if err := writeFile(path, content); err != nil {
		t.Fatal(err)
	}

	result, err := Parse(path, map[string]bool{"word": true})
	if err != nil {
		t.Fatalf("Parse should not error on empty graph: %v", err)
	}
	if len(result.Relations) != 0 {
		t.Errorf("expected 0 relations, got %d", len(result.Relations))
	}
}

func TestParse_NilKnownWords(t *testing.T) {
	path := testdataPath(t, "sample.json")

	result, err := Parse(path, nil)
	if err != nil {
		t.Fatalf("Parse should not error with nil knownWords: %v", err)
	}
	if len(result.Relations) != 0 {
		t.Errorf("expected 0 relations with nil knownWords, got %d", len(result.Relations))
	}
}

func TestParse_EmptyKnownWords(t *testing.T) {
	path := testdataPath(t, "sample.json")

	result, err := Parse(path, map[string]bool{})
	if err != nil {
		t.Fatalf("Parse should not error with empty knownWords: %v", err)
	}
	if len(result.Relations) != 0 {
		t.Errorf("expected 0 relations with empty knownWords, got %d", len(result.Relations))
	}
}

// --- Parse: synonym extraction ---

func TestParse_Synonyms(t *testing.T) {
	path := testdataPath(t, "sample.json")

	// "happy", "glad", and "joyful" share synset ewn-01148283-a.
	// "xylophone" is also in that synset but not in knownWords.
	knownWords := map[string]bool{
		"happy":  true,
		"glad":   true,
		"joyful": true,
	}

	result, err := Parse(path, knownWords)
	if err != nil {
		t.Fatalf("Parse returned error: %v", err)
	}

	// Expect synonym pairs: (glad, happy), (glad, joyful), (happy, joyful)
	// Alphabetically ordered source < target.
	synonyms := filterRelations(result.Relations, "synonym")
	sort.Slice(synonyms, func(i, j int) bool {
		if synonyms[i].SourceWord == synonyms[j].SourceWord {
			return synonyms[i].TargetWord < synonyms[j].TargetWord
		}
		return synonyms[i].SourceWord < synonyms[j].SourceWord
	})

	expected := []Relation{
		{SourceWord: "glad", TargetWord: "happy", RelationType: "synonym"},
		{SourceWord: "glad", TargetWord: "joyful", RelationType: "synonym"},
		{SourceWord: "happy", TargetWord: "joyful", RelationType: "synonym"},
	}

	if len(synonyms) != len(expected) {
		t.Fatalf("synonyms: got %d, want %d; got: %+v", len(synonyms), len(expected), synonyms)
	}
	for i, got := range synonyms {
		want := expected[i]
		if got.SourceWord != want.SourceWord || got.TargetWord != want.TargetWord {
			t.Errorf("synonym[%d]: got (%q,%q), want (%q,%q)",
				i, got.SourceWord, got.TargetWord, want.SourceWord, want.TargetWord)
		}
	}
}

func TestParse_Synonyms_FilteredByKnownWords(t *testing.T) {
	path := testdataPath(t, "sample.json")

	// Only "happy" known; "glad", "joyful", "xylophone" not known.
	// No synonym pairs can be formed with just one known word.
	knownWords := map[string]bool{
		"happy": true,
	}

	result, err := Parse(path, knownWords)
	if err != nil {
		t.Fatalf("Parse returned error: %v", err)
	}

	synonyms := filterRelations(result.Relations, "synonym")
	if len(synonyms) != 0 {
		t.Errorf("expected 0 synonyms with only one known word, got %d: %+v", len(synonyms), synonyms)
	}
}

// --- Parse: antonym extraction ---

func TestParse_Antonyms(t *testing.T) {
	path := testdataPath(t, "sample.json")

	knownWords := map[string]bool{
		"happy": true,
		"sad":   true,
	}

	result, err := Parse(path, knownWords)
	if err != nil {
		t.Fatalf("Parse returned error: %v", err)
	}

	// happy↔sad antonym. Alphabetical: happy < sad.
	// Both sides declare antonym, but dedup should give us one pair.
	antonyms := filterRelations(result.Relations, "antonym")
	if len(antonyms) != 1 {
		t.Fatalf("antonyms: got %d, want 1; got: %+v", len(antonyms), antonyms)
	}
	if antonyms[0].SourceWord != "happy" || antonyms[0].TargetWord != "sad" {
		t.Errorf("antonym: got (%q,%q), want (happy,sad)",
			antonyms[0].SourceWord, antonyms[0].TargetWord)
	}
}

// --- Parse: hypernym extraction ---

func TestParse_Hypernyms(t *testing.T) {
	path := testdataPath(t, "sample.json")

	knownWords := map[string]bool{
		"dog":    true,
		"animal": true,
	}

	result, err := Parse(path, knownWords)
	if err != nil {
		t.Fatalf("Parse returned error: %v", err)
	}

	// dog's synset has hypernym pointing to animal's synset.
	// Direction: specific → general, so source=dog, target=animal.
	hypernyms := filterRelations(result.Relations, "hypernym")
	if len(hypernyms) != 1 {
		t.Fatalf("hypernyms: got %d, want 1; got: %+v", len(hypernyms), hypernyms)
	}
	if hypernyms[0].SourceWord != "dog" || hypernyms[0].TargetWord != "animal" {
		t.Errorf("hypernym: got (%q,%q), want (dog,animal)",
			hypernyms[0].SourceWord, hypernyms[0].TargetWord)
	}
}

// --- Parse: derived form extraction ---

func TestParse_DerivedForms(t *testing.T) {
	path := testdataPath(t, "sample.json")

	knownWords := map[string]bool{
		"happy":     true,
		"happiness": true,
	}

	result, err := Parse(path, knownWords)
	if err != nil {
		t.Fatalf("Parse returned error: %v", err)
	}

	// happy↔happiness derivation. Alphabetical ordering: happy < happiness.
	// Both sides declare derivation, but dedup should give us one pair.
	derived := filterRelations(result.Relations, "derived")
	if len(derived) != 1 {
		t.Fatalf("derived: got %d, want 1; got: %+v", len(derived), derived)
	}
	if derived[0].SourceWord != "happiness" || derived[0].TargetWord != "happy" {
		t.Errorf("derived: got (%q,%q), want (happiness,happy)",
			derived[0].SourceWord, derived[0].TargetWord)
	}
}

// --- Parse: full parse with stats ---

func TestParse_FullSample(t *testing.T) {
	path := testdataPath(t, "sample.json")

	knownWords := map[string]bool{
		"happy":     true,
		"glad":      true,
		"sad":       true,
		"happiness": true,
		"dog":       true,
		"animal":    true,
		"joyful":    true,
		// "xylophone" intentionally absent
	}

	result, err := Parse(path, knownWords)
	if err != nil {
		t.Fatalf("Parse returned error: %v", err)
	}

	// Stats checks.
	if result.Stats.TotalSynsets != 5 {
		t.Errorf("TotalSynsets: got %d, want 5", result.Stats.TotalSynsets)
	}
	if result.Stats.TotalEntries != 8 {
		t.Errorf("TotalEntries: got %d, want 8", result.Stats.TotalEntries)
	}

	// Verify we have relations of each type.
	synonyms := filterRelations(result.Relations, "synonym")
	antonyms := filterRelations(result.Relations, "antonym")
	hypernyms := filterRelations(result.Relations, "hypernym")
	derived := filterRelations(result.Relations, "derived")

	if len(synonyms) == 0 {
		t.Error("expected some synonyms")
	}
	if len(antonyms) == 0 {
		t.Error("expected some antonyms")
	}
	if len(hypernyms) == 0 {
		t.Error("expected some hypernyms")
	}
	if len(derived) == 0 {
		t.Error("expected some derived relations")
	}

	// Stats: filtered by known should count xylophone being excluded from synonym pairs.
	if result.Stats.FilteredByKnown == 0 {
		t.Error("expected FilteredByKnown > 0 (xylophone should be filtered)")
	}

	// Stats: duplicates should be > 0 because antonym and derivation are declared on both sides.
	if result.Stats.Duplicates == 0 {
		t.Error("expected Duplicates > 0 (bidirectional antonym/derivation)")
	}

	// Total relations should match the sum of all types.
	if result.Stats.TotalRelations != len(result.Relations) {
		t.Errorf("TotalRelations stat (%d) does not match actual relations count (%d)",
			result.Stats.TotalRelations, len(result.Relations))
	}
}

// --- Parse: self-referential filtering ---

func TestParse_SelfReferential(t *testing.T) {
	// Create a JSON with a sense that has a derivation pointing to itself.
	dir := t.TempDir()
	path := filepath.Join(dir, "self_ref.json")
	content := `{
		"@context": "...",
		"@graph": [{
			"@id": "ewn",
			"entry": [
				{
					"@id": "ewn-run-v",
					"lemma": {"writtenForm": "run"},
					"sense": [
						{
							"@id": "ewn-run-v-01",
							"synset": "ewn-99999999-v",
							"relations": [
								{"relType": "derivation", "target": "ewn-run-v-01"}
							]
						}
					]
				}
			],
			"synset": [
				{
					"@id": "ewn-99999999-v",
					"relations": []
				}
			]
		}]
	}`
	if err := writeFile(path, content); err != nil {
		t.Fatal(err)
	}

	knownWords := map[string]bool{"run": true}
	result, err := Parse(path, knownWords)
	if err != nil {
		t.Fatalf("Parse returned error: %v", err)
	}

	if len(result.Relations) != 0 {
		t.Errorf("expected 0 relations (self-referential should be skipped), got %d: %+v",
			len(result.Relations), result.Relations)
	}
	if result.Stats.SelfReferential != 1 {
		t.Errorf("SelfReferential stat: got %d, want 1", result.Stats.SelfReferential)
	}
}

// --- Parse: normalization ---

func TestParse_Normalization(t *testing.T) {
	// Words with different casing should be normalized.
	dir := t.TempDir()
	path := filepath.Join(dir, "casing.json")
	content := `{
		"@context": "...",
		"@graph": [{
			"@id": "ewn",
			"entry": [
				{
					"@id": "ewn-Hello-n",
					"lemma": {"writtenForm": "Hello"},
					"sense": [
						{
							"@id": "ewn-Hello-n-01",
							"synset": "ewn-88888888-n",
							"relations": []
						}
					]
				},
				{
					"@id": "ewn-World-n",
					"lemma": {"writtenForm": "  World  "},
					"sense": [
						{
							"@id": "ewn-World-n-01",
							"synset": "ewn-88888888-n",
							"relations": []
						}
					]
				}
			],
			"synset": [
				{
					"@id": "ewn-88888888-n",
					"relations": []
				}
			]
		}]
	}`
	if err := writeFile(path, content); err != nil {
		t.Fatal(err)
	}

	knownWords := map[string]bool{
		"hello": true,
		"world": true,
	}

	result, err := Parse(path, knownWords)
	if err != nil {
		t.Fatalf("Parse returned error: %v", err)
	}

	synonyms := filterRelations(result.Relations, "synonym")
	if len(synonyms) != 1 {
		t.Fatalf("synonyms: got %d, want 1", len(synonyms))
	}
	if synonyms[0].SourceWord != "hello" || synonyms[0].TargetWord != "world" {
		t.Errorf("synonym: got (%q,%q), want (hello,world)",
			synonyms[0].SourceWord, synonyms[0].TargetWord)
	}
}

// --- Parse: deduplication ---

func TestParse_Deduplication(t *testing.T) {
	path := testdataPath(t, "sample.json")

	// happy and sad both declare antonym to each other.
	// After dedup, we should get exactly 1 antonym pair.
	knownWords := map[string]bool{
		"happy": true,
		"sad":   true,
	}

	result, err := Parse(path, knownWords)
	if err != nil {
		t.Fatalf("Parse returned error: %v", err)
	}

	antonyms := filterRelations(result.Relations, "antonym")
	if len(antonyms) != 1 {
		t.Errorf("expected exactly 1 antonym after dedup, got %d: %+v", len(antonyms), antonyms)
	}
}

// --- ToDomainRelations ---

func TestToDomainRelations(t *testing.T) {
	path := testdataPath(t, "sample.json")

	knownWords := map[string]bool{
		"happy":     true,
		"glad":      true,
		"sad":       true,
		"happiness": true,
		"dog":       true,
		"animal":    true,
		"joyful":    true,
	}

	result, err := Parse(path, knownWords)
	if err != nil {
		t.Fatalf("Parse returned error: %v", err)
	}

	happyID := uuid.New()
	gladID := uuid.New()
	sadID := uuid.New()
	happinessID := uuid.New()
	dogID := uuid.New()
	animalID := uuid.New()
	joyfulID := uuid.New()

	entryIDMap := map[string]uuid.UUID{
		"happy":     happyID,
		"glad":      gladID,
		"sad":       sadID,
		"happiness": happinessID,
		"dog":       dogID,
		"animal":    animalID,
		"joyful":    joyfulID,
	}

	domainRels := result.ToDomainRelations(entryIDMap)

	if len(domainRels) != len(result.Relations) {
		t.Fatalf("domain relations count: got %d, want %d", len(domainRels), len(result.Relations))
	}

	// Verify all have non-zero IDs and correct source_slug.
	for i, r := range domainRels {
		if r.ID == uuid.Nil {
			t.Errorf("DomainRelation[%d] ID should be non-zero", i)
		}
		if r.SourceSlug != "wordnet" {
			t.Errorf("DomainRelation[%d] SourceSlug: got %q, want %q", i, r.SourceSlug, "wordnet")
		}
		if r.SourceEntryID == uuid.Nil {
			t.Errorf("DomainRelation[%d] SourceEntryID should be non-zero", i)
		}
		if r.TargetEntryID == uuid.Nil {
			t.Errorf("DomainRelation[%d] TargetEntryID should be non-zero", i)
		}
	}

	// Verify relation types are valid.
	validTypes := map[string]bool{"synonym": true, "hypernym": true, "antonym": true, "derived": true}
	for i, r := range domainRels {
		if !validTypes[r.RelationType] {
			t.Errorf("DomainRelation[%d] RelationType: got %q, want one of %v", i, r.RelationType, validTypes)
		}
	}
}

func TestToDomainRelations_EmptyEntryIDMap(t *testing.T) {
	result := ParseResult{
		Relations: []Relation{
			{SourceWord: "happy", TargetWord: "glad", RelationType: "synonym"},
		},
	}

	domainRels := result.ToDomainRelations(map[string]uuid.UUID{})
	if len(domainRels) != 0 {
		t.Errorf("expected 0 domain relations for empty entryIDMap, got %d", len(domainRels))
	}
}

func TestToDomainRelations_NilEntryIDMap(t *testing.T) {
	result := ParseResult{
		Relations: []Relation{
			{SourceWord: "happy", TargetWord: "glad", RelationType: "synonym"},
		},
	}

	domainRels := result.ToDomainRelations(nil)
	if len(domainRels) != 0 {
		t.Errorf("expected 0 domain relations for nil entryIDMap, got %d", len(domainRels))
	}
}

func TestToDomainRelations_PartialEntryIDMap(t *testing.T) {
	result := ParseResult{
		Relations: []Relation{
			{SourceWord: "glad", TargetWord: "happy", RelationType: "synonym"},
			{SourceWord: "happy", TargetWord: "sad", RelationType: "antonym"},
		},
	}

	happyID := uuid.New()
	gladID := uuid.New()

	entryIDMap := map[string]uuid.UUID{
		"happy": happyID,
		"glad":  gladID,
		// "sad" intentionally absent
	}

	domainRels := result.ToDomainRelations(entryIDMap)
	// Only the synonym should be included (both words have IDs).
	if len(domainRels) != 1 {
		t.Fatalf("expected 1 domain relation, got %d", len(domainRels))
	}
	if domainRels[0].RelationType != "synonym" {
		t.Errorf("expected synonym, got %q", domainRels[0].RelationType)
	}
	if domainRels[0].SourceEntryID != gladID {
		t.Errorf("SourceEntryID: got %s, want %s", domainRels[0].SourceEntryID, gladID)
	}
	if domainRels[0].TargetEntryID != happyID {
		t.Errorf("TargetEntryID: got %s, want %s", domainRels[0].TargetEntryID, happyID)
	}
}

func TestToDomainRelations_EmptyParseResult(t *testing.T) {
	result := ParseResult{
		Relations: nil,
	}

	domainRels := result.ToDomainRelations(map[string]uuid.UUID{
		"happy": uuid.New(),
	})
	if len(domainRels) != 0 {
		t.Errorf("expected 0 domain relations for empty parse result, got %d", len(domainRels))
	}
}

// --- Helper ---

func filterRelations(relations []Relation, relType string) []Relation {
	var result []Relation
	for _, r := range relations {
		if r.RelationType == relType {
			result = append(result, r)
		}
	}
	return result
}
