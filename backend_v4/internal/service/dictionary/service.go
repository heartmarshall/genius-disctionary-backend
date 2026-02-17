package dictionary

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/google/uuid"
	"github.com/heartmarshall/myenglish-backend/internal/config"
	"github.com/heartmarshall/myenglish-backend/internal/domain"
	"github.com/heartmarshall/myenglish-backend/pkg/ctxutil"
)

// ---------------------------------------------------------------------------
// Consumer-defined interfaces (private)
// ---------------------------------------------------------------------------

type entryRepo interface {
	GetByID(ctx context.Context, userID, entryID uuid.UUID) (*domain.Entry, error)
	GetByText(ctx context.Context, userID uuid.UUID, textNormalized string) (*domain.Entry, error)
	GetByIDs(ctx context.Context, userID uuid.UUID, ids []uuid.UUID) ([]domain.Entry, error)
	Find(ctx context.Context, userID uuid.UUID, filter domain.EntryFilter) ([]domain.Entry, int, error)
	FindCursor(ctx context.Context, userID uuid.UUID, filter domain.EntryFilter) ([]domain.Entry, bool, error)
	FindDeleted(ctx context.Context, userID uuid.UUID, limit, offset int) ([]domain.Entry, int, error)
	CountByUser(ctx context.Context, userID uuid.UUID) (int, error)
	Create(ctx context.Context, entry *domain.Entry) (*domain.Entry, error)
	UpdateNotes(ctx context.Context, userID, entryID uuid.UUID, notes *string) (*domain.Entry, error)
	SoftDelete(ctx context.Context, userID, entryID uuid.UUID) error
	Restore(ctx context.Context, userID, entryID uuid.UUID) (*domain.Entry, error)
	HardDeleteOld(ctx context.Context, threshold time.Time) (int64, error)
}

type senseRepo interface {
	GetByEntryIDs(ctx context.Context, entryIDs []uuid.UUID) ([]domain.Sense, error)
	CreateFromRef(ctx context.Context, entryID, refSenseID uuid.UUID, sourceSlug string) (*domain.Sense, error)
	CreateCustom(ctx context.Context, entryID uuid.UUID, definition *string, pos *domain.PartOfSpeech, cefr *string, sourceSlug string) (*domain.Sense, error)
}

type translationRepo interface {
	GetBySenseIDs(ctx context.Context, senseIDs []uuid.UUID) ([]domain.Translation, error)
	CreateFromRef(ctx context.Context, senseID, refTranslationID uuid.UUID, sourceSlug string) (*domain.Translation, error)
	CreateCustom(ctx context.Context, senseID uuid.UUID, text string, sourceSlug string) (*domain.Translation, error)
}

type exampleRepo interface {
	GetBySenseIDs(ctx context.Context, senseIDs []uuid.UUID) ([]domain.Example, error)
	CreateFromRef(ctx context.Context, senseID, refExampleID uuid.UUID, sourceSlug string) (*domain.Example, error)
	CreateCustom(ctx context.Context, senseID uuid.UUID, sentence string, translation *string, sourceSlug string) (*domain.Example, error)
}

type pronunciationRepo interface {
	Link(ctx context.Context, entryID, refPronunciationID uuid.UUID) error
}

type imageRepo interface {
	LinkCatalog(ctx context.Context, entryID, refImageID uuid.UUID) error
}

type cardRepo interface {
	GetByEntryIDs(ctx context.Context, entryIDs []uuid.UUID) ([]domain.Card, error)
	Create(ctx context.Context, userID, entryID uuid.UUID, status domain.LearningStatus, easeFactor float64) (*domain.Card, error)
}

type auditRepo interface {
	Create(ctx context.Context, record domain.AuditRecord) error
}

type txManager interface {
	RunInTx(ctx context.Context, fn func(ctx context.Context) error) error
}

type refCatalogService interface {
	GetOrFetchEntry(ctx context.Context, text string) (*domain.RefEntry, error)
	GetRefEntry(ctx context.Context, refEntryID uuid.UUID) (*domain.RefEntry, error)
	Search(ctx context.Context, query string, limit int) ([]domain.RefEntry, error)
}

