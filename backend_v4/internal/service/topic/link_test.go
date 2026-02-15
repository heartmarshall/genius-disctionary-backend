package topic

import (
	"context"
	"errors"
	"log/slog"
	"testing"

	"github.com/google/uuid"
	"github.com/heartmarshall/myenglish-backend/internal/domain"
	"github.com/heartmarshall/myenglish-backend/pkg/ctxutil"
)

func newLinkTestService(t *testing.T, topics *topicRepoMock, entries *entryRepoMock) *Service {
	t.Helper()
	return NewService(
		slog.Default(),
		topics,
		entries,
		&auditLoggerMock{LogFunc: func(ctx context.Context, r domain.AuditRecord) error { return nil }},
		&txManagerMock{RunInTxFunc: func(ctx context.Context, fn func(context.Context) error) error { return fn(ctx) }},
	)
}

// --- LinkEntry tests ---

func TestLinkEntry_Success(t *testing.T) {
	t.Parallel()

	userID := uuid.New()
	topicID := uuid.New()
	entryID := uuid.New()
	ctx := ctxutil.WithUserID(context.Background(), userID)

	topicsMock := &topicRepoMock{
		GetByIDFunc: func(_ context.Context, uid, tid uuid.UUID) (*domain.Topic, error) {
			return &domain.Topic{ID: tid, UserID: uid, Name: "test"}, nil
		},
		LinkEntryFunc: func(_ context.Context, eid, tid uuid.UUID) error {
			return nil
		},
	}
	entriesMock := &entryRepoMock{
		GetByIDFunc: func(_ context.Context, uid, eid uuid.UUID) (*domain.Entry, error) {
			return &domain.Entry{ID: eid, UserID: uid, Text: "hello"}, nil
		},
	}

	svc := newLinkTestService(t, topicsMock, entriesMock)
	err := svc.LinkEntry(ctx, LinkEntryInput{TopicID: topicID, EntryID: entryID})

	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if len(topicsMock.GetByIDCalls()) != 1 {
		t.Fatalf("expected 1 topic GetByID call, got %d", len(topicsMock.GetByIDCalls()))
	}
	if len(entriesMock.GetByIDCalls()) != 1 {
		t.Fatalf("expected 1 entry GetByID call, got %d", len(entriesMock.GetByIDCalls()))
	}
	if len(topicsMock.LinkEntryCalls()) != 1 {
		t.Fatalf("expected 1 LinkEntry call, got %d", len(topicsMock.LinkEntryCalls()))
	}
}

func TestLinkEntry_TopicNotFound(t *testing.T) {
	t.Parallel()

	userID := uuid.New()
	ctx := ctxutil.WithUserID(context.Background(), userID)

	topicsMock := &topicRepoMock{
		GetByIDFunc: func(_ context.Context, _, _ uuid.UUID) (*domain.Topic, error) {
			return nil, domain.ErrNotFound
		},
	}
	entriesMock := &entryRepoMock{}

	svc := newLinkTestService(t, topicsMock, entriesMock)
	err := svc.LinkEntry(ctx, LinkEntryInput{TopicID: uuid.New(), EntryID: uuid.New()})

	if !errors.Is(err, domain.ErrNotFound) {
		t.Fatalf("expected ErrNotFound, got %v", err)
	}
}

func TestLinkEntry_EntryNotFound(t *testing.T) {
	t.Parallel()

	userID := uuid.New()
	ctx := ctxutil.WithUserID(context.Background(), userID)

	topicsMock := &topicRepoMock{
		GetByIDFunc: func(_ context.Context, uid, tid uuid.UUID) (*domain.Topic, error) {
			return &domain.Topic{ID: tid, UserID: uid, Name: "test"}, nil
		},
	}
	entriesMock := &entryRepoMock{
		GetByIDFunc: func(_ context.Context, _, _ uuid.UUID) (*domain.Entry, error) {
			return nil, domain.ErrNotFound
		},
	}

	svc := newLinkTestService(t, topicsMock, entriesMock)
	err := svc.LinkEntry(ctx, LinkEntryInput{TopicID: uuid.New(), EntryID: uuid.New()})

	if !errors.Is(err, domain.ErrNotFound) {
		t.Fatalf("expected ErrNotFound, got %v", err)
	}
}

