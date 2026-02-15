package resolver

//go:generate moq -out dictionary_service_mock_test.go -pkg resolver . dictionaryService

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"
	"github.com/heartmarshall/myenglish-backend/internal/domain"
	"github.com/heartmarshall/myenglish-backend/internal/service/dictionary"
	"github.com/heartmarshall/myenglish-backend/internal/transport/graphql/generated"
	"github.com/heartmarshall/myenglish-backend/pkg/ctxutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func ptr[T any](v T) *T {
	return &v
}

// TestSearchCatalog_Success tests successful catalog search.
func TestSearchCatalog_Success(t *testing.T) {
	t.Parallel()

	refEntryID := uuid.New()
	mock := &dictionaryServiceMock{
		SearchCatalogFunc: func(ctx context.Context, query string, limit int) ([]domain.RefEntry, error) {
			return []domain.RefEntry{
				{ID: refEntryID, Text: "test", TextNormalized: "test"},
			}, nil
		},
	}

	resolver := &queryResolver{&Resolver{dictionary: mock}}
	result, err := resolver.SearchCatalog(context.Background(), "test", ptr(10))

	require.NoError(t, err)
	require.Len(t, result, 1)
	assert.Equal(t, "test", result[0].Text)
	assert.Equal(t, refEntryID, result[0].ID)
}

// TestSearchCatalog_DefaultLimit tests that default limit is applied.
func TestSearchCatalog_DefaultLimit(t *testing.T) {
	t.Parallel()

	mock := &dictionaryServiceMock{
		SearchCatalogFunc: func(ctx context.Context, query string, limit int) ([]domain.RefEntry, error) {
			assert.Equal(t, 10, limit) // verify default
			return []domain.RefEntry{}, nil
		},
	}

	resolver := &queryResolver{&Resolver{dictionary: mock}}
	_, err := resolver.SearchCatalog(context.Background(), "test", nil)

	require.NoError(t, err)
}

// TestSearchCatalog_ServiceError tests service error propagation.
func TestSearchCatalog_ServiceError(t *testing.T) {
	t.Parallel()

	mock := &dictionaryServiceMock{
		SearchCatalogFunc: func(ctx context.Context, query string, limit int) ([]domain.RefEntry, error) {
			return nil, errors.New("service error")
		},
	}

	resolver := &queryResolver{&Resolver{dictionary: mock}}
	_, err := resolver.SearchCatalog(context.Background(), "test", ptr(10))

	require.Error(t, err)
}

// TestPreviewRefEntry_Success tests successful preview.
func TestPreviewRefEntry_Success(t *testing.T) {
	t.Parallel()

	refEntryID := uuid.New()
	mock := &dictionaryServiceMock{
		PreviewRefEntryFunc: func(ctx context.Context, text string) (*domain.RefEntry, error) {
			return &domain.RefEntry{ID: refEntryID, Text: text, TextNormalized: text}, nil
		},
	}

	resolver := &queryResolver{&Resolver{dictionary: mock}}
	result, err := resolver.PreviewRefEntry(context.Background(), "hello")

	require.NoError(t, err)
	assert.Equal(t, "hello", result.Text)
}

// TestPreviewRefEntry_NotFound tests not found case.
func TestPreviewRefEntry_NotFound(t *testing.T) {
	t.Parallel()

	mock := &dictionaryServiceMock{
		PreviewRefEntryFunc: func(ctx context.Context, text string) (*domain.RefEntry, error) {
			return nil, domain.ErrNotFound
		},
	}

	resolver := &queryResolver{&Resolver{dictionary: mock}}
	_, err := resolver.PreviewRefEntry(context.Background(), "unknown")

	require.ErrorIs(t, err, domain.ErrNotFound)
}

