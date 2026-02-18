package resolver

import (
	"context"
	"log/slog"

	"github.com/google/uuid"
	"github.com/heartmarshall/myenglish-backend/internal/domain"
	"github.com/heartmarshall/myenglish-backend/internal/service/content"
	"github.com/heartmarshall/myenglish-backend/internal/service/dictionary"
	"github.com/heartmarshall/myenglish-backend/internal/service/inbox"
	"github.com/heartmarshall/myenglish-backend/internal/service/study"
	"github.com/heartmarshall/myenglish-backend/internal/service/topic"
	"github.com/heartmarshall/myenglish-backend/internal/service/user"
)

// dictionaryService defines what resolver needs from Dictionary service.
type dictionaryService interface {
	SearchCatalog(ctx context.Context, query string, limit int) ([]domain.RefEntry, error)
	PreviewRefEntry(ctx context.Context, text string) (*domain.RefEntry, error)
	CreateEntryFromCatalog(ctx context.Context, input dictionary.CreateFromCatalogInput) (*domain.Entry, error)
	CreateEntryCustom(ctx context.Context, input dictionary.CreateCustomInput) (*domain.Entry, error)
	FindEntries(ctx context.Context, input dictionary.FindInput) (*dictionary.FindResult, error)
	GetEntry(ctx context.Context, entryID uuid.UUID) (*domain.Entry, error)
	UpdateNotes(ctx context.Context, input dictionary.UpdateNotesInput) (*domain.Entry, error)
	DeleteEntry(ctx context.Context, entryID uuid.UUID) error
	FindDeletedEntries(ctx context.Context, limit, offset int) ([]domain.Entry, int, error)
	RestoreEntry(ctx context.Context, entryID uuid.UUID) (*domain.Entry, error)
	BatchDeleteEntries(ctx context.Context, entryIDs []uuid.UUID) (*dictionary.BatchResult, error)
	ImportEntries(ctx context.Context, input dictionary.ImportInput) (*dictionary.ImportResult, error)
	ExportEntries(ctx context.Context) (*dictionary.ExportResult, error)
}

// contentService defines what resolver needs from Content service.
type contentService interface {
	AddSense(ctx context.Context, input content.AddSenseInput) (*domain.Sense, error)
	UpdateSense(ctx context.Context, input content.UpdateSenseInput) (*domain.Sense, error)
	DeleteSense(ctx context.Context, senseID uuid.UUID) error
	ReorderSenses(ctx context.Context, input content.ReorderSensesInput) error
	AddTranslation(ctx context.Context, input content.AddTranslationInput) (*domain.Translation, error)
	UpdateTranslation(ctx context.Context, input content.UpdateTranslationInput) (*domain.Translation, error)
	DeleteTranslation(ctx context.Context, translationID uuid.UUID) error
	ReorderTranslations(ctx context.Context, input content.ReorderTranslationsInput) error
	AddExample(ctx context.Context, input content.AddExampleInput) (*domain.Example, error)
	UpdateExample(ctx context.Context, input content.UpdateExampleInput) (*domain.Example, error)
	DeleteExample(ctx context.Context, exampleID uuid.UUID) error
	ReorderExamples(ctx context.Context, input content.ReorderExamplesInput) error
	AddUserImage(ctx context.Context, input content.AddUserImageInput) (*domain.UserImage, error)
	DeleteUserImage(ctx context.Context, imageID uuid.UUID) error
}

// studyService defines what resolver needs from Study service.
type studyService interface {
	GetStudyQueue(ctx context.Context, input study.GetQueueInput) ([]*domain.Card, error)
	ReviewCard(ctx context.Context, input study.ReviewCardInput) (*domain.Card, error)
	UndoReview(ctx context.Context, input study.UndoReviewInput) (*domain.Card, error)
	StartSession(ctx context.Context) (*domain.StudySession, error)
	FinishSession(ctx context.Context, input study.FinishSessionInput) (*domain.StudySession, error)
	AbandonSession(ctx context.Context) error
	GetActiveSession(ctx context.Context) (*domain.StudySession, error)
	CreateCard(ctx context.Context, input study.CreateCardInput) (*domain.Card, error)
	DeleteCard(ctx context.Context, input study.DeleteCardInput) error
	BatchCreateCards(ctx context.Context, input study.BatchCreateCardsInput) (study.BatchCreateResult, error)
	GetDashboard(ctx context.Context) (domain.Dashboard, error)
	GetCardHistory(ctx context.Context, input study.GetCardHistoryInput) ([]*domain.ReviewLog, int, error)
	GetCardStats(ctx context.Context, input study.GetCardHistoryInput) (domain.CardStats, error)
}

// topicService defines what resolver needs from Topic service.
type topicService interface {
	CreateTopic(ctx context.Context, input topic.CreateTopicInput) (*domain.Topic, error)
	UpdateTopic(ctx context.Context, input topic.UpdateTopicInput) (*domain.Topic, error)
	DeleteTopic(ctx context.Context, input topic.DeleteTopicInput) error
	ListTopics(ctx context.Context) ([]*domain.Topic, error)
	LinkEntry(ctx context.Context, input topic.LinkEntryInput) error
	UnlinkEntry(ctx context.Context, input topic.UnlinkEntryInput) error
	BatchLinkEntries(ctx context.Context, input topic.BatchLinkEntriesInput) (*topic.BatchLinkResult, error)
}

// inboxService defines what resolver needs from Inbox service.
type inboxService interface {
	CreateItem(ctx context.Context, input inbox.CreateItemInput) (*domain.InboxItem, error)
	ListItems(ctx context.Context, input inbox.ListItemsInput) ([]*domain.InboxItem, int, error)
	GetItem(ctx context.Context, itemID uuid.UUID) (*domain.InboxItem, error)
	DeleteItem(ctx context.Context, input inbox.DeleteItemInput) error
	DeleteAll(ctx context.Context) (int, error)
}

// userService defines what resolver needs from User service.
type userService interface {
	GetProfile(ctx context.Context) (*domain.User, error)
	GetSettings(ctx context.Context) (*domain.UserSettings, error)
	UpdateSettings(ctx context.Context, input user.UpdateSettingsInput) (*domain.UserSettings, error)
}

// Resolver is the root resolver containing all service dependencies.
type Resolver struct {
	dictionary dictionaryService
	content    contentService
	study      studyService
	topic      topicService
	inbox      inboxService
	user       userService
	log        *slog.Logger
}

// NewResolver creates a new Resolver with all service dependencies.
func NewResolver(
	log *slog.Logger,
	dictionary dictionaryService,
	content contentService,
	study studyService,
	topic topicService,
	inbox inboxService,
	user userService,
) *Resolver {
	return &Resolver{
		dictionary: dictionary,
		content:    content,
		study:      study,
		topic:      topic,
		inbox:      inbox,
		user:       user,
		log:        log.With("component", "graphql"),
	}
}
