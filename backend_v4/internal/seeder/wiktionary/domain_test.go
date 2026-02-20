package wiktionary

import (
	"testing"

	"github.com/google/uuid"
	"github.com/heartmarshall/myenglish-backend/internal/domain"
)

func TestToDomainEntries_EmptyInput(t *testing.T) {
	result := ToDomainEntries(nil)

	if len(result.Entries) != 0 {
		t.Errorf("Entries: got %d, want 0", len(result.Entries))
	}
	if len(result.Senses) != 0 {
		t.Errorf("Senses: got %d, want 0", len(result.Senses))
	}
	if len(result.Translations) != 0 {
		t.Errorf("Translations: got %d, want 0", len(result.Translations))
	}
	if len(result.Examples) != 0 {
		t.Errorf("Examples: got %d, want 0", len(result.Examples))
	}
	if len(result.Pronunciations) != 0 {
		t.Errorf("Pronunciations: got %d, want 0", len(result.Pronunciations))
	}

	// Also test empty slice.
	result2 := ToDomainEntries([]ParsedEntry{})
	if len(result2.Entries) != 0 {
		t.Errorf("Entries (empty slice): got %d, want 0", len(result2.Entries))
	}
}

func TestToDomainEntries_SingleEntryOneSense(t *testing.T) {
	entries := []ParsedEntry{
		{
			Word: "Hello",
			POSGroups: []POSGroup{
				{
					POS: "noun",
					Senses: []ParsedSense{
						{
							Glosses: []string{"a greeting"},
						},
					},
				},
			},
		},
	}

	result := ToDomainEntries(entries)

	// One entry.
	if len(result.Entries) != 1 {
		t.Fatalf("Entries: got %d, want 1", len(result.Entries))
	}
	entry := result.Entries[0]

	if entry.Text != "Hello" {
		t.Errorf("Text: got %q, want %q", entry.Text, "Hello")
	}
	if entry.TextNormalized != "hello" {
		t.Errorf("TextNormalized: got %q, want %q", entry.TextNormalized, "hello")
	}
	if entry.IsCoreLexicon != false {
		t.Errorf("IsCoreLexicon: got %v, want false", entry.IsCoreLexicon)
	}
	if entry.ID == uuid.Nil {
		t.Error("Entry ID should be a non-zero UUID")
	}
	if entry.CreatedAt.IsZero() {
		t.Error("CreatedAt should not be zero")
	}

	// One sense.
	if len(result.Senses) != 1 {
		t.Fatalf("Senses: got %d, want 1", len(result.Senses))
	}
	sense := result.Senses[0]

	if sense.ID == uuid.Nil {
		t.Error("Sense ID should be a non-zero UUID")
	}
	if sense.RefEntryID != entry.ID {
		t.Errorf("Sense RefEntryID: got %s, want %s", sense.RefEntryID, entry.ID)
	}
	if sense.Definition != "a greeting" {
		t.Errorf("Definition: got %q, want %q", sense.Definition, "a greeting")
	}
	if sense.PartOfSpeech == nil {
		t.Fatal("PartOfSpeech should not be nil")
	}
	if *sense.PartOfSpeech != domain.PartOfSpeechNoun {
		t.Errorf("PartOfSpeech: got %q, want %q", *sense.PartOfSpeech, domain.PartOfSpeechNoun)
	}
	if sense.SourceSlug != "wiktionary" {
		t.Errorf("SourceSlug: got %q, want %q", sense.SourceSlug, "wiktionary")
	}
	if sense.Position != 0 {
		t.Errorf("Position: got %d, want 0", sense.Position)
	}
}

