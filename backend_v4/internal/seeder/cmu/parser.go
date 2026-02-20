// Package cmu parses CMU Pronouncing Dictionary files into IPA transcriptions.
// Pure function: file path in, domain structs out. No database dependencies.
package cmu

import (
	"bufio"
	"errors"
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/google/uuid"
	"github.com/heartmarshall/myenglish-backend/internal/domain"
)

const (
	sourceSlug = "cmu"
	region     = "US"
)

// errSkipLine signals that a line should be skipped (comment, empty, etc.).
var errSkipLine = errors.New("skip line")

// arpabetMap maps ARPAbet phonemes (without stress markers) to IPA symbols.
var arpabetMap = map[string]string{
	"AA": "\u0251",     // ɑ
	"AE": "\u00e6",     // æ
	"AH": "\u028c",     // ʌ
	"AO": "\u0254",     // ɔ
	"AW": "a\u028a",    // aʊ
	"AY": "a\u026a",    // aɪ
	"B":  "b",
	"CH": "t\u0283",    // tʃ
	"D":  "d",
	"DH": "\u00f0",     // ð
	"EH": "\u025b",     // ɛ
	"ER": "\u025d",     // ɝ
	"EY": "e\u026a",    // eɪ
	"F":  "f",
	"G":  "\u0261",     // ɡ
	"HH": "h",
	"IH": "\u026a",     // ɪ
	"IY": "i",
	"JH": "d\u0292",    // dʒ
	"K":  "k",
	"L":  "l",
	"M":  "m",
	"N":  "n",
	"NG": "\u014b",     // ŋ
	"OW": "o\u028a",    // oʊ
	"OY": "\u0254\u026a", // ɔɪ
	"P":  "p",
	"R":  "\u0279",     // ɹ
	"S":  "s",
	"SH": "\u0283",     // ʃ
	"T":  "t",
	"TH": "\u03b8",     // θ
	"UH": "\u028a",     // ʊ
	"UW": "u",
	"V":  "v",
	"W":  "w",
	"Y":  "j",
	"Z":  "z",
	"ZH": "\u0292",     // ʒ
}

// IPATranscription holds a single IPA transcription with its variant index.
type IPATranscription struct {
	IPA          string // e.g., "/haʊs/"
	VariantIndex int    // 0 for primary, 1 for (2), 2 for (3), etc.
}

// ParseResult holds the parsed CMU dictionary data.
type ParseResult struct {
	Pronunciations map[string][]IPATranscription // normalizedWord → pronunciations
	Stats          Stats
}

// Stats holds parser statistics for logging.
type Stats struct {
	TotalLines   int
	CommentLines int
	ParsedLines  int
	UniqueWords  int
}

// Parse reads a CMU dict file and returns parsed pronunciations.
func Parse(filePath string) (ParseResult, error) {
	f, err := os.Open(filePath)
	if err != nil {
		return ParseResult{}, fmt.Errorf("open file: %w", err)
	}
	defer f.Close()

	result := ParseResult{
		Pronunciations: make(map[string][]IPATranscription),
	}

	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		result.Stats.TotalLines++
		line := scanner.Text()

		word, ipa, err := parseLine(line)
		if err == errSkipLine {
			if strings.HasPrefix(line, ";;;") {
				result.Stats.CommentLines++
			}
			continue
		}
		if err != nil {
			continue
		}

		result.Stats.ParsedLines++
		result.Pronunciations[word] = append(result.Pronunciations[word], ipa)
	}

	if err := scanner.Err(); err != nil {
		return ParseResult{}, fmt.Errorf("scanner error: %w", err)
	}

	result.Stats.UniqueWords = len(result.Pronunciations)
	return result, nil
}

