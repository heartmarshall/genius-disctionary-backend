package study

import (
	"context"
	"errors"
	"fmt"
	"log/slog"

	"github.com/google/uuid"
	"github.com/heartmarshall/myenglish-backend/internal/domain"
	"github.com/heartmarshall/myenglish-backend/pkg/ctxutil"
)

// GetActiveSession returns the user's active study session, or nil if none.
func (s *Service) GetActiveSession(ctx context.Context) (*domain.StudySession, error) {
	userID, ok := ctxutil.UserIDFromCtx(ctx)
	if !ok {
		return nil, domain.ErrUnauthorized
	}

	session, err := s.sessions.GetActive(ctx, userID)
	if err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			return nil, nil
		}
		return nil, fmt.Errorf("get active session: %w", err)
	}
	return session, nil
}

// StartSession starts a new study session or returns existing ACTIVE session (idempotent).
func (s *Service) StartSession(ctx context.Context) (*domain.StudySession, error) {
	userID, ok := ctxutil.UserIDFromCtx(ctx)
	if !ok {
		return nil, domain.ErrUnauthorized
	}

	// Check for existing ACTIVE session first
	existing, err := s.sessions.GetActive(ctx, userID)
	if err == nil {
		// Found existing ACTIVE session - return it (idempotent)
		s.log.InfoContext(ctx, "returning existing session",
			slog.String("user_id", userID.String()),
			slog.String("session_id", existing.ID.String()),
		)
		return existing, nil
	}
	if err != nil && !errors.Is(err, domain.ErrNotFound) {
		return nil, fmt.Errorf("check active session: %w", err)
	}

	// No active session - create new one
	session := &domain.StudySession{
		ID:        uuid.New(),
		UserID:    userID,
		Status:    domain.SessionStatusActive,
		StartedAt: s.clock.Now(),
	}

	created, err := s.sessions.Create(ctx, session)
	if err != nil {
		// Race condition: another request created session between check and create
		if errors.Is(err, domain.ErrAlreadyExists) {
			// Retry: fetch the session that was just created
			existing, getErr := s.sessions.GetActive(ctx, userID)
			if getErr != nil {
				return nil, fmt.Errorf("get active after race: %w", getErr)
			}
			s.log.InfoContext(ctx, "race condition detected, returning existing session",
				slog.String("user_id", userID.String()),
				slog.String("session_id", existing.ID.String()),
			)
			return existing, nil
		}
		return nil, fmt.Errorf("create session: %w", err)
	}

	s.log.InfoContext(ctx, "session started",
		slog.String("user_id", userID.String()),
		slog.String("session_id", created.ID.String()),
	)

	return created, nil
}

// FinishActiveSession finishes the user's current ACTIVE session.
func (s *Service) FinishActiveSession(ctx context.Context) (*domain.StudySession, error) {
	userID, ok := ctxutil.UserIDFromCtx(ctx)
	if !ok {
		return nil, domain.ErrUnauthorized
	}

	session, err := s.sessions.GetActive(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("get active session: %w", err)
	}

	return s.finishSession(ctx, userID, session)
}

// FinishSession finishes a specific session by ID.
func (s *Service) FinishSession(ctx context.Context, input FinishSessionInput) (*domain.StudySession, error) {
	userID, ok := ctxutil.UserIDFromCtx(ctx)
	if !ok {
		return nil, domain.ErrUnauthorized
	}

	if err := input.Validate(); err != nil {
		return nil, err
	}

	session, err := s.sessions.GetByID(ctx, userID, input.SessionID)
	if err != nil {
		return nil, fmt.Errorf("get session: %w", err)
	}

	return s.finishSession(ctx, userID, session)
}

// finishSession aggregates review logs and finishes the session.
func (s *Service) finishSession(ctx context.Context, userID uuid.UUID, session *domain.StudySession) (*domain.StudySession, error) {
	if session.Status != domain.SessionStatusActive {
		return nil, domain.NewValidationError("session", "session already finished")
	}

	now := s.clock.Now()
	var finishedSession *domain.StudySession

	err := s.tx.RunInTx(ctx, func(txCtx context.Context) error {
		logs, logErr := s.reviews.GetByPeriod(txCtx, userID, session.StartedAt, now)
		if logErr != nil {
			return fmt.Errorf("get review logs: %w", logErr)
		}

		result := aggregateSessionResult(logs, session.StartedAt, now)

		var finErr error
		finishedSession, finErr = s.sessions.Finish(txCtx, userID, session.ID, result)
		return finErr
	})

	if err != nil {
		return nil, fmt.Errorf("finish session: %w", err)
	}

	s.log.InfoContext(ctx, "session finished",
		slog.String("user_id", userID.String()),
		slog.String("session_id", session.ID.String()),
	)

	return finishedSession, nil
}

// AbandonSession abandons the current ACTIVE session (idempotent noop if no active session).
func (s *Service) AbandonSession(ctx context.Context) error {
	userID, ok := ctxutil.UserIDFromCtx(ctx)
	if !ok {
		return domain.ErrUnauthorized
	}

	// Try to get active session
	session, err := s.sessions.GetActive(ctx, userID)
	if err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			// No active session - idempotent noop
			return nil
		}
		return fmt.Errorf("get active session: %w", err)
	}

	// Abandon the active session
	if err := s.sessions.Abandon(ctx, userID, session.ID); err != nil {
		return fmt.Errorf("abandon session: %w", err)
	}

	s.log.InfoContext(ctx, "session abandoned",
		slog.String("user_id", userID.String()),
		slog.String("session_id", session.ID.String()),
	)

	return nil
}
