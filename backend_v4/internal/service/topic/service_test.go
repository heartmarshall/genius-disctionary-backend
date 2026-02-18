package topic

import (
	"context"
	"errors"
	"log/slog"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/heartmarshall/myenglish-backend/internal/domain"
	"github.com/heartmarshall/myenglish-backend/pkg/ctxutil"
)

// newCRUDTestService creates a Service with the given mocks and a default logger.
func newCRUDTestService(
	t *testing.T,
	topicMock *topicRepoMock,
	auditMock *auditLoggerMock,
	txMock *txManagerMock,
) *Service {
	t.Helper()
	return NewService(
		slog.Default(),
		topicMock,
		&entryRepoMock{},
		auditMock,
		txMock,
	)
}

// defaultTxMock returns a txManagerMock that simply calls the function with the same context.
func defaultTxMock() *txManagerMock {
	return &txManagerMock{
		RunInTxFunc: func(ctx context.Context, fn func(context.Context) error) error {
			return fn(ctx)
		},
	}
}

// defaultAuditMock returns an auditLoggerMock that always succeeds.
func defaultAuditMock() *auditLoggerMock {
	return &auditLoggerMock{
		LogFunc: func(ctx context.Context, record domain.AuditRecord) error {
			return nil
		},
	}
}

// ---------------------------------------------------------------------------
// CreateTopic Tests (11 tests)
// ---------------------------------------------------------------------------

