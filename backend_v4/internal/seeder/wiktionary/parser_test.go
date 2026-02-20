package wiktionary

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

func testdataPath(t *testing.T, name string) string {
	t.Helper()
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("cannot determine test file path")
	}
	return filepath.Join(filepath.Dir(file), "testdata", name)
}

// --- Pass 1: Scoring ---

func TestScoringPass(t *testing.T) {
	path := testdataPath(t, "sample.jsonl")
	coreWords := map[string]bool{"water": true}

	scores, stats, err := scoringPass(path, coreWords)
	if err != nil {
		t.Fatalf("scoringPass returned error: %v", err)
	}

	// Stats checks.
	if stats.TotalLines != 10 {
		t.Errorf("TotalLines: got %d, want 10", stats.TotalLines)
	}
	if stats.MalformedLines != 1 {
		t.Errorf("MalformedLines: got %d, want 1", stats.MalformedLines)
	}
	// English lines: run(verb) + run(noun) + house + beautiful + water + set(verb) + set(noun) + xyz = 8
	if stats.EnglishLines != 8 {
		t.Errorf("EnglishLines: got %d, want 8", stats.EnglishLines)
	}

	// Score map should contain normalized words.
	if _, ok := scores["run"]; !ok {
		t.Error("scores should contain 'run'")
	}
	if _, ok := scores["house"]; !ok {
		t.Error("scores should contain 'house'")
	}
	if _, ok := scores["set"]; !ok {
		t.Error("scores should contain 'set'")
	}

	// "maison" (French) should be absent.
	if _, ok := scores["maison"]; ok {
		t.Error("scores should NOT contain 'maison' (non-English)")
	}

	// "run" appears on two lines, score should be cumulative.
	if scores["run"] <= scores["xyz"] {
		t.Errorf("run (multi-POS) should score higher than xyz; run=%f xyz=%f",
			scores["run"], scores["xyz"])
	}

	// Core word "water" should have a large bonus.
	if scores["water"] <= scores["beautiful"] {
		t.Errorf("water (core word) should score higher than beautiful; water=%f beautiful=%f",
			scores["water"], scores["beautiful"])
	}
}

// --- Top N selection ---

func TestSelectTopN(t *testing.T) {
	scores := map[string]float64{
		"alpha":   10.0,
		"beta":    5.0,
		"gamma":   3.0,
		"delta":   1.0,
		"epsilon": 0.5,
	}
	coreWords := map[string]bool{"delta": true}

	// Request top 3. "delta" is a core word, so guaranteed in.
	topN := selectTopN(scores, coreWords, 3)

	if !topN["alpha"] {
		t.Error("alpha (highest score) should be in top N")
	}
	if !topN["delta"] {
		t.Error("delta (core word) should be in top N regardless of score")
	}
	if len(topN) != 3 {
		t.Errorf("selectTopN should return 3 entries, got %d", len(topN))
	}
	// epsilon (lowest) should not be in top N.
	if topN["epsilon"] {
		t.Error("epsilon (lowest score, not core) should NOT be in top N")
	}
}

func TestSelectTopN_AllCoreWords(t *testing.T) {
	scores := map[string]float64{
		"a": 1.0,
		"b": 2.0,
		"c": 3.0,
	}
	// All core words, request top 2 → all 3 should be included.
	coreWords := map[string]bool{"a": true, "b": true, "c": true}
	topN := selectTopN(scores, coreWords, 2)
	if len(topN) != 3 {
		t.Errorf("all core words should be included even if > topN; got %d", len(topN))
	}
}

func TestSelectTopN_EmptyScores(t *testing.T) {
	topN := selectTopN(map[string]float64{}, nil, 10)
	if len(topN) != 0 {
		t.Errorf("empty scores should return empty set; got %d", len(topN))
	}
}

// --- Pass 2: Full Parse ---

