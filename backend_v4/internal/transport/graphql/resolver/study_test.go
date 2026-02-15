package resolver

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"
	"github.com/heartmarshall/myenglish-backend/internal/domain"
	"github.com/heartmarshall/myenglish-backend/internal/service/study"
	"github.com/heartmarshall/myenglish-backend/internal/transport/graphql/generated"
	"github.com/heartmarshall/myenglish-backend/pkg/ctxutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ---------------------------------------------------------------------------
// Query Tests
// ---------------------------------------------------------------------------

// TestStudyQueue_Success tests successful queue retrieval.
func TestStudyQueue_Success(t *testing.T) {
	t.Parallel()

	userID := uuid.New()
	entryID := uuid.New()
	cardID := uuid.New()

	studyMock := &studyServiceMock{
		GetStudyQueueFunc: func(ctx context.Context, input study.GetQueueInput) ([]*domain.Card, error) {
			assert.Equal(t, 20, input.Limit)
			return []*domain.Card{
				{ID: cardID, EntryID: entryID, UserID: userID},
			}, nil
		},
	}

	dictMock := &dictionaryServiceMock{
		GetEntryFunc: func(ctx context.Context, id uuid.UUID) (*domain.Entry, error) {
			return &domain.Entry{ID: entryID, Text: "test"}, nil
		},
	}

	resolver := &queryResolver{&Resolver{study: studyMock, dictionary: dictMock}}
	ctx := ctxutil.WithUserID(context.Background(), userID)

	result, err := resolver.StudyQueue(ctx, nil)

	require.NoError(t, err)
	require.Len(t, result, 1)
	assert.Equal(t, entryID, result[0].ID)
}

// TestStudyQueue_CustomLimit tests custom limit.
func TestStudyQueue_CustomLimit(t *testing.T) {
	t.Parallel()

	userID := uuid.New()

	studyMock := &studyServiceMock{
		GetStudyQueueFunc: func(ctx context.Context, input study.GetQueueInput) ([]*domain.Card, error) {
			assert.Equal(t, 50, input.Limit)
			return []*domain.Card{}, nil
		},
	}

	resolver := &queryResolver{&Resolver{study: studyMock}}
	ctx := ctxutil.WithUserID(context.Background(), userID)

	_, err := resolver.StudyQueue(ctx, ptr(50))

	require.NoError(t, err)
}

// TestStudyQueue_Unauthorized tests missing user ID.
func TestStudyQueue_Unauthorized(t *testing.T) {
	t.Parallel()

	resolver := &queryResolver{&Resolver{study: &studyServiceMock{}}}
	_, err := resolver.StudyQueue(context.Background(), nil)

	require.ErrorIs(t, err, domain.ErrUnauthorized)
}

// TestDashboard_Success tests successful dashboard retrieval.
func TestDashboard_Success(t *testing.T) {
	t.Parallel()

	userID := uuid.New()

	studyMock := &studyServiceMock{
		GetDashboardFunc: func(ctx context.Context) (domain.Dashboard, error) {
			return domain.Dashboard{
				DueCount:      10,
				NewCount:      5,
				ReviewedToday: 3,
				Streak:        7,
			}, nil
		},
	}

	resolver := &queryResolver{&Resolver{study: studyMock}}
	ctx := ctxutil.WithUserID(context.Background(), userID)

	result, err := resolver.Dashboard(ctx)

	require.NoError(t, err)
	assert.Equal(t, 10, result.DueCount)
	assert.Equal(t, 5, result.NewCount)
	assert.Equal(t, 7, result.Streak)
}

// TestDashboard_Unauthorized tests missing user ID.
func TestDashboard_Unauthorized(t *testing.T) {
	t.Parallel()

	resolver := &queryResolver{&Resolver{study: &studyServiceMock{}}}
	_, err := resolver.Dashboard(context.Background())

	require.ErrorIs(t, err, domain.ErrUnauthorized)
}