// ---------------------------------------------------------------------------
// Service
// ---------------------------------------------------------------------------

// Service implements the dictionary business logic.
type Service struct {
	log            *slog.Logger
	entries        entryRepo
	senses         senseRepo
	translations   translationRepo
	examples       exampleRepo
	pronunciations pronunciationRepo
	images         imageRepo
	cards          cardRepo
	audit          auditRepo
	tx             txManager
	refCatalog     refCatalogService
	cfg            config.DictionaryConfig
}

// NewService creates a new Dictionary service.
func NewService(
	logger *slog.Logger,
	entries entryRepo,
	senses senseRepo,
	translations translationRepo,
	examples exampleRepo,
	pronunciations pronunciationRepo,
	images imageRepo,
	cards cardRepo,
	audit auditRepo,
	tx txManager,
	refCatalog refCatalogService,
	cfg config.DictionaryConfig,
) *Service {
	return &Service{
		log:            logger.With("service", "dictionary"),
		entries:        entries,
		senses:         senses,
		translations:   translations,
		examples:       examples,
		pronunciations: pronunciations,
		images:         images,
		cards:          cards,
		audit:          audit,
		tx:             tx,
		refCatalog:     refCatalog,
		cfg:            cfg,
	}
}

// ---------------------------------------------------------------------------
// 1. SearchCatalog
// ---------------------------------------------------------------------------

// SearchCatalog searches the reference catalog for entries matching the query.
func (s *Service) SearchCatalog(ctx context.Context, query string, limit int) ([]domain.RefEntry, error) {
	if _, ok := ctxutil.UserIDFromCtx(ctx); !ok {
		return nil, domain.ErrUnauthorized
	}

	if query == "" {
		return []domain.RefEntry{}, nil
	}

	limit = clampLimit(limit, 1, 50, 20)

	return s.refCatalog.Search(ctx, query, limit)
}

// ---------------------------------------------------------------------------
// 2. PreviewRefEntry
// ---------------------------------------------------------------------------

// PreviewRefEntry fetches or retrieves a reference entry by text.
func (s *Service) PreviewRefEntry(ctx context.Context, text string) (*domain.RefEntry, error) {
	if _, ok := ctxutil.UserIDFromCtx(ctx); !ok {
		return nil, domain.ErrUnauthorized
	}

	return s.refCatalog.GetOrFetchEntry(ctx, text)
}

// ---------------------------------------------------------------------------
// 3. CreateEntryFromCatalog
// ---------------------------------------------------------------------------

