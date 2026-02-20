package cmu

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

// --- ARPAbet to IPA conversion ---

func TestArpabetToIPA(t *testing.T) {
	tests := []struct {
		name    string
		phoneme string
		want    string
	}{
		{"consonant B", "B", "b"},
		{"consonant CH", "CH", "t\u0283"},
		{"consonant D", "D", "d"},
		{"consonant DH", "DH", "\u00f0"},
		{"consonant F", "F", "f"},
		{"consonant G", "G", "\u0261"},
		{"consonant HH", "HH", "h"},
		{"consonant JH", "JH", "d\u0292"},
		{"consonant K", "K", "k"},
		{"consonant L", "L", "l"},
		{"consonant M", "M", "m"},
		{"consonant N", "N", "n"},
		{"consonant NG", "NG", "\u014b"},
		{"consonant P", "P", "p"},
		{"consonant R", "R", "\u0279"},
		{"consonant S", "S", "s"},
		{"consonant SH", "SH", "\u0283"},
		{"consonant T", "T", "t"},
		{"consonant TH", "TH", "\u03b8"},
		{"consonant V", "V", "v"},
		{"consonant W", "W", "w"},
		{"consonant Y", "Y", "j"},
		{"consonant Z", "Z", "z"},
		{"consonant ZH", "ZH", "\u0292"},
		{"vowel AA", "AA", "\u0251"},
		{"vowel AE", "AE", "\u00e6"},
		{"vowel AH", "AH", "\u028c"},
		{"vowel AO", "AO", "\u0254"},
		{"vowel AW", "AW", "a\u028a"},
		{"vowel AY", "AY", "a\u026a"},
		{"vowel EH", "EH", "\u025b"},
		{"vowel ER", "ER", "\u025d"},
		{"vowel EY", "EY", "e\u026a"},
		{"vowel IH", "IH", "\u026a"},
		{"vowel IY", "IY", "i"},
		{"vowel OW", "OW", "o\u028a"},
		{"vowel OY", "OY", "\u0254\u026a"},
		{"vowel UH", "UH", "\u028a"},
		{"vowel UW", "UW", "u"},
		{"unknown phoneme", "XX", ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, ok := arpabetToIPA(tt.phoneme)
			if tt.want == "" {
				if ok {
					t.Errorf("arpabetToIPA(%q) should return ok=false for unknown phoneme", tt.phoneme)
				}
				return
			}
			if !ok {
				t.Errorf("arpabetToIPA(%q) returned ok=false, want ok=true", tt.phoneme)
				return
			}
			if got != tt.want {
				t.Errorf("arpabetToIPA(%q) = %q, want %q", tt.phoneme, got, tt.want)
			}
		})
	}
}

// --- Stress marker stripping ---

