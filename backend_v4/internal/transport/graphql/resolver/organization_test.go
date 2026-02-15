package resolver

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/heartmarshall/myenglish-backend/internal/domain"
	"github.com/heartmarshall/myenglish-backend/internal/service/inbox"
	"github.com/heartmarshall/myenglish-backend/internal/service/topic"
	"github.com/heartmarshall/myenglish-backend/internal/transport/graphql/generated"
	"github.com/heartmarshall/myenglish-backend/pkg/ctxutil"
	"github.com/stretchr/testify/require"
)

// Topic Tests

func TestCreateTopic_Success(t *testing.T) {
	t.Parallel()

	mock := &topicServiceMock{
		CreateTopicFunc: func(ctx context.Context, input topic.CreateTopicInput) (*domain.Topic, error) {
			desc := "test description"
			return &domain.Topic{
				ID:          uuid.New(),
				Name:        input.Name,
				Description: &desc,
			}, nil
		},
	}

	resolver := &mutationResolver{&Resolver{topic: mock}}
	ctx := ctxutil.WithUserID(context.Background(), uuid.New())

	desc := "test description"
	result, err := resolver.CreateTopic(ctx, generated.CreateTopicInput{
		Name:        "Test Topic",
		Description: &desc,
	})

	require.NoError(t, err)
	require.NotNil(t, result.Topic)
	require.Equal(t, "Test Topic", result.Topic.Name)
}

func TestCreateTopic_Unauthorized(t *testing.T) {
	t.Parallel()

	resolver := &mutationResolver{&Resolver{topic: &topicServiceMock{}}}
	ctx := context.Background()

	_, err := resolver.CreateTopic(ctx, generated.CreateTopicInput{Name: "Test"})

	require.ErrorIs(t, err, domain.ErrUnauthorized)
}

func TestUpdateTopic_Success(t *testing.T) {
	t.Parallel()

	topicID := uuid.New()
	newName := "Updated Topic"

	mock := &topicServiceMock{
		UpdateTopicFunc: func(ctx context.Context, input topic.UpdateTopicInput) (*domain.Topic, error) {
			return &domain.Topic{
				ID:   input.TopicID,
				Name: *input.Name,
			}, nil
		},
	}

	resolver := &mutationResolver{&Resolver{topic: mock}}
	ctx := ctxutil.WithUserID(context.Background(), uuid.New())

	result, err := resolver.UpdateTopic(ctx, generated.UpdateTopicInput{
		TopicID: topicID,
		Name:    &newName,
	})

	require.NoError(t, err)
	require.NotNil(t, result.Topic)
	require.Equal(t, topicID, result.Topic.ID)
	require.Equal(t, newName, result.Topic.Name)
}

func TestDeleteTopic_Success(t *testing.T) {
	t.Parallel()

	topicID := uuid.New()

	mock := &topicServiceMock{
		DeleteTopicFunc: func(ctx context.Context, input topic.DeleteTopicInput) error {
			require.Equal(t, topicID, input.TopicID)
			return nil
		},
	}

	resolver := &mutationResolver{&Resolver{topic: mock}}
	ctx := ctxutil.WithUserID(context.Background(), uuid.New())

	result, err := resolver.DeleteTopic(ctx, topicID)

	require.NoError(t, err)
	require.Equal(t, topicID, result.TopicID)
}

func TestLinkEntryToTopic_Success(t *testing.T) {
	t.Parallel()

	topicID := uuid.New()
	entryID := uuid.New()

	mock := &topicServiceMock{
		LinkEntryFunc: func(ctx context.Context, input topic.LinkEntryInput) error {
			require.Equal(t, topicID, input.TopicID)
			require.Equal(t, entryID, input.EntryID)
			return nil
		},
	}

	resolver := &mutationResolver{&Resolver{topic: mock}}
	ctx := ctxutil.WithUserID(context.Background(), uuid.New())

	result, err := resolver.LinkEntryToTopic(ctx, generated.LinkEntryInput{
		TopicID: topicID,
		EntryID: entryID,
	})

	require.NoError(t, err)
	require.True(t, result.Success)
}

func TestUnlinkEntryFromTopic_Success(t *testing.T) {
	t.Parallel()

	topicID := uuid.New()
	entryID := uuid.New()

	mock := &topicServiceMock{
		UnlinkEntryFunc: func(ctx context.Context, input topic.UnlinkEntryInput) error {
			require.Equal(t, topicID, input.TopicID)
			require.Equal(t, entryID, input.EntryID)
			return nil
		},
	}

	resolver := &mutationResolver{&Resolver{topic: mock}}
	ctx := ctxutil.WithUserID(context.Background(), uuid.New())

	result, err := resolver.UnlinkEntryFromTopic(ctx, generated.UnlinkEntryInput{
		TopicID: topicID,
		EntryID: entryID,
	})

	require.NoError(t, err)
	require.True(t, result.Success)
}

func TestBatchLinkEntriesToTopic_Success(t *testing.T) {
	t.Parallel()

	topicID := uuid.New()
	entryIDs := []uuid.UUID{uuid.New(), uuid.New(), uuid.New()}

	mock := &topicServiceMock{
		BatchLinkEntriesFunc: func(ctx context.Context, input topic.BatchLinkEntriesInput) (*topic.BatchLinkResult, error) {
			require.Equal(t, topicID, input.TopicID)
			require.Equal(t, entryIDs, input.EntryIDs)
			return &topic.BatchLinkResult{
				Linked:  2,
				Skipped: 1,
			}, nil
		},
	}

	resolver := &mutationResolver{&Resolver{topic: mock}}
	ctx := ctxutil.WithUserID(context.Background(), uuid.New())

	result, err := resolver.BatchLinkEntriesToTopic(ctx, generated.BatchLinkEntriesInput{
		TopicID:  topicID,
		EntryIds: entryIDs,
	})

	require.NoError(t, err)
	require.Equal(t, 2, result.Linked)
	require.Equal(t, 1, result.Skipped)
}