func TestCreateTopic_Success(t *testing.T) {
	t.Parallel()

	userID := uuid.New()
	topicID := uuid.New()
	desc := "A test topic"

	topicMock := &topicRepoMock{
		CountFunc: func(ctx context.Context, uid uuid.UUID) (int, error) {
			return 5, nil
		},
		CreateFunc: func(ctx context.Context, uid uuid.UUID, topic *domain.Topic) (*domain.Topic, error) {
			return &domain.Topic{
				ID:          topicID,
				UserID:      uid,
				Name:        topic.Name,
				Description: topic.Description,
				CreatedAt:   time.Now(),
				UpdatedAt:   time.Now(),
			}, nil
		},
	}

	auditMock := defaultAuditMock()
	txMock := defaultTxMock()
	svc := newCRUDTestService(t, topicMock, auditMock, txMock)
	ctx := ctxutil.WithUserID(context.Background(), userID)

	result, err := svc.CreateTopic(ctx, CreateTopicInput{
		Name:        "Travel",
		Description: &desc,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result.ID != topicID {
		t.Errorf("topic ID: got %v, want %v", result.ID, topicID)
	}
	if result.Name != "Travel" {
		t.Errorf("name: got %q, want %q", result.Name, "Travel")
	}
	if result.Description == nil || *result.Description != desc {
		t.Errorf("description: got %v, want %q", result.Description, desc)
	}
	if len(topicMock.CountCalls()) != 1 {
		t.Errorf("Count calls: got %d, want 1", len(topicMock.CountCalls()))
	}
	if len(topicMock.CreateCalls()) != 1 {
		t.Errorf("Create calls: got %d, want 1", len(topicMock.CreateCalls()))
	}
	if len(auditMock.LogCalls()) != 1 {
		t.Errorf("Audit Log calls: got %d, want 1", len(auditMock.LogCalls()))
	}
}

func TestCreateTopic_WithoutDescription(t *testing.T) {
	t.Parallel()

	userID := uuid.New()
	topicID := uuid.New()

	topicMock := &topicRepoMock{
		CountFunc: func(ctx context.Context, uid uuid.UUID) (int, error) {
			return 0, nil
		},
		CreateFunc: func(ctx context.Context, uid uuid.UUID, topic *domain.Topic) (*domain.Topic, error) {
			if topic.Description != nil {
				t.Errorf("description should be nil, got %v", topic.Description)
			}
			return &domain.Topic{
				ID:          topicID,
				UserID:      uid,
				Name:        topic.Name,
				Description: nil,
				CreatedAt:   time.Now(),
				UpdatedAt:   time.Now(),
			}, nil
		},
	}

	svc := newCRUDTestService(t, topicMock, defaultAuditMock(), defaultTxMock())
	ctx := ctxutil.WithUserID(context.Background(), userID)

	result, err := svc.CreateTopic(ctx, CreateTopicInput{
		Name:        "Food",
		Description: nil,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Description != nil {
		t.Errorf("expected nil description, got %v", result.Description)
	}
}

func TestCreateTopic_EmptyName(t *testing.T) {
	t.Parallel()

	userID := uuid.New()
	svc := newCRUDTestService(t, &topicRepoMock{}, defaultAuditMock(), defaultTxMock())
	ctx := ctxutil.WithUserID(context.Background(), userID)

	_, err := svc.CreateTopic(ctx, CreateTopicInput{
		Name: "",
	})
	if err == nil {
		t.Fatal("expected error, got nil")
	}

	var ve *domain.ValidationError
	if !errors.As(err, &ve) {
		t.Fatalf("expected ValidationError, got %T: %v", err, err)
	}
	if ve.Errors[0].Field != "name" || ve.Errors[0].Message != "required" {
		t.Errorf("expected name/required, got %s/%s", ve.Errors[0].Field, ve.Errors[0].Message)
	}
}

func TestCreateTopic_WhitespaceOnlyName(t *testing.T) {
	t.Parallel()

	userID := uuid.New()
	svc := newCRUDTestService(t, &topicRepoMock{}, defaultAuditMock(), defaultTxMock())
	ctx := ctxutil.WithUserID(context.Background(), userID)

	_, err := svc.CreateTopic(ctx, CreateTopicInput{
		Name: "   \t\n  ",
	})
	if err == nil {
		t.Fatal("expected error, got nil")
	}

	var ve *domain.ValidationError
	if !errors.As(err, &ve) {
		t.Fatalf("expected ValidationError, got %T: %v", err, err)
	}
	if ve.Errors[0].Field != "name" || ve.Errors[0].Message != "required" {
		t.Errorf("expected name/required, got %s/%s", ve.Errors[0].Field, ve.Errors[0].Message)
	}
}

func TestCreateTopic_NameTooLong(t *testing.T) {
	t.Parallel()

	userID := uuid.New()
	svc := newCRUDTestService(t, &topicRepoMock{}, defaultAuditMock(), defaultTxMock())
	ctx := ctxutil.WithUserID(context.Background(), userID)

	longName := strings.Repeat("a", 101)
	_, err := svc.CreateTopic(ctx, CreateTopicInput{
		Name: longName,
	})
	if err == nil {
		t.Fatal("expected error, got nil")
	}

	var ve *domain.ValidationError
	if !errors.As(err, &ve) {
		t.Fatalf("expected ValidationError, got %T: %v", err, err)
	}
	found := false
	for _, fe := range ve.Errors {
		if fe.Field == "name" && fe.Message == "max 100 characters" {
			found = true
		}
	}
	if !found {
		t.Errorf("expected name/max 100 characters error, got %v", ve.Errors)
	}
}

func TestCreateTopic_DescriptionTooLong(t *testing.T) {
	t.Parallel()

	userID := uuid.New()
	svc := newCRUDTestService(t, &topicRepoMock{}, defaultAuditMock(), defaultTxMock())
	ctx := ctxutil.WithUserID(context.Background(), userID)

	longDesc := strings.Repeat("b", 501)
	_, err := svc.CreateTopic(ctx, CreateTopicInput{
		Name:        "Valid",
		Description: &longDesc,
	})
	if err == nil {
		t.Fatal("expected error, got nil")
	}

	var ve *domain.ValidationError
	if !errors.As(err, &ve) {
		t.Fatalf("expected ValidationError, got %T: %v", err, err)
	}
	found := false
	for _, fe := range ve.Errors {
		if fe.Field == "description" && fe.Message == "max 500 characters" {
			found = true
		}
	}
	if !found {
		t.Errorf("expected description/max 500 characters error, got %v", ve.Errors)
	}
}

func TestCreateTopic_DuplicateName(t *testing.T) {
	t.Parallel()

	userID := uuid.New()

	topicMock := &topicRepoMock{
		CountFunc: func(ctx context.Context, uid uuid.UUID) (int, error) {
			return 5, nil
		},
		CreateFunc: func(ctx context.Context, uid uuid.UUID, topic *domain.Topic) (*domain.Topic, error) {
			return nil, domain.ErrAlreadyExists
		},
	}

	svc := newCRUDTestService(t, topicMock, defaultAuditMock(), defaultTxMock())
	ctx := ctxutil.WithUserID(context.Background(), userID)

	_, err := svc.CreateTopic(ctx, CreateTopicInput{
		Name: "Duplicate",
	})
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !errors.Is(err, domain.ErrAlreadyExists) {
		t.Errorf("error: got %v, want ErrAlreadyExists", err)
	}
}

func TestCreateTopic_LimitReached(t *testing.T) {
	t.Parallel()

	userID := uuid.New()

	topicMock := &topicRepoMock{
		CountFunc: func(ctx context.Context, uid uuid.UUID) (int, error) {
			return 100, nil
		},
	}

	svc := newCRUDTestService(t, topicMock, defaultAuditMock(), defaultTxMock())
	ctx := ctxutil.WithUserID(context.Background(), userID)

	_, err := svc.CreateTopic(ctx, CreateTopicInput{
		Name: "OneMore",
	})
	if err == nil {
		t.Fatal("expected error, got nil")
	}

	var ve *domain.ValidationError
	if !errors.As(err, &ve) {
		t.Fatalf("expected ValidationError, got %T: %v", err, err)
	}
	if ve.Errors[0].Field != "topics" {
		t.Errorf("field: got %q, want %q", ve.Errors[0].Field, "topics")
	}
	if ve.Errors[0].Message != "limit reached (max 100)" {
		t.Errorf("message: got %q, want %q", ve.Errors[0].Message, "limit reached (max 100)")
	}
	// Create should not be called
	if len(topicMock.CreateCalls()) != 0 {
		t.Errorf("Create calls: got %d, want 0", len(topicMock.CreateCalls()))
	}
}

func TestCreateTopic_Audit(t *testing.T) {
	t.Parallel()

	userID := uuid.New()
	topicID := uuid.New()

	topicMock := &topicRepoMock{
		CountFunc: func(ctx context.Context, uid uuid.UUID) (int, error) {
			return 0, nil
		},
		CreateFunc: func(ctx context.Context, uid uuid.UUID, topic *domain.Topic) (*domain.Topic, error) {
			return &domain.Topic{
				ID:        topicID,
				UserID:    uid,
				Name:      topic.Name,
				CreatedAt: time.Now(),
				UpdatedAt: time.Now(),
			}, nil
		},
	}

	var capturedRecord domain.AuditRecord
	auditMock := &auditLoggerMock{
		LogFunc: func(ctx context.Context, record domain.AuditRecord) error {
			capturedRecord = record
			return nil
		},
	}

	svc := newCRUDTestService(t, topicMock, auditMock, defaultTxMock())
	ctx := ctxutil.WithUserID(context.Background(), userID)

	_, err := svc.CreateTopic(ctx, CreateTopicInput{Name: "Audit Test"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if capturedRecord.UserID != userID {
		t.Errorf("audit userID: got %v, want %v", capturedRecord.UserID, userID)
	}
	if capturedRecord.EntityType != domain.EntityTypeTopic {
		t.Errorf("audit entity type: got %v, want %v", capturedRecord.EntityType, domain.EntityTypeTopic)
	}
	if capturedRecord.EntityID == nil || *capturedRecord.EntityID != topicID {
		t.Errorf("audit entity ID: got %v, want %v", capturedRecord.EntityID, topicID)
	}
	if capturedRecord.Action != domain.AuditActionCreate {
		t.Errorf("audit action: got %v, want %v", capturedRecord.Action, domain.AuditActionCreate)
	}
	nameChange, ok := capturedRecord.Changes["name"].(map[string]any)
	if !ok {
		t.Fatalf("audit changes[name]: expected map, got %T", capturedRecord.Changes["name"])
	}
	if nameChange["new"] != "Audit Test" {
		t.Errorf("audit changes[name][new]: got %v, want %q", nameChange["new"], "Audit Test")
	}
}

func TestCreateTopic_Unauthorized(t *testing.T) {
	t.Parallel()

	svc := newCRUDTestService(t, &topicRepoMock{}, defaultAuditMock(), defaultTxMock())
	ctx := context.Background() // no user ID

	_, err := svc.CreateTopic(ctx, CreateTopicInput{Name: "hello"})
	if !errors.Is(err, domain.ErrUnauthorized) {
		t.Errorf("error: got %v, want ErrUnauthorized", err)
	}
}

func TestCreateTopic_EmptyDescription(t *testing.T) {
	t.Parallel()

	userID := uuid.New()
	topicID := uuid.New()
	emptyDesc := ""

	topicMock := &topicRepoMock{
		CountFunc: func(ctx context.Context, uid uuid.UUID) (int, error) {
			return 0, nil
		},
		CreateFunc: func(ctx context.Context, uid uuid.UUID, topic *domain.Topic) (*domain.Topic, error) {
			if topic.Description != nil {
				t.Errorf("expected description to be nil after trimOrNil, got %q", *topic.Description)
			}
			return &domain.Topic{
				ID:          topicID,
				UserID:      uid,
				Name:        topic.Name,
				Description: topic.Description,
				CreatedAt:   time.Now(),
				UpdatedAt:   time.Now(),
			}, nil
		},
	}

	svc := newCRUDTestService(t, topicMock, defaultAuditMock(), defaultTxMock())
	ctx := ctxutil.WithUserID(context.Background(), userID)

	result, err := svc.CreateTopic(ctx, CreateTopicInput{
		Name:        "NoDesc",
		Description: &emptyDesc,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Description != nil {
		t.Errorf("expected nil description, got %v", result.Description)
	}
}

// ---------------------------------------------------------------------------
// UpdateTopic Tests (10 tests)
// ---------------------------------------------------------------------------

func TestUpdateTopic_NameOnly(t *testing.T) {
	t.Parallel()

	userID := uuid.New()
	topicID := uuid.New()
	newName := "Updated Name"
	oldDesc := "old description"

	topicMock := &topicRepoMock{
		GetByIDFunc: func(ctx context.Context, uid, tid uuid.UUID) (*domain.Topic, error) {
			return &domain.Topic{
				ID:          topicID,
				UserID:      uid,
				Name:        "Old Name",
				Description: &oldDesc,
			}, nil
		},
		UpdateFunc: func(ctx context.Context, uid, tid uuid.UUID, params domain.TopicUpdateParams) (*domain.Topic, error) {
			if params.Name == nil || *params.Name != newName {
				t.Errorf("expected name %q, got %v", newName, params.Name)
			}
			if params.Description != nil {
				t.Errorf("expected description to be nil (unchanged), got %v", params.Description)
			}
			return &domain.Topic{
				ID:          topicID,
				UserID:      uid,
				Name:        *params.Name,
				Description: &oldDesc,
			}, nil
		},
	}

	var capturedRecord domain.AuditRecord
	auditMock := &auditLoggerMock{
		LogFunc: func(ctx context.Context, record domain.AuditRecord) error {
			capturedRecord = record
			return nil
		},
	}

	svc := newCRUDTestService(t, topicMock, auditMock, defaultTxMock())
	ctx := ctxutil.WithUserID(context.Background(), userID)

	result, err := svc.UpdateTopic(ctx, UpdateTopicInput{
		TopicID: topicID,
		Name:    &newName,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Name != newName {
		t.Errorf("name: got %q, want %q", result.Name, newName)
	}

	// Audit should have name change
	nameChange, ok := capturedRecord.Changes["name"].(map[string]any)
	if !ok {
		t.Fatalf("audit changes[name]: expected map, got %T", capturedRecord.Changes["name"])
	}
	if nameChange["old"] != "Old Name" {
		t.Errorf("audit old name: got %v, want %q", nameChange["old"], "Old Name")
	}
	if nameChange["new"] != newName {
		t.Errorf("audit new name: got %v, want %q", nameChange["new"], newName)
	}
}

func TestUpdateTopic_DescriptionOnly(t *testing.T) {
	t.Parallel()

	userID := uuid.New()
	topicID := uuid.New()
	newDesc := "New description"

	topicMock := &topicRepoMock{
		GetByIDFunc: func(ctx context.Context, uid, tid uuid.UUID) (*domain.Topic, error) {
			return &domain.Topic{
				ID:          topicID,
				UserID:      uid,
				Name:        "Name",
				Description: nil,
			}, nil
		},
		UpdateFunc: func(ctx context.Context, uid, tid uuid.UUID, params domain.TopicUpdateParams) (*domain.Topic, error) {
			if params.Name != nil {
				t.Errorf("expected name to be nil (unchanged), got %v", params.Name)
			}
			return &domain.Topic{
				ID:          topicID,
				UserID:      uid,
				Name:        "Name",
				Description: &newDesc,
			}, nil
		},
	}

	svc := newCRUDTestService(t, topicMock, defaultAuditMock(), defaultTxMock())
	ctx := ctxutil.WithUserID(context.Background(), userID)

	result, err := svc.UpdateTopic(ctx, UpdateTopicInput{
		TopicID:     topicID,
		Description: &newDesc,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Description == nil || *result.Description != newDesc {
		t.Errorf("description: got %v, want %q", result.Description, newDesc)
	}
}

func TestUpdateTopic_ClearDescription(t *testing.T) {
	t.Parallel()

	userID := uuid.New()
	topicID := uuid.New()
	emptyStr := ""
	oldDesc := "had a description"

	topicMock := &topicRepoMock{
		GetByIDFunc: func(ctx context.Context, uid, tid uuid.UUID) (*domain.Topic, error) {
			return &domain.Topic{
				ID:          topicID,
				UserID:      uid,
				Name:        "Name",
				Description: &oldDesc,
			}, nil
		},
		UpdateFunc: func(ctx context.Context, uid, tid uuid.UUID, params domain.TopicUpdateParams) (*domain.Topic, error) {
			if params.Description == nil {
				t.Error("expected description to be set (ptr to empty string), got nil")
			} else if *params.Description != "" {
				t.Errorf("expected empty string for clearing, got %q", *params.Description)
			}
			return &domain.Topic{
				ID:          topicID,
				UserID:      uid,
				Name:        "Name",
				Description: nil, // cleared
			}, nil
		},
	}

	svc := newCRUDTestService(t, topicMock, defaultAuditMock(), defaultTxMock())
	ctx := ctxutil.WithUserID(context.Background(), userID)

	result, err := svc.UpdateTopic(ctx, UpdateTopicInput{
		TopicID:     topicID,
		Description: &emptyStr,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Description != nil {
		t.Errorf("expected nil description after clearing, got %v", result.Description)
	}
}

func TestUpdateTopic_BothFields(t *testing.T) {
	t.Parallel()

	userID := uuid.New()
	topicID := uuid.New()
	newName := "New Name"
	newDesc := "New Desc"

	topicMock := &topicRepoMock{
		GetByIDFunc: func(ctx context.Context, uid, tid uuid.UUID) (*domain.Topic, error) {
			return &domain.Topic{
				ID:     topicID,
				UserID: uid,
				Name:   "Old",
			}, nil
		},
		UpdateFunc: func(ctx context.Context, uid, tid uuid.UUID, params domain.TopicUpdateParams) (*domain.Topic, error) {
			if params.Name == nil {
				t.Error("expected name to be set")
			}
			if params.Description == nil {
				t.Error("expected description to be set")
			}
			return &domain.Topic{
				ID:          topicID,
				UserID:      uid,
				Name:        *params.Name,
				Description: params.Description,
			}, nil
		},
	}

	svc := newCRUDTestService(t, topicMock, defaultAuditMock(), defaultTxMock())
	ctx := ctxutil.WithUserID(context.Background(), userID)

	result, err := svc.UpdateTopic(ctx, UpdateTopicInput{
		TopicID:     topicID,
		Name:        &newName,
		Description: &newDesc,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Name != newName {
		t.Errorf("name: got %q, want %q", result.Name, newName)
	}
	if result.Description == nil || *result.Description != newDesc {
		t.Errorf("description: got %v, want %q", result.Description, newDesc)
	}
}

func TestUpdateTopic_NoFields(t *testing.T) {
	t.Parallel()

	userID := uuid.New()
	topicID := uuid.New()
	svc := newCRUDTestService(t, &topicRepoMock{}, defaultAuditMock(), defaultTxMock())
	ctx := ctxutil.WithUserID(context.Background(), userID)

	_, err := svc.UpdateTopic(ctx, UpdateTopicInput{
		TopicID: topicID,
		Name:    nil,
	})
	if err == nil {
		t.Fatal("expected error, got nil")
	}

	var ve *domain.ValidationError
	if !errors.As(err, &ve) {
		t.Fatalf("expected ValidationError, got %T: %v", err, err)
	}
	found := false
	for _, fe := range ve.Errors {
		if fe.Field == "input" && fe.Message == "at least one field must be provided" {
			found = true
		}
	}
	if !found {
		t.Errorf("expected input/at least one field must be provided, got %v", ve.Errors)
	}
}

func TestUpdateTopic_NotFound(t *testing.T) {
	t.Parallel()

	userID := uuid.New()
	topicID := uuid.New()
	newName := "New"

	topicMock := &topicRepoMock{
		GetByIDFunc: func(ctx context.Context, uid, tid uuid.UUID) (*domain.Topic, error) {
			return nil, domain.ErrNotFound
		},
	}

	svc := newCRUDTestService(t, topicMock, defaultAuditMock(), defaultTxMock())
	ctx := ctxutil.WithUserID(context.Background(), userID)

	_, err := svc.UpdateTopic(ctx, UpdateTopicInput{
		TopicID: topicID,
		Name:    &newName,
	})
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !errors.Is(err, domain.ErrNotFound) {
		t.Errorf("error: got %v, want ErrNotFound", err)
	}
}

func TestUpdateTopic_DuplicateName(t *testing.T) {
	t.Parallel()

	userID := uuid.New()
	topicID := uuid.New()
	newName := "Existing Name"

	topicMock := &topicRepoMock{
		GetByIDFunc: func(ctx context.Context, uid, tid uuid.UUID) (*domain.Topic, error) {
			return &domain.Topic{
				ID:     topicID,
				UserID: uid,
				Name:   "Old Name",
			}, nil
		},
		UpdateFunc: func(ctx context.Context, uid, tid uuid.UUID, params domain.TopicUpdateParams) (*domain.Topic, error) {
			return nil, domain.ErrAlreadyExists
		},
	}

	svc := newCRUDTestService(t, topicMock, defaultAuditMock(), defaultTxMock())
	ctx := ctxutil.WithUserID(context.Background(), userID)

	_, err := svc.UpdateTopic(ctx, UpdateTopicInput{
		TopicID: topicID,
		Name:    &newName,
	})
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !errors.Is(err, domain.ErrAlreadyExists) {
		t.Errorf("error: got %v, want ErrAlreadyExists", err)
	}
}

func TestUpdateTopic_EmptyName(t *testing.T) {
	t.Parallel()

	userID := uuid.New()
	topicID := uuid.New()
	emptyName := ""

	svc := newCRUDTestService(t, &topicRepoMock{}, defaultAuditMock(), defaultTxMock())
	ctx := ctxutil.WithUserID(context.Background(), userID)

	_, err := svc.UpdateTopic(ctx, UpdateTopicInput{
		TopicID: topicID,
		Name:    &emptyName,
	})
	if err == nil {
		t.Fatal("expected error, got nil")
	}

	var ve *domain.ValidationError
	if !errors.As(err, &ve) {
		t.Fatalf("expected ValidationError, got %T: %v", err, err)
	}
	if ve.Errors[0].Field != "name" || ve.Errors[0].Message != "required" {
		t.Errorf("expected name/required, got %s/%s", ve.Errors[0].Field, ve.Errors[0].Message)
	}
}

func TestUpdateTopic_Audit(t *testing.T) {
	t.Parallel()

	userID := uuid.New()
	topicID := uuid.New()
	newName := "New Name"
	oldDesc := "old desc"
	newDesc := "new desc"

	topicMock := &topicRepoMock{
		GetByIDFunc: func(ctx context.Context, uid, tid uuid.UUID) (*domain.Topic, error) {
			return &domain.Topic{
				ID:          topicID,
				UserID:      uid,
				Name:        "Old Name",
				Description: &oldDesc,
			}, nil
		},
		UpdateFunc: func(ctx context.Context, uid, tid uuid.UUID, params domain.TopicUpdateParams) (*domain.Topic, error) {
			return &domain.Topic{
				ID:          topicID,
				UserID:      uid,
				Name:        newName,
				Description: &newDesc,
			}, nil
		},
	}

	var capturedRecord domain.AuditRecord
	auditMock := &auditLoggerMock{
		LogFunc: func(ctx context.Context, record domain.AuditRecord) error {
			capturedRecord = record
			return nil
		},
	}

	svc := newCRUDTestService(t, topicMock, auditMock, defaultTxMock())
	ctx := ctxutil.WithUserID(context.Background(), userID)

	_, err := svc.UpdateTopic(ctx, UpdateTopicInput{
		TopicID:     topicID,
		Name:        &newName,
		Description: &newDesc,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if capturedRecord.Action != domain.AuditActionUpdate {
		t.Errorf("audit action: got %v, want %v", capturedRecord.Action, domain.AuditActionUpdate)
	}
	if capturedRecord.EntityType != domain.EntityTypeTopic {
		t.Errorf("audit entity type: got %v, want %v", capturedRecord.EntityType, domain.EntityTypeTopic)
	}

	// Audit should contain only changed fields
	nameChange, ok := capturedRecord.Changes["name"].(map[string]any)
	if !ok {
		t.Fatalf("audit changes[name]: expected map, got %T", capturedRecord.Changes["name"])
	}
	if nameChange["old"] != "Old Name" {
		t.Errorf("audit old name: got %v, want %q", nameChange["old"], "Old Name")
	}
	if nameChange["new"] != newName {
		t.Errorf("audit new name: got %v, want %q", nameChange["new"], newName)
	}

	descChange, ok := capturedRecord.Changes["description"].(map[string]any)
	if !ok {
		t.Fatalf("audit changes[description]: expected map, got %T", capturedRecord.Changes["description"])
	}
	if *(descChange["old"].(*string)) != oldDesc {
		t.Errorf("audit old desc: got %v, want %q", descChange["old"], oldDesc)
	}
	if *(descChange["new"].(*string)) != newDesc {
		t.Errorf("audit new desc: got %v, want %q", descChange["new"], newDesc)
	}
}

func TestUpdateTopic_Unauthorized(t *testing.T) {
	t.Parallel()

	svc := newCRUDTestService(t, &topicRepoMock{}, defaultAuditMock(), defaultTxMock())
	ctx := context.Background()

	name := "test"
	_, err := svc.UpdateTopic(ctx, UpdateTopicInput{
		TopicID: uuid.New(),
		Name:    &name,
	})
	if !errors.Is(err, domain.ErrUnauthorized) {
		t.Errorf("error: got %v, want ErrUnauthorized", err)
	}
}

// ---------------------------------------------------------------------------
// DeleteTopic Tests (4 tests)
// ---------------------------------------------------------------------------

func TestDeleteTopic_Success(t *testing.T) {
	t.Parallel()

	userID := uuid.New()
	topicID := uuid.New()

	topicMock := &topicRepoMock{
		GetByIDFunc: func(ctx context.Context, uid, tid uuid.UUID) (*domain.Topic, error) {
			return &domain.Topic{
				ID:     topicID,
				UserID: uid,
				Name:   "ToDelete",
			}, nil
		},
		DeleteFunc: func(ctx context.Context, uid, tid uuid.UUID) error {
			if uid != userID {
				t.Errorf("userID: got %v, want %v", uid, userID)
			}
			if tid != topicID {
				t.Errorf("topicID: got %v, want %v", tid, topicID)
			}
			return nil
		},
	}

	svc := newCRUDTestService(t, topicMock, defaultAuditMock(), defaultTxMock())
	ctx := ctxutil.WithUserID(context.Background(), userID)

	err := svc.DeleteTopic(ctx, DeleteTopicInput{TopicID: topicID})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(topicMock.DeleteCalls()) != 1 {
		t.Errorf("Delete calls: got %d, want 1", len(topicMock.DeleteCalls()))
	}
}

func TestDeleteTopic_NotFound(t *testing.T) {
	t.Parallel()

	userID := uuid.New()
	topicID := uuid.New()

	topicMock := &topicRepoMock{
		GetByIDFunc: func(ctx context.Context, uid, tid uuid.UUID) (*domain.Topic, error) {
			return nil, domain.ErrNotFound
		},
	}

	svc := newCRUDTestService(t, topicMock, defaultAuditMock(), defaultTxMock())
	ctx := ctxutil.WithUserID(context.Background(), userID)

	err := svc.DeleteTopic(ctx, DeleteTopicInput{TopicID: topicID})
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !errors.Is(err, domain.ErrNotFound) {
		t.Errorf("error: got %v, want ErrNotFound", err)
	}
}

func TestDeleteTopic_Audit(t *testing.T) {
	t.Parallel()

	userID := uuid.New()
	topicID := uuid.New()

	topicMock := &topicRepoMock{
		GetByIDFunc: func(ctx context.Context, uid, tid uuid.UUID) (*domain.Topic, error) {
			return &domain.Topic{
				ID:     topicID,
				UserID: uid,
				Name:   "Deleted Topic",
			}, nil
		},
		DeleteFunc: func(ctx context.Context, uid, tid uuid.UUID) error {
			return nil
		},
	}

	var capturedRecord domain.AuditRecord
	auditMock := &auditLoggerMock{
		LogFunc: func(ctx context.Context, record domain.AuditRecord) error {
			capturedRecord = record
			return nil
		},
	}

	svc := newCRUDTestService(t, topicMock, auditMock, defaultTxMock())
	ctx := ctxutil.WithUserID(context.Background(), userID)

	err := svc.DeleteTopic(ctx, DeleteTopicInput{TopicID: topicID})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if capturedRecord.Action != domain.AuditActionDelete {
		t.Errorf("audit action: got %v, want %v", capturedRecord.Action, domain.AuditActionDelete)
	}
	if capturedRecord.EntityType != domain.EntityTypeTopic {
		t.Errorf("audit entity type: got %v, want %v", capturedRecord.EntityType, domain.EntityTypeTopic)
	}
	if capturedRecord.EntityID == nil || *capturedRecord.EntityID != topicID {
		t.Errorf("audit entity ID: got %v, want %v", capturedRecord.EntityID, topicID)
	}
	nameChange, ok := capturedRecord.Changes["name"].(map[string]any)
	if !ok {
		t.Fatalf("audit changes[name]: expected map, got %T", capturedRecord.Changes["name"])
	}
	if nameChange["old"] != "Deleted Topic" {
		t.Errorf("audit name old: got %v, want %q", nameChange["old"], "Deleted Topic")
	}
}

func TestDeleteTopic_Unauthorized(t *testing.T) {
	t.Parallel()

	svc := newCRUDTestService(t, &topicRepoMock{}, defaultAuditMock(), defaultTxMock())
	ctx := context.Background()

	err := svc.DeleteTopic(ctx, DeleteTopicInput{TopicID: uuid.New()})
	if !errors.Is(err, domain.ErrUnauthorized) {
		t.Errorf("error: got %v, want ErrUnauthorized", err)
	}
}

// ---------------------------------------------------------------------------
// ListTopics Tests (4 tests)
// ---------------------------------------------------------------------------

func TestListTopics_Success(t *testing.T) {
	t.Parallel()

	userID := uuid.New()
	topics := []*domain.Topic{
		{ID: uuid.New(), UserID: userID, Name: "Topic 1", CreatedAt: time.Now(), UpdatedAt: time.Now()},
		{ID: uuid.New(), UserID: userID, Name: "Topic 2", CreatedAt: time.Now(), UpdatedAt: time.Now()},
	}

	topicMock := &topicRepoMock{
		ListFunc: func(ctx context.Context, uid uuid.UUID) ([]*domain.Topic, error) {
			if uid != userID {
				t.Errorf("userID: got %v, want %v", uid, userID)
			}
			return topics, nil
		},
	}

	svc := newCRUDTestService(t, topicMock, defaultAuditMock(), defaultTxMock())
	ctx := ctxutil.WithUserID(context.Background(), userID)

	result, err := svc.ListTopics(ctx)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result) != 2 {
		t.Errorf("result length: got %d, want 2", len(result))
	}
	if result[0].Name != "Topic 1" {
		t.Errorf("first topic name: got %q, want %q", result[0].Name, "Topic 1")
	}
	if result[1].Name != "Topic 2" {
		t.Errorf("second topic name: got %q, want %q", result[1].Name, "Topic 2")
	}
}

func TestListTopics_Empty(t *testing.T) {
	t.Parallel()

	userID := uuid.New()

	topicMock := &topicRepoMock{
		ListFunc: func(ctx context.Context, uid uuid.UUID) ([]*domain.Topic, error) {
			return []*domain.Topic{}, nil
		},
	}

	svc := newCRUDTestService(t, topicMock, defaultAuditMock(), defaultTxMock())
	ctx := ctxutil.WithUserID(context.Background(), userID)

	result, err := svc.ListTopics(ctx)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result == nil {
		t.Fatal("expected empty slice, got nil")
	}
	if len(result) != 0 {
		t.Errorf("result length: got %d, want 0", len(result))
	}
}

func TestListTopics_WithEntryCounts(t *testing.T) {
	t.Parallel()

	userID := uuid.New()
	topics := []*domain.Topic{
		{ID: uuid.New(), UserID: userID, Name: "Topic 1", EntryCount: 5},
		{ID: uuid.New(), UserID: userID, Name: "Topic 2", EntryCount: 12},
		{ID: uuid.New(), UserID: userID, Name: "Topic 3", EntryCount: 0},
	}

	topicMock := &topicRepoMock{
		ListFunc: func(ctx context.Context, uid uuid.UUID) ([]*domain.Topic, error) {
			return topics, nil
		},
	}

	svc := newCRUDTestService(t, topicMock, defaultAuditMock(), defaultTxMock())
	ctx := ctxutil.WithUserID(context.Background(), userID)

	result, err := svc.ListTopics(ctx)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result) != 3 {
		t.Fatalf("result length: got %d, want 3", len(result))
	}
	if result[0].EntryCount != 5 {
		t.Errorf("topic 1 entry count: got %d, want 5", result[0].EntryCount)
	}
	if result[1].EntryCount != 12 {
		t.Errorf("topic 2 entry count: got %d, want 12", result[1].EntryCount)
	}
	if result[2].EntryCount != 0 {
		t.Errorf("topic 3 entry count: got %d, want 0", result[2].EntryCount)
	}
}

func TestListTopics_Unauthorized(t *testing.T) {
	t.Parallel()

	svc := newCRUDTestService(t, &topicRepoMock{}, defaultAuditMock(), defaultTxMock())
	ctx := context.Background()

	_, err := svc.ListTopics(ctx)
	if !errors.Is(err, domain.ErrUnauthorized) {
		t.Errorf("error: got %v, want ErrUnauthorized", err)
	}
}

// ---------------------------------------------------------------------------
// UpdateTopic: skip audit when nothing changed (issue #9)
// ---------------------------------------------------------------------------

func TestUpdateTopic_NoChanges_SkipsAudit(t *testing.T) {
	t.Parallel()

	userID := uuid.New()
	topicID := uuid.New()
	sameName := "Same Name"
	sameDesc := "Same Desc"

	topicMock := &topicRepoMock{
		GetByIDFunc: func(ctx context.Context, uid, tid uuid.UUID) (*domain.Topic, error) {
			return &domain.Topic{
				ID:          topicID,
				UserID:      uid,
				Name:        sameName,
				Description: &sameDesc,
			}, nil
		},
		UpdateFunc: func(ctx context.Context, uid, tid uuid.UUID, params domain.TopicUpdateParams) (*domain.Topic, error) {
			return &domain.Topic{
				ID:          topicID,
				UserID:      uid,
				Name:        sameName,
				Description: &sameDesc,
			}, nil
		},
	}

	auditMock := &auditLoggerMock{
		LogFunc: func(ctx context.Context, record domain.AuditRecord) error {
			t.Error("audit.Log should not be called for no-op update")
			return nil
		},
	}

	svc := newCRUDTestService(t, topicMock, auditMock, defaultTxMock())
	ctx := ctxutil.WithUserID(context.Background(), userID)

	_, err := svc.UpdateTopic(ctx, UpdateTopicInput{
		TopicID: topicID,
		Name:    &sameName,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

// ---------------------------------------------------------------------------
// GetTopic Tests (issue #10)
// ---------------------------------------------------------------------------

func TestGetTopic_Success(t *testing.T) {
	t.Parallel()

	userID := uuid.New()
	topicID := uuid.New()
	desc := "test description"

	topicMock := &topicRepoMock{
		GetByIDFunc: func(ctx context.Context, uid, tid uuid.UUID) (*domain.Topic, error) {
			if uid != userID {
				t.Errorf("userID: got %v, want %v", uid, userID)
			}
			if tid != topicID {
				t.Errorf("topicID: got %v, want %v", tid, topicID)
			}
			return &domain.Topic{
				ID:          topicID,
				UserID:      uid,
				Name:        "Travel",
				Description: &desc,
				CreatedAt:   time.Now(),
				UpdatedAt:   time.Now(),
			}, nil
		},
	}

	svc := newCRUDTestService(t, topicMock, defaultAuditMock(), defaultTxMock())
	ctx := ctxutil.WithUserID(context.Background(), userID)

	result, err := svc.GetTopic(ctx, topicID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.ID != topicID {
		t.Errorf("topic ID: got %v, want %v", result.ID, topicID)
	}
	if result.Name != "Travel" {
		t.Errorf("name: got %q, want %q", result.Name, "Travel")
	}
	if result.Description == nil || *result.Description != desc {
		t.Errorf("description: got %v, want %q", result.Description, desc)
	}
}

func TestGetTopic_NotFound(t *testing.T) {
	t.Parallel()

	userID := uuid.New()
	topicID := uuid.New()

	topicMock := &topicRepoMock{
		GetByIDFunc: func(ctx context.Context, uid, tid uuid.UUID) (*domain.Topic, error) {
			return nil, domain.ErrNotFound
		},
	}

	svc := newCRUDTestService(t, topicMock, defaultAuditMock(), defaultTxMock())
	ctx := ctxutil.WithUserID(context.Background(), userID)

	_, err := svc.GetTopic(ctx, topicID)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !errors.Is(err, domain.ErrNotFound) {
		t.Errorf("error: got %v, want ErrNotFound", err)
	}
}

func TestGetTopic_Unauthorized(t *testing.T) {
	t.Parallel()

	svc := newCRUDTestService(t, &topicRepoMock{}, defaultAuditMock(), defaultTxMock())
	ctx := context.Background()

	_, err := svc.GetTopic(ctx, uuid.New())
	if !errors.Is(err, domain.ErrUnauthorized) {
		t.Errorf("error: got %v, want ErrUnauthorized", err)
	}
}

func TestGetTopic_NilID(t *testing.T) {
	t.Parallel()

	userID := uuid.New()
	svc := newCRUDTestService(t, &topicRepoMock{}, defaultAuditMock(), defaultTxMock())
	ctx := ctxutil.WithUserID(context.Background(), userID)

	_, err := svc.GetTopic(ctx, uuid.Nil)
	if err == nil {
		t.Fatal("expected error, got nil")
	}

	var ve *domain.ValidationError
	if !errors.As(err, &ve) {
		t.Fatalf("expected ValidationError, got %T: %v", err, err)
	}
	if ve.Errors[0].Field != "topic_id" {
		t.Errorf("field: got %q, want %q", ve.Errors[0].Field, "topic_id")
	}
}
