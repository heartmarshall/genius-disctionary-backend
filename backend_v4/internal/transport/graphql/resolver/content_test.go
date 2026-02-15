package resolver

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"
	"github.com/heartmarshall/myenglish-backend/internal/domain"
	"github.com/heartmarshall/myenglish-backend/internal/service/content"
	"github.com/heartmarshall/myenglish-backend/internal/transport/graphql/generated"
	"github.com/heartmarshall/myenglish-backend/pkg/ctxutil"
	"github.com/stretchr/testify/require"
)

// ============================================================================
// Sense tests
// ============================================================================

func TestAddSense_Success(t *testing.T) {
	t.Parallel()

	senseID := uuid.New()
	entryID := uuid.New()

	mock := &contentServiceMock{
		AddSenseFunc: func(ctx context.Context, input content.AddSenseInput) (*domain.Sense, error) {
			return &domain.Sense{
				ID:         senseID,
				EntryID:    entryID,
				Definition: ptr("test definition"),
			}, nil
		},
	}

	resolver := &mutationResolver{&Resolver{content: mock}}
	ctx := ctxutil.WithUserID(context.Background(), uuid.New())

	result, err := resolver.AddSense(ctx, generated.AddSenseInput{
		EntryID:    entryID,
		Definition: ptr("test definition"),
	})

	require.NoError(t, err)
	require.NotNil(t, result.Sense)
	require.Equal(t, senseID, result.Sense.ID)
	require.Equal(t, 1, len(mock.AddSenseCalls()))
}

func TestAddSense_Unauthorized(t *testing.T) {
	t.Parallel()

	mock := &contentServiceMock{}
	resolver := &mutationResolver{&Resolver{content: mock}}
	ctx := context.Background() // No user ID

	result, err := resolver.AddSense(ctx, generated.AddSenseInput{
		EntryID: uuid.New(),
	})

	require.ErrorIs(t, err, domain.ErrUnauthorized)
	require.Nil(t, result)
	require.Equal(t, 0, len(mock.AddSenseCalls()))
}

func TestUpdateSense_Success(t *testing.T) {
	t.Parallel()

	senseID := uuid.New()

	mock := &contentServiceMock{
		UpdateSenseFunc: func(ctx context.Context, input content.UpdateSenseInput) (*domain.Sense, error) {
			return &domain.Sense{
				ID:         senseID,
				Definition: ptr("updated definition"),
			}, nil
		},
	}

	resolver := &mutationResolver{&Resolver{content: mock}}
	ctx := ctxutil.WithUserID(context.Background(), uuid.New())

	result, err := resolver.UpdateSense(ctx, generated.UpdateSenseInput{
		SenseID:    senseID,
		Definition: ptr("updated definition"),
	})

	require.NoError(t, err)
	require.NotNil(t, result.Sense)
	require.Equal(t, "updated definition", *result.Sense.Definition)
	require.Equal(t, 1, len(mock.UpdateSenseCalls()))
}

func TestUpdateSense_NotFound(t *testing.T) {
	t.Parallel()

	mock := &contentServiceMock{
		UpdateSenseFunc: func(ctx context.Context, input content.UpdateSenseInput) (*domain.Sense, error) {
			return nil, domain.ErrNotFound
		},
	}

	resolver := &mutationResolver{&Resolver{content: mock}}
	ctx := ctxutil.WithUserID(context.Background(), uuid.New())

	result, err := resolver.UpdateSense(ctx, generated.UpdateSenseInput{
		SenseID: uuid.New(),
	})

	require.ErrorIs(t, err, domain.ErrNotFound)
	require.Nil(t, result)
}

func TestDeleteSense_Success(t *testing.T) {
	t.Parallel()

	senseID := uuid.New()

	mock := &contentServiceMock{
		DeleteSenseFunc: func(ctx context.Context, id uuid.UUID) error {
			return nil
		},
	}

	resolver := &mutationResolver{&Resolver{content: mock}}
	ctx := ctxutil.WithUserID(context.Background(), uuid.New())

	result, err := resolver.DeleteSense(ctx, senseID)

	require.NoError(t, err)
	require.NotNil(t, result)
	require.Equal(t, senseID, result.SenseID)
	require.Equal(t, 1, len(mock.DeleteSenseCalls()))
}

func TestDeleteSense_ServiceError(t *testing.T) {
	t.Parallel()

	mock := &contentServiceMock{
		DeleteSenseFunc: func(ctx context.Context, id uuid.UUID) error {
			return errors.New("db error")
		},
	}

	resolver := &mutationResolver{&Resolver{content: mock}}
	ctx := ctxutil.WithUserID(context.Background(), uuid.New())

	result, err := resolver.DeleteSense(ctx, uuid.New())

	require.Error(t, err)
	require.Nil(t, result)
}