// TestCardHistory_Success tests successful history retrieval.
func TestCardHistory_Success(t *testing.T) {
	t.Parallel()

	userID := uuid.New()
	cardID := uuid.New()

	studyMock := &studyServiceMock{
		GetCardHistoryFunc: func(ctx context.Context, input study.GetCardHistoryInput) ([]*domain.ReviewLog, int, error) {
			assert.Equal(t, cardID, input.CardID)
			assert.Equal(t, 50, input.Limit)
			assert.Equal(t, 0, input.Offset)
			return []*domain.ReviewLog{
				{CardID: cardID, Grade: domain.ReviewGradeGood},
			}, 1, nil
		},
	}

	resolver := &queryResolver{&Resolver{study: studyMock}}
	ctx := ctxutil.WithUserID(context.Background(), userID)

	result, err := resolver.CardHistory(ctx, generated.GetCardHistoryInput{CardID: cardID})

	require.NoError(t, err)
	require.Len(t, result, 1)
}

// TestCardHistory_CustomPagination tests custom limit and offset.
func TestCardHistory_CustomPagination(t *testing.T) {
	t.Parallel()

	userID := uuid.New()
	cardID := uuid.New()

	studyMock := &studyServiceMock{
		GetCardHistoryFunc: func(ctx context.Context, input study.GetCardHistoryInput) ([]*domain.ReviewLog, int, error) {
			assert.Equal(t, 100, input.Limit)
			assert.Equal(t, 20, input.Offset)
			return []*domain.ReviewLog{}, 0, nil
		},
	}

	resolver := &queryResolver{&Resolver{study: studyMock}}
	ctx := ctxutil.WithUserID(context.Background(), userID)

	_, err := resolver.CardHistory(ctx, generated.GetCardHistoryInput{
		CardID: cardID,
		Limit:  ptr(100),
		Offset: ptr(20),
	})

	require.NoError(t, err)
}

// TestCardStats_Success tests successful stats retrieval.
func TestCardStats_Success(t *testing.T) {
	t.Parallel()

	userID := uuid.New()
	cardID := uuid.New()

	avgTime := 1500
	studyMock := &studyServiceMock{
		GetCardStatsFunc: func(ctx context.Context, input study.GetCardHistoryInput) (domain.CardStats, error) {
			assert.Equal(t, cardID, input.CardID)
			return domain.CardStats{
				TotalReviews:  10,
				AccuracyRate:  85.5,
				AverageTimeMs: &avgTime,
			}, nil
		},
	}

	resolver := &queryResolver{&Resolver{study: studyMock}}
	ctx := ctxutil.WithUserID(context.Background(), userID)

	result, err := resolver.CardStats(ctx, cardID)

	require.NoError(t, err)
	assert.Equal(t, 10, result.TotalReviews)
	assert.Equal(t, 85.5, result.AccuracyRate)
}

// ---------------------------------------------------------------------------
// Mutation Tests
// ---------------------------------------------------------------------------

// TestReviewCard_Success tests successful card review.
func TestReviewCard_Success(t *testing.T) {
	t.Parallel()

	userID := uuid.New()
	cardID := uuid.New()

	studyMock := &studyServiceMock{
		ReviewCardFunc: func(ctx context.Context, input study.ReviewCardInput) (*domain.Card, error) {
			assert.Equal(t, cardID, input.CardID)
			assert.Equal(t, domain.ReviewGradeGood, input.Grade)
			return &domain.Card{ID: cardID}, nil
		},
	}

	resolver := &mutationResolver{&Resolver{study: studyMock}}
	ctx := ctxutil.WithUserID(context.Background(), userID)

	result, err := resolver.ReviewCard(ctx, generated.ReviewCardInput{
		CardID: cardID,
		Grade:  domain.ReviewGradeGood,
	})

	require.NoError(t, err)
	assert.Equal(t, cardID, result.Card.ID)
}

// TestReviewCard_Unauthorized tests missing user ID.
func TestReviewCard_Unauthorized(t *testing.T) {
	t.Parallel()

	resolver := &mutationResolver{&Resolver{study: &studyServiceMock{}}}
	_, err := resolver.ReviewCard(context.Background(), generated.ReviewCardInput{})

	require.ErrorIs(t, err, domain.ErrUnauthorized)
}