func TestTopics_Success(t *testing.T) {
	t.Parallel()

	topics := []*domain.Topic{
		{ID: uuid.New(), Name: "Topic 1"},
		{ID: uuid.New(), Name: "Topic 2"},
	}

	mock := &topicServiceMock{
		ListTopicsFunc: func(ctx context.Context) ([]*domain.Topic, error) {
			return topics, nil
		},
	}

	resolver := &queryResolver{&Resolver{topic: mock}}
	ctx := ctxutil.WithUserID(context.Background(), uuid.New())

	result, err := resolver.Topics(ctx)

	require.NoError(t, err)
	require.Len(t, result, 2)
	require.Equal(t, "Topic 1", result[0].Name)
}

// Inbox Tests

func TestCreateInboxItem_Success(t *testing.T) {
	t.Parallel()

	ctx := ctxutil.WithUserID(context.Background(), uuid.New())
	itemContext := "test context"

	mock := &inboxServiceMock{
		CreateItemFunc: func(ctx context.Context, input inbox.CreateItemInput) (*domain.InboxItem, error) {
			return &domain.InboxItem{
				ID:      uuid.New(),
				Text:    input.Text,
				Context: input.Context,
			}, nil
		},
	}

	resolver := &mutationResolver{&Resolver{inbox: mock}}

	result, err := resolver.CreateInboxItem(ctx, generated.CreateInboxItemInput{
		Text:    "test item",
		Context: &itemContext,
	})

	require.NoError(t, err)
	require.NotNil(t, result.Item)
	require.Equal(t, "test item", result.Item.Text)
}

func TestDeleteInboxItem_Success(t *testing.T) {
	t.Parallel()

	itemID := uuid.New()

	mock := &inboxServiceMock{
		DeleteItemFunc: func(ctx context.Context, input inbox.DeleteItemInput) error {
			require.Equal(t, itemID, input.ItemID)
			return nil
		},
	}

	resolver := &mutationResolver{&Resolver{inbox: mock}}
	ctx := ctxutil.WithUserID(context.Background(), uuid.New())

	result, err := resolver.DeleteInboxItem(ctx, itemID)

	require.NoError(t, err)
	require.Equal(t, itemID, result.ItemID)
}

func TestClearInbox_Success(t *testing.T) {
	t.Parallel()

	mock := &inboxServiceMock{
		DeleteAllFunc: func(ctx context.Context) (int, error) {
			return 5, nil
		},
	}

	resolver := &mutationResolver{&Resolver{inbox: mock}}
	ctx := ctxutil.WithUserID(context.Background(), uuid.New())

	result, err := resolver.ClearInbox(ctx)

	require.NoError(t, err)
	require.Equal(t, 5, result.DeletedCount)
}

func TestInboxItems_Success(t *testing.T) {
	t.Parallel()

	items := []*domain.InboxItem{
		{ID: uuid.New(), Text: "Item 1"},
		{ID: uuid.New(), Text: "Item 2"},
	}

	mock := &inboxServiceMock{
		ListItemsFunc: func(ctx context.Context, input inbox.ListItemsInput) ([]*domain.InboxItem, int, error) {
			return items, 2, nil
		},
	}

	resolver := &queryResolver{&Resolver{inbox: mock}}
	ctx := ctxutil.WithUserID(context.Background(), uuid.New())

	limit := 10
	offset := 0
	result, err := resolver.InboxItems(ctx, &limit, &offset)

	require.NoError(t, err)
	require.Len(t, result.Items, 2)
	require.Equal(t, 2, result.TotalCount)
}

func TestInboxItems_DefaultPagination(t *testing.T) {
	t.Parallel()

	mock := &inboxServiceMock{
		ListItemsFunc: func(ctx context.Context, input inbox.ListItemsInput) ([]*domain.InboxItem, int, error) {
			require.Equal(t, 50, input.Limit)
			require.Equal(t, 0, input.Offset)
			return []*domain.InboxItem{}, 0, nil
		},
	}

	resolver := &queryResolver{&Resolver{inbox: mock}}
	ctx := ctxutil.WithUserID(context.Background(), uuid.New())

	_, err := resolver.InboxItems(ctx, nil, nil)

	require.NoError(t, err)
}

func TestInboxItem_Success(t *testing.T) {
	t.Parallel()

	itemID := uuid.New()

	mock := &inboxServiceMock{
		GetItemFunc: func(ctx context.Context, id uuid.UUID) (*domain.InboxItem, error) {
			require.Equal(t, itemID, id)
			return &domain.InboxItem{
				ID:   id,
				Text: "test item",
			}, nil
		},
	}

	resolver := &queryResolver{&Resolver{inbox: mock}}
	ctx := ctxutil.WithUserID(context.Background(), uuid.New())

	result, err := resolver.InboxItem(ctx, itemID)

	require.NoError(t, err)
	require.NotNil(t, result)
	require.Equal(t, itemID, result.ID)
}

func TestInboxItem_NotFound(t *testing.T) {
	t.Parallel()

	mock := &inboxServiceMock{
		GetItemFunc: func(ctx context.Context, id uuid.UUID) (*domain.InboxItem, error) {
			return nil, domain.ErrNotFound
		},
	}

	resolver := &queryResolver{&Resolver{inbox: mock}}
	ctx := ctxutil.WithUserID(context.Background(), uuid.New())

	_, err := resolver.InboxItem(ctx, uuid.New())

	require.ErrorIs(t, err, domain.ErrNotFound)
}
