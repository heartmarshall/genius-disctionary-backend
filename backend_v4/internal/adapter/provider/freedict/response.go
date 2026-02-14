package freedict

// apiEntry represents a single entry from the FreeDictionary API response.
// The API returns an array of entries (one per etymology).
type apiEntry struct {
	Word      string        `json:"word"`
	Phonetics []apiPhonetic `json:"phonetics"`
	Meanings  []apiMeaning  `json:"meanings"`
}

// apiPhonetic represents phonetic/pronunciation data from the API.
type apiPhonetic struct {
	Text  string `json:"text"`
	Audio string `json:"audio"`
}

// apiMeaning represents a group of definitions sharing a part of speech.
type apiMeaning struct {
	PartOfSpeech string          `json:"partOfSpeech"`
	Definitions  []apiDefinition `json:"definitions"`
}

// apiDefinition represents a single definition with an optional example.
type apiDefinition struct {
	Definition string `json:"definition"`
	Example    string `json:"example"`
}