// ToDomainPronunciations converts parsed CMU data to domain RefPronunciation records.
// Only words present in entryIDMap are included.
// source_slug = "cmu", region = "US" (CMU is American English).
func (r ParseResult) ToDomainPronunciations(entryIDMap map[string]uuid.UUID) []domain.RefPronunciation {
	if len(entryIDMap) == 0 {
		return nil
	}

	var result []domain.RefPronunciation
	regionStr := region

	for word, entryID := range entryIDMap {
		ipas, ok := r.Pronunciations[word]
		if !ok {
			continue
		}

		for _, ipa := range ipas {
			transcription := ipa.IPA
			result = append(result, domain.RefPronunciation{
				ID:            uuid.New(),
				RefEntryID:    entryID,
				Transcription: &transcription,
				AudioURL:      nil,
				Region:        &regionStr,
				SourceSlug:    sourceSlug,
			})
		}
	}

	return result
}

// arpabetToIPA converts an ARPAbet phoneme (without stress) to its IPA equivalent.
func arpabetToIPA(phoneme string) (string, bool) {
	ipa, ok := arpabetMap[phoneme]
	return ipa, ok
}

// stripStress removes the trailing stress marker (0, 1, 2) from an ARPAbet phoneme.
func stripStress(phoneme string) string {
	if len(phoneme) == 0 {
		return phoneme
	}
	last := phoneme[len(phoneme)-1]
	if last == '0' || last == '1' || last == '2' {
		return phoneme[:len(phoneme)-1]
	}
	return phoneme
}

// phonemesToIPA converts a slice of ARPAbet phonemes to an IPA transcription string.
// Stress markers are stripped before lookup. Result is wrapped in slashes.
func phonemesToIPA(phonemes []string) string {
	var b strings.Builder
	b.WriteByte('/')
	for _, p := range phonemes {
		stripped := stripStress(p)
		ipa, ok := arpabetToIPA(stripped)
		if ok {
			b.WriteString(ipa)
		}
	}
	b.WriteByte('/')
	return b.String()
}

// parseLine parses a single line from a CMU dict file.
// Returns the normalized word, an IPATranscription, or errSkipLine for comments/empty lines.
func parseLine(line string) (string, IPATranscription, error) {
	// Skip empty lines.
	if line == "" {
		return "", IPATranscription{}, errSkipLine
	}

	// Skip comment lines.
	if strings.HasPrefix(line, ";;;") {
		return "", IPATranscription{}, errSkipLine
	}

	// CMU format: WORD  PHONEME1 PHONEME2 ... (two spaces between word and phonemes).
	parts := strings.SplitN(line, "  ", 2)
	if len(parts) != 2 {
		return "", IPATranscription{}, errSkipLine
	}

	rawWord := strings.TrimSpace(parts[0])
	phonemesStr := strings.TrimSpace(parts[1])

	if rawWord == "" || phonemesStr == "" {
		return "", IPATranscription{}, errSkipLine
	}

	word, variantIdx := parseWordAndVariant(rawWord)
	phonemes := strings.Fields(phonemesStr)
	ipa := phonemesToIPA(phonemes)

	return word, IPATranscription{
		IPA:          ipa,
		VariantIndex: variantIdx,
	}, nil
}

// parseWordAndVariant splits a raw CMU word like "HOUSE(2)" into
// the normalized word and variant index.
// Primary pronunciation has variant index 0, "(2)" maps to 1, "(3)" to 2, etc.
func parseWordAndVariant(raw string) (string, int) {
	idx := strings.IndexByte(raw, '(')
	if idx == -1 {
		return domain.NormalizeText(raw), 0
	}

	word := raw[:idx]
	// Extract number between parentheses.
	end := strings.IndexByte(raw[idx:], ')')
	if end == -1 {
		return domain.NormalizeText(raw), 0
	}

	numStr := raw[idx+1 : idx+end]
	n, err := strconv.Atoi(numStr)
	if err != nil {
		return domain.NormalizeText(raw), 0
	}

	// (2) → variant index 1, (3) → variant index 2, etc.
	return domain.NormalizeText(word), n - 1
}