// CreateEntryFromCatalog creates a new dictionary entry from a reference catalog entry.
func (s *Service) CreateEntryFromCatalog(ctx context.Context, input CreateFromCatalogInput) (*domain.Entry, error) {
	userID, ok := ctxutil.UserIDFromCtx(ctx)
	if !ok {
		return nil, domain.ErrUnauthorized
	}

	if err := input.Validate(); err != nil {
		return nil, err
	}

	// Get reference entry.
	refEntry, err := s.refCatalog.GetRefEntry(ctx, input.RefEntryID)
	if err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			return nil, domain.NewValidationError("ref_entry_id", "reference entry not found")
		}
		return nil, fmt.Errorf("get ref entry: %w", err)
	}

	// Check entry limit.
	count, err := s.entries.CountByUser(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("count entries: %w", err)
	}
	if count >= s.cfg.MaxEntriesPerUser {
		return nil, domain.NewValidationError("entries", "limit reached")
	}

	// Duplicate check.
	_, err = s.entries.GetByText(ctx, userID, refEntry.TextNormalized)
	if err == nil {
		return nil, domain.ErrAlreadyExists
	}
	if !errors.Is(err, domain.ErrNotFound) {
		return nil, fmt.Errorf("check duplicate: %w", err)
	}

	// Determine which senses to use.
	var selectedSenses []domain.RefSense
	if len(input.SenseIDs) == 0 {
		// Use all senses from the reference entry.
		selectedSenses = refEntry.Senses
	} else {
		// Filter to only requested senses, preserving order.
		senseMap := make(map[uuid.UUID]domain.RefSense, len(refEntry.Senses))
		for _, rs := range refEntry.Senses {
			senseMap[rs.ID] = rs
		}
		for _, senseID := range input.SenseIDs {
			rs, found := senseMap[senseID]
			if !found {
				return nil, domain.NewValidationError("sense_ids", "sense not found: "+senseID.String())
			}
			selectedSenses = append(selectedSenses, rs)
		}
	}

	// Create entry in transaction.
	var created *domain.Entry
	txErr := s.tx.RunInTx(ctx, func(txCtx context.Context) error {
		entry := &domain.Entry{
			UserID:         userID,
			RefEntryID:     &refEntry.ID,
			Text:           refEntry.Text,
			TextNormalized: refEntry.TextNormalized,
			Notes:          input.Notes,
		}

		var createErr error
		created, createErr = s.entries.Create(txCtx, entry)
		if createErr != nil {
			return fmt.Errorf("create entry: %w", createErr)
		}

		// Create senses and their children.
		for _, rs := range selectedSenses {
			sense, senseErr := s.senses.CreateFromRef(txCtx, created.ID, rs.ID, rs.SourceSlug)
			if senseErr != nil {
				return fmt.Errorf("create sense from ref: %w", senseErr)
			}

			// Translations for this sense.
			for _, rt := range rs.Translations {
				if _, trErr := s.translations.CreateFromRef(txCtx, sense.ID, rt.ID, rt.SourceSlug); trErr != nil {
					return fmt.Errorf("create translation from ref: %w", trErr)
				}
			}

			// Examples for this sense.
			for _, re := range rs.Examples {
				if _, exErr := s.examples.CreateFromRef(txCtx, sense.ID, re.ID, re.SourceSlug); exErr != nil {
					return fmt.Errorf("create example from ref: %w", exErr)
				}
			}
		}

		// Link pronunciations.
		for _, rp := range refEntry.Pronunciations {
			if linkErr := s.pronunciations.Link(txCtx, created.ID, rp.ID); linkErr != nil {
				return fmt.Errorf("link pronunciation: %w", linkErr)
			}
		}

		// Link images.
		for _, ri := range refEntry.Images {
			if linkErr := s.images.LinkCatalog(txCtx, created.ID, ri.ID); linkErr != nil {
				return fmt.Errorf("link image: %w", linkErr)
			}
		}

		// Create card if requested.
		if input.CreateCard {
			if _, cardErr := s.cards.Create(txCtx, userID, created.ID, domain.LearningStatusNew, s.cfg.DefaultEaseFactor); cardErr != nil {
				return fmt.Errorf("create card: %w", cardErr)
			}
		}

		// Audit.
		auditErr := s.audit.Create(txCtx, domain.AuditRecord{
			UserID:     userID,
			EntityType: domain.EntityTypeEntry,
			EntityID:   &created.ID,
			Action:     domain.AuditActionCreate,
			Changes:    map[string]any{"text": created.Text, "source": "catalog"},
		})
		if auditErr != nil {
			return fmt.Errorf("audit create: %w", auditErr)
		}

		return nil
	})

	if txErr != nil {
		// Handle unique constraint violation from concurrent create.
		if errors.Is(txErr, domain.ErrAlreadyExists) {
			return nil, domain.ErrAlreadyExists
		}
		return nil, txErr
	}

	return created, nil
}

// ---------------------------------------------------------------------------
// 4. CreateEntryCustom
// ---------------------------------------------------------------------------