func TestLinkEntry_EntryDeleted(t *testing.T) {
	t.Parallel()

	userID := uuid.New()
	ctx := ctxutil.WithUserID(context.Background(), userID)

	topicsMock := &topicRepoMock{
		GetByIDFunc: func(_ context.Context, uid, tid uuid.UUID) (*domain.Topic, error) {
			return &domain.Topic{ID: tid, UserID: uid, Name: "test"}, nil
		},
	}
	// entryRepo.GetByID filters soft-deleted entries, so it returns ErrNotFound
	entriesMock := &entryRepoMock{
		GetByIDFunc: func(_ context.Context, _, _ uuid.UUID) (*domain.Entry, error) {
			return nil, domain.ErrNotFound
		},
	}

	svc := newLinkTestService(t, topicsMock, entriesMock)
	err := svc.LinkEntry(ctx, LinkEntryInput{TopicID: uuid.New(), EntryID: uuid.New()})

	if !errors.Is(err, domain.ErrNotFound) {
		t.Fatalf("expected ErrNotFound, got %v", err)
	}
}

func TestLinkEntry_AlreadyLinked(t *testing.T) {
	t.Parallel()

	userID := uuid.New()
	topicID := uuid.New()
	entryID := uuid.New()
	ctx := ctxutil.WithUserID(context.Background(), userID)

	topicsMock := &topicRepoMock{
		GetByIDFunc: func(_ context.Context, uid, tid uuid.UUID) (*domain.Topic, error) {
			return &domain.Topic{ID: tid, UserID: uid, Name: "test"}, nil
		},
		LinkEntryFunc: func(_ context.Context, _, _ uuid.UUID) error {
			return nil // ON CONFLICT DO NOTHING — idempotent
		},
	}
	entriesMock := &entryRepoMock{
		GetByIDFunc: func(_ context.Context, uid, eid uuid.UUID) (*domain.Entry, error) {
			return &domain.Entry{ID: eid, UserID: uid, Text: "hello"}, nil
		},
	}

	svc := newLinkTestService(t, topicsMock, entriesMock)
	err := svc.LinkEntry(ctx, LinkEntryInput{TopicID: topicID, EntryID: entryID})

	if err != nil {
		t.Fatalf("expected no error for idempotent link, got %v", err)
	}
}

func TestLinkEntry_WrongUserTopic(t *testing.T) {
	t.Parallel()

	userID := uuid.New()
	ctx := ctxutil.WithUserID(context.Background(), userID)

	// Topic belongs to another user — repo returns ErrNotFound (filtered by user_id)
	topicsMock := &topicRepoMock{
		GetByIDFunc: func(_ context.Context, _, _ uuid.UUID) (*domain.Topic, error) {
			return nil, domain.ErrNotFound
		},
	}
	entriesMock := &entryRepoMock{}

	svc := newLinkTestService(t, topicsMock, entriesMock)
	err := svc.LinkEntry(ctx, LinkEntryInput{TopicID: uuid.New(), EntryID: uuid.New()})

	if !errors.Is(err, domain.ErrNotFound) {
		t.Fatalf("expected ErrNotFound, got %v", err)
	}
}

func TestLinkEntry_WrongUserEntry(t *testing.T) {
	t.Parallel()

	userID := uuid.New()
	ctx := ctxutil.WithUserID(context.Background(), userID)

	topicsMock := &topicRepoMock{
		GetByIDFunc: func(_ context.Context, uid, tid uuid.UUID) (*domain.Topic, error) {
			return &domain.Topic{ID: tid, UserID: uid, Name: "test"}, nil
		},
	}
	// Entry belongs to another user — repo returns ErrNotFound (filtered by user_id)
	entriesMock := &entryRepoMock{
		GetByIDFunc: func(_ context.Context, _, _ uuid.UUID) (*domain.Entry, error) {
			return nil, domain.ErrNotFound
		},
	}

	svc := newLinkTestService(t, topicsMock, entriesMock)
	err := svc.LinkEntry(ctx, LinkEntryInput{TopicID: uuid.New(), EntryID: uuid.New()})

	if !errors.Is(err, domain.ErrNotFound) {
		t.Fatalf("expected ErrNotFound, got %v", err)
	}
}

