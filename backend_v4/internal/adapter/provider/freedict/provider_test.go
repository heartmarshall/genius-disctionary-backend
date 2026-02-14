package freedict

import (
	"context"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
)

func newTestLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(io.Discard, nil))
}

func strPtr(s string) *string { return &s }

func TestProvider_FetchEntry_Success(t *testing.T) {
	t.Parallel()

	body := `[{
		"word": "hello",
		"phonetics": [
			{"text": "/həˈloʊ/", "audio": "https://example.com/hello-us.mp3"},
			{"text": "/hɛˈləʊ/", "audio": "https://example.com/hello-uk.mp3"}
		],
		"meanings": [
			{
				"partOfSpeech": "noun",
				"definitions": [
					{"definition": "A greeting.", "example": "She gave a cheerful hello."}
				]
			},
			{
				"partOfSpeech": "interjection",
				"definitions": [
					{"definition": "Used as a greeting.", "example": "Hello, how are you?"},
					{"definition": "Used to attract attention.", "example": ""}
				]
			}
		]
	}]`

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/hello" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(body))
	}))
	defer srv.Close()

	p := NewProviderWithURL(srv.URL, newTestLogger())
	result, err := p.FetchEntry(context.Background(), "hello")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result == nil {
		t.Fatal("expected non-nil result")
	}

	if result.Word != "hello" {
		t.Errorf("Word = %q, want %q", result.Word, "hello")
	}

	// 3 senses total: 1 noun + 2 interjection
	if len(result.Senses) != 3 {
		t.Fatalf("len(Senses) = %d, want 3", len(result.Senses))
	}

	// First sense: noun with example
	s0 := result.Senses[0]
	if s0.Definition != "A greeting." {
		t.Errorf("Senses[0].Definition = %q, want %q", s0.Definition, "A greeting.")
	}
	if s0.PartOfSpeech == nil || *s0.PartOfSpeech != "noun" {
		t.Errorf("Senses[0].PartOfSpeech = %v, want %q", s0.PartOfSpeech, "noun")
	}
	if len(s0.Examples) != 1 || s0.Examples[0].Sentence != "She gave a cheerful hello." {
		t.Errorf("Senses[0].Examples = %v, want one example", s0.Examples)
	}

	// Third sense: interjection without example
	s2 := result.Senses[2]
	if s2.PartOfSpeech == nil || *s2.PartOfSpeech != "interjection" {
		t.Errorf("Senses[2].PartOfSpeech = %v, want %q", s2.PartOfSpeech, "interjection")
	}
	if len(s2.Examples) != 0 {
		t.Errorf("Senses[2].Examples = %v, want empty", s2.Examples)
	}

	// 2 pronunciations
	if len(result.Pronunciations) != 2 {
		t.Fatalf("len(Pronunciations) = %d, want 2", len(result.Pronunciations))
	}

	p0 := result.Pronunciations[0]
	if p0.Transcription == nil || *p0.Transcription != "/həˈloʊ/" {
		t.Errorf("Pronunciations[0].Transcription = %v, want %q", p0.Transcription, "/həˈloʊ/")
	}
	if p0.AudioURL == nil || *p0.AudioURL != "https://example.com/hello-us.mp3" {
		t.Errorf("Pronunciations[0].AudioURL = %v", p0.AudioURL)
	}
	if p0.Region == nil || *p0.Region != "US" {
		t.Errorf("Pronunciations[0].Region = %v, want US", p0.Region)
	}

	p1 := result.Pronunciations[1]
	if p1.Region == nil || *p1.Region != "UK" {
		t.Errorf("Pronunciations[1].Region = %v, want UK", p1.Region)
	}
}

func TestProvider_FetchEntry_NotFound(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		w.Write([]byte(`{"title":"No Definitions Found"}`))
	}))
	defer srv.Close()

	p := NewProviderWithURL(srv.URL, newTestLogger())
	result, err := p.FetchEntry(context.Background(), "asdfxyz")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != nil {
		t.Fatalf("expected nil result for 404, got %+v", result)
	}
}

func TestProvider_FetchEntry_ServerErrorRetrySuccess(t *testing.T) {
	t.Parallel()

	var callCount atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		n := callCount.Add(1)
		if n == 1 {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`[{"word":"test","phonetics":[],"meanings":[]}]`))
	}))
	defer srv.Close()

	p := NewProviderWithURL(srv.URL, newTestLogger())
	result, err := p.FetchEntry(context.Background(), "test")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result == nil {
		t.Fatal("expected non-nil result after retry")
	}
	if result.Word != "test" {
		t.Errorf("Word = %q, want %q", result.Word, "test")
	}
	if got := callCount.Load(); got != 2 {
		t.Errorf("call count = %d, want 2", got)
	}
}

