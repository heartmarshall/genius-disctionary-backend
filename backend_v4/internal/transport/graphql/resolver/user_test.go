package resolver

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/heartmarshall/myenglish-backend/internal/domain"
	"github.com/heartmarshall/myenglish-backend/internal/service/user"
	"github.com/heartmarshall/myenglish-backend/internal/transport/graphql/generated"
	"github.com/heartmarshall/myenglish-backend/pkg/ctxutil"
	"github.com/stretchr/testify/require"
)

func TestMe_Success(t *testing.T) {
	t.Parallel()

	userID := uuid.New()
	expectedUser := &domain.User{
		ID:       userID,
		Email:    "test@example.com",
		Username: "testuser",
	}

	mock := &userServiceMock{
		GetProfileFunc: func(ctx context.Context) (*domain.User, error) {
			return expectedUser, nil
		},
	}

	resolver := &queryResolver{&Resolver{user: mock}}
	ctx := ctxutil.WithUserID(context.Background(), userID)

	result, err := resolver.Me(ctx)

	require.NoError(t, err)
	require.NotNil(t, result)
	require.Equal(t, userID, result.ID)
	require.Equal(t, "test@example.com", result.Email)
}

func TestMe_Unauthorized(t *testing.T) {
	t.Parallel()

	resolver := &queryResolver{&Resolver{user: &userServiceMock{}}}
	ctx := context.Background()

	_, err := resolver.Me(ctx)

	require.ErrorIs(t, err, domain.ErrUnauthorized)
}

func TestUpdateSettings_Success(t *testing.T) {
	t.Parallel()

	newCards := 20
	reviews := 100
	maxInterval := 365
	timezone := "Europe/London"

	mock := &userServiceMock{
		UpdateSettingsFunc: func(ctx context.Context, input user.UpdateSettingsInput) (*domain.UserSettings, error) {
			return &domain.UserSettings{
				NewCardsPerDay:  *input.NewCardsPerDay,
				ReviewsPerDay:   *input.ReviewsPerDay,
				MaxIntervalDays: *input.MaxIntervalDays,
				Timezone:        *input.Timezone,
			}, nil
		},
	}

	resolver := &mutationResolver{&Resolver{user: mock}}
	ctx := ctxutil.WithUserID(context.Background(), uuid.New())

	result, err := resolver.UpdateSettings(ctx, generated.UpdateSettingsInput{
		NewCardsPerDay:  &newCards,
		ReviewsPerDay:   &reviews,
		MaxIntervalDays: &maxInterval,
		Timezone:        &timezone,
	})

	require.NoError(t, err)
	require.NotNil(t, result.Settings)
	require.Equal(t, 20, result.Settings.NewCardsPerDay)
	require.Equal(t, 100, result.Settings.ReviewsPerDay)
	require.Equal(t, 365, result.Settings.MaxIntervalDays)
	require.Equal(t, "Europe/London", result.Settings.Timezone)
}

func TestUpdateSettings_PartialUpdate(t *testing.T) {
	t.Parallel()

	newCards := 15

	mock := &userServiceMock{
		UpdateSettingsFunc: func(ctx context.Context, input user.UpdateSettingsInput) (*domain.UserSettings, error) {
			require.NotNil(t, input.NewCardsPerDay)
			require.Nil(t, input.ReviewsPerDay)
			require.Nil(t, input.MaxIntervalDays)
			require.Nil(t, input.Timezone)

			return &domain.UserSettings{
				NewCardsPerDay:  *input.NewCardsPerDay,
				ReviewsPerDay:   200,
				MaxIntervalDays: 180,
				Timezone:        "UTC",
			}, nil
		},
	}

	resolver := &mutationResolver{&Resolver{user: mock}}
	ctx := ctxutil.WithUserID(context.Background(), uuid.New())

	result, err := resolver.UpdateSettings(ctx, generated.UpdateSettingsInput{
		NewCardsPerDay: &newCards,
	})

	require.NoError(t, err)
	require.NotNil(t, result.Settings)
	require.Equal(t, 15, result.Settings.NewCardsPerDay)
}

func TestUpdateSettings_Unauthorized(t *testing.T) {
	t.Parallel()

	resolver := &mutationResolver{&Resolver{user: &userServiceMock{}}}
	ctx := context.Background()

	newCards := 10
	_, err := resolver.UpdateSettings(ctx, generated.UpdateSettingsInput{
		NewCardsPerDay: &newCards,
	})

	require.ErrorIs(t, err, domain.ErrUnauthorized)
}

func TestUserResolver_Settings_Success(t *testing.T) {
	t.Parallel()

	expectedSettings := &domain.UserSettings{
		NewCardsPerDay:  10,
		ReviewsPerDay:   50,
		MaxIntervalDays: 180,
		Timezone:        "UTC",
	}

	mock := &userServiceMock{
		GetSettingsFunc: func(ctx context.Context) (*domain.UserSettings, error) {
			return expectedSettings, nil
		},
	}

	resolver := &userResolver{&Resolver{user: mock}}
	ctx := ctxutil.WithUserID(context.Background(), uuid.New())

	result, err := resolver.Settings(ctx, &domain.User{ID: uuid.New()})

	require.NoError(t, err)
	require.NotNil(t, result)
	require.Equal(t, 10, result.NewCardsPerDay)
	require.Equal(t, "UTC", result.Timezone)
}

