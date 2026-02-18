package dictionary

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/heartmarshall/myenglish-backend/internal/domain"
	"github.com/heartmarshall/myenglish-backend/pkg/ctxutil"
)

// ---------------------------------------------------------------------------
// 12. ImportEntries
// ---------------------------------------------------------------------------

// ImportEntries imports entries from an external source. Per-chunk transactions.
func (s *Service) ImportEntries(ctx context.Context, input ImportInput) (*ImportResult, error) {
	userID, ok := ctxutil.UserIDFromCtx(ctx)
	if !ok {
		return nil, domain.ErrUnauthorized
	}

	if err := input.Validate(); err != nil {
		return nil, err
	}

	// Check limit: current + new items.
	count, err := s.entries.CountByUser(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("count entries: %w", err)
	}
	if count+len(input.Items) > s.cfg.MaxEntriesPerUser {
		return nil, domain.NewValidationError("items", "importing these items would exceed entry limit")
	}

	const sourceSlug = "import"

	result := &ImportResult{}
	seen := make(map[string]bool)

	// Split into chunks.
	chunkSize := s.cfg.ImportChunkSize
	if chunkSize <= 0 {
		chunkSize = 50
	}

	for chunkStart := 0; chunkStart < len(input.Items); chunkStart += chunkSize {
		chunkEnd := chunkStart + chunkSize
		if chunkEnd > len(input.Items) {
			chunkEnd = len(input.Items)
		}
		chunk := input.Items[chunkStart:chunkEnd]

		// Track per-chunk results separately so we can revert on tx failure.
		var chunkImported int
		var chunkSkipped int
		var chunkErrors []ImportError
		// Track texts added to "seen" in this chunk, so we can remove them on failure.
		var chunkSeenTexts []string

		txErr := s.tx.RunInTx(ctx, func(txCtx context.Context) error {
			for i, item := range chunk {
				lineNumber := chunkStart + i + 1 // 1-based

				normalized := domain.NormalizeText(item.Text)
				if normalized == "" {
					chunkErrors = append(chunkErrors, ImportError{
						LineNumber: lineNumber,
						Text:       item.Text,
						Reason:     "empty text after normalization",
					})
					chunkSkipped++
					continue
				}

				// Deduplicate within file.
				if seen[normalized] {
					chunkErrors = append(chunkErrors, ImportError{
						LineNumber: lineNumber,
						Text:       item.Text,
						Reason:     "duplicate within import",
					})
					chunkSkipped++
					continue
				}

				// Check if entry already exists.
				_, getErr := s.entries.GetByText(txCtx, userID, normalized)
				if getErr == nil {
					chunkErrors = append(chunkErrors, ImportError{
						LineNumber: lineNumber,
						Text:       item.Text,
						Reason:     "entry already exists",
					})
					chunkSkipped++
					seen[normalized] = true
					chunkSeenTexts = append(chunkSeenTexts, normalized)
					continue
				}
				if !errors.Is(getErr, domain.ErrNotFound) {
					return fmt.Errorf("check duplicate: %w", getErr)
				}

				seen[normalized] = true
				chunkSeenTexts = append(chunkSeenTexts, normalized)

				now := time.Now().UTC()
				entry := &domain.Entry{
					ID:             uuid.New(),
					UserID:         userID,
					Text:           item.Text,
					TextNormalized: normalized,
					Notes:          item.Notes,
					CreatedAt:      now,
					UpdatedAt:      now,
				}

				created, createErr := s.entries.Create(txCtx, entry)
				if createErr != nil {
					return fmt.Errorf("create entry: %w", createErr)
				}

				// If translations provided, create a single sense with them.
				if len(item.Translations) > 0 {
					sense, senseErr := s.senses.CreateCustom(txCtx, created.ID, nil, nil, nil, sourceSlug)
					if senseErr != nil {
						return fmt.Errorf("create sense: %w", senseErr)
					}

					for _, tr := range item.Translations {
						if _, trErr := s.translations.CreateCustom(txCtx, sense.ID, tr, sourceSlug); trErr != nil {
							return fmt.Errorf("create translation: %w", trErr)
						}
					}
				}

				chunkImported++
			}
			return nil
		})

		if txErr != nil {
			// Chunk failed — the entire chunk is rolled back.
			// Remove all seen texts from this chunk since tx rolled back.
			for _, text := range chunkSeenTexts {
				delete(seen, text)
			}

			// Mark all items in the chunk as errors.
			for i, item := range chunk {
				lineNumber := chunkStart + i + 1
				result.Errors = append(result.Errors, ImportError{
					LineNumber: lineNumber,
					Text:       item.Text,
					Reason:     "chunk transaction failed: " + txErr.Error(),
				})
			}
			result.Skipped += len(chunk)
		} else {
			// Chunk succeeded — commit the per-chunk results.
			result.Imported += chunkImported
			result.Skipped += chunkSkipped
			result.Errors = append(result.Errors, chunkErrors...)
		}
	}

	return result, nil
}