func TestProvider_FetchEntry_ServerErrorBothAttemptsFail(t *testing.T) {
	t.Parallel()

	var callCount atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		callCount.Add(1)
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer srv.Close()

	p := NewProviderWithURL(srv.URL, newTestLogger())
	_, err := p.FetchEntry(context.Background(), "fail")
	if err == nil {
		t.Fatal("expected error when both attempts fail")
	}
	if got := callCount.Load(); got != 2 {
		t.Errorf("call count = %d, want 2", got)
	}
}

func TestProvider_FetchEntry_InvalidJSON(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`not valid json`))
	}))
	defer srv.Close()

	p := NewProviderWithURL(srv.URL, newTestLogger())
	_, err := p.FetchEntry(context.Background(), "bad")
	if err == nil {
		t.Fatal("expected error for invalid JSON")
	}
}

func TestProvider_FetchEntry_MultipleEntries(t *testing.T) {
	t.Parallel()

	// Two entries (different etymologies) with overlapping phonetics.
	body := `[
		{
			"word": "run",
			"phonetics": [{"text": "/rʌn/", "audio": "https://example.com/run-us.mp3"}],
			"meanings": [
				{
					"partOfSpeech": "verb",
					"definitions": [{"definition": "To move fast.", "example": "She runs every day."}]
				}
			]
		},
		{
			"word": "run",
			"phonetics": [{"text": "/rʌn/", "audio": ""}],
			"meanings": [
				{
					"partOfSpeech": "noun",
					"definitions": [{"definition": "An act of running.", "example": ""}]
				}
			]
		}
	]`

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(body))
	}))
	defer srv.Close()

	p := NewProviderWithURL(srv.URL, newTestLogger())
	result, err := p.FetchEntry(context.Background(), "run")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Senses should be concatenated: 1 verb + 1 noun = 2.
	if len(result.Senses) != 2 {
		t.Fatalf("len(Senses) = %d, want 2", len(result.Senses))
	}
	if result.Senses[0].PartOfSpeech == nil || *result.Senses[0].PartOfSpeech != "verb" {
		t.Errorf("Senses[0].PartOfSpeech = %v, want verb", result.Senses[0].PartOfSpeech)
	}
	if result.Senses[1].PartOfSpeech == nil || *result.Senses[1].PartOfSpeech != "noun" {
		t.Errorf("Senses[1].PartOfSpeech = %v, want noun", result.Senses[1].PartOfSpeech)
	}

	// Pronunciations should be deduplicated: both have "/rʌn/", keep first (with audio).
	if len(result.Pronunciations) != 1 {
		t.Fatalf("len(Pronunciations) = %d, want 1 (deduplicated)", len(result.Pronunciations))
	}
	if result.Pronunciations[0].AudioURL == nil {
		t.Error("expected AudioURL to be kept from first entry")
	}
}

func TestProvider_FetchEntry_PhoneticsDeduplication(t *testing.T) {
	t.Parallel()

	body := `[{
		"word": "test",
		"phonetics": [
			{"text": "/tɛst/", "audio": ""},
			{"text": "/tɛst/", "audio": "https://example.com/test-uk.mp3"},
			{"text": "/tɛst/", "audio": "https://example.com/test-us.mp3"},
			{"text": "", "audio": ""},
			{"text": "", "audio": "https://example.com/other.mp3"}
		],
		"meanings": []
	}]`

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(body))
	}))
	defer srv.Close()

	p := NewProviderWithURL(srv.URL, newTestLogger())
	result, err := p.FetchEntry(context.Background(), "test")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// "/tɛst/" appears 3 times → deduplicated to 1.
	// First has no audio, second has audio → second's audio should be used.
	// Empty text with empty audio is skipped.
	// Empty text with audio is kept (no transcription to dedup by).
	if len(result.Pronunciations) != 2 {
		t.Fatalf("len(Pronunciations) = %d, want 2", len(result.Pronunciations))
	}

	// First: "/tɛst/" with UK audio (first with audio wins update).
	pr0 := result.Pronunciations[0]
	if pr0.Transcription == nil || *pr0.Transcription != "/tɛst/" {
		t.Errorf("Pronunciations[0].Transcription = %v, want /tɛst/", pr0.Transcription)
	}
	if pr0.AudioURL == nil || *pr0.AudioURL != "https://example.com/test-uk.mp3" {
		t.Errorf("Pronunciations[0].AudioURL = %v, want test-uk.mp3", pr0.AudioURL)
	}

	// Second: no transcription, has audio.
	pr1 := result.Pronunciations[1]
	if pr1.Transcription != nil {
		t.Errorf("Pronunciations[1].Transcription = %v, want nil", pr1.Transcription)
	}
	if pr1.AudioURL == nil || *pr1.AudioURL != "https://example.com/other.mp3" {
		t.Errorf("Pronunciations[1].AudioURL = %v, want other.mp3", pr1.AudioURL)
	}
}

