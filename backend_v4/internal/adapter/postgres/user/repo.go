// Package user implements the User repository using PostgreSQL.
package user

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"

	postgres "github.com/heartmarshall/myenglish-backend/internal/adapter/postgres"
	"github.com/heartmarshall/myenglish-backend/internal/adapter/postgres/user/sqlc"
	"github.com/heartmarshall/myenglish-backend/internal/domain"
)

// Repo provides user and user-settings persistence backed by PostgreSQL.
type Repo struct {
	pool *pgxpool.Pool
}

// New creates a new user repository.
func New(pool *pgxpool.Pool) *Repo {
	return &Repo{pool: pool}
}

// ---------------------------------------------------------------------------
// User operations
// ---------------------------------------------------------------------------

// GetByID returns a user by primary key.
func (r *Repo) GetByID(ctx context.Context, id uuid.UUID) (*domain.User, error) {
	q := sqlc.New(postgres.QuerierFromCtx(ctx, r.pool))

	row, err := q.GetUserByID(ctx, id)
	if err != nil {
		return nil, mapError(err, "user", id)
	}

	u := toDomainUser(fromGetByID(row))
	return &u, nil
}

// GetByEmail returns a user by email address.
func (r *Repo) GetByEmail(ctx context.Context, email string) (*domain.User, error) {
	q := sqlc.New(postgres.QuerierFromCtx(ctx, r.pool))

	row, err := q.GetUserByEmail(ctx, email)
	if err != nil {
		return nil, mapError(err, "user", uuid.Nil)
	}

	u := toDomainUser(fromGetByEmail(row))
	return &u, nil
}

// GetByUsername returns a user by username.
func (r *Repo) GetByUsername(ctx context.Context, username string) (*domain.User, error) {
	q := sqlc.New(postgres.QuerierFromCtx(ctx, r.pool))

	row, err := q.GetUserByUsername(ctx, username)
	if err != nil {
		return nil, mapError(err, "user", uuid.Nil)
	}

	u := toDomainUser(fromGetByUsername(row))
	return &u, nil
}

// Create inserts a new user and returns the persisted domain.User.
func (r *Repo) Create(ctx context.Context, u *domain.User) (*domain.User, error) {
	q := sqlc.New(postgres.QuerierFromCtx(ctx, r.pool))

	row, err := q.CreateUser(ctx, sqlc.CreateUserParams{
		ID:        u.ID,
		Email:     u.Email,
		Username:  u.Username,
		Name:      stringToPgText(u.Name),
		AvatarUrl: ptrStringToPgText(u.AvatarURL),
		CreatedAt: u.CreatedAt,
		UpdatedAt: u.UpdatedAt,
	})
	if err != nil {
		return nil, mapError(err, "user", u.ID)
	}

	result := toDomainUser(fromCreate(row))
	return &result, nil
}

// Update modifies name and avatar_url for the given user.
func (r *Repo) Update(ctx context.Context, id uuid.UUID, name *string, avatarURL *string) (*domain.User, error) {
	q := sqlc.New(postgres.QuerierFromCtx(ctx, r.pool))

	row, err := q.UpdateUser(ctx, sqlc.UpdateUserParams{
		ID:        id,
		Name:      ptrStringToPgText(name),
		AvatarUrl: ptrStringToPgText(avatarURL),
	})
	if err != nil {
		return nil, mapError(err, "user", id)
	}

	u := toDomainUser(fromUpdate(row))
	return &u, nil
}

// ---------------------------------------------------------------------------
// UserSettings operations
// ---------------------------------------------------------------------------

// GetSettings returns the settings for the given user.
func (r *Repo) GetSettings(ctx context.Context, userID uuid.UUID) (*domain.UserSettings, error) {
	q := sqlc.New(postgres.QuerierFromCtx(ctx, r.pool))

	row, err := q.GetUserSettings(ctx, userID)
	if err != nil {
		return nil, mapError(err, "user_settings", userID)
	}

	s := toDomainSettings(row)
	return &s, nil
}

// CreateSettings inserts new user settings.
func (r *Repo) CreateSettings(ctx context.Context, s *domain.UserSettings) error {
	q := sqlc.New(postgres.QuerierFromCtx(ctx, r.pool))

	_, err := q.CreateUserSettings(ctx, sqlc.CreateUserSettingsParams{
		UserID:          s.UserID,
		NewCardsPerDay:  int32(s.NewCardsPerDay),
		ReviewsPerDay:   int32(s.ReviewsPerDay),
		MaxIntervalDays: int32(s.MaxIntervalDays),
		Timezone:        s.Timezone,
	})
	if err != nil {
		return mapError(err, "user_settings", s.UserID)
	}

	return nil
}

// UpdateSettings updates the settings for the given user.
func (r *Repo) UpdateSettings(ctx context.Context, userID uuid.UUID, s domain.UserSettings) (*domain.UserSettings, error) {
	q := sqlc.New(postgres.QuerierFromCtx(ctx, r.pool))

	row, err := q.UpdateUserSettings(ctx, sqlc.UpdateUserSettingsParams{
		UserID:          userID,
		NewCardsPerDay:  int32(s.NewCardsPerDay),
		ReviewsPerDay:   int32(s.ReviewsPerDay),
		MaxIntervalDays: int32(s.MaxIntervalDays),
		Timezone:        s.Timezone,
	})
	if err != nil {
		return nil, mapError(err, "user_settings", userID)
	}

	result := toDomainSettings(row)
	return &result, nil
}

