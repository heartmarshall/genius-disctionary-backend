package llm_importer

// LLMWordEntry is the top-level JSON document produced by the LLM.
// One file per word: llm-output/<word>.json
type LLMWordEntry struct {
	Word       string     `json:"word"`
	SourceSlug string     `json:"source_slug"`
	Senses     []LLMSense `json:"senses"`
}

// LLMSense is one sense within an LLM word entry.
type LLMSense struct {
	POS          string       `json:"pos"`
	Definition   string       `json:"definition"`
	CEFRLevel    string       `json:"cefr_level,omitempty"`
	Notes        string       `json:"notes,omitempty"`
	Translations []string     `json:"translations"`
	Examples     []LLMExample `json:"examples,omitempty"`
}

// LLMExample is one usage example within a sense.
type LLMExample struct {
	Sentence    string `json:"sentence"`
	Translation string `json:"translation,omitempty"`
}
