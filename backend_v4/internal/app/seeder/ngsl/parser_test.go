package ngsl

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/heartmarshall/myenglish-backend/internal/domain"
)

func testdataPath(t *testing.T, name string) string {
	t.Helper()
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("cannot determine test file path")
	}
	return filepath.Join(filepath.Dir(file), "testdata", name)
}

// --- CEFR mapping ---

func TestCefrForRank(t *testing.T) {
	tests := []struct {
		name string
		rank int
		want string
	}{
		{"rank 1 → A1", 1, "A1"},
		{"rank 250 → A1", 250, "A1"},
		{"rank 500 → A1 (boundary)", 500, "A1"},
		{"rank 501 → A2", 501, "A2"},
		{"rank 800 → A2", 800, "A2"},
		{"rank 1200 → A2 (boundary)", 1200, "A2"},
		{"rank 1201 → B1", 1201, "B1"},
		{"rank 1500 → B1", 1500, "B1"},
		{"rank 2000 → B1 (boundary)", 2000, "B1"},
		{"rank 2001 → B2", 2001, "B2"},
		{"rank 2809 → B2", 2809, "B2"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := cefrForRank(tt.rank)
			if got != tt.want {
				t.Errorf("cefrForRank(%d) = %q, want %q", tt.rank, got, tt.want)
			}
		})
	}
}

// --- NGSL parsing ---

func TestParseNGSL(t *testing.T) {
	f, err := os.Open(testdataPath(t, "ngsl_sample.csv"))
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()

	entries, err := parseNGSL(f)
	if err != nil {
		t.Fatalf("parseNGSL returned error: %v", err)
	}

	// Sample has 7 words.
	if len(entries) != 7 {
		t.Fatalf("expected 7 entries, got %d", len(entries))
	}

	// First entry: "the" at rank 1.
	first := entries[0]
	if first.TextNormalized != "the" {
		t.Errorf("first entry TextNormalized = %q, want %q", first.TextNormalized, "the")
	}
	if first.FrequencyRank == nil || *first.FrequencyRank != 1 {
		t.Errorf("first entry FrequencyRank = %v, want 1", first.FrequencyRank)
	}
	if first.CEFRLevel == nil || *first.CEFRLevel != "A1" {
		t.Errorf("first entry CEFRLevel = %v, want A1", first.CEFRLevel)
	}
	if first.IsCoreLexicon == nil || !*first.IsCoreLexicon {
		t.Error("first entry IsCoreLexicon should be true")
	}

	// Check normalization: "Be" → "be", "OF" → "of".
	if entries[1].TextNormalized != "be" {
		t.Errorf("entries[1].TextNormalized = %q, want %q", entries[1].TextNormalized, "be")
	}
	if entries[2].TextNormalized != "of" {
		t.Errorf("entries[2].TextNormalized = %q, want %q", entries[2].TextNormalized, "of")
	}

	// All entries should have FrequencyRank set (1-based sequential).
	for i, e := range entries {
		expectedRank := i + 1
		if e.FrequencyRank == nil {
			t.Errorf("entry %d (%s): FrequencyRank is nil, want %d", i, e.TextNormalized, expectedRank)
		} else if *e.FrequencyRank != expectedRank {
			t.Errorf("entry %d (%s): FrequencyRank = %d, want %d", i, e.TextNormalized, *e.FrequencyRank, expectedRank)
		}
	}

	// All entries should be core lexicon.
	for i, e := range entries {
		if e.IsCoreLexicon == nil || !*e.IsCoreLexicon {
			t.Errorf("entry %d (%s): IsCoreLexicon should be true", i, e.TextNormalized)
		}
	}
}

// --- NAWL parsing ---

func TestParseNAWL(t *testing.T) {
	f, err := os.Open(testdataPath(t, "nawl_sample.csv"))
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()

	entries, err := parseNAWL(f)
	if err != nil {
		t.Fatalf("parseNAWL returned error: %v", err)
	}

	// Sample has 3 words.
	if len(entries) != 3 {
		t.Fatalf("expected 3 entries, got %d", len(entries))
	}

	// All NAWL entries should have nil FrequencyRank and "C1" CEFR.
	for i, e := range entries {
		if e.FrequencyRank != nil {
			t.Errorf("NAWL entry %d (%s): FrequencyRank should be nil, got %d",
				i, e.TextNormalized, *e.FrequencyRank)
		}
		if e.CEFRLevel == nil || *e.CEFRLevel != "C1" {
			t.Errorf("NAWL entry %d (%s): CEFRLevel should be C1, got %v",
				i, e.TextNormalized, e.CEFRLevel)
		}
		if e.IsCoreLexicon == nil || !*e.IsCoreLexicon {
			t.Errorf("NAWL entry %d (%s): IsCoreLexicon should be true",
				i, e.TextNormalized)
		}
	}

	// Check normalization: "Abstract" → "abstract", "Analyze" → "analyze".
	if entries[0].TextNormalized != "abstract" {
		t.Errorf("entries[0].TextNormalized = %q, want %q", entries[0].TextNormalized, "abstract")
	}
	if entries[2].TextNormalized != "analyze" {
		t.Errorf("entries[2].TextNormalized = %q, want %q", entries[2].TextNormalized, "analyze")
	}
}

// --- Combined Parse ---

