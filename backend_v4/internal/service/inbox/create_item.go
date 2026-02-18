package inbox

import (
	"context"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/heartmarshall/myenglish-backend/internal/domain"
	"github.com/heartmarshall/myenglish-backend/pkg/ctxutil"
)

// CreateItem creates a new inbox item.
func (s *Service) CreateItem(ctx context.Context, input CreateItemInput) (*domain.InboxItem, error) {
	userID, ok := ctxutil.UserIDFromCtx(ctx)
	if !ok {
		return nil, domain.ErrUnauthorized
	}

	if err := input.Validate(); err != nil {
		return nil, err
	}

	text := strings.TrimSpace(input.Text)
	ctxField := trimOrNil(input.Context)

	count, err := s.inbox.Count(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("count inbox items: %w", err)
	}
	if count >= MaxInboxItems {
		return nil, domain.NewValidationError("inbox", "inbox is full (max 500 items)")
	}

	item, err := s.inbox.Create(ctx, userID, &domain.InboxItem{
		ID:        uuid.New(),
		Text:      text,
		Context:   ctxField,
		CreatedAt: time.Now().UTC(),
	})
	if err != nil {
		return nil, fmt.Errorf("create inbox item: %w", err)
	}

	textPreview := text
	if len(textPreview) > 50 {
		textPreview = textPreview[:50]
	}

	s.log.InfoContext(ctx, "inbox item created",
		slog.String("user_id", userID.String()),
		slog.String("item_id", item.ID.String()),
		slog.String("text", textPreview),
	)

	return item, nil
}
