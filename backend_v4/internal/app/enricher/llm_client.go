package enricher

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"

	anthropic "github.com/anthropics/anthropic-sdk-go"
	"github.com/heartmarshall/myenglish-backend/internal/domain"
)

// callLLM sends one EnrichContext to Claude and saves the LLM output JSON.
// Output is saved to llmOutputDir/<normalized_word>.json.
// If the file already exists, it is skipped (resume support).
func callLLM(ctx context.Context, client anthropic.Client, cfg *Config, enrichCtx EnrichContext, log *slog.Logger) error {
	normalized := domain.NormalizeText(enrichCtx.Word)
	outPath := filepath.Join(cfg.LLMOutputDir, normalized+".json")

	// Resume: skip if already done.
	if _, err := os.Stat(outPath); err == nil {
		return nil
	}

	if err := os.MkdirAll(cfg.LLMOutputDir, 0755); err != nil {
		return fmt.Errorf("create llm-output dir: %w", err)
	}

	contextJSON, err := json.MarshalIndent(enrichCtx, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal context: %w", err)
	}

	prompt := buildPrompt(enrichCtx.Word, string(contextJSON))

	msg, err := client.Messages.New(ctx, anthropic.MessageNewParams{
		Model:     anthropic.Model(cfg.LLMModel),
		MaxTokens: 2048,
		Messages: []anthropic.MessageParam{
			anthropic.NewUserMessage(anthropic.NewTextBlock(prompt)),
		},
	})
	if err != nil {
		return fmt.Errorf("llm api call for %q: %w", enrichCtx.Word, err)
	}

	if len(msg.Content) == 0 {
		return fmt.Errorf("empty response for %q", enrichCtx.Word)
	}

	responseText := msg.Content[0].Text

	// Extract JSON from the response (between first { and last }).
	jsonStr, err := extractJSON(responseText)
	if err != nil {
		return fmt.Errorf("extract json from response for %q: %w", enrichCtx.Word, err)
	}

	// Verify the extracted string is actually valid JSON before persisting.
	if !json.Valid([]byte(jsonStr)) {
		return fmt.Errorf("response for %q does not contain valid JSON", enrichCtx.Word)
	}

	if err := os.WriteFile(outPath, []byte(jsonStr), 0644); err != nil {
		return fmt.Errorf("write llm output for %q: %w", enrichCtx.Word, err)
	}

	log.Info("llm output saved", slog.String("word", enrichCtx.Word), slog.String("path", outPath))
	return nil
}

// buildPrompt creates the LLM prompt for a single word.
func buildPrompt(word, contextJSON string) string {
	return fmt.Sprintf(`You are a professional English-Russian dictionary editor.

Given the word "%s" and its context data from reference datasets, produce an improved dictionary entry in JSON format.

Context data:
%s

Output ONLY a valid JSON object matching this exact schema:
{
  "word": "<word>",
  "source_slug": "llm",
  "senses": [
    {
      "pos": "<NOUN|VERB|ADJECTIVE|ADVERB|...>",
      "definition": "<clear English definition suitable for B1+ learners>",
      "cefr_level": "<A1|A2|B1|B2|C1|C2 or empty>",
      "notes": "<learning note in Russian: usage tips, collocations, common mistakes>",
      "translations": ["<Russian translation 1>", "<Russian translation 2>"],
      "examples": [
        {"sentence": "<English example>", "translation": "<Russian translation>"}
      ]
    }
  ]
}

Rules:
- Improve definitions to be clearer and more useful for language learners
- Provide 2-4 high-quality Russian translations per sense
- Write notes in Russian, focusing on practical usage
- Generate 1-3 natural example sentences with Russian translations
- Use uppercase POS values matching: NOUN, VERB, ADJECTIVE, ADVERB, PRONOUN, PREPOSITION, CONJUNCTION, INTERJECTION, PHRASE, IDIOM, OTHER
- Output ONLY the JSON, no markdown, no explanations`, word, contextJSON)
}

// extractJSON finds the first complete JSON object in a string.
func extractJSON(s string) (string, error) {
	start := strings.Index(s, "{")
	end := strings.LastIndex(s, "}")
	if start == -1 || end == -1 || end <= start {
		return "", fmt.Errorf("no JSON object found in response")
	}
	return s[start : end+1], nil
}