// TestDictionary_Success tests successful dictionary query with connection.
func TestDictionary_Success(t *testing.T) {
	t.Parallel()

	userID := uuid.New()
	entryID := uuid.New()
	ctx := ctxutil.WithUserID(context.Background(), userID)

	startCursor := "start"
	endCursor := "end"

	mock := &dictionaryServiceMock{
		FindEntriesFunc: func(ctx context.Context, input dictionary.FindInput) (*dictionary.FindResult, error) {
			return &dictionary.FindResult{
				Entries: []domain.Entry{
					{ID: entryID, Text: "test"},
				},
				TotalCount:  100,
				HasNextPage: true,
				PageInfo: &dictionary.PageInfo{
					StartCursor: &startCursor,
					EndCursor:   &endCursor,
				},
			}, nil
		},
	}

	resolver := &queryResolver{&Resolver{dictionary: mock}}
	input := generated.DictionaryFilterInput{
		Search: ptr("test"),
		Limit:  ptr(20),
	}

	result, err := resolver.Dictionary(ctx, input)

	require.NoError(t, err)
	assert.Len(t, result.Edges, 1)
	assert.Equal(t, entryID, result.Edges[0].Node.ID)
	assert.Equal(t, 100, result.TotalCount)
	assert.True(t, result.PageInfo.HasNextPage)
	assert.False(t, result.PageInfo.HasPreviousPage)
	assert.Equal(t, &startCursor, result.PageInfo.StartCursor)
	assert.Equal(t, &endCursor, result.PageInfo.EndCursor)
}

// TestDictionary_Unauthorized tests unauthorized access.
func TestDictionary_Unauthorized(t *testing.T) {
	t.Parallel()

	resolver := &queryResolver{&Resolver{dictionary: &dictionaryServiceMock{}}}
	_, err := resolver.Dictionary(context.Background(), generated.DictionaryFilterInput{})

	require.ErrorIs(t, err, domain.ErrUnauthorized)
}

// TestDictionary_CursorPagination tests cursor-based pagination.
func TestDictionary_CursorPagination(t *testing.T) {
	t.Parallel()

	userID := uuid.New()
	ctx := ctxutil.WithUserID(context.Background(), userID)

	mock := &dictionaryServiceMock{
		FindEntriesFunc: func(ctx context.Context, input dictionary.FindInput) (*dictionary.FindResult, error) {
			// Verify cursor-based fields are set
			assert.NotNil(t, input.Cursor)
			assert.Equal(t, "cursor123", *input.Cursor)
			assert.Equal(t, 15, input.Limit)
			return &dictionary.FindResult{Entries: []domain.Entry{}, TotalCount: 0}, nil
		},
	}

	resolver := &queryResolver{&Resolver{dictionary: mock}}
	input := generated.DictionaryFilterInput{
		After: ptr("cursor123"),
		First: ptr(15),
	}

	_, err := resolver.Dictionary(ctx, input)
	require.NoError(t, err)
}

// TestDictionaryEntry_Success tests successful single entry retrieval.
func TestDictionaryEntry_Success(t *testing.T) {
	t.Parallel()

	userID := uuid.New()
	entryID := uuid.New()
	ctx := ctxutil.WithUserID(context.Background(), userID)

	mock := &dictionaryServiceMock{
		GetEntryFunc: func(ctx context.Context, id uuid.UUID) (*domain.Entry, error) {
			return &domain.Entry{ID: id, Text: "test"}, nil
		},
	}

	resolver := &queryResolver{&Resolver{dictionary: mock}}
	result, err := resolver.DictionaryEntry(ctx, entryID)

	require.NoError(t, err)
	assert.Equal(t, entryID, result.ID)
}

// TestDictionaryEntry_Unauthorized tests unauthorized access.
func TestDictionaryEntry_Unauthorized(t *testing.T) {
	t.Parallel()

	resolver := &queryResolver{&Resolver{dictionary: &dictionaryServiceMock{}}}
	_, err := resolver.DictionaryEntry(context.Background(), uuid.New())

	require.ErrorIs(t, err, domain.ErrUnauthorized)
}

// TestDeletedEntries_Success tests successful deleted entries retrieval.
func TestDeletedEntries_Success(t *testing.T) {
	t.Parallel()

	userID := uuid.New()
	ctx := ctxutil.WithUserID(context.Background(), userID)

	mock := &dictionaryServiceMock{
		FindDeletedEntriesFunc: func(ctx context.Context, limit, offset int) ([]domain.Entry, int, error) {
			return []domain.Entry{
				{ID: uuid.New(), Text: "deleted1"},
				{ID: uuid.New(), Text: "deleted2"},
			}, 10, nil
		},
	}

	resolver := &queryResolver{&Resolver{dictionary: mock}}
	result, err := resolver.DeletedEntries(ctx, ptr(50), ptr(0))

	require.NoError(t, err)
	assert.Len(t, result.Entries, 2)
	assert.Equal(t, 10, result.TotalCount)
}