func TestToDomainEntries_MultiplePOSGroups(t *testing.T) {
	entries := []ParsedEntry{
		{
			Word: "run",
			POSGroups: []POSGroup{
				{
					POS: "noun",
					Senses: []ParsedSense{
						{Glosses: []string{"an act of running"}},
						{Glosses: []string{"a flow of liquid"}},
					},
				},
				{
					POS: "verb",
					Senses: []ParsedSense{
						{Glosses: []string{"to move quickly on foot"}},
					},
				},
			},
		},
	}

	result := ToDomainEntries(entries)

	if len(result.Senses) != 3 {
		t.Fatalf("Senses: got %d, want 3", len(result.Senses))
	}

	// Verify sequential positions across POS groups.
	for i, sense := range result.Senses {
		if sense.Position != i {
			t.Errorf("Sense[%d] Position: got %d, want %d", i, sense.Position, i)
		}
		if sense.RefEntryID != result.Entries[0].ID {
			t.Errorf("Sense[%d] RefEntryID mismatch", i)
		}
	}

	// First two senses should be noun, third should be verb.
	if *result.Senses[0].PartOfSpeech != domain.PartOfSpeechNoun {
		t.Errorf("Sense[0] POS: got %q, want noun", *result.Senses[0].PartOfSpeech)
	}
	if *result.Senses[1].PartOfSpeech != domain.PartOfSpeechNoun {
		t.Errorf("Sense[1] POS: got %q, want noun", *result.Senses[1].PartOfSpeech)
	}
	if *result.Senses[2].PartOfSpeech != domain.PartOfSpeechVerb {
		t.Errorf("Sense[2] POS: got %q, want verb", *result.Senses[2].PartOfSpeech)
	}

	// Verify definitions.
	if result.Senses[0].Definition != "an act of running" {
		t.Errorf("Sense[0] Definition: got %q", result.Senses[0].Definition)
	}
	if result.Senses[1].Definition != "a flow of liquid" {
		t.Errorf("Sense[1] Definition: got %q", result.Senses[1].Definition)
	}
	if result.Senses[2].Definition != "to move quickly on foot" {
		t.Errorf("Sense[2] Definition: got %q", result.Senses[2].Definition)
	}
}

func TestToDomainEntries_TranslationsAndExamples(t *testing.T) {
	entries := []ParsedEntry{
		{
			Word: "book",
			POSGroups: []POSGroup{
				{
					POS: "noun",
					Senses: []ParsedSense{
						{
							Glosses:      []string{"a written work"},
							Translations: []string{"книга", "книжка"},
							Examples:     []string{"I read a <b>book</b>.", "The [[book]] was great."},
						},
					},
				},
			},
		},
	}

	result := ToDomainEntries(entries)

	// Translations.
	if len(result.Translations) != 2 {
		t.Fatalf("Translations: got %d, want 2", len(result.Translations))
	}
	senseID := result.Senses[0].ID
	for i, tr := range result.Translations {
		if tr.RefSenseID != senseID {
			t.Errorf("Translation[%d] RefSenseID mismatch", i)
		}
		if tr.SourceSlug != "wiktionary" {
			t.Errorf("Translation[%d] SourceSlug: got %q", i, tr.SourceSlug)
		}
		if tr.Position != i {
			t.Errorf("Translation[%d] Position: got %d, want %d", i, tr.Position, i)
		}
		if tr.ID == uuid.Nil {
			t.Errorf("Translation[%d] ID should be non-zero", i)
		}
	}
	if result.Translations[0].Text != "книга" {
		t.Errorf("Translation[0] Text: got %q, want %q", result.Translations[0].Text, "книга")
	}
	if result.Translations[1].Text != "книжка" {
		t.Errorf("Translation[1] Text: got %q, want %q", result.Translations[1].Text, "книжка")
	}

	// Examples (markup should be stripped).
	if len(result.Examples) != 2 {
		t.Fatalf("Examples: got %d, want 2", len(result.Examples))
	}
	for i, ex := range result.Examples {
		if ex.RefSenseID != senseID {
			t.Errorf("Example[%d] RefSenseID mismatch", i)
		}
		if ex.SourceSlug != "wiktionary" {
			t.Errorf("Example[%d] SourceSlug: got %q", i, ex.SourceSlug)
		}
		if ex.Position != i {
			t.Errorf("Example[%d] Position: got %d, want %d", i, ex.Position, i)
		}
		if ex.Translation != nil {
			t.Errorf("Example[%d] Translation should be nil", i)
		}
		if ex.ID == uuid.Nil {
			t.Errorf("Example[%d] ID should be non-zero", i)
		}
	}
	if result.Examples[0].Sentence != "I read a book." {
		t.Errorf("Example[0] Sentence: got %q, want %q", result.Examples[0].Sentence, "I read a book.")
	}
	if result.Examples[1].Sentence != "The book was great." {
		t.Errorf("Example[1] Sentence: got %q, want %q", result.Examples[1].Sentence, "The book was great.")
	}
}