// TestUndoReview_Success tests successful review undo.
func TestUndoReview_Success(t *testing.T) {
	t.Parallel()

	userID := uuid.New()
	cardID := uuid.New()

	studyMock := &studyServiceMock{
		UndoReviewFunc: func(ctx context.Context, input study.UndoReviewInput) (*domain.Card, error) {
			assert.Equal(t, cardID, input.CardID)
			return &domain.Card{ID: cardID}, nil
		},
	}

	resolver := &mutationResolver{&Resolver{study: studyMock}}
	ctx := ctxutil.WithUserID(context.Background(), userID)

	result, err := resolver.UndoReview(ctx, cardID)

	require.NoError(t, err)
	assert.Equal(t, cardID, result.Card.ID)
}

// TestCreateCard_Success tests successful card creation.
func TestCreateCard_Success(t *testing.T) {
	t.Parallel()

	userID := uuid.New()
	entryID := uuid.New()
	cardID := uuid.New()

	studyMock := &studyServiceMock{
		CreateCardFunc: func(ctx context.Context, input study.CreateCardInput) (*domain.Card, error) {
			assert.Equal(t, entryID, input.EntryID)
			return &domain.Card{ID: cardID, EntryID: entryID}, nil
		},
	}

	resolver := &mutationResolver{&Resolver{study: studyMock}}
	ctx := ctxutil.WithUserID(context.Background(), userID)

	result, err := resolver.CreateCard(ctx, entryID)

	require.NoError(t, err)
	assert.Equal(t, cardID, result.Card.ID)
}

// TestDeleteCard_Success tests successful card deletion.
func TestDeleteCard_Success(t *testing.T) {
	t.Parallel()

	userID := uuid.New()
	cardID := uuid.New()

	studyMock := &studyServiceMock{
		DeleteCardFunc: func(ctx context.Context, input study.DeleteCardInput) error {
			assert.Equal(t, cardID, input.CardID)
			return nil
		},
	}

	resolver := &mutationResolver{&Resolver{study: studyMock}}
	ctx := ctxutil.WithUserID(context.Background(), userID)

	result, err := resolver.DeleteCard(ctx, cardID)

	require.NoError(t, err)
	assert.Equal(t, cardID, result.CardID)
}

// TestBatchCreateCards_Success tests successful batch creation.
func TestBatchCreateCards_Success(t *testing.T) {
	t.Parallel()

	userID := uuid.New()
	entryIDs := []uuid.UUID{uuid.New(), uuid.New()}

	studyMock := &studyServiceMock{
		BatchCreateCardsFunc: func(ctx context.Context, input study.BatchCreateCardsInput) (study.BatchCreateResult, error) {
			return study.BatchCreateResult{
				Created:         2,
				SkippedExisting: 1,
				SkippedNoSenses: 2,
				Errors:          []study.BatchCreateError{},
			}, nil
		},
	}

	resolver := &mutationResolver{&Resolver{study: studyMock}}
	ctx := ctxutil.WithUserID(context.Background(), userID)

	result, err := resolver.BatchCreateCards(ctx, entryIDs)

	require.NoError(t, err)
	assert.Equal(t, 2, result.CreatedCount)
	assert.Equal(t, 3, result.SkippedCount) // 1 + 2
}

// TestBatchCreateCards_WithErrors tests batch creation with errors.
func TestBatchCreateCards_WithErrors(t *testing.T) {
	t.Parallel()

	userID := uuid.New()
	entryID := uuid.New()

	studyMock := &studyServiceMock{
		BatchCreateCardsFunc: func(ctx context.Context, input study.BatchCreateCardsInput) (study.BatchCreateResult, error) {
			return study.BatchCreateResult{
				Created:         0,
				SkippedExisting: 0,
				SkippedNoSenses: 0,
				Errors: []study.BatchCreateError{
					{EntryID: entryID, Reason: "entry not found"},
				},
			}, nil
		},
	}

	resolver := &mutationResolver{&Resolver{study: studyMock}}
	ctx := ctxutil.WithUserID(context.Background(), userID)

	result, err := resolver.BatchCreateCards(ctx, []uuid.UUID{entryID})

	require.NoError(t, err)
	assert.Equal(t, 0, result.CreatedCount)
	require.Len(t, result.Errors, 1)
	assert.Equal(t, entryID, result.Errors[0].EntryID)
}

