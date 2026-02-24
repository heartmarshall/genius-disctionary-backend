// Package wiktionary parses Kaikki JSONL dumps into domain structs.
// Pure function: file path in, domain structs out. No database dependencies.
package wiktionary

// ParsedEntry is the intermediate struct produced by the two-pass parser.
// It groups all POS lines for the same normalized word.
type ParsedEntry struct {
	Word      string
	Score     float64
	POSGroups []POSGroup // one per Kaikki line
	Sounds    []Sound
}

// POSGroup holds senses from a single Kaikki line (one POS).
type POSGroup struct {
	POS    string
	Senses []ParsedSense
}

// ParsedSense holds the useful data extracted from a Kaikki sense.
type ParsedSense struct {
	Glosses      []string
	Examples     []string
	Translations []string // Russian only
}

// Sound holds a single IPA pronunciation with optional region.
type Sound struct {
	IPA    string
	Region string // "US", "UK", or ""
}

// Stats holds parser statistics for logging.
type Stats struct {
	TotalLines    int
	MalformedLines int
	EnglishLines  int
	EntriesParsed int
}

// kaikkiEntry mirrors the Kaikki JSONL structure (only fields we need).
type kaikkiEntry struct {
	Word   string        `json:"word"`
	POS    string        `json:"pos"`
	Lang   string        `json:"lang"`
	Senses []kaikkiSense `json:"senses"`
	Sounds []kaikkiSound `json:"sounds"`
}

// kaikkiSense mirrors one sense from a Kaikki entry.
type kaikkiSense struct {
	Glosses      []string             `json:"glosses"`
	Examples     []kaikkiExample      `json:"examples"`
	Translations []kaikkiTranslation  `json:"translations"`
}

// kaikkiExample mirrors an example from Kaikki.
type kaikkiExample struct {
	Text string `json:"text"`
}

// kaikkiTranslation mirrors a translation from Kaikki.
type kaikkiTranslation struct {
	Code string `json:"code"`
	Word string `json:"word"`
}

// kaikkiSound mirrors a sound entry from Kaikki.
type kaikkiSound struct {
	IPA  string   `json:"ipa"`
	Tags []string `json:"tags"`
}
