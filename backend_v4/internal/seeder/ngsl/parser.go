// Package ngsl parses NGSL and NAWL CSV files into domain metadata updates.
// Pure function: file paths in, domain structs out. No database dependencies.
package ngsl

import (
	"encoding/csv"
	"fmt"
	"io"
	"os"

	"github.com/heartmarshall/myenglish-backend/internal/domain"
)

// Parse reads NGSL and NAWL CSV files and returns combined metadata updates
// and a set of all core words (normalized).
func Parse(ngslPath, nawlPath string) ([]domain.EntryMetadataUpdate, map[string]bool, error) {
	ngslFile, err := os.Open(ngslPath)
	if err != nil {
		return nil, nil, fmt.Errorf("open NGSL file: %w", err)
	}
	defer ngslFile.Close()

	ngslEntries, err := parseNGSL(ngslFile)
	if err != nil {
		return nil, nil, fmt.Errorf("parse NGSL: %w", err)
	}

	nawlFile, err := os.Open(nawlPath)
	if err != nil {
		return nil, nil, fmt.Errorf("open NAWL file: %w", err)
	}
	defer nawlFile.Close()

	nawlEntries, err := parseNAWL(nawlFile)
	if err != nil {
		return nil, nil, fmt.Errorf("parse NAWL: %w", err)
	}

	updates := make([]domain.EntryMetadataUpdate, 0, len(ngslEntries)+len(nawlEntries))
	updates = append(updates, ngslEntries...)
	updates = append(updates, nawlEntries...)

	coreWords := make(map[string]bool, len(updates))
	for i := range updates {
		coreWords[updates[i].TextNormalized] = true
	}

	return updates, coreWords, nil
}

// parseNGSL reads an NGSL CSV from the given reader. Each non-empty row after
// the header becomes an entry with a 1-based frequency rank and a CEFR level
// derived from the rank.
func parseNGSL(r io.Reader) ([]domain.EntryMetadataUpdate, error) {
	reader := csv.NewReader(r)
	reader.FieldsPerRecord = -1 // allow variable column count

	// Skip header row.
	if _, err := reader.Read(); err != nil {
		if err == io.EOF {
			return nil, nil
		}
		return nil, fmt.Errorf("read header: %w", err)
	}

	coreLexicon := true
	var entries []domain.EntryMetadataUpdate
	rank := 0

	for {
		record, err := reader.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("read row: %w", err)
		}

		if len(record) == 0 {
			continue
		}

		word := domain.NormalizeText(record[0])
		if word == "" {
			continue
		}

		rank++
		freqRank := rank
		cefr := cefrForRank(rank)

		entries = append(entries, domain.EntryMetadataUpdate{
			TextNormalized: word,
			FrequencyRank:  &freqRank,
			CEFRLevel:      &cefr,
			IsCoreLexicon:  &coreLexicon,
		})
	}

	return entries, nil
}

// parseNAWL reads a NAWL CSV from the given reader. All entries get CEFR "C1",
// no frequency rank, and IsCoreLexicon = true.
func parseNAWL(r io.Reader) ([]domain.EntryMetadataUpdate, error) {
	reader := csv.NewReader(r)
	reader.FieldsPerRecord = -1

	// Skip header row.
	if _, err := reader.Read(); err != nil {
		if err == io.EOF {
			return nil, nil
		}
		return nil, fmt.Errorf("read header: %w", err)
	}

	coreLexicon := true
	cefrC1 := "C1"
	var entries []domain.EntryMetadataUpdate

	for {
		record, err := reader.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("read row: %w", err)
		}

		if len(record) == 0 {
			continue
		}

		word := domain.NormalizeText(record[0])
		if word == "" {
			continue
		}

		entries = append(entries, domain.EntryMetadataUpdate{
			TextNormalized: word,
			FrequencyRank:  nil,
			CEFRLevel:      &cefrC1,
			IsCoreLexicon:  &coreLexicon,
		})
	}

	return entries, nil
}

// cefrForRank maps a 1-based NGSL frequency rank to a CEFR level.
//
//	1-500   → A1
//	501-1200  → A2
//	1201-2000 → B1
//	2001+     → B2
func cefrForRank(rank int) string {
	switch {
	case rank <= 500:
		return "A1"
	case rank <= 1200:
		return "A2"
	case rank <= 2000:
		return "B1"
	default:
		return "B2"
	}
}