func TestStripStress(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"AH0", "AH"},
		{"AW1", "AW"},
		{"IY2", "IY"},
		{"HH", "HH"},  // no stress marker
		{"S", "S"},     // single char, no stress
		{"ER1", "ER"},
		{"OW0", "OW"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := stripStress(tt.input)
			if got != tt.want {
				t.Errorf("stripStress(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

// --- Full line parsing ---

func TestParseLine(t *testing.T) {
	tests := []struct {
		name         string
		line         string
		wantWord     string
		wantIPA      string
		wantVariant  int
		wantSkip     bool
	}{
		{
			name:     "simple word",
			line:     "HELLO  HH AH0 L OW1",
			wantWord: "hello",
			wantIPA:  "/h\u028clo\u028a/",
			wantVariant: 0,
		},
		{
			name:     "variant 2",
			line:     "HOUSE(2)  HH AW1 Z",
			wantWord: "house",
			wantIPA:  "/ha\u028az/",
			wantVariant: 1,
		},
		{
			name:     "variant 3",
			line:     "THE(3)  DH IY0",
			wantWord: "the",
			wantIPA:  "/\u00f0i/",
			wantVariant: 2,
		},
		{
			name:     "comment line",
			line:     ";;; This is a comment",
			wantSkip: true,
		},
		{
			name:     "empty line",
			line:     "",
			wantSkip: true,
		},
		{
			name:     "word with stress 2",
			line:     "CAT  K AE1 T",
			wantWord: "cat",
			wantIPA:  "/k\u00e6t/",
			wantVariant: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			word, ipa, err := parseLine(tt.line)
			if tt.wantSkip {
				if err != errSkipLine {
					t.Errorf("parseLine(%q) should return errSkipLine, got err=%v", tt.line, err)
				}
				return
			}
			if err != nil {
				t.Fatalf("parseLine(%q) returned unexpected error: %v", tt.line, err)
			}
			if word != tt.wantWord {
				t.Errorf("word: got %q, want %q", word, tt.wantWord)
			}
			if ipa.IPA != tt.wantIPA {
				t.Errorf("IPA: got %q, want %q", ipa.IPA, tt.wantIPA)
			}
			if ipa.VariantIndex != tt.wantVariant {
				t.Errorf("VariantIndex: got %d, want %d", ipa.VariantIndex, tt.wantVariant)
			}
		})
	}
}

// --- phonemesToIPA ---

func TestPhonemesToIPA(t *testing.T) {
	tests := []struct {
		name     string
		phonemes []string
		want     string
	}{
		{
			name:     "HOUSE",
			phonemes: []string{"HH", "AW1", "S"},
			want:     "/ha\u028as/",
		},
		{
			name:     "WORLD",
			phonemes: []string{"W", "ER1", "L", "D"},
			want:     "/w\u025dld/",
		},
		{
			name:     "THE",
			phonemes: []string{"DH", "AH0"},
			want:     "/\u00f0\u028c/",
		},
		{
			name:     "READ variant 1",
			phonemes: []string{"R", "IY1", "D"},
			want:     "/\u0279id/",
		},
		{
			name:     "READ variant 2",
			phonemes: []string{"R", "EH1", "D"},
			want:     "/\u0279\u025bd/",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := phonemesToIPA(tt.phonemes)
			if got != tt.want {
				t.Errorf("phonemesToIPA(%v) = %q, want %q", tt.phonemes, got, tt.want)
			}
		})
	}
}

// --- parseWordAndVariant ---

func TestParseWordAndVariant(t *testing.T) {
	tests := []struct {
		name        string
		raw         string
		wantWord    string
		wantVariant int
	}{
		{"no variant", "HELLO", "hello", 0},
		{"variant 2", "HOUSE(2)", "house", 1},
		{"variant 3", "THE(3)", "the", 2},
		{"variant 10", "WORD(10)", "word", 9},
		{"lowercase preserved", "hello", "hello", 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			word, variant := parseWordAndVariant(tt.raw)
			if word != tt.wantWord {
				t.Errorf("word: got %q, want %q", word, tt.wantWord)
			}
			if variant != tt.wantVariant {
				t.Errorf("variant: got %d, want %d", variant, tt.wantVariant)
			}
		})
	}
}

// --- Full file parsing ---