// TestDeletedEntries_DefaultValues tests default limit and offset.
func TestDeletedEntries_DefaultValues(t *testing.T) {
	t.Parallel()

	userID := uuid.New()
	ctx := ctxutil.WithUserID(context.Background(), userID)

	mock := &dictionaryServiceMock{
		FindDeletedEntriesFunc: func(ctx context.Context, limit, offset int) ([]domain.Entry, int, error) {
			assert.Equal(t, 50, limit)  // default limit
			assert.Equal(t, 0, offset)   // default offset
			return []domain.Entry{}, 0, nil
		},
	}

	resolver := &queryResolver{&Resolver{dictionary: mock}}
	_, err := resolver.DeletedEntries(ctx, nil, nil)

	require.NoError(t, err)
}

// TestExportEntries_Success tests successful export.
func TestExportEntries_Success(t *testing.T) {
	t.Parallel()

	userID := uuid.New()
	ctx := ctxutil.WithUserID(context.Background(), userID)

	mock := &dictionaryServiceMock{
		ExportEntriesFunc: func(ctx context.Context) (*dictionary.ExportResult, error) {
			return &dictionary.ExportResult{
				Items: []dictionary.ExportItem{
					{Text: "word1"},
					{Text: "word2"},
				},
			}, nil
		},
	}

	resolver := &queryResolver{&Resolver{dictionary: mock}}
	result, err := resolver.ExportEntries(ctx)

	require.NoError(t, err)
	assert.Len(t, result.Items, 2)
}

// TestExportEntries_Unauthorized tests unauthorized access.
func TestExportEntries_Unauthorized(t *testing.T) {
	t.Parallel()

	resolver := &queryResolver{&Resolver{dictionary: &dictionaryServiceMock{}}}
	_, err := resolver.ExportEntries(context.Background())

	require.ErrorIs(t, err, domain.ErrUnauthorized)
}

// TestCreateEntryFromCatalog_Success tests successful entry creation from catalog.
func TestCreateEntryFromCatalog_Success(t *testing.T) {
	t.Parallel()

	userID := uuid.New()
	entryID := uuid.New()
	refEntryID := uuid.New()
	ctx := ctxutil.WithUserID(context.Background(), userID)

	mock := &dictionaryServiceMock{
		CreateEntryFromCatalogFunc: func(ctx context.Context, input dictionary.CreateFromCatalogInput) (*domain.Entry, error) {
			assert.Equal(t, refEntryID, input.RefEntryID)
			assert.True(t, input.CreateCard)
			return &domain.Entry{ID: entryID, Text: "test"}, nil
		},
	}

	resolver := &mutationResolver{&Resolver{dictionary: mock}}
	input := generated.CreateEntryFromCatalogInput{
		RefEntryID: refEntryID,
		SenseIds:   []uuid.UUID{uuid.New()},
		CreateCard: ptr(true),
	}

	result, err := resolver.CreateEntryFromCatalog(ctx, input)

	require.NoError(t, err)
	assert.Equal(t, entryID, result.Entry.ID)
}

// TestCreateEntryFromCatalog_Unauthorized tests unauthorized creation.
func TestCreateEntryFromCatalog_Unauthorized(t *testing.T) {
	t.Parallel()

	resolver := &mutationResolver{&Resolver{dictionary: &dictionaryServiceMock{}}}
	_, err := resolver.CreateEntryFromCatalog(context.Background(), generated.CreateEntryFromCatalogInput{})

	require.ErrorIs(t, err, domain.ErrUnauthorized)
}

// TestCreateEntryCustom_Success tests successful custom entry creation.
func TestCreateEntryCustom_Success(t *testing.T) {
	t.Parallel()

	userID := uuid.New()
	entryID := uuid.New()
	ctx := ctxutil.WithUserID(context.Background(), userID)

	mock := &dictionaryServiceMock{
		CreateEntryCustomFunc: func(ctx context.Context, input dictionary.CreateCustomInput) (*domain.Entry, error) {
			assert.Equal(t, "custom word", input.Text)
			assert.Len(t, input.Senses, 1)
			return &domain.Entry{ID: entryID, Text: input.Text}, nil
		},
	}

	resolver := &mutationResolver{&Resolver{dictionary: mock}}
	input := generated.CreateEntryCustomInput{
		Text: "custom word",
		Senses: []*generated.CustomSenseInput{
			{
				Definition:   ptr("a definition"),
				Translations: []string{"translation1"},
				Examples: []*generated.CustomExampleInput{
					{Sentence: "example sentence"},
				},
			},
		},
	}

	result, err := resolver.CreateEntryCustom(ctx, input)

	require.NoError(t, err)
	assert.Equal(t, entryID, result.Entry.ID)
}

