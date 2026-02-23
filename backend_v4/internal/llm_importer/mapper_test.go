package llm_importer

import (
	"testing"

	"github.com/heartmarshall/myenglish-backend/internal/domain"
)

func TestMap_basicEntry(t *testing.T) {
	input := LLMWordEntry{
		Word:       "Abandon",
		SourceSlug: "llm",
		Senses: []LLMSense{
			{
				POS:          "VERB",
				Definition:   "To leave permanently.",
				CEFRLevel:    "B1",
				Notes:        "Often used with 'to'.",
				Translations: []string{"бросать", "покидать"},
				Examples: []LLMExample{
					{Sentence: "She abandoned the car.", Translation: "Она бросила машину."},
				},
			},
		},
	}

	result := Map(input)

	if result.Entry.Text != "Abandon" {
		t.Errorf("Entry.Text = %q, want %q", result.Entry.Text, "Abandon")
	}
	if result.Entry.TextNormalized != "abandon" {
		t.Errorf("Entry.TextNormalized = %q, want %q", result.Entry.TextNormalized, "abandon")
	}

	if len(result.Senses) != 1 {
		t.Fatalf("len(Senses) = %d, want 1", len(result.Senses))
	}
	s := result.Senses[0]
	if s.Definition != "To leave permanently." {
		t.Errorf("Sense.Definition = %q", s.Definition)
	}
	if *s.PartOfSpeech != domain.PartOfSpeechVerb {
		t.Errorf("Sense.PartOfSpeech = %q, want VERB", *s.PartOfSpeech)
	}
	if *s.CEFRLevel != "B1" {
		t.Errorf("Sense.CEFRLevel = %q, want B1", *s.CEFRLevel)
	}
	if *s.Notes != "Often used with 'to'." {
		t.Errorf("Sense.Notes = %q", *s.Notes)
	}
	if s.SourceSlug != "llm" {
		t.Errorf("Sense.SourceSlug = %q, want llm", s.SourceSlug)
	}

	if len(result.Translations) != 2 {
		t.Fatalf("len(Translations) = %d, want 2", len(result.Translations))
	}
	if result.Translations[0].Text != "бросать" {
		t.Errorf("Translations[0].Text = %q", result.Translations[0].Text)
	}

	if len(result.Examples) != 1 {
		t.Fatalf("len(Examples) = %d, want 1", len(result.Examples))
	}
	ex := result.Examples[0]
	if ex.Sentence != "She abandoned the car." {
		t.Errorf("Example.Sentence = %q", ex.Sentence)
	}
	if *ex.Translation != "Она бросила машину." {
		t.Errorf("Example.Translation = %q", *ex.Translation)
	}
}

func TestMap_emptyCEFRAndNotes(t *testing.T) {
	input := LLMWordEntry{
		Word: "run", SourceSlug: "llm",
		Senses: []LLMSense{{POS: "VERB", Definition: "To move fast."}},
	}
	result := Map(input)
	if result.Senses[0].CEFRLevel != nil {
		t.Error("CEFRLevel should be nil when empty string in JSON")
	}
	if result.Senses[0].Notes != nil {
		t.Error("Notes should be nil when empty string in JSON")
	}
}