func TestParse(t *testing.T) {
	ngslPath := testdataPath(t, "ngsl_sample.csv")
	nawlPath := testdataPath(t, "nawl_sample.csv")

	updates, coreWords, err := Parse(ngslPath, nawlPath)
	if err != nil {
		t.Fatalf("Parse returned error: %v", err)
	}

	// 7 NGSL + 3 NAWL = 10 total.
	if len(updates) != 10 {
		t.Fatalf("expected 10 updates, got %d", len(updates))
	}

	// First 7 should be NGSL (with frequency ranks).
	for i := 0; i < 7; i++ {
		if updates[i].FrequencyRank == nil {
			t.Errorf("updates[%d]: NGSL entry should have FrequencyRank", i)
		}
	}

	// Last 3 should be NAWL (nil frequency rank, "C1").
	for i := 7; i < 10; i++ {
		if updates[i].FrequencyRank != nil {
			t.Errorf("updates[%d]: NAWL entry should have nil FrequencyRank", i)
		}
		if updates[i].CEFRLevel == nil || *updates[i].CEFRLevel != "C1" {
			t.Errorf("updates[%d]: NAWL entry should have C1 CEFR", i)
		}
	}

	// Core words map should contain all 10 words.
	if len(coreWords) != 10 {
		t.Errorf("coreWords: expected 10 entries, got %d", len(coreWords))
	}

	// Check specific words in the core set.
	expectedCoreWords := []string{"the", "be", "of", "and", "have", "to", "it", "abstract", "achieve", "analyze"}
	for _, w := range expectedCoreWords {
		if !coreWords[w] {
			t.Errorf("coreWords should contain %q", w)
		}
	}
}

func TestParse_FileNotFound(t *testing.T) {
	nawlPath := testdataPath(t, "nawl_sample.csv")

	// Missing NGSL file.
	_, _, err := Parse("/nonexistent/ngsl.csv", nawlPath)
	if err == nil {
		t.Error("Parse should return error for missing NGSL file")
	}

	// Missing NAWL file.
	ngslPath := testdataPath(t, "ngsl_sample.csv")
	_, _, err = Parse(ngslPath, "/nonexistent/nawl.csv")
	if err == nil {
		t.Error("Parse should return error for missing NAWL file")
	}
}

func TestParse_EmptyFiles(t *testing.T) {
	dir := t.TempDir()

	ngslFile := filepath.Join(dir, "empty_ngsl.csv")
	nawlFile := filepath.Join(dir, "empty_nawl.csv")

	if err := os.WriteFile(ngslFile, []byte("Headword\n"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(nawlFile, []byte("Headword\n"), 0644); err != nil {
		t.Fatal(err)
	}

	updates, coreWords, err := Parse(ngslFile, nawlFile)
	if err != nil {
		t.Fatalf("Parse should not error on empty files: %v", err)
	}
	if len(updates) != 0 {
		t.Errorf("expected 0 updates, got %d", len(updates))
	}
	if len(coreWords) != 0 {
		t.Errorf("expected 0 core words, got %d", len(coreWords))
	}
}

// --- Edge cases ---

func TestParseNGSL_EmptyLines(t *testing.T) {
	csv := "Headword\nhello\n\n  \nworld\n"
	r := strings.NewReader(csv)

	entries, err := parseNGSL(r)
	if err != nil {
		t.Fatalf("parseNGSL returned error: %v", err)
	}

	// Only "hello" (rank 1) and "world" (rank 2) — empty lines skipped.
	if len(entries) != 2 {
		t.Fatalf("expected 2 entries, got %d", len(entries))
	}

	if entries[0].TextNormalized != "hello" {
		t.Errorf("entries[0].TextNormalized = %q, want %q", entries[0].TextNormalized, "hello")
	}
	if entries[0].FrequencyRank == nil || *entries[0].FrequencyRank != 1 {
		t.Errorf("entries[0].FrequencyRank = %v, want 1", entries[0].FrequencyRank)
	}

	if entries[1].TextNormalized != "world" {
		t.Errorf("entries[1].TextNormalized = %q, want %q", entries[1].TextNormalized, "world")
	}
	if entries[1].FrequencyRank == nil || *entries[1].FrequencyRank != 2 {
		t.Errorf("entries[1].FrequencyRank = %v, want 2", entries[1].FrequencyRank)
	}
}

func TestParseNAWL_EmptyLines(t *testing.T) {
	csv := "Headword\nfoo\n\nbar\n"
	r := strings.NewReader(csv)

	entries, err := parseNAWL(r)
	if err != nil {
		t.Fatalf("parseNAWL returned error: %v", err)
	}

	if len(entries) != 2 {
		t.Fatalf("expected 2 entries, got %d", len(entries))
	}

	if entries[0].TextNormalized != "foo" {
		t.Errorf("entries[0].TextNormalized = %q, want %q", entries[0].TextNormalized, "foo")
	}
	if entries[1].TextNormalized != "bar" {
		t.Errorf("entries[1].TextNormalized = %q, want %q", entries[1].TextNormalized, "bar")
	}
}

// Ensure domain.NormalizeText is used (whitespace trimming, lowercase, space compression).
func TestNormalizationApplied(t *testing.T) {
	csv := "Headword\n  HELLO  \n"
	r := strings.NewReader(csv)

	entries, err := parseNGSL(r)
	if err != nil {
		t.Fatal(err)
	}

	if len(entries) != 1 {
		t.Fatalf("expected 1 entry, got %d", len(entries))
	}

	want := domain.NormalizeText("  HELLO  ")
	if entries[0].TextNormalized != want {
		t.Errorf("TextNormalized = %q, want %q", entries[0].TextNormalized, want)
	}
}