func TestToDomainEntries_EmptyGlossesSkipped(t *testing.T) {
	entries := []ParsedEntry{
		{
			Word: "test",
			POSGroups: []POSGroup{
				{
					POS: "noun",
					Senses: []ParsedSense{
						{Glosses: []string{}},              // empty glosses, skip
						{Glosses: nil},                     // nil glosses, skip
						{Glosses: []string{"a procedure"}}, // valid
					},
				},
			},
		},
	}

	result := ToDomainEntries(entries)

	if len(result.Senses) != 1 {
		t.Fatalf("Senses: got %d, want 1 (empty glosses should be skipped)", len(result.Senses))
	}
	if result.Senses[0].Definition != "a procedure" {
		t.Errorf("Definition: got %q, want %q", result.Senses[0].Definition, "a procedure")
	}
	// Position should still be 0 since skipped senses don't increment.
	if result.Senses[0].Position != 0 {
		t.Errorf("Position: got %d, want 0", result.Senses[0].Position)
	}
}

func TestToDomainEntries_PronunciationsWithRegion(t *testing.T) {
	entries := []ParsedEntry{
		{
			Word: "water",
			POSGroups: []POSGroup{
				{
					POS:    "noun",
					Senses: []ParsedSense{{Glosses: []string{"a liquid"}}},
				},
			},
			Sounds: []Sound{
				{IPA: "/ˈwɔːtər/", Region: "UK"},
				{IPA: "/ˈwɑːtɚ/", Region: "US"},
			},
		},
	}

	result := ToDomainEntries(entries)

	if len(result.Pronunciations) != 2 {
		t.Fatalf("Pronunciations: got %d, want 2", len(result.Pronunciations))
	}

	entryID := result.Entries[0].ID
	for i, p := range result.Pronunciations {
		if p.RefEntryID != entryID {
			t.Errorf("Pronunciation[%d] RefEntryID mismatch", i)
		}
		if p.SourceSlug != "wiktionary" {
			t.Errorf("Pronunciation[%d] SourceSlug: got %q", i, p.SourceSlug)
		}
		if p.ID == uuid.Nil {
			t.Errorf("Pronunciation[%d] ID should be non-zero", i)
		}
		if p.AudioURL != nil {
			t.Errorf("Pronunciation[%d] AudioURL should be nil", i)
		}
	}

	// First pronunciation.
	if result.Pronunciations[0].Transcription == nil || *result.Pronunciations[0].Transcription != "/ˈwɔːtər/" {
		t.Errorf("Pronunciation[0] Transcription: got %v", result.Pronunciations[0].Transcription)
	}
	if result.Pronunciations[0].Region == nil || *result.Pronunciations[0].Region != "UK" {
		t.Errorf("Pronunciation[0] Region: got %v", result.Pronunciations[0].Region)
	}

	// Second pronunciation.
	if result.Pronunciations[1].Transcription == nil || *result.Pronunciations[1].Transcription != "/ˈwɑːtɚ/" {
		t.Errorf("Pronunciation[1] Transcription: got %v", result.Pronunciations[1].Transcription)
	}
	if result.Pronunciations[1].Region == nil || *result.Pronunciations[1].Region != "US" {
		t.Errorf("Pronunciation[1] Region: got %v", result.Pronunciations[1].Region)
	}
}

func TestToDomainEntries_PronunciationWithoutRegion(t *testing.T) {
	entries := []ParsedEntry{
		{
			Word: "cat",
			POSGroups: []POSGroup{
				{
					POS:    "noun",
					Senses: []ParsedSense{{Glosses: []string{"a feline"}}},
				},
			},
			Sounds: []Sound{
				{IPA: "/kæt/", Region: ""},
			},
		},
	}

	result := ToDomainEntries(entries)

	if len(result.Pronunciations) != 1 {
		t.Fatalf("Pronunciations: got %d, want 1", len(result.Pronunciations))
	}

	p := result.Pronunciations[0]
	if p.Transcription == nil || *p.Transcription != "/kæt/" {
		t.Errorf("Transcription: got %v, want /kæt/", p.Transcription)
	}
	if p.Region != nil {
		t.Errorf("Region: got %v, want nil (empty region should map to nil)", p.Region)
	}
}

