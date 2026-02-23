package llm_importer

import (
	"fmt"

	"github.com/heartmarshall/myenglish-backend/internal/domain"
)

var validCEFR = map[string]bool{
	"A1": true, "A2": true, "B1": true, "B2": true, "C1": true, "C2": true,
}

// Validate checks that an LLMWordEntry has all required fields with valid values.
func Validate(e LLMWordEntry) error {
	if e.Word == "" {
		return fmt.Errorf("word is empty")
	}
	if len(e.Senses) == 0 {
		return fmt.Errorf("word %q has no senses", e.Word)
	}
	for i, s := range e.Senses {
		if s.Definition == "" {
			return fmt.Errorf("sense %d of %q has empty definition", i, e.Word)
		}
		pos := domain.PartOfSpeech(s.POS)
		if !pos.IsValid() {
			return fmt.Errorf("sense %d of %q has invalid POS %q", i, e.Word, s.POS)
		}
		if s.CEFRLevel != "" && !validCEFR[s.CEFRLevel] {
			return fmt.Errorf("sense %d of %q has invalid CEFR level %q", i, e.Word, s.CEFRLevel)
		}
	}
	return nil
}
