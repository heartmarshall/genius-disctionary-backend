package enricher

import (
	"testing"

	"github.com/heartmarshall/myenglish-backend/internal/app/seeder/cmu"
	"github.com/heartmarshall/myenglish-backend/internal/app/seeder/wiktionary"
)

func TestBuildContext_fullWord(t *testing.T) {
	wiktMap := map[string]*wiktionary.ParsedEntry{
		"run": {
			Word: "run",
			POSGroups: []wiktionary.POSGroup{
				{
					POS: "verb",
					Senses: []wiktionary.ParsedSense{
						{Glosses: []string{"To move fast."}, Translations: []string{"бежать"}},
					},
				},
			},
			Sounds: []wiktionary.Sound{{IPA: "/rʌn/", Region: "US"}},
		},
	}

	relMap := map[string]map[string][]string{
		"run": {
			"synonym":  {"sprint", "jog"},
			"antonym":  {"walk"},
			"hypernym": {"move"},
		},
	}

	cmuResult := cmu.ParseResult{
		Pronunciations: map[string][]cmu.IPATranscription{
			"run": {{IPA: "/rʌn/", VariantIndex: 0}},
		},
	}

	ctx := BuildContext("run", wiktMap, relMap, cmuResult)

	if ctx.Word != "run" {
		t.Errorf("Word = %q, want %q", ctx.Word, "run")
	}
	if ctx.IPA != "/rʌn/" {
		t.Errorf("IPA = %q, want %q", ctx.IPA, "/rʌn/")
	}
	if len(ctx.WiktionarySenses) != 1 {
		t.Fatalf("len(WiktionarySenses) = %d, want 1", len(ctx.WiktionarySenses))
	}
	if ctx.WiktionarySenses[0].Definition != "To move fast." {
		t.Errorf("Sense.Definition = %q", ctx.WiktionarySenses[0].Definition)
	}
	if len(ctx.WiktionarySenses[0].Translations) != 1 || ctx.WiktionarySenses[0].Translations[0] != "бежать" {
		t.Errorf("Sense.Translations = %v", ctx.WiktionarySenses[0].Translations)
	}
	if len(ctx.Relations.Synonyms) != 2 {
		t.Errorf("Synonyms = %v", ctx.Relations.Synonyms)
	}
	if len(ctx.Relations.Antonyms) != 1 {
		t.Errorf("Antonyms = %v", ctx.Relations.Antonyms)
	}
	if len(ctx.Relations.Hypernyms) != 1 {
		t.Errorf("Hypernyms = %v, want 1 element", ctx.Relations.Hypernyms)
	}
}

func TestBuildContext_unknownWord(t *testing.T) {
	ctx := BuildContext("xyzzy",
		map[string]*wiktionary.ParsedEntry{},
		map[string]map[string][]string{},
		cmu.ParseResult{Pronunciations: map[string][]cmu.IPATranscription{}},
	)
	if ctx.Word != "xyzzy" {
		t.Errorf("Word = %q", ctx.Word)
	}
	if len(ctx.WiktionarySenses) != 0 {
		t.Errorf("expected no senses for unknown word")
	}
	if ctx.IPA != "" {
		t.Errorf("expected empty IPA for unknown word")
	}
	if len(ctx.Relations.Synonyms) != 0 {
		t.Errorf("expected no Synonyms for unknown word, got %v", ctx.Relations.Synonyms)
	}
	if len(ctx.Relations.Antonyms) != 0 {
		t.Errorf("expected no Antonyms for unknown word, got %v", ctx.Relations.Antonyms)
	}
	if len(ctx.Relations.Hypernyms) != 0 {
		t.Errorf("expected no Hypernyms for unknown word, got %v", ctx.Relations.Hypernyms)
	}
}