// CreateEntryCustom creates a new custom dictionary entry.
func (s *Service) CreateEntryCustom(ctx context.Context, input CreateCustomInput) (*domain.Entry, error) {
	userID, ok := ctxutil.UserIDFromCtx(ctx)
	if !ok {
		return nil, domain.ErrUnauthorized
	}

	if err := input.Validate(); err != nil {
		return nil, err
	}

	normalized := domain.NormalizeText(input.Text)
	if normalized == "" {
		return nil, domain.NewValidationError("text", "required")
	}

	// Check entry limit.
	count, err := s.entries.CountByUser(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("count entries: %w", err)
	}
	if count >= s.cfg.MaxEntriesPerUser {
		return nil, domain.NewValidationError("entries", "limit reached")
	}

	// Duplicate check.
	_, err = s.entries.GetByText(ctx, userID, normalized)
	if err == nil {
		return nil, domain.ErrAlreadyExists
	}
	if !errors.Is(err, domain.ErrNotFound) {
		return nil, fmt.Errorf("check duplicate: %w", err)
	}

	const sourceSlug = "user"

	var created *domain.Entry
	txErr := s.tx.RunInTx(ctx, func(txCtx context.Context) error {
		entry := &domain.Entry{
			UserID:         userID,
			Text:           input.Text,
			TextNormalized: normalized,
			Notes:          input.Notes,
		}

		var createErr error
		created, createErr = s.entries.Create(txCtx, entry)
		if createErr != nil {
			return fmt.Errorf("create entry: %w", createErr)
		}

		// Create senses and their children.
		for _, si := range input.Senses {
			sense, senseErr := s.senses.CreateCustom(txCtx, created.ID, si.Definition, si.PartOfSpeech, nil, sourceSlug)
			if senseErr != nil {
				return fmt.Errorf("create custom sense: %w", senseErr)
			}

			for _, tr := range si.Translations {
				if _, trErr := s.translations.CreateCustom(txCtx, sense.ID, tr, sourceSlug); trErr != nil {
					return fmt.Errorf("create custom translation: %w", trErr)
				}
			}

			for _, ex := range si.Examples {
				if _, exErr := s.examples.CreateCustom(txCtx, sense.ID, ex.Sentence, ex.Translation, sourceSlug); exErr != nil {
					return fmt.Errorf("create custom example: %w", exErr)
				}
			}
		}

		// Create card if requested.
		if input.CreateCard {
			if _, cardErr := s.cards.Create(txCtx, userID, created.ID, domain.LearningStatusNew, s.cfg.DefaultEaseFactor); cardErr != nil {
				return fmt.Errorf("create card: %w", cardErr)
			}
		}

		// Audit.
		auditErr := s.audit.Create(txCtx, domain.AuditRecord{
			UserID:     userID,
			EntityType: domain.EntityTypeEntry,
			EntityID:   &created.ID,
			Action:     domain.AuditActionCreate,
			Changes:    map[string]any{"text": created.Text, "source": sourceSlug},
		})
		if auditErr != nil {
			return fmt.Errorf("audit create: %w", auditErr)
		}

		return nil
	})

	if txErr != nil {
		if errors.Is(txErr, domain.ErrAlreadyExists) {
			return nil, domain.ErrAlreadyExists
		}
		return nil, txErr
	}

	return created, nil
}

// ---------------------------------------------------------------------------
// 5. FindEntries
// ---------------------------------------------------------------------------

// FindEntries searches and paginates user dictionary entries.
func (s *Service) FindEntries(ctx context.Context, input FindInput) (*FindResult, error) {
	userID, ok := ctxutil.UserIDFromCtx(ctx)
	if !ok {
		return nil, domain.ErrUnauthorized
	}

	if err := input.Validate(); err != nil {
		return nil, err
	}

	// Normalize search text.
	var normalizedSearch *string
	if input.Search != nil {
		n := domain.NormalizeText(*input.Search)
		if n != "" {
			normalizedSearch = &n
		}
	}

	// Clamp limit.
	limit := clampLimit(input.Limit, 1, 200, 20)

	// Defaults.
	sortBy := input.SortBy
	if sortBy == "" {
		sortBy = "created_at"
	}
	sortOrder := input.SortOrder
	if sortOrder == "" {
		sortOrder = "DESC"
	}

	filter := domain.EntryFilter{
		Search:       normalizedSearch,
		HasCard:      input.HasCard,
		PartOfSpeech: input.PartOfSpeech,
		TopicID:      input.TopicID,
		Status:       input.Status,
		SortBy:       sortBy,
		SortOrder:    sortOrder,
		Limit:        limit,
		Cursor:       input.Cursor,
		Offset:       input.Offset,
	}

	result := &FindResult{}

	if input.Cursor != nil {
		// Cursor-based pagination.
		entries, hasNext, err := s.entries.FindCursor(ctx, userID, filter)
		if err != nil {
			return nil, fmt.Errorf("find entries (cursor): %w", err)
		}
		result.Entries = entries
		result.HasNextPage = hasNext

		if len(entries) > 0 {
			startID := entries[0].ID.String()
			endID := entries[len(entries)-1].ID.String()
			result.PageInfo = &PageInfo{
				StartCursor: &startID,
				EndCursor:   &endID,
			}
		}
	} else {
		// Offset-based pagination.
		entries, total, err := s.entries.Find(ctx, userID, filter)
		if err != nil {
			return nil, fmt.Errorf("find entries (offset): %w", err)
		}
		result.Entries = entries
		result.TotalCount = total

		offset := 0
		if input.Offset != nil {
			offset = *input.Offset
		}
		result.HasNextPage = offset+len(entries) < total

		if len(entries) > 0 {
			startID := entries[0].ID.String()
			endID := entries[len(entries)-1].ID.String()
			result.PageInfo = &PageInfo{
				StartCursor: &startID,
				EndCursor:   &endID,
			}
		}
	}

	return result, nil
}