func TestReorderSenses_Success(t *testing.T) {
	t.Parallel()

	entryID := uuid.New()

	mock := &contentServiceMock{
		ReorderSensesFunc: func(ctx context.Context, input content.ReorderSensesInput) error {
			return nil
		},
	}

	resolver := &mutationResolver{&Resolver{content: mock}}
	ctx := ctxutil.WithUserID(context.Background(), uuid.New())

	result, err := resolver.ReorderSenses(ctx, generated.ReorderSensesInput{
		EntryID: entryID,
		Items: []*generated.ReorderItemInput{
			{ID: uuid.New(), Position: 0},
			{ID: uuid.New(), Position: 1},
		},
	})

	require.NoError(t, err)
	require.NotNil(t, result)
	require.True(t, result.Success)
	require.Equal(t, 1, len(mock.ReorderSensesCalls()))
}

func TestReorderSenses_Unauthorized(t *testing.T) {
	t.Parallel()

	mock := &contentServiceMock{}
	resolver := &mutationResolver{&Resolver{content: mock}}
	ctx := context.Background() // No user ID

	result, err := resolver.ReorderSenses(ctx, generated.ReorderSensesInput{
		EntryID: uuid.New(),
		Items: []*generated.ReorderItemInput{
			{ID: uuid.New(), Position: 0},
		},
	})

	require.ErrorIs(t, err, domain.ErrUnauthorized)
	require.Nil(t, result)
	require.Equal(t, 0, len(mock.ReorderSensesCalls()))
}

// ============================================================================
// Translation tests
// ============================================================================

func TestAddTranslation_Success(t *testing.T) {
	t.Parallel()

	translationID := uuid.New()
	senseID := uuid.New()

	mock := &contentServiceMock{
		AddTranslationFunc: func(ctx context.Context, input content.AddTranslationInput) (*domain.Translation, error) {
			return &domain.Translation{
				ID:       translationID,
				SenseID:  senseID,
				Text:     ptr("translation text"),
				Position: 0,
			}, nil
		},
	}

	resolver := &mutationResolver{&Resolver{content: mock}}
	ctx := ctxutil.WithUserID(context.Background(), uuid.New())

	result, err := resolver.AddTranslation(ctx, generated.AddTranslationInput{
		SenseID: senseID,
		Text:    "translation text",
	})

	require.NoError(t, err)
	require.NotNil(t, result.Translation)
	require.Equal(t, translationID, result.Translation.ID)
}

func TestAddTranslation_Unauthorized(t *testing.T) {
	t.Parallel()

	mock := &contentServiceMock{}
	resolver := &mutationResolver{&Resolver{content: mock}}
	ctx := context.Background()

	result, err := resolver.AddTranslation(ctx, generated.AddTranslationInput{
		SenseID: uuid.New(),
		Text:    "text",
	})

	require.ErrorIs(t, err, domain.ErrUnauthorized)
	require.Nil(t, result)
}

func TestUpdateTranslation_Success(t *testing.T) {
	t.Parallel()

	translationID := uuid.New()

	mock := &contentServiceMock{
		UpdateTranslationFunc: func(ctx context.Context, input content.UpdateTranslationInput) (*domain.Translation, error) {
			return &domain.Translation{
				ID:       translationID,
				Text:     ptr("updated text"),
				Position: 0,
			}, nil
		},
	}

	resolver := &mutationResolver{&Resolver{content: mock}}
	ctx := ctxutil.WithUserID(context.Background(), uuid.New())

	result, err := resolver.UpdateTranslation(ctx, generated.UpdateTranslationInput{
		TranslationID: translationID,
		Text:          "updated text",
	})

	require.NoError(t, err)
	require.NotNil(t, result.Translation)
	require.Equal(t, "updated text", *result.Translation.Text)
}

func TestDeleteTranslation_Success(t *testing.T) {
	t.Parallel()

	translationID := uuid.New()

	mock := &contentServiceMock{
		DeleteTranslationFunc: func(ctx context.Context, id uuid.UUID) error {
			return nil
		},
	}

	resolver := &mutationResolver{&Resolver{content: mock}}
	ctx := ctxutil.WithUserID(context.Background(), uuid.New())

	result, err := resolver.DeleteTranslation(ctx, translationID)

	require.NoError(t, err)
	require.NotNil(t, result)
	require.Equal(t, translationID, result.TranslationID)
}

func TestReorderTranslations_Success(t *testing.T) {
	t.Parallel()

	senseID := uuid.New()

	mock := &contentServiceMock{
		ReorderTranslationsFunc: func(ctx context.Context, input content.ReorderTranslationsInput) error {
			return nil
		},
	}

	resolver := &mutationResolver{&Resolver{content: mock}}
	ctx := ctxutil.WithUserID(context.Background(), uuid.New())

	result, err := resolver.ReorderTranslations(ctx, generated.ReorderTranslationsInput{
		SenseID: senseID,
		Items: []*generated.ReorderItemInput{
			{ID: uuid.New(), Position: 0},
		},
	})

	require.NoError(t, err)
	require.NotNil(t, result)
	require.True(t, result.Success)
}

// ============================================================================
// Example tests
// ============================================================================