func TestProvider_FetchEntry_EmptyDefinitions(t *testing.T) {
	t.Parallel()

	body := `[{"word": "rare", "phonetics": [], "meanings": []}]`

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(body))
	}))
	defer srv.Close()

	p := NewProviderWithURL(srv.URL, newTestLogger())
	result, err := p.FetchEntry(context.Background(), "rare")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Word != "rare" {
		t.Errorf("Word = %q, want %q", result.Word, "rare")
	}
	if len(result.Senses) != 0 {
		t.Errorf("len(Senses) = %d, want 0", len(result.Senses))
	}
	if len(result.Pronunciations) != 0 {
		t.Errorf("len(Pronunciations) = %d, want 0", len(result.Pronunciations))
	}
}

func TestProvider_FetchEntry_DefinitionWithExample(t *testing.T) {
	t.Parallel()

	body := `[{
		"word": "book",
		"phonetics": [],
		"meanings": [{
			"partOfSpeech": "noun",
			"definitions": [{"definition": "A written work.", "example": "I read a book."}]
		}]
	}]`

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(body))
	}))
	defer srv.Close()

	p := NewProviderWithURL(srv.URL, newTestLogger())
	result, err := p.FetchEntry(context.Background(), "book")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(result.Senses) != 1 {
		t.Fatalf("len(Senses) = %d, want 1", len(result.Senses))
	}
	s := result.Senses[0]
	if len(s.Examples) != 1 {
		t.Fatalf("len(Examples) = %d, want 1", len(s.Examples))
	}
	if s.Examples[0].Sentence != "I read a book." {
		t.Errorf("Example.Sentence = %q, want %q", s.Examples[0].Sentence, "I read a book.")
	}
	if s.Examples[0].Translation != nil {
		t.Errorf("Example.Translation = %v, want nil", s.Examples[0].Translation)
	}
}

func TestProvider_FetchEntry_DefinitionWithoutExample(t *testing.T) {
	t.Parallel()

	body := `[{
		"word": "cat",
		"phonetics": [],
		"meanings": [{
			"partOfSpeech": "noun",
			"definitions": [{"definition": "A small domesticated carnivore.", "example": ""}]
		}]
	}]`

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(body))
	}))
	defer srv.Close()

	p := NewProviderWithURL(srv.URL, newTestLogger())
	result, err := p.FetchEntry(context.Background(), "cat")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(result.Senses) != 1 {
		t.Fatalf("len(Senses) = %d, want 1", len(result.Senses))
	}
	if len(result.Senses[0].Examples) != 0 {
		t.Errorf("len(Examples) = %d, want 0", len(result.Senses[0].Examples))
	}
}

func TestInferRegion(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		audioURL string
		want     *string
	}{
		{
			name:     "US with dot separator",
			audioURL: "https://example.com/hello-us.mp3",
			want:     strPtr("US"),
		},
		{
			name:     "US with dash separator",
			audioURL: "https://example.com/audio-us-hello.mp3",
			want:     strPtr("US"),
		},
		{
			name:     "UK with dot separator",
			audioURL: "https://example.com/hello-uk.mp3",
			want:     strPtr("UK"),
		},
		{
			name:     "UK with dash separator",
			audioURL: "https://example.com/audio-uk-hello.mp3",
			want:     strPtr("UK"),
		},
		{
			name:     "no region",
			audioURL: "https://example.com/hello.mp3",
			want:     nil,
		},
		{
			name:     "case insensitive US",
			audioURL: "https://example.com/Hello-US.mp3",
			want:     strPtr("US"),
		},
		{
			name:     "empty URL",
			audioURL: "",
			want:     nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := inferRegion(tt.audioURL)
			if tt.want == nil {
				if got != nil {
					t.Errorf("inferRegion(%q) = %q, want nil", tt.audioURL, *got)
				}
				return
			}
			if got == nil {
				t.Errorf("inferRegion(%q) = nil, want %q", tt.audioURL, *tt.want)
				return
			}
			if *got != *tt.want {
				t.Errorf("inferRegion(%q) = %q, want %q", tt.audioURL, *got, *tt.want)
			}
		})
	}
}

// TestProvider_FetchEntry_EmptyArray verifies that an empty JSON array returns
// a DictionaryResult with empty Senses and Pronunciations.
func TestProvider_FetchEntry_EmptyArray(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`[]`))
	}))
	defer srv.Close()

	p := NewProviderWithURL(srv.URL, newTestLogger())
	result, err := p.FetchEntry(context.Background(), "empty")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result == nil {
		t.Fatal("expected non-nil result for empty array")
	}
	if result.Word != "" {
		t.Errorf("Word = %q, want empty", result.Word)
	}
	if len(result.Senses) != 0 {
		t.Errorf("len(Senses) = %d, want 0", len(result.Senses))
	}
	if len(result.Pronunciations) != 0 {
		t.Errorf("len(Pronunciations) = %d, want 0", len(result.Pronunciations))
	}
}