func TestParsingPass(t *testing.T) {
	path := testdataPath(t, "sample.jsonl")
	selected := map[string]bool{"run": true, "house": true}

	entries, err := parsingPass(path, selected)
	if err != nil {
		t.Fatalf("parsingPass returned error: %v", err)
	}

	if len(entries) != 2 {
		t.Fatalf("expected 2 entries, got %d", len(entries))
	}

	// Find "run" entry.
	var runEntry *ParsedEntry
	var houseEntry *ParsedEntry
	for i := range entries {
		switch entries[i].Word {
		case "run":
			runEntry = &entries[i]
		case "house":
			houseEntry = &entries[i]
		}
	}

	if runEntry == nil {
		t.Fatal("run entry not found")
	}
	if houseEntry == nil {
		t.Fatal("house entry not found")
	}

	// "run" should have 2 POS groups (verb + noun) merged from 2 lines.
	if len(runEntry.POSGroups) != 2 {
		t.Errorf("run: expected 2 POS groups, got %d", len(runEntry.POSGroups))
	}

	// Check POS group POS values.
	posSet := map[string]bool{}
	for _, pg := range runEntry.POSGroups {
		posSet[pg.POS] = true
	}
	if !posSet["verb"] || !posSet["noun"] {
		t.Errorf("run: expected verb and noun POS groups, got %v", posSet)
	}

	// "run" verb should have 1 sense with glosses, examples, translations.
	var verbGroup *POSGroup
	for i := range runEntry.POSGroups {
		if runEntry.POSGroups[i].POS == "verb" {
			verbGroup = &runEntry.POSGroups[i]
			break
		}
	}
	if verbGroup == nil {
		t.Fatal("run: verb POS group not found")
	}
	if len(verbGroup.Senses) != 1 {
		t.Fatalf("run verb: expected 1 sense, got %d", len(verbGroup.Senses))
	}
	sense := verbGroup.Senses[0]
	if len(sense.Glosses) != 1 {
		t.Errorf("run verb sense: expected 1 gloss, got %d", len(sense.Glosses))
	}
	// Example should have markup stripped.
	if len(sense.Examples) != 1 {
		t.Errorf("run verb sense: expected 1 example, got %d", len(sense.Examples))
	}
	// Russian translations should be deduplicated.
	if len(sense.Translations) != 2 {
		t.Errorf("run verb sense: expected 2 Russian translations (бежать, бегать), got %d: %v",
			len(sense.Translations), sense.Translations)
	}

	// "run" sounds should be merged from both lines (dedup by IPA+Region).
	if len(runEntry.Sounds) < 2 {
		t.Errorf("run: expected at least 2 sounds, got %d", len(runEntry.Sounds))
	}

	// Check IPA region extraction.
	hasUS := false
	hasUK := false
	for _, s := range runEntry.Sounds {
		if s.Region == "US" {
			hasUS = true
		}
		if s.Region == "UK" {
			hasUK = true
		}
	}
	if !hasUS || !hasUK {
		t.Errorf("run: expected US and UK regions; sounds=%+v", runEntry.Sounds)
	}

	// "house" should have 1 POS group and markup-stripped definition.
	if len(houseEntry.POSGroups) != 1 {
		t.Errorf("house: expected 1 POS group, got %d", len(houseEntry.POSGroups))
	}
	if len(houseEntry.POSGroups[0].Senses) != 1 {
		t.Errorf("house: expected 1 sense, got %d", len(houseEntry.POSGroups[0].Senses))
	}
	def := houseEntry.POSGroups[0].Senses[0].Glosses[0]
	// Should have HTML stripped (the <a> tag).
	if containsHTML(def) {
		t.Errorf("house definition still contains HTML: %s", def)
	}
}

// --- Integration: Full Parse function ---

func TestParse(t *testing.T) {
	path := testdataPath(t, "sample.jsonl")
	coreWords := map[string]bool{"water": true}

	entries, stats, err := Parse(path, coreWords, 5)
	if err != nil {
		t.Fatalf("Parse returned error: %v", err)
	}

	if stats.MalformedLines != 1 {
		t.Errorf("MalformedLines: got %d, want 1", stats.MalformedLines)
	}

	// With topN=5, all 6 unique English words should fit.
	// Unique words: run, house, beautiful, water, set, xyz = 6
	// topN=5 means top 5 by score (water gets core bonus).
	if len(entries) < 5 {
		t.Errorf("expected at least 5 entries, got %d", len(entries))
	}

	// "water" (core word) should definitely be in.
	found := false
	for _, e := range entries {
		if e.Word == "water" {
			found = true
			break
		}
	}
	if !found {
		t.Error("core word 'water' should be in parse results")
	}

	// All entries should have non-empty Word.
	for _, e := range entries {
		if e.Word == "" {
			t.Error("entry has empty Word")
		}
	}
}

func TestParse_FileNotFound(t *testing.T) {
	_, _, err := Parse("/nonexistent/file.jsonl", nil, 100)
	if err == nil {
		t.Error("Parse should return error for missing file")
	}
}

func TestParse_EmptyFile(t *testing.T) {
	f, err := os.CreateTemp(t.TempDir(), "empty-*.jsonl")
	if err != nil {
		t.Fatal(err)
	}
	f.Close()

	entries, stats, err := Parse(f.Name(), nil, 100)
	if err != nil {
		t.Fatalf("Parse should not error on empty file: %v", err)
	}
	if len(entries) != 0 {
		t.Errorf("expected 0 entries, got %d", len(entries))
	}
	if stats.TotalLines != 0 {
		t.Errorf("TotalLines: got %d, want 0", stats.TotalLines)
	}
}

// --- IPA region extraction ---

func TestExtractRegion(t *testing.T) {
	tests := []struct {
		name   string
		tags   []string
		want   string
	}{
		{"US tag", []string{"US"}, "US"},
		{"UK tag", []string{"UK"}, "UK"},
		{"General-American", []string{"General-American"}, "US"},
		{"Received-Pronunciation", []string{"Received-Pronunciation"}, "UK"},
		{"US in mixed tags", []string{"phoneme", "US", "standard"}, "US"},
		{"UK in mixed tags", []string{"UK", "phoneme"}, "UK"},
		{"no region tags", []string{"phoneme", "standard"}, ""},
		{"empty tags", []string{}, ""},
		{"nil tags", nil, ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := extractRegion(tt.tags)
			if got != tt.want {
				t.Errorf("extractRegion(%v) = %q, want %q", tt.tags, got, tt.want)
			}
		})
	}
}

// containsHTML checks if a string contains HTML tags.
func containsHTML(s string) bool {
	for i := 0; i < len(s); i++ {
		if s[i] == '<' {
			for j := i + 1; j < len(s); j++ {
				if s[j] == '>' {
					return true
				}
			}
		}
	}
	return false
}