// TestCreateEntryCustom_DefaultCreateCard tests default createCard value.
func TestCreateEntryCustom_DefaultCreateCard(t *testing.T) {
	t.Parallel()

	userID := uuid.New()
	ctx := ctxutil.WithUserID(context.Background(), userID)

	mock := &dictionaryServiceMock{
		CreateEntryCustomFunc: func(ctx context.Context, input dictionary.CreateCustomInput) (*domain.Entry, error) {
			assert.False(t, input.CreateCard) // default should be false
			return &domain.Entry{ID: uuid.New()}, nil
		},
	}

	resolver := &mutationResolver{&Resolver{dictionary: mock}}
	input := generated.CreateEntryCustomInput{
		Text:   "word",
		Senses: []*generated.CustomSenseInput{},
	}

	_, err := resolver.CreateEntryCustom(ctx, input)
	require.NoError(t, err)
}

// TestUpdateEntryNotes_Success tests successful notes update.
func TestUpdateEntryNotes_Success(t *testing.T) {
	t.Parallel()

	userID := uuid.New()
	entryID := uuid.New()
	ctx := ctxutil.WithUserID(context.Background(), userID)

	notes := "updated notes"
	mock := &dictionaryServiceMock{
		UpdateNotesFunc: func(ctx context.Context, input dictionary.UpdateNotesInput) (*domain.Entry, error) {
			assert.Equal(t, entryID, input.EntryID)
			assert.Equal(t, &notes, input.Notes)
			return &domain.Entry{ID: entryID, Notes: input.Notes}, nil
		},
	}

	resolver := &mutationResolver{&Resolver{dictionary: mock}}
	input := generated.UpdateEntryNotesInput{
		EntryID: entryID,
		Notes:   &notes,
	}

	result, err := resolver.UpdateEntryNotes(ctx, input)

	require.NoError(t, err)
	assert.Equal(t, &notes, result.Entry.Notes)
}

// TestUpdateEntryNotes_Unauthorized tests unauthorized update.
func TestUpdateEntryNotes_Unauthorized(t *testing.T) {
	t.Parallel()

	resolver := &mutationResolver{&Resolver{dictionary: &dictionaryServiceMock{}}}
	_, err := resolver.UpdateEntryNotes(context.Background(), generated.UpdateEntryNotesInput{})

	require.ErrorIs(t, err, domain.ErrUnauthorized)
}

// TestDeleteEntry_Success tests successful entry deletion.
func TestDeleteEntry_Success(t *testing.T) {
	t.Parallel()

	userID := uuid.New()
	entryID := uuid.New()
	ctx := ctxutil.WithUserID(context.Background(), userID)

	mock := &dictionaryServiceMock{
		DeleteEntryFunc: func(ctx context.Context, id uuid.UUID) error {
			assert.Equal(t, entryID, id)
			return nil
		},
	}

	resolver := &mutationResolver{&Resolver{dictionary: mock}}
	result, err := resolver.DeleteEntry(ctx, entryID)

	require.NoError(t, err)
	assert.Equal(t, entryID, result.EntryID)
}

// TestDeleteEntry_NotFound tests deletion of non-existent entry.
func TestDeleteEntry_NotFound(t *testing.T) {
	t.Parallel()

	userID := uuid.New()
	ctx := ctxutil.WithUserID(context.Background(), userID)

	mock := &dictionaryServiceMock{
		DeleteEntryFunc: func(ctx context.Context, id uuid.UUID) error {
			return domain.ErrNotFound
		},
	}

	resolver := &mutationResolver{&Resolver{dictionary: mock}}
	_, err := resolver.DeleteEntry(ctx, uuid.New())

	require.ErrorIs(t, err, domain.ErrNotFound)
}