// ---------------------------------------------------------------------------
// 6. GetEntry
// ---------------------------------------------------------------------------

// GetEntry returns a single entry by ID.
func (s *Service) GetEntry(ctx context.Context, entryID uuid.UUID) (*domain.Entry, error) {
	userID, ok := ctxutil.UserIDFromCtx(ctx)
	if !ok {
		return nil, domain.ErrUnauthorized
	}

	return s.entries.GetByID(ctx, userID, entryID)
}

// ---------------------------------------------------------------------------
// 7. UpdateNotes
// ---------------------------------------------------------------------------

// UpdateNotes updates the notes for an entry.
func (s *Service) UpdateNotes(ctx context.Context, input UpdateNotesInput) (*domain.Entry, error) {
	userID, ok := ctxutil.UserIDFromCtx(ctx)
	if !ok {
		return nil, domain.ErrUnauthorized
	}

	if err := input.Validate(); err != nil {
		return nil, err
	}

	// Get old entry for audit diff.
	oldEntry, err := s.entries.GetByID(ctx, userID, input.EntryID)
	if err != nil {
		return nil, err
	}

	var updated *domain.Entry
	txErr := s.tx.RunInTx(ctx, func(txCtx context.Context) error {
		var updateErr error
		updated, updateErr = s.entries.UpdateNotes(txCtx, userID, input.EntryID, input.Notes)
		if updateErr != nil {
			return fmt.Errorf("update notes: %w", updateErr)
		}

		changes := map[string]any{
			"old_notes": oldEntry.Notes,
			"new_notes": input.Notes,
		}

		auditErr := s.audit.Create(txCtx, domain.AuditRecord{
			UserID:     userID,
			EntityType: domain.EntityTypeEntry,
			EntityID:   &input.EntryID,
			Action:     domain.AuditActionUpdate,
			Changes:    changes,
		})
		if auditErr != nil {
			return fmt.Errorf("audit update: %w", auditErr)
		}

		return nil
	})

	if txErr != nil {
		return nil, txErr
	}

	return updated, nil
}

// ---------------------------------------------------------------------------
// 8. DeleteEntry
// ---------------------------------------------------------------------------

// DeleteEntry soft-deletes an entry.
func (s *Service) DeleteEntry(ctx context.Context, entryID uuid.UUID) error {
	userID, ok := ctxutil.UserIDFromCtx(ctx)
	if !ok {
		return domain.ErrUnauthorized
	}

	// Get entry for audit text.
	entry, err := s.entries.GetByID(ctx, userID, entryID)
	if err != nil {
		return err
	}

	txErr := s.tx.RunInTx(ctx, func(txCtx context.Context) error {
		if delErr := s.entries.SoftDelete(txCtx, userID, entryID); delErr != nil {
			return fmt.Errorf("soft delete: %w", delErr)
		}

		auditErr := s.audit.Create(txCtx, domain.AuditRecord{
			UserID:     userID,
			EntityType: domain.EntityTypeEntry,
			EntityID:   &entryID,
			Action:     domain.AuditActionDelete,
			Changes:    map[string]any{"text": entry.Text},
		})
		if auditErr != nil {
			return fmt.Errorf("audit delete: %w", auditErr)
		}

		return nil
	})

	return txErr
}

