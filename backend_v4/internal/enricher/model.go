package enricher

// EnrichContext is the per-word context file written to enrich-output/<word>.json.
// It is consumed by the LLM as input for generating LLMWordEntry JSON.
type EnrichContext struct {
	Word             string            `json:"word"`
	IPA              string            `json:"ipa,omitempty"`
	WiktionarySenses []WiktionarySense `json:"wiktionary_senses,omitempty"`
	Relations        Relations         `json:"relations,omitempty"`
}

// WiktionarySense holds one sense from Wiktionary for the LLM context.
type WiktionarySense struct {
	POS          string   `json:"pos"`
	Definition   string   `json:"definition"`
	Translations []string `json:"translations_ru,omitempty"`
}

// Relations holds semantic relations for a word from WordNet.
type Relations struct {
	Synonyms  []string `json:"synonyms,omitempty"`
	Antonyms  []string `json:"antonyms,omitempty"`
	Hypernyms []string `json:"hypernyms,omitempty"`
}
