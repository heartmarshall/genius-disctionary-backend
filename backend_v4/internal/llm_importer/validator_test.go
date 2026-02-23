package llm_importer

import "testing"

func TestValidate_valid(t *testing.T) {
	entry := LLMWordEntry{
		Word:       "abandon",
		SourceSlug: "llm",
		Senses: []LLMSense{
			{
				POS:          "VERB",
				Definition:   "To leave permanently.",
				CEFRLevel:    "B1",
				Notes:        "Often used with 'to'.",
				Translations: []string{"бросать"},
				Examples: []LLMExample{
					{Sentence: "She abandoned the car.", Translation: "Она бросила машину."},
				},
			},
		},
	}
	if err := Validate(entry); err != nil {
		t.Errorf("Validate() unexpected error: %v", err)
	}
}

func TestValidate_missingWord(t *testing.T) {
	entry := LLMWordEntry{SourceSlug: "llm", Senses: []LLMSense{{POS: "NOUN", Definition: "x"}}}
	if err := Validate(entry); err == nil {
		t.Error("Validate() expected error for empty word")
	}
}

func TestValidate_noSenses(t *testing.T) {
	entry := LLMWordEntry{Word: "run", SourceSlug: "llm", Senses: nil}
	if err := Validate(entry); err == nil {
		t.Error("Validate() expected error for empty senses")
	}
}

func TestValidate_invalidPOS(t *testing.T) {
	entry := LLMWordEntry{
		Word: "run", SourceSlug: "llm",
		Senses: []LLMSense{{POS: "BANANA", Definition: "x"}},
	}
	if err := Validate(entry); err == nil {
		t.Error("Validate() expected error for invalid POS")
	}
}

func TestValidate_invalidCEFR(t *testing.T) {
	entry := LLMWordEntry{
		Word: "run", SourceSlug: "llm",
		Senses: []LLMSense{{POS: "VERB", Definition: "x", CEFRLevel: "Z9"}},
	}
	if err := Validate(entry); err == nil {
		t.Error("Validate() expected error for invalid CEFR level")
	}
}

func TestValidate_emptyDefinition(t *testing.T) {
	entry := LLMWordEntry{
		Word: "run", SourceSlug: "llm",
		Senses: []LLMSense{{POS: "VERB", Definition: ""}},
	}
	if err := Validate(entry); err == nil {
		t.Error("Validate() expected error for empty definition")
	}
}