func TestToDomainEntries_SourceSlugConsistency(t *testing.T) {
	entries := []ParsedEntry{
		{
			Word: "fly",
			POSGroups: []POSGroup{
				{
					POS: "verb",
					Senses: []ParsedSense{
						{
							Glosses:      []string{"to move through air"},
							Translations: []string{"летать"},
							Examples:     []string{"Birds fly."},
						},
					},
				},
			},
			Sounds: []Sound{
				{IPA: "/flaɪ/", Region: "US"},
			},
		},
	}

	result := ToDomainEntries(entries)

	const want = "wiktionary"

	for i, s := range result.Senses {
		if s.SourceSlug != want {
			t.Errorf("Sense[%d] SourceSlug: got %q, want %q", i, s.SourceSlug, want)
		}
	}
	for i, tr := range result.Translations {
		if tr.SourceSlug != want {
			t.Errorf("Translation[%d] SourceSlug: got %q, want %q", i, tr.SourceSlug, want)
		}
	}
	for i, ex := range result.Examples {
		if ex.SourceSlug != want {
			t.Errorf("Example[%d] SourceSlug: got %q, want %q", i, ex.SourceSlug, want)
		}
	}
	for i, p := range result.Pronunciations {
		if p.SourceSlug != want {
			t.Errorf("Pronunciation[%d] SourceSlug: got %q, want %q", i, p.SourceSlug, want)
		}
	}
}

func TestToDomainEntries_TextNormalized(t *testing.T) {
	entries := []ParsedEntry{
		{
			Word: "  Hello  World  ",
			POSGroups: []POSGroup{
				{
					POS:    "noun",
					Senses: []ParsedSense{{Glosses: []string{"test"}}},
				},
			},
		},
	}

	result := ToDomainEntries(entries)

	if result.Entries[0].Text != "  Hello  World  " {
		t.Errorf("Text should be preserved as-is: got %q", result.Entries[0].Text)
	}
	if result.Entries[0].TextNormalized != "hello world" {
		t.Errorf("TextNormalized: got %q, want %q", result.Entries[0].TextNormalized, "hello world")
	}
}

func TestToDomainEntries_AllIDsNonZero(t *testing.T) {
	entries := []ParsedEntry{
		{
			Word: "test",
			POSGroups: []POSGroup{
				{
					POS: "noun",
					Senses: []ParsedSense{
						{
							Glosses:      []string{"a trial"},
							Translations: []string{"тест"},
							Examples:     []string{"Take a test."},
						},
					},
				},
			},
			Sounds: []Sound{{IPA: "/tɛst/", Region: "US"}},
		},
	}

	result := ToDomainEntries(entries)

	if result.Entries[0].ID == uuid.Nil {
		t.Error("Entry ID should not be zero")
	}
	if result.Senses[0].ID == uuid.Nil {
		t.Error("Sense ID should not be zero")
	}
	if result.Translations[0].ID == uuid.Nil {
		t.Error("Translation ID should not be zero")
	}
	if result.Examples[0].ID == uuid.Nil {
		t.Error("Example ID should not be zero")
	}
	if result.Pronunciations[0].ID == uuid.Nil {
		t.Error("Pronunciation ID should not be zero")
	}
}