// ---------------------------------------------------------------------------
// 9. FindDeletedEntries
// ---------------------------------------------------------------------------

// FindDeletedEntries returns soft-deleted entries for the user.
func (s *Service) FindDeletedEntries(ctx context.Context, limit, offset int) ([]domain.Entry, int, error) {
	userID, ok := ctxutil.UserIDFromCtx(ctx)
	if !ok {
		return nil, 0, domain.ErrUnauthorized
	}

	limit = clampLimit(limit, 1, 200, 20)

	return s.entries.FindDeleted(ctx, userID, limit, offset)
}

// ---------------------------------------------------------------------------
// 10. RestoreEntry
// ---------------------------------------------------------------------------

// RestoreEntry restores a soft-deleted entry.
func (s *Service) RestoreEntry(ctx context.Context, entryID uuid.UUID) (*domain.Entry, error) {
	userID, ok := ctxutil.UserIDFromCtx(ctx)
	if !ok {
		return nil, domain.ErrUnauthorized
	}

	restored, err := s.entries.Restore(ctx, userID, entryID)
	if err != nil {
		if errors.Is(err, domain.ErrAlreadyExists) {
			return nil, domain.NewValidationError("text", "active entry with this text already exists")
		}
		return nil, err
	}

	return restored, nil
}

// ---------------------------------------------------------------------------
// 11. BatchDeleteEntries
// ---------------------------------------------------------------------------