func TestAddExample_Success(t *testing.T) {
	t.Parallel()

	exampleID := uuid.New()
	senseID := uuid.New()

	mock := &contentServiceMock{
		AddExampleFunc: func(ctx context.Context, input content.AddExampleInput) (*domain.Example, error) {
			return &domain.Example{
				ID:       exampleID,
				SenseID:  senseID,
				Sentence: ptr("example sentence"),
				Position: 0,
			}, nil
		},
	}

	resolver := &mutationResolver{&Resolver{content: mock}}
	ctx := ctxutil.WithUserID(context.Background(), uuid.New())

	result, err := resolver.AddExample(ctx, generated.AddExampleInput{
		SenseID:  senseID,
		Sentence: "example sentence",
	})

	require.NoError(t, err)
	require.NotNil(t, result.Example)
	require.Equal(t, exampleID, result.Example.ID)
}

func TestUpdateExample_Success(t *testing.T) {
	t.Parallel()

	exampleID := uuid.New()

	mock := &contentServiceMock{
		UpdateExampleFunc: func(ctx context.Context, input content.UpdateExampleInput) (*domain.Example, error) {
			return &domain.Example{
				ID:       exampleID,
				Sentence: ptr("updated sentence"),
				Position: 0,
			}, nil
		},
	}

	resolver := &mutationResolver{&Resolver{content: mock}}
	ctx := ctxutil.WithUserID(context.Background(), uuid.New())

	result, err := resolver.UpdateExample(ctx, generated.UpdateExampleInput{
		ExampleID: exampleID,
		Sentence:  ptr("updated sentence"),
	})

	require.NoError(t, err)
	require.NotNil(t, result.Example)
	require.Equal(t, "updated sentence", *result.Example.Sentence)
}

func TestDeleteExample_Success(t *testing.T) {
	t.Parallel()

	exampleID := uuid.New()

	mock := &contentServiceMock{
		DeleteExampleFunc: func(ctx context.Context, id uuid.UUID) error {
			return nil
		},
	}

	resolver := &mutationResolver{&Resolver{content: mock}}
	ctx := ctxutil.WithUserID(context.Background(), uuid.New())

	result, err := resolver.DeleteExample(ctx, exampleID)

	require.NoError(t, err)
	require.NotNil(t, result)
	require.Equal(t, exampleID, result.ExampleID)
}

func TestReorderExamples_Success(t *testing.T) {
	t.Parallel()

	senseID := uuid.New()

	mock := &contentServiceMock{
		ReorderExamplesFunc: func(ctx context.Context, input content.ReorderExamplesInput) error {
			return nil
		},
	}

	resolver := &mutationResolver{&Resolver{content: mock}}
	ctx := ctxutil.WithUserID(context.Background(), uuid.New())

	result, err := resolver.ReorderExamples(ctx, generated.ReorderExamplesInput{
		SenseID: senseID,
		Items: []*generated.ReorderItemInput{
			{ID: uuid.New(), Position: 0},
		},
	})

	require.NoError(t, err)
	require.NotNil(t, result)
	require.True(t, result.Success)
}

// ============================================================================
// UserImage tests
// ============================================================================

func TestAddUserImage_Success(t *testing.T) {
	t.Parallel()

	imageID := uuid.New()
	entryID := uuid.New()

	mock := &contentServiceMock{
		AddUserImageFunc: func(ctx context.Context, input content.AddUserImageInput) (*domain.UserImage, error) {
			return &domain.UserImage{
				ID:      imageID,
				EntryID: entryID,
				URL:     "https://example.com/image.jpg",
			}, nil
		},
	}

	resolver := &mutationResolver{&Resolver{content: mock}}
	ctx := ctxutil.WithUserID(context.Background(), uuid.New())

	result, err := resolver.AddUserImage(ctx, generated.AddUserImageInput{
		EntryID: entryID,
		URL:     "https://example.com/image.jpg",
	})

	require.NoError(t, err)
	require.NotNil(t, result.Image)
	require.Equal(t, imageID, result.Image.ID)
}

func TestAddUserImage_Unauthorized(t *testing.T) {
	t.Parallel()

	mock := &contentServiceMock{}
	resolver := &mutationResolver{&Resolver{content: mock}}
	ctx := context.Background()

	result, err := resolver.AddUserImage(ctx, generated.AddUserImageInput{
		EntryID: uuid.New(),
		URL:     "https://example.com/image.jpg",
	})

	require.ErrorIs(t, err, domain.ErrUnauthorized)
	require.Nil(t, result)
}

func TestDeleteUserImage_Success(t *testing.T) {
	t.Parallel()

	imageID := uuid.New()

	mock := &contentServiceMock{
		DeleteUserImageFunc: func(ctx context.Context, id uuid.UUID) error {
			return nil
		},
	}

	resolver := &mutationResolver{&Resolver{content: mock}}
	ctx := ctxutil.WithUserID(context.Background(), uuid.New())

	result, err := resolver.DeleteUserImage(ctx, imageID)

	require.NoError(t, err)
	require.NotNil(t, result)
	require.Equal(t, imageID, result.ImageID)
}