// TestStartStudySession_Success tests successful session start.
func TestStartStudySession_Success(t *testing.T) {
	t.Parallel()

	userID := uuid.New()
	sessionID := uuid.New()

	studyMock := &studyServiceMock{
		StartSessionFunc: func(ctx context.Context) (*domain.StudySession, error) {
			return &domain.StudySession{ID: sessionID}, nil
		},
	}

	resolver := &mutationResolver{&Resolver{study: studyMock}}
	ctx := ctxutil.WithUserID(context.Background(), userID)

	result, err := resolver.StartStudySession(ctx)

	require.NoError(t, err)
	assert.Equal(t, sessionID, result.Session.ID)
}

// TestFinishStudySession_Success tests successful session finish.
func TestFinishStudySession_Success(t *testing.T) {
	t.Parallel()

	userID := uuid.New()
	sessionID := uuid.New()

	studyMock := &studyServiceMock{
		FinishSessionFunc: func(ctx context.Context, input study.FinishSessionInput) (*domain.StudySession, error) {
			assert.Equal(t, sessionID, input.SessionID)
			return &domain.StudySession{ID: sessionID}, nil
		},
	}

	resolver := &mutationResolver{&Resolver{study: studyMock}}
	ctx := ctxutil.WithUserID(context.Background(), userID)

	result, err := resolver.FinishStudySession(ctx, generated.FinishSessionInput{SessionID: sessionID})

	require.NoError(t, err)
	assert.Equal(t, sessionID, result.Session.ID)
}

// TestAbandonStudySession_Success tests successful session abandon.
func TestAbandonStudySession_Success(t *testing.T) {
	t.Parallel()

	userID := uuid.New()

	studyMock := &studyServiceMock{
		AbandonSessionFunc: func(ctx context.Context) error {
			return nil
		},
	}

	resolver := &mutationResolver{&Resolver{study: studyMock}}
	ctx := ctxutil.WithUserID(context.Background(), userID)

	result, err := resolver.AbandonStudySession(ctx)

	require.NoError(t, err)
	assert.True(t, result.Success)
}

// TestAbandonStudySession_Error tests session abandon with error.
func TestAbandonStudySession_Error(t *testing.T) {
	t.Parallel()

	userID := uuid.New()

	studyMock := &studyServiceMock{
		AbandonSessionFunc: func(ctx context.Context) error {
			return errors.New("no active session")
		},
	}

	resolver := &mutationResolver{&Resolver{study: studyMock}}
	ctx := ctxutil.WithUserID(context.Background(), userID)

	_, err := resolver.AbandonStudySession(ctx)

	require.Error(t, err)
}

// ---------------------------------------------------------------------------
// Field Resolver Tests
// ---------------------------------------------------------------------------

// TestCardStats_AverageDurationMs_WithValue tests average duration field.
func TestCardStats_AverageDurationMs_WithValue(t *testing.T) {
	t.Parallel()

	avgTime := 2500
	stats := &domain.CardStats{AverageTimeMs: &avgTime}
	resolver := &cardStatsResolver{}

	result, err := resolver.AverageDurationMs(context.Background(), stats)

	require.NoError(t, err)
	assert.Equal(t, 2500, result)
}

// TestCardStats_AverageDurationMs_Nil tests average duration field with nil.
func TestCardStats_AverageDurationMs_Nil(t *testing.T) {
	t.Parallel()

	stats := &domain.CardStats{AverageTimeMs: nil}
	resolver := &cardStatsResolver{}

	result, err := resolver.AverageDurationMs(context.Background(), stats)

	require.NoError(t, err)
	assert.Equal(t, 0, result)
}

// TestSessionResult_TotalReviews tests total reviews field.
func TestSessionResult_TotalReviews(t *testing.T) {
	t.Parallel()

	sessionResult := &domain.SessionResult{TotalReviewed: 15}
	resolver := &sessionResultResolver{}

	result, err := resolver.TotalReviews(context.Background(), sessionResult)

	require.NoError(t, err)
	assert.Equal(t, 15, result)
}