// TestRestoreEntry_Success tests successful entry restoration.
func TestRestoreEntry_Success(t *testing.T) {
	t.Parallel()

	userID := uuid.New()
	entryID := uuid.New()
	ctx := ctxutil.WithUserID(context.Background(), userID)

	mock := &dictionaryServiceMock{
		RestoreEntryFunc: func(ctx context.Context, id uuid.UUID) (*domain.Entry, error) {
			return &domain.Entry{ID: id, Text: "restored"}, nil
		},
	}

	resolver := &mutationResolver{&Resolver{dictionary: mock}}
	result, err := resolver.RestoreEntry(ctx, entryID)

	require.NoError(t, err)
	assert.Equal(t, entryID, result.Entry.ID)
}

// TestRestoreEntry_Unauthorized tests unauthorized restoration.
func TestRestoreEntry_Unauthorized(t *testing.T) {
	t.Parallel()

	resolver := &mutationResolver{&Resolver{dictionary: &dictionaryServiceMock{}}}
	_, err := resolver.RestoreEntry(context.Background(), uuid.New())

	require.ErrorIs(t, err, domain.ErrUnauthorized)
}

// TestBatchDeleteEntries_Success tests successful batch deletion.
func TestBatchDeleteEntries_Success(t *testing.T) {
	t.Parallel()

	userID := uuid.New()
	ctx := ctxutil.WithUserID(context.Background(), userID)

	id1 := uuid.New()
	id2 := uuid.New()

	mock := &dictionaryServiceMock{
		BatchDeleteEntriesFunc: func(ctx context.Context, ids []uuid.UUID) (*dictionary.BatchResult, error) {
			return &dictionary.BatchResult{
				Deleted: 1,
				Errors: []dictionary.BatchError{
					{EntryID: id2, Error: "not found"},
				},
			}, nil
		},
	}

	resolver := &mutationResolver{&Resolver{dictionary: mock}}
	result, err := resolver.BatchDeleteEntries(ctx, []uuid.UUID{id1, id2})

	require.NoError(t, err)
	assert.Equal(t, 1, result.DeletedCount)
	assert.Len(t, result.Errors, 1)
	assert.Equal(t, id2, result.Errors[0].ID)
	assert.Equal(t, "not found", result.Errors[0].Message)
}

// TestBatchDeleteEntries_Unauthorized tests unauthorized batch deletion.
func TestBatchDeleteEntries_Unauthorized(t *testing.T) {
	t.Parallel()

	resolver := &mutationResolver{&Resolver{dictionary: &dictionaryServiceMock{}}}
	_, err := resolver.BatchDeleteEntries(context.Background(), []uuid.UUID{})

	require.ErrorIs(t, err, domain.ErrUnauthorized)
}

// TestImportEntries_Success tests successful import.
func TestImportEntries_Success(t *testing.T) {
	t.Parallel()

	userID := uuid.New()
	ctx := ctxutil.WithUserID(context.Background(), userID)

	mock := &dictionaryServiceMock{
		ImportEntriesFunc: func(ctx context.Context, input dictionary.ImportInput) (*dictionary.ImportResult, error) {
			return &dictionary.ImportResult{
				Imported: 8,
				Skipped:  2,
				Errors: []dictionary.ImportError{
					{LineNumber: 3, Text: "word3", Reason: "duplicate"},
				},
			}, nil
		},
	}

	resolver := &mutationResolver{&Resolver{dictionary: mock}}
	input := generated.ImportEntriesInput{
		Items: []*generated.ImportItemInput{
			{Text: "word1", Translations: []string{"tr1"}},
			{Text: "word2", Translations: []string{"tr2"}},
		},
	}

	result, err := resolver.ImportEntries(ctx, input)

	require.NoError(t, err)
	assert.Equal(t, 8, result.ImportedCount)
	assert.Equal(t, 2, result.SkippedCount)
	assert.Len(t, result.Errors, 1)
	assert.Equal(t, 3, result.Errors[0].Index)
	assert.Equal(t, "word3", result.Errors[0].Text)
	assert.Equal(t, "duplicate", result.Errors[0].Message)
}

// TestImportEntries_Unauthorized tests unauthorized import.
func TestImportEntries_Unauthorized(t *testing.T) {
	t.Parallel()

	resolver := &mutationResolver{&Resolver{dictionary: &dictionaryServiceMock{}}}
	_, err := resolver.ImportEntries(context.Background(), generated.ImportEntriesInput{})

	require.ErrorIs(t, err, domain.ErrUnauthorized)
}