func TestParse(t *testing.T) {
	path := testdataPath(t, "sample.dict")

	result, err := Parse(path)
	if err != nil {
		t.Fatalf("Parse returned error: %v", err)
	}

	// Stats checks.
	if result.Stats.TotalLines != 12 {
		t.Errorf("TotalLines: got %d, want 12", result.Stats.TotalLines)
	}
	if result.Stats.CommentLines != 2 {
		t.Errorf("CommentLines: got %d, want 2", result.Stats.CommentLines)
	}
	if result.Stats.ParsedLines != 10 {
		t.Errorf("ParsedLines: got %d, want 10", result.Stats.ParsedLines)
	}
	// Unique words: hello, house, read, the, world, cat = 6.
	if result.Stats.UniqueWords != 6 {
		t.Errorf("UniqueWords: got %d, want 6", result.Stats.UniqueWords)
	}

	// Check pronunciations map size.
	if len(result.Pronunciations) != 6 {
		t.Errorf("Pronunciations map size: got %d, want 6", len(result.Pronunciations))
	}

	// HELLO: single pronunciation.
	hello, ok := result.Pronunciations["hello"]
	if !ok {
		t.Fatal("expected 'hello' in pronunciations")
	}
	if len(hello) != 1 {
		t.Errorf("hello: expected 1 pronunciation, got %d", len(hello))
	}
	if hello[0].IPA != "/h\u028clo\u028a/" {
		t.Errorf("hello IPA: got %q, want %q", hello[0].IPA, "/h\u028clo\u028a/")
	}
	if hello[0].VariantIndex != 0 {
		t.Errorf("hello VariantIndex: got %d, want 0", hello[0].VariantIndex)
	}

	// HOUSE: two variants.
	house, ok := result.Pronunciations["house"]
	if !ok {
		t.Fatal("expected 'house' in pronunciations")
	}
	if len(house) != 2 {
		t.Errorf("house: expected 2 pronunciations, got %d", len(house))
	}
	if house[0].VariantIndex != 0 {
		t.Errorf("house[0] VariantIndex: got %d, want 0", house[0].VariantIndex)
	}
	if house[1].VariantIndex != 1 {
		t.Errorf("house[1] VariantIndex: got %d, want 1", house[1].VariantIndex)
	}

	// READ: two variants with different IPA.
	read, ok := result.Pronunciations["read"]
	if !ok {
		t.Fatal("expected 'read' in pronunciations")
	}
	if len(read) != 2 {
		t.Errorf("read: expected 2 pronunciations, got %d", len(read))
	}
	if read[0].IPA != "/\u0279id/" {
		t.Errorf("read[0] IPA: got %q, want %q", read[0].IPA, "/\u0279id/")
	}
	if read[1].IPA != "/\u0279\u025bd/" {
		t.Errorf("read[1] IPA: got %q, want %q", read[1].IPA, "/\u0279\u025bd/")
	}

	// THE: three variants.
	the, ok := result.Pronunciations["the"]
	if !ok {
		t.Fatal("expected 'the' in pronunciations")
	}
	if len(the) != 3 {
		t.Errorf("the: expected 3 pronunciations, got %d", len(the))
	}
	if the[2].VariantIndex != 2 {
		t.Errorf("the[2] VariantIndex: got %d, want 2", the[2].VariantIndex)
	}

	// WORLD: single pronunciation.
	world, ok := result.Pronunciations["world"]
	if !ok {
		t.Fatal("expected 'world' in pronunciations")
	}
	if len(world) != 1 {
		t.Errorf("world: expected 1 pronunciation, got %d", len(world))
	}
	if world[0].IPA != "/w\u025dld/" {
		t.Errorf("world IPA: got %q, want %q", world[0].IPA, "/w\u025dld/")
	}

	// CAT: single pronunciation.
	cat, ok := result.Pronunciations["cat"]
	if !ok {
		t.Fatal("expected 'cat' in pronunciations")
	}
	if len(cat) != 1 {
		t.Errorf("cat: expected 1 pronunciation, got %d", len(cat))
	}
	if cat[0].IPA != "/k\u00e6t/" {
		t.Errorf("cat IPA: got %q, want %q", cat[0].IPA, "/k\u00e6t/")
	}
}

func TestParse_FileNotFound(t *testing.T) {
	_, err := Parse("/nonexistent/file.dict")
	if err == nil {
		t.Error("Parse should return error for missing file")
	}
}

func TestParse_EmptyFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "empty.dict")
	if err := writeFile(path, ""); err != nil {
		t.Fatal(err)
	}

	result, err := Parse(path)
	if err != nil {
		t.Fatalf("Parse should not error on empty file: %v", err)
	}
	if len(result.Pronunciations) != 0 {
		t.Errorf("expected 0 pronunciations, got %d", len(result.Pronunciations))
	}
	if result.Stats.TotalLines != 0 {
		t.Errorf("TotalLines: got %d, want 0", result.Stats.TotalLines)
	}
}

func TestParse_OnlyComments(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "comments.dict")
	if err := writeFile(path, ";;; comment 1\n;;; comment 2\n"); err != nil {
		t.Fatal(err)
	}

	result, err := Parse(path)
	if err != nil {
		t.Fatalf("Parse should not error on comments-only file: %v", err)
	}
	if len(result.Pronunciations) != 0 {
		t.Errorf("expected 0 pronunciations, got %d", len(result.Pronunciations))
	}
	if result.Stats.TotalLines != 2 {
		t.Errorf("TotalLines: got %d, want 2", result.Stats.TotalLines)
	}
	if result.Stats.CommentLines != 2 {
		t.Errorf("CommentLines: got %d, want 2", result.Stats.CommentLines)
	}
}

