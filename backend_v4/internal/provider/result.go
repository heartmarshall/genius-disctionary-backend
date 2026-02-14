package provider

// DictionaryResult is the structured result from a dictionary API provider.
type DictionaryResult struct {
	Word           string
	Senses         []SenseResult
	Pronunciations []PronunciationResult
}

// SenseResult represents a single word sense from an external dictionary.
type SenseResult struct {
	Definition   string
	PartOfSpeech *string
	Examples     []ExampleResult
}

// ExampleResult represents a usage example from an external dictionary.
type ExampleResult struct {
	Sentence    string
	Translation *string
}

// PronunciationResult represents pronunciation data from an external dictionary.
type PronunciationResult struct {
	Transcription *string
	AudioURL      *string
	Region        *string
}