// GetByUserID is an alias for GetSettings, satisfying the study service's settingsRepo interface.
func (r *Repo) GetByUserID(ctx context.Context, userID uuid.UUID) (*domain.UserSettings, error) {
	return r.GetSettings(ctx, userID)
}

// ---------------------------------------------------------------------------
// Error mapping
// ---------------------------------------------------------------------------

// mapError converts pgx/pgconn errors into domain errors.
func mapError(err error, entity string, id uuid.UUID) error {
	if err == nil {
		return nil
	}

	// context errors pass through as-is
	if errors.Is(err, context.DeadlineExceeded) || errors.Is(err, context.Canceled) {
		return fmt.Errorf("%s %s: %w", entity, id, err)
	}

	// pgx.ErrNoRows -> domain.ErrNotFound
	if errors.Is(err, pgx.ErrNoRows) {
		return fmt.Errorf("%s %s: %w", entity, id, domain.ErrNotFound)
	}

	// PgError codes
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) {
		switch pgErr.Code {
		case "23505": // unique_violation
			return fmt.Errorf("%s %s: %w", entity, id, domain.ErrAlreadyExists)
		case "23503": // foreign_key_violation
			return fmt.Errorf("%s %s: %w", entity, id, domain.ErrNotFound)
		case "23514": // check_violation
			return fmt.Errorf("%s %s: %w", entity, id, domain.ErrValidation)
		}
	}

	// Everything else: wrap with context
	return fmt.Errorf("%s %s: %w", entity, id, err)
}

// ---------------------------------------------------------------------------
// Mapping helpers: sqlc → domain
// ---------------------------------------------------------------------------

// userRow is the common field set returned by all user queries.
type userRow struct {
	ID        uuid.UUID
	Email     string
	Username  string
	Name      pgtype.Text
	AvatarUrl pgtype.Text
	Role      string
	CreatedAt time.Time
	UpdatedAt time.Time
}

func fromGetByID(r sqlc.GetUserByIDRow) userRow {
	return userRow{r.ID, r.Email, r.Username, r.Name, r.AvatarUrl, r.Role, r.CreatedAt, r.UpdatedAt}
}

func fromGetByEmail(r sqlc.GetUserByEmailRow) userRow {
	return userRow{r.ID, r.Email, r.Username, r.Name, r.AvatarUrl, r.Role, r.CreatedAt, r.UpdatedAt}
}

func fromGetByUsername(r sqlc.GetUserByUsernameRow) userRow {
	return userRow{r.ID, r.Email, r.Username, r.Name, r.AvatarUrl, r.Role, r.CreatedAt, r.UpdatedAt}
}

func fromCreate(r sqlc.CreateUserRow) userRow {
	return userRow{r.ID, r.Email, r.Username, r.Name, r.AvatarUrl, r.Role, r.CreatedAt, r.UpdatedAt}
}

func fromUpdate(r sqlc.UpdateUserRow) userRow {
	return userRow{r.ID, r.Email, r.Username, r.Name, r.AvatarUrl, r.Role, r.CreatedAt, r.UpdatedAt}
}

// toDomainUser converts a userRow into a domain.User.
func toDomainUser(row userRow) domain.User {
	return domain.User{
		ID:        row.ID,
		Email:     row.Email,
		Username:  row.Username,
		Name:      pgTextToString(row.Name),
		AvatarURL: pgTextToPtr(row.AvatarUrl),
		Role:      domain.UserRole(row.Role),
		CreatedAt: row.CreatedAt,
		UpdatedAt: row.UpdatedAt,
	}
}

// toDomainSettings converts a sqlc.UserSetting row into a domain.UserSettings.
func toDomainSettings(row sqlc.UserSetting) domain.UserSettings {
	return domain.UserSettings{
		UserID:          row.UserID,
		NewCardsPerDay:  int(row.NewCardsPerDay),
		ReviewsPerDay:   int(row.ReviewsPerDay),
		MaxIntervalDays: int(row.MaxIntervalDays),
		Timezone:        row.Timezone,
		UpdatedAt:       row.UpdatedAt,
	}
}

// ---------------------------------------------------------------------------
// pgtype helpers
// ---------------------------------------------------------------------------

// pgTextToString returns the string value or empty string if invalid (NULL).
func pgTextToString(t pgtype.Text) string {
	if t.Valid {
		return t.String
	}
	return ""
}

// pgTextToPtr returns a *string (nil when NULL).
func pgTextToPtr(t pgtype.Text) *string {
	if t.Valid {
		return &t.String
	}
	return nil
}

// stringToPgText converts a Go string to pgtype.Text.
// An empty string is stored as a valid empty text, not NULL.
func stringToPgText(s string) pgtype.Text {
	return pgtype.Text{String: s, Valid: true}
}

// ptrStringToPgText converts a *string to pgtype.Text (nil → NULL).
func ptrStringToPgText(s *string) pgtype.Text {
	if s == nil {
		return pgtype.Text{}
	}
	return pgtype.Text{String: *s, Valid: true}
}