// BatchDeleteEntries soft-deletes multiple entries. NOT transactional, partial failure OK.
func (s *Service) BatchDeleteEntries(ctx context.Context, entryIDs []uuid.UUID) (*BatchResult, error) {
	userID, ok := ctxutil.UserIDFromCtx(ctx)
	if !ok {
		return nil, domain.ErrUnauthorized
	}

	if len(entryIDs) == 0 {
		return nil, domain.NewValidationError("entry_ids", "required (at least 1)")
	}
	if len(entryIDs) > 200 {
		return nil, domain.NewValidationError("entry_ids", "too many (max 200)")
	}

	result := &BatchResult{}

	for _, eid := range entryIDs {
		if delErr := s.entries.SoftDelete(ctx, userID, eid); delErr != nil {
			result.Errors = append(result.Errors, BatchError{
				EntryID: eid,
				Error:   delErr.Error(),
			})
		} else {
			result.Deleted++
		}
	}

	// Write a single audit record if any were deleted.
	if result.Deleted > 0 {
		ids := make([]string, 0, result.Deleted)
		for _, eid := range entryIDs {
			// Only include successfully deleted IDs.
			failed := false
			for _, be := range result.Errors {
				if be.EntryID == eid {
					failed = true
					break
				}
			}
			if !failed {
				ids = append(ids, eid.String())
			}
		}

		auditErr := s.audit.Create(ctx, domain.AuditRecord{
			UserID:     userID,
			EntityType: domain.EntityTypeEntry,
			Action:     domain.AuditActionDelete,
			Changes:    map[string]any{"batch_delete": ids, "count": result.Deleted},
		})
		if auditErr != nil {
			s.log.ErrorContext(ctx, "batch delete audit error",
				slog.String("error", auditErr.Error()),
			)
		}
	}

	return result, nil
}

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

				entry := &domain.Entry{
					UserID:         userID,
					Text:           item.Text,
					TextNormalized: normalized,
					Notes:          item.Notes,
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

// ---------------------------------------------------------------------------
// 13. ExportEntries
// ---------------------------------------------------------------------------

// ExportEntries exports all dictionary entries for the user.
func (s *Service) ExportEntries(ctx context.Context) (*ExportResult, error) {
	userID, ok := ctxutil.UserIDFromCtx(ctx)
	if !ok {
		return nil, domain.ErrUnauthorized
	}

	// Find all entries.
	entries, _, err := s.entries.Find(ctx, userID, domain.EntryFilter{
		SortBy:    "created_at",
		SortOrder: "ASC",
		Limit:     s.cfg.ExportMaxEntries,
	})
	if err != nil {
		return nil, fmt.Errorf("find entries for export: %w", err)
	}

	if len(entries) == 0 {
		return &ExportResult{
			Items:      []ExportItem{},
			ExportedAt: time.Now(),
		}, nil
	}

	// Collect entry IDs.
	entryIDs := make([]uuid.UUID, len(entries))
	for i, e := range entries {
		entryIDs[i] = e.ID
	}

	// Batch load senses.
	senses, err := s.senses.GetByEntryIDs(ctx, entryIDs)
	if err != nil {
		return nil, fmt.Errorf("get senses: %w", err)
	}

	// Collect sense IDs and build sense-by-entry map.
	sensesByEntry := make(map[uuid.UUID][]domain.Sense)
	var senseIDs []uuid.UUID
	for _, sense := range senses {
		sensesByEntry[sense.EntryID] = append(sensesByEntry[sense.EntryID], sense)
		senseIDs = append(senseIDs, sense.ID)
	}

	// Batch load translations and examples.
	var translations []domain.Translation
	var examples []domain.Example
	if len(senseIDs) > 0 {
		translations, err = s.translations.GetBySenseIDs(ctx, senseIDs)
		if err != nil {
			return nil, fmt.Errorf("get translations: %w", err)
		}
		examples, err = s.examples.GetBySenseIDs(ctx, senseIDs)
		if err != nil {
			return nil, fmt.Errorf("get examples: %w", err)
		}
	}

	translationsBySense := make(map[uuid.UUID][]domain.Translation)
	for _, tr := range translations {
		translationsBySense[tr.SenseID] = append(translationsBySense[tr.SenseID], tr)
	}

	examplesBySense := make(map[uuid.UUID][]domain.Example)
	for _, ex := range examples {
		examplesBySense[ex.SenseID] = append(examplesBySense[ex.SenseID], ex)
	}

	// Batch load cards.
	cards, err := s.cards.GetByEntryIDs(ctx, entryIDs)
	if err != nil {
		return nil, fmt.Errorf("get cards: %w", err)
	}
	cardByEntry := make(map[uuid.UUID]domain.Card)
	for _, c := range cards {
		cardByEntry[c.EntryID] = c
	}

	// Build export items.
	items := make([]ExportItem, 0, len(entries))
	for _, entry := range entries {
		item := ExportItem{
			Text:      entry.Text,
			Notes:     entry.Notes,
			CreatedAt: entry.CreatedAt,
		}

		// Card status.
		if card, found := cardByEntry[entry.ID]; found {
			status := card.Status
			item.CardStatus = &status
		}

		// Senses.
		for _, sense := range sensesByEntry[entry.ID] {
			exportSense := ExportSense{
				Definition:   sense.Definition,
				PartOfSpeech: sense.PartOfSpeech,
			}

			// Translations.
			for _, tr := range translationsBySense[sense.ID] {
				if tr.Text != nil {
					exportSense.Translations = append(exportSense.Translations, *tr.Text)
				}
			}

			// Examples.
			for _, ex := range examplesBySense[sense.ID] {
				exportEx := ExportExample{
					Translation: ex.Translation,
				}
				if ex.Sentence != nil {
					exportEx.Sentence = *ex.Sentence
				}
				exportSense.Examples = append(exportSense.Examples, exportEx)
			}

			item.Senses = append(item.Senses, exportSense)
		}

		items = append(items, item)
	}

	return &ExportResult{
		Items:      items,
		ExportedAt: time.Now(),
	}, nil
}

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

// clampLimit ensures a limit is within [min, max], defaulting from 0 to defaultVal.
func clampLimit(limit, min, max, defaultVal int) int {
	if limit <= 0 {
		return defaultVal
	}
	if limit < min {
		return min
	}
	if limit > max {
		return max
	}
	return limit
}
