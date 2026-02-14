package freedict

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/heartmarshall/myenglish-backend/internal/provider"
)

const defaultBaseURL = "https://api.dictionaryapi.dev/api/v2/entries/en"

// Provider fetches dictionary data from the FreeDictionary API.
type Provider struct {
	baseURL    string
	httpClient *http.Client
	log        *slog.Logger
}

// NewProvider creates a Provider with the default FreeDictionary API URL.
func NewProvider(logger *slog.Logger) *Provider {
	return &Provider{
		baseURL:    defaultBaseURL,
		httpClient: &http.Client{Timeout: 10 * time.Second},
		log:        logger.With("adapter", "freedict"),
	}
}

// NewProviderWithURL creates a Provider with a custom base URL (for testing).
func NewProviderWithURL(baseURL string, logger *slog.Logger) *Provider {
	return &Provider{
		baseURL:    baseURL,
		httpClient: &http.Client{Timeout: 10 * time.Second},
		log:        logger.With("adapter", "freedict"),
	}
}

// FetchEntry fetches a dictionary entry for the given word.
// Returns nil, nil if the word is not found (HTTP 404).
func (p *Provider) FetchEntry(ctx context.Context, word string) (*provider.DictionaryResult, error) {
	reqURL := p.baseURL + "/" + url.PathEscape(word)

	p.log.DebugContext(ctx, "freedict request", slog.String("word", word))

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, reqURL, nil)
	if err != nil {
		return nil, fmt.Errorf("freedict: create request: %w", err)
	}

	resp, err := p.doWithRetry(ctx, req, word)
	if err != nil {
		p.log.ErrorContext(ctx, "freedict request failed", slog.String("word", word), slog.String("error", err.Error()))
		return nil, fmt.Errorf("freedict: request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return nil, nil
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("freedict: unexpected status %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("freedict: read body: %w", err)
	}

	var entries []apiEntry
	if err := json.Unmarshal(body, &entries); err != nil {
		return nil, fmt.Errorf("freedict: decode json: %w", err)
	}

	result := mapAPIResponse(entries)

	p.log.DebugContext(ctx, "freedict response",
		slog.String("word", word),
		slog.Int("status", resp.StatusCode),
		slog.Int("senses", len(result.Senses)),
		slog.Int("pronunciations", len(result.Pronunciations)),
	)

	return result, nil
}

// doWithRetry executes the request with a single retry on 5xx or network errors.
func (p *Provider) doWithRetry(ctx context.Context, req *http.Request, word string) (*http.Response, error) {
	resp, err := p.httpClient.Do(req)

	shouldRetry := err != nil || (resp != nil && resp.StatusCode >= 500)
	if !shouldRetry {
		return resp, err
	}

	// Don't retry if context is already cancelled.
	if ctx.Err() != nil {
		return resp, err
	}

	reason := "network error"
	if err == nil && resp != nil {
		reason = fmt.Sprintf("status %d", resp.StatusCode)
	}
	p.log.WarnContext(ctx, "freedict retry", slog.String("word", word), slog.String("reason", reason))

	// Close body from the failed attempt before retrying.
	if resp != nil && resp.Body != nil {
		resp.Body.Close()
	}

	time.Sleep(500 * time.Millisecond)

	resp, err = p.httpClient.Do(req)
	return resp, err
}

// mapAPIResponse converts the API entries into a provider.DictionaryResult.
// Multiple entries (different etymologies) are merged: senses concatenated,
// pronunciations deduplicated by transcription text.
func mapAPIResponse(entries []apiEntry) *provider.DictionaryResult {
	result := &provider.DictionaryResult{
		Senses:         []provider.SenseResult{},
		Pronunciations: []provider.PronunciationResult{},
	}

	if len(entries) == 0 {
		return result
	}

	result.Word = entries[0].Word

	// Track seen transcriptions for deduplication.
	// Key: transcription text, Value: index in result.Pronunciations.
	seenTranscriptions := make(map[string]int)

	for _, entry := range entries {
		// Collect senses from all entries.
		for _, meaning := range entry.Meanings {
			pos := meaning.PartOfSpeech
			for _, def := range meaning.Definitions {
				sense := provider.SenseResult{
					Definition: def.Definition,
					Examples:   []provider.ExampleResult{},
				}
				if pos != "" {
					posCopy := pos
					sense.PartOfSpeech = &posCopy
				}
				if def.Example != "" {
					sense.Examples = append(sense.Examples, provider.ExampleResult{
						Sentence: def.Example,
					})
				}
				result.Senses = append(result.Senses, sense)
			}
		}

		// Collect pronunciations, deduplicating by transcription.
		for _, ph := range entry.Phonetics {
			pron := mapPhonetic(ph)
			if pron == nil {
				continue
			}

			// Deduplicate by transcription text.
			if pron.Transcription != nil {
				key := *pron.Transcription
				if idx, exists := seenTranscriptions[key]; exists {
					// If existing entry has no audio but this one does, update it.
					if result.Pronunciations[idx].AudioURL == nil && pron.AudioURL != nil {
						result.Pronunciations[idx].AudioURL = pron.AudioURL
						result.Pronunciations[idx].Region = pron.Region
					}
					continue
				}
				seenTranscriptions[key] = len(result.Pronunciations)
			}

			result.Pronunciations = append(result.Pronunciations, *pron)
		}
	}

	return result
}

// mapPhonetic converts an API phonetic to a PronunciationResult.
// Returns nil if both text and audio are empty.
func mapPhonetic(ph apiPhonetic) *provider.PronunciationResult {
	if ph.Text == "" && ph.Audio == "" {
		return nil
	}

	pron := &provider.PronunciationResult{}

	if ph.Text != "" {
		t := ph.Text
		pron.Transcription = &t
	}

	if ph.Audio != "" {
		a := ph.Audio
		pron.AudioURL = &a
		pron.Region = inferRegion(ph.Audio)
	}

	return pron
}

// inferRegion attempts to determine the pronunciation region from the audio URL.
func inferRegion(audioURL string) *string {
	lower := strings.ToLower(audioURL)
	if strings.Contains(lower, "-us.") || strings.Contains(lower, "-us-") {
		r := "US"
		return &r
	}
	if strings.Contains(lower, "-uk.") || strings.Contains(lower, "-uk-") {
		r := "UK"
		return &r
	}
	return nil
}