func TestLinkEntry_Unauthorized(t *testing.T) {
	t.Parallel()

	ctx := context.Background() // no userID in context

	topicsMock := &topicRepoMock{}
	entriesMock := &entryRepoMock{}

	svc := newLinkTestService(t, topicsMock, entriesMock)
	err := svc.LinkEntry(ctx, LinkEntryInput{TopicID: uuid.New(), EntryID: uuid.New()})

	if !errors.Is(err, domain.ErrUnauthorized) {
		t.Fatalf("expected ErrUnauthorized, got %v", err)
	}
}

func TestLinkEntry_NilTopicID(t *testing.T) {
	t.Parallel()

	userID := uuid.New()
	ctx := ctxutil.WithUserID(context.Background(), userID)

	topicsMock := &topicRepoMock{}
	entriesMock := &entryRepoMock{}

	svc := newLinkTestService(t, topicsMock, entriesMock)
	err := svc.LinkEntry(ctx, LinkEntryInput{TopicID: uuid.Nil, EntryID: uuid.New()})

	var ve *domain.ValidationError
	if !errors.As(err, &ve) {
		t.Fatalf("expected ValidationError, got %v", err)
	}
	found := false
	for _, fe := range ve.Errors {
		if fe.Field == "topic_id" {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("expected field error for topic_id, got %v", ve.Errors)
	}
}

func TestLinkEntry_NilEntryID(t *testing.T) {
	t.Parallel()

	userID := uuid.New()
	ctx := ctxutil.WithUserID(context.Background(), userID)

	topicsMock := &topicRepoMock{}
	entriesMock := &entryRepoMock{}

	svc := newLinkTestService(t, topicsMock, entriesMock)
	err := svc.LinkEntry(ctx, LinkEntryInput{TopicID: uuid.New(), EntryID: uuid.Nil})

	var ve *domain.ValidationError
	if !errors.As(err, &ve) {
		t.Fatalf("expected ValidationError, got %v", err)
	}
	found := false
	for _, fe := range ve.Errors {
		if fe.Field == "entry_id" {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("expected field error for entry_id, got %v", ve.Errors)
	}
}

// --- UnlinkEntry tests ---

func TestUnlinkEntry_Success(t *testing.T) {
	t.Parallel()

	userID := uuid.New()
	topicID := uuid.New()
	entryID := uuid.New()
	ctx := ctxutil.WithUserID(context.Background(), userID)

	topicsMock := &topicRepoMock{
		GetByIDFunc: func(_ context.Context, uid, tid uuid.UUID) (*domain.Topic, error) {
			return &domain.Topic{ID: tid, UserID: uid, Name: "test"}, nil
		},
		UnlinkEntryFunc: func(_ context.Context, eid, tid uuid.UUID) error {
			return nil
		},
	}
	entriesMock := &entryRepoMock{}

	svc := newLinkTestService(t, topicsMock, entriesMock)
	err := svc.UnlinkEntry(ctx, UnlinkEntryInput{TopicID: topicID, EntryID: entryID})

	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if len(topicsMock.GetByIDCalls()) != 1 {
		t.Fatalf("expected 1 topic GetByID call, got %d", len(topicsMock.GetByIDCalls()))
	}
	// UnlinkEntry does NOT check entry ownership
	if len(entriesMock.GetByIDCalls()) != 0 {
		t.Fatalf("expected 0 entry GetByID calls, got %d", len(entriesMock.GetByIDCalls()))
	}
	if len(topicsMock.UnlinkEntryCalls()) != 1 {
		t.Fatalf("expected 1 UnlinkEntry call, got %d", len(topicsMock.UnlinkEntryCalls()))
	}
}

func TestUnlinkEntry_TopicNotFound(t *testing.T) {
	t.Parallel()

	userID := uuid.New()
	ctx := ctxutil.WithUserID(context.Background(), userID)

	topicsMock := &topicRepoMock{
		GetByIDFunc: func(_ context.Context, _, _ uuid.UUID) (*domain.Topic, error) {
			return nil, domain.ErrNotFound
		},
	}
	entriesMock := &entryRepoMock{}

	svc := newLinkTestService(t, topicsMock, entriesMock)
	err := svc.UnlinkEntry(ctx, UnlinkEntryInput{TopicID: uuid.New(), EntryID: uuid.New()})

	if !errors.Is(err, domain.ErrNotFound) {
		t.Fatalf("expected ErrNotFound, got %v", err)
	}
}

func TestUnlinkEntry_NotLinked(t *testing.T) {
	t.Parallel()

	userID := uuid.New()
	ctx := ctxutil.WithUserID(context.Background(), userID)

	topicsMock := &topicRepoMock{
		GetByIDFunc: func(_ context.Context, uid, tid uuid.UUID) (*domain.Topic, error) {
			return &domain.Topic{ID: tid, UserID: uid, Name: "test"}, nil
		},
		UnlinkEntryFunc: func(_ context.Context, _, _ uuid.UUID) error {
			return nil // 0 affected rows — idempotent
		},
	}
	entriesMock := &entryRepoMock{}

	svc := newLinkTestService(t, topicsMock, entriesMock)
	err := svc.UnlinkEntry(ctx, UnlinkEntryInput{TopicID: uuid.New(), EntryID: uuid.New()})

	if err != nil {
		t.Fatalf("expected no error for idempotent unlink, got %v", err)
	}
}

func TestUnlinkEntry_Unauthorized(t *testing.T) {
	t.Parallel()

	ctx := context.Background()

	topicsMock := &topicRepoMock{}
	entriesMock := &entryRepoMock{}

	svc := newLinkTestService(t, topicsMock, entriesMock)
	err := svc.UnlinkEntry(ctx, UnlinkEntryInput{TopicID: uuid.New(), EntryID: uuid.New()})

	if !errors.Is(err, domain.ErrUnauthorized) {
		t.Fatalf("expected ErrUnauthorized, got %v", err)
	}
}

// --- BatchLinkEntries tests ---

func TestBatchLinkEntries_Success(t *testing.T) {
	t.Parallel()

	userID := uuid.New()
	topicID := uuid.New()
	entryIDs := make([]uuid.UUID, 5)
	existMap := make(map[uuid.UUID]bool)
	for i := range entryIDs {
		entryIDs[i] = uuid.New()
		existMap[entryIDs[i]] = true
	}
	ctx := ctxutil.WithUserID(context.Background(), userID)

	topicsMock := &topicRepoMock{
		GetByIDFunc: func(_ context.Context, uid, tid uuid.UUID) (*domain.Topic, error) {
			return &domain.Topic{ID: tid, UserID: uid, Name: "test"}, nil
		},
		BatchLinkEntriesFunc: func(_ context.Context, eids []uuid.UUID, tid uuid.UUID) (int, error) {
			return len(eids), nil // all newly linked
		},
	}
	entriesMock := &entryRepoMock{
		ExistByIDsFunc: func(_ context.Context, _ uuid.UUID, _ []uuid.UUID) (map[uuid.UUID]bool, error) {
			return existMap, nil
		},
	}

	svc := newLinkTestService(t, topicsMock, entriesMock)
	result, err := svc.BatchLinkEntries(ctx, BatchLinkEntriesInput{TopicID: topicID, EntryIDs: entryIDs})

	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if result.Linked != 5 {
		t.Fatalf("expected 5 linked, got %d", result.Linked)
	}
	if result.Skipped != 0 {
		t.Fatalf("expected 0 skipped, got %d", result.Skipped)
	}
}

func TestBatchLinkEntries_SomeAlreadyLinked(t *testing.T) {
	t.Parallel()

	userID := uuid.New()
	topicID := uuid.New()
	entryIDs := make([]uuid.UUID, 5)
	existMap := make(map[uuid.UUID]bool)
	for i := range entryIDs {
		entryIDs[i] = uuid.New()
		existMap[entryIDs[i]] = true
	}
	ctx := ctxutil.WithUserID(context.Background(), userID)

	topicsMock := &topicRepoMock{
		GetByIDFunc: func(_ context.Context, uid, tid uuid.UUID) (*domain.Topic, error) {
			return &domain.Topic{ID: tid, UserID: uid, Name: "test"}, nil
		},
		BatchLinkEntriesFunc: func(_ context.Context, eids []uuid.UUID, tid uuid.UUID) (int, error) {
			return 3, nil // 2 already linked, 3 newly linked
		},
	}
	entriesMock := &entryRepoMock{
		ExistByIDsFunc: func(_ context.Context, _ uuid.UUID, _ []uuid.UUID) (map[uuid.UUID]bool, error) {
			return existMap, nil
		},
	}

	svc := newLinkTestService(t, topicsMock, entriesMock)
	result, err := svc.BatchLinkEntries(ctx, BatchLinkEntriesInput{TopicID: topicID, EntryIDs: entryIDs})

	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if result.Linked != 3 {
		t.Fatalf("expected 3 linked, got %d", result.Linked)
	}
	if result.Skipped != 2 {
		t.Fatalf("expected 2 skipped, got %d", result.Skipped)
	}
}

func TestBatchLinkEntries_SomeNotFound(t *testing.T) {
	t.Parallel()

	userID := uuid.New()
	topicID := uuid.New()
	entryIDs := make([]uuid.UUID, 5)
	existMap := make(map[uuid.UUID]bool)
	for i := range entryIDs {
		entryIDs[i] = uuid.New()
		if i < 3 {
			existMap[entryIDs[i]] = true // only 3 exist
		}
	}
	ctx := ctxutil.WithUserID(context.Background(), userID)

	topicsMock := &topicRepoMock{
		GetByIDFunc: func(_ context.Context, uid, tid uuid.UUID) (*domain.Topic, error) {
			return &domain.Topic{ID: tid, UserID: uid, Name: "test"}, nil
		},
		BatchLinkEntriesFunc: func(_ context.Context, eids []uuid.UUID, tid uuid.UUID) (int, error) {
			if len(eids) != 3 {
				t.Errorf("expected 3 valid entry IDs, got %d", len(eids))
			}
			return 3, nil
		},
	}
	entriesMock := &entryRepoMock{
		ExistByIDsFunc: func(_ context.Context, _ uuid.UUID, _ []uuid.UUID) (map[uuid.UUID]bool, error) {
			return existMap, nil
		},
	}

	svc := newLinkTestService(t, topicsMock, entriesMock)
	result, err := svc.BatchLinkEntries(ctx, BatchLinkEntriesInput{TopicID: topicID, EntryIDs: entryIDs})

	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if result.Linked != 3 {
		t.Fatalf("expected 3 linked, got %d", result.Linked)
	}
	if result.Skipped != 2 {
		t.Fatalf("expected 2 skipped, got %d", result.Skipped)
	}
}

func TestBatchLinkEntries_AllAlreadyLinked(t *testing.T) {
	t.Parallel()

	userID := uuid.New()
	topicID := uuid.New()
	entryIDs := make([]uuid.UUID, 5)
	existMap := make(map[uuid.UUID]bool)
	for i := range entryIDs {
		entryIDs[i] = uuid.New()
		existMap[entryIDs[i]] = true
	}
	ctx := ctxutil.WithUserID(context.Background(), userID)

	topicsMock := &topicRepoMock{
		GetByIDFunc: func(_ context.Context, uid, tid uuid.UUID) (*domain.Topic, error) {
			return &domain.Topic{ID: tid, UserID: uid, Name: "test"}, nil
		},
		BatchLinkEntriesFunc: func(_ context.Context, eids []uuid.UUID, tid uuid.UUID) (int, error) {
			return 0, nil // all already linked
		},
	}
	entriesMock := &entryRepoMock{
		ExistByIDsFunc: func(_ context.Context, _ uuid.UUID, _ []uuid.UUID) (map[uuid.UUID]bool, error) {
			return existMap, nil
		},
	}

	svc := newLinkTestService(t, topicsMock, entriesMock)
	result, err := svc.BatchLinkEntries(ctx, BatchLinkEntriesInput{TopicID: topicID, EntryIDs: entryIDs})

	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if result.Linked != 0 {
		t.Fatalf("expected 0 linked, got %d", result.Linked)
	}
	if result.Skipped != 5 {
		t.Fatalf("expected 5 skipped, got %d", result.Skipped)
	}
}

func TestBatchLinkEntries_TopicNotFound(t *testing.T) {
	t.Parallel()

	userID := uuid.New()
	ctx := ctxutil.WithUserID(context.Background(), userID)

	topicsMock := &topicRepoMock{
		GetByIDFunc: func(_ context.Context, _, _ uuid.UUID) (*domain.Topic, error) {
			return nil, domain.ErrNotFound
		},
	}
	entriesMock := &entryRepoMock{}

	svc := newLinkTestService(t, topicsMock, entriesMock)
	_, err := svc.BatchLinkEntries(ctx, BatchLinkEntriesInput{TopicID: uuid.New(), EntryIDs: []uuid.UUID{uuid.New()}})

	if !errors.Is(err, domain.ErrNotFound) {
		t.Fatalf("expected ErrNotFound, got %v", err)
	}
}

func TestBatchLinkEntries_EmptyInput(t *testing.T) {
	t.Parallel()

	userID := uuid.New()
	ctx := ctxutil.WithUserID(context.Background(), userID)

	topicsMock := &topicRepoMock{}
	entriesMock := &entryRepoMock{}

	svc := newLinkTestService(t, topicsMock, entriesMock)
	_, err := svc.BatchLinkEntries(ctx, BatchLinkEntriesInput{TopicID: uuid.New(), EntryIDs: []uuid.UUID{}})

	var ve *domain.ValidationError
	if !errors.As(err, &ve) {
		t.Fatalf("expected ValidationError, got %v", err)
	}
	found := false
	for _, fe := range ve.Errors {
		if fe.Field == "entry_ids" {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("expected field error for entry_ids, got %v", ve.Errors)
	}
}

func TestBatchLinkEntries_TooMany(t *testing.T) {
	t.Parallel()

	userID := uuid.New()
	ctx := ctxutil.WithUserID(context.Background(), userID)

	entryIDs := make([]uuid.UUID, 201)
	for i := range entryIDs {
		entryIDs[i] = uuid.New()
	}

	topicsMock := &topicRepoMock{}
	entriesMock := &entryRepoMock{}

	svc := newLinkTestService(t, topicsMock, entriesMock)
	_, err := svc.BatchLinkEntries(ctx, BatchLinkEntriesInput{TopicID: uuid.New(), EntryIDs: entryIDs})

	var ve *domain.ValidationError
	if !errors.As(err, &ve) {
		t.Fatalf("expected ValidationError, got %v", err)
	}
	found := false
	for _, fe := range ve.Errors {
		if fe.Field == "entry_ids" {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("expected field error for entry_ids, got %v", ve.Errors)
	}
}

func TestBatchLinkEntries_Unauthorized(t *testing.T) {
	t.Parallel()

	ctx := context.Background()

	topicsMock := &topicRepoMock{}
	entriesMock := &entryRepoMock{}

	svc := newLinkTestService(t, topicsMock, entriesMock)
	_, err := svc.BatchLinkEntries(ctx, BatchLinkEntriesInput{TopicID: uuid.New(), EntryIDs: []uuid.UUID{uuid.New()}})

	if !errors.Is(err, domain.ErrUnauthorized) {
		t.Fatalf("expected ErrUnauthorized, got %v", err)
	}
}

func TestBatchLinkEntries_AllEntriesNotFound(t *testing.T) {
	t.Parallel()

	userID := uuid.New()
	topicID := uuid.New()
	entryIDs := make([]uuid.UUID, 3)
	for i := range entryIDs {
		entryIDs[i] = uuid.New()
	}
	ctx := ctxutil.WithUserID(context.Background(), userID)

	topicsMock := &topicRepoMock{
		GetByIDFunc: func(_ context.Context, uid, tid uuid.UUID) (*domain.Topic, error) {
			return &domain.Topic{ID: tid, UserID: uid, Name: "test"}, nil
		},
	}
	entriesMock := &entryRepoMock{
		ExistByIDsFunc: func(_ context.Context, _ uuid.UUID, _ []uuid.UUID) (map[uuid.UUID]bool, error) {
			return map[uuid.UUID]bool{}, nil // none exist
		},
	}

	svc := newLinkTestService(t, topicsMock, entriesMock)
	result, err := svc.BatchLinkEntries(ctx, BatchLinkEntriesInput{TopicID: topicID, EntryIDs: entryIDs})

	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if result.Linked != 0 {
		t.Fatalf("expected 0 linked, got %d", result.Linked)
	}
	if result.Skipped != 3 {
		t.Fatalf("expected 3 skipped, got %d", result.Skipped)
	}
	// BatchLinkEntries should NOT have been called since no valid IDs
	if len(topicsMock.BatchLinkEntriesCalls()) != 0 {
		t.Fatalf("expected 0 BatchLinkEntries calls, got %d", len(topicsMock.BatchLinkEntriesCalls()))
	}
}