// --- ToDomainPronunciations ---

func TestToDomainPronunciations(t *testing.T) {
	path := testdataPath(t, "sample.dict")

	result, err := Parse(path)
	if err != nil {
		t.Fatalf("Parse returned error: %v", err)
	}

	helloID := uuid.New()
	houseID := uuid.New()
	readID := uuid.New()

	entryIDMap := map[string]uuid.UUID{
		"hello": helloID,
		"house": houseID,
		"read":  readID,
		// "the", "world", "cat" intentionally absent
	}

	pronunciations := result.ToDomainPronunciations(entryIDMap)

	// hello(1) + house(2) + read(2) = 5
	if len(pronunciations) != 5 {
		t.Fatalf("expected 5 pronunciations, got %d", len(pronunciations))
	}

	// Verify all have non-zero IDs.
	for i, p := range pronunciations {
		if p.ID == uuid.Nil {
			t.Errorf("Pronunciation[%d] ID should be non-zero", i)
		}
	}

	// Verify source_slug and region.
	for i, p := range pronunciations {
		if p.SourceSlug != "cmu" {
			t.Errorf("Pronunciation[%d] SourceSlug: got %q, want %q", i, p.SourceSlug, "cmu")
		}
		if p.Region == nil || *p.Region != "US" {
			t.Errorf("Pronunciation[%d] Region: got %v, want US", i, p.Region)
		}
		if p.AudioURL != nil {
			t.Errorf("Pronunciation[%d] AudioURL should be nil", i)
		}
	}

	// Verify ref entry IDs.
	helloCount := 0
	houseCount := 0
	readCount := 0
	for _, p := range pronunciations {
		switch p.RefEntryID {
		case helloID:
			helloCount++
		case houseID:
			houseCount++
		case readID:
			readCount++
		default:
			t.Errorf("unexpected RefEntryID: %s", p.RefEntryID)
		}
	}

	if helloCount != 1 {
		t.Errorf("hello pronunciations: got %d, want 1", helloCount)
	}
	if houseCount != 2 {
		t.Errorf("house pronunciations: got %d, want 2", houseCount)
	}
	if readCount != 2 {
		t.Errorf("read pronunciations: got %d, want 2", readCount)
	}

	// Verify transcription values.
	for _, p := range pronunciations {
		if p.Transcription == nil {
			t.Error("Transcription should not be nil")
			continue
		}
		if *p.Transcription == "" {
			t.Error("Transcription should not be empty")
		}
	}
}

func TestToDomainPronunciations_EmptyEntryIDMap(t *testing.T) {
	path := testdataPath(t, "sample.dict")

	result, err := Parse(path)
	if err != nil {
		t.Fatalf("Parse returned error: %v", err)
	}

	pronunciations := result.ToDomainPronunciations(map[string]uuid.UUID{})
	if len(pronunciations) != 0 {
		t.Errorf("expected 0 pronunciations for empty entryIDMap, got %d", len(pronunciations))
	}
}

func TestToDomainPronunciations_NilEntryIDMap(t *testing.T) {
	path := testdataPath(t, "sample.dict")

	result, err := Parse(path)
	if err != nil {
		t.Fatalf("Parse returned error: %v", err)
	}

	pronunciations := result.ToDomainPronunciations(nil)
	if len(pronunciations) != 0 {
		t.Errorf("expected 0 pronunciations for nil entryIDMap, got %d", len(pronunciations))
	}
}

func TestToDomainPronunciations_EmptyParseResult(t *testing.T) {
	result := ParseResult{
		Pronunciations: map[string][]IPATranscription{},
	}

	pronunciations := result.ToDomainPronunciations(map[string]uuid.UUID{
		"hello": uuid.New(),
	})
	if len(pronunciations) != 0 {
		t.Errorf("expected 0 pronunciations for empty parse result, got %d", len(pronunciations))
	}
}

// writeFile is a test helper that creates a file with given content.
func writeFile(path, content string) error {
	return os.WriteFile(path, []byte(content), 0o644)
}