func TestToDomainEntries_ForeignKeysCorrect(t *testing.T) {
	entries := []ParsedEntry{
		{
			Word: "go",
			POSGroups: []POSGroup{
				{
					POS: "verb",
					Senses: []ParsedSense{
						{
							Glosses:      []string{"to move"},
							Translations: []string{"идти", "ехать"},
							Examples:     []string{"Let's go."},
						},
						{
							Glosses:      []string{"to function"},
							Translations: []string{"работать"},
						},
					},
				},
			},
			Sounds: []Sound{{IPA: "/ɡoʊ/", Region: "US"}},
		},
	}

	result := ToDomainEntries(entries)

	entryID := result.Entries[0].ID

	// All senses reference the entry.
	for i, s := range result.Senses {
		if s.RefEntryID != entryID {
			t.Errorf("Sense[%d] RefEntryID: got %s, want %s", i, s.RefEntryID, entryID)
		}
	}

	// Pronunciations reference the entry.
	for i, p := range result.Pronunciations {
		if p.RefEntryID != entryID {
			t.Errorf("Pronunciation[%d] RefEntryID: got %s, want %s", i, p.RefEntryID, entryID)
		}
	}

	// Translations for first sense (2 translations).
	sense0ID := result.Senses[0].ID
	sense1ID := result.Senses[1].ID

	if len(result.Translations) != 3 {
		t.Fatalf("Translations: got %d, want 3", len(result.Translations))
	}

	// First two translations belong to sense 0.
	if result.Translations[0].RefSenseID != sense0ID {
		t.Errorf("Translation[0] RefSenseID: got %s, want %s", result.Translations[0].RefSenseID, sense0ID)
	}
	if result.Translations[1].RefSenseID != sense0ID {
		t.Errorf("Translation[1] RefSenseID: got %s, want %s", result.Translations[1].RefSenseID, sense0ID)
	}
	// Third translation belongs to sense 1.
	if result.Translations[2].RefSenseID != sense1ID {
		t.Errorf("Translation[2] RefSenseID: got %s, want %s", result.Translations[2].RefSenseID, sense1ID)
	}

	// Example belongs to first sense.
	if len(result.Examples) != 1 {
		t.Fatalf("Examples: got %d, want 1", len(result.Examples))
	}
	if result.Examples[0].RefSenseID != sense0ID {
		t.Errorf("Example[0] RefSenseID: got %s, want %s", result.Examples[0].RefSenseID, sense0ID)
	}
}

func TestToDomainEntries_DefinitionUsesFirstGlossOnly(t *testing.T) {
	entries := []ParsedEntry{
		{
			Word: "set",
			POSGroups: []POSGroup{
				{
					POS: "verb",
					Senses: []ParsedSense{
						{Glosses: []string{"to put in place", "to arrange"}},
					},
				},
			},
		},
	}

	result := ToDomainEntries(entries)

	if result.Senses[0].Definition != "to put in place" {
		t.Errorf("Definition should use first gloss only: got %q, want %q",
			result.Senses[0].Definition, "to put in place")
	}
}

func TestToDomainEntries_DefinitionMarkupStripped(t *testing.T) {
	entries := []ParsedEntry{
		{
			Word: "test",
			POSGroups: []POSGroup{
				{
					POS: "noun",
					Senses: []ParsedSense{
						{Glosses: []string{"a <b>formal</b> [[examination]]"}},
					},
				},
			},
		},
	}

	result := ToDomainEntries(entries)

	if result.Senses[0].Definition != "a formal examination" {
		t.Errorf("Definition markup not stripped: got %q, want %q",
			result.Senses[0].Definition, "a formal examination")
	}
}

func TestToDomainEntries_MultipleEntries(t *testing.T) {
	entries := []ParsedEntry{
		{
			Word: "cat",
			POSGroups: []POSGroup{
				{POS: "noun", Senses: []ParsedSense{{Glosses: []string{"a feline"}}}},
			},
		},
		{
			Word: "dog",
			POSGroups: []POSGroup{
				{POS: "noun", Senses: []ParsedSense{{Glosses: []string{"a canine"}}}},
			},
		},
	}

	result := ToDomainEntries(entries)

	if len(result.Entries) != 2 {
		t.Fatalf("Entries: got %d, want 2", len(result.Entries))
	}
	if len(result.Senses) != 2 {
		t.Fatalf("Senses: got %d, want 2", len(result.Senses))
	}

	// Each sense references its own entry.
	if result.Senses[0].RefEntryID != result.Entries[0].ID {
		t.Error("Sense[0] should reference Entry[0]")
	}
	if result.Senses[1].RefEntryID != result.Entries[1].ID {
		t.Error("Sense[1] should reference Entry[1]")
	}

	// Each entry gets its own ID.
	if result.Entries[0].ID == result.Entries[1].ID {
		t.Error("Entries should have different IDs")
	}
}
