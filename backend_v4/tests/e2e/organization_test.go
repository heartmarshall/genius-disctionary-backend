//go:build e2e

package e2e_test

import (
	"net/http"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ---------------------------------------------------------------------------
// Scenario 9: Topic with linked entries shows correct entry count.
// ---------------------------------------------------------------------------

func TestE2E_TopicWithEntries_ShowsEntryCount(t *testing.T) {
	ts := setupTestServer(t)
	token := createTestUserAndGetToken(t, ts)

	topicName := "Animals " + uuid.New().String()[:8]

	// Create topic.
	createTopicQuery := `mutation($input: CreateTopicInput!) {
		createTopic(input: $input) { topic { id name entryCount } }
	}`
	status, result := ts.graphqlQuery(t, createTopicQuery, map[string]any{
		"input": map[string]any{"name": topicName},
	}, token)
	require.Equal(t, http.StatusOK, status)
	requireNoErrors(t, result)
	topic := gqlPayload(t, result, "createTopic")["topic"].(map[string]any)
	topicID := topic["id"].(string)
	assert.Equal(t, float64(0), topic["entryCount"])

	// Create 2 entries.
	var entryIDs []string
	createEntryQuery := `mutation($input: CreateEntryCustomInput!) {
		createEntryCustom(input: $input) { entry { id } }
	}`
	for i := 0; i < 2; i++ {
		status, result = ts.graphqlQuery(t, createEntryQuery, map[string]any{
			"input": map[string]any{
				"text":   "animal-" + uuid.New().String()[:8],
				"senses": []any{map[string]any{"definition": "a creature"}},
			},
		}, token)
		require.Equal(t, http.StatusOK, status)
		requireNoErrors(t, result)
		entryIDs = append(entryIDs, gqlPayload(t, result, "createEntryCustom")["entry"].(map[string]any)["id"].(string))
	}

	// Link both entries to topic.
	linkQuery := `mutation($input: LinkEntryInput!) {
		linkEntryToTopic(input: $input) { success }
	}`
	for _, eid := range entryIDs {
		status, result = ts.graphqlQuery(t, linkQuery, map[string]any{
			"input": map[string]any{"topicId": topicID, "entryId": eid},
		}, token)
		assert.Equal(t, http.StatusOK, status)
		requireNoErrors(t, result)
	}

	// Verify topic entry count.
	listQuery := `query { topics { id name entryCount } }`
	status, result = ts.graphqlQuery(t, listQuery, nil, token)
	assert.Equal(t, http.StatusOK, status)
	requireNoErrors(t, result)

	topics := gqlData(t, result)["topics"].([]any)
	for _, tp := range topics {
		tMap := tp.(map[string]any)
		if tMap["id"] == topicID {
			assert.Equal(t, float64(2), tMap["entryCount"])
			break
		}
	}

	// Verify entry shows topic in its topics field.
	getEntryQuery := `query($id: UUID!) { dictionaryEntry(id: $id) { topics { id name } } }`
	status, result = ts.graphqlQuery(t, getEntryQuery, map[string]any{"id": entryIDs[0]}, token)
	assert.Equal(t, http.StatusOK, status)
	requireNoErrors(t, result)
	entryTopics := gqlPayload(t, result, "dictionaryEntry")["topics"].([]any)
	require.Len(t, entryTopics, 1)
	assert.Equal(t, topicID, entryTopics[0].(map[string]any)["id"])
}

// ---------------------------------------------------------------------------
// Scenario 9: Duplicate topic name → ALREADY_EXISTS.
// ---------------------------------------------------------------------------

func TestE2E_DuplicateTopicName_ReturnsAlreadyExists(t *testing.T) {
	ts := setupTestServer(t)
	token := createTestUserAndGetToken(t, ts)

	name := "Unique Topic " + uuid.New().String()[:8]
	createQuery := `mutation($input: CreateTopicInput!) {
		createTopic(input: $input) { topic { id } }
	}`
	input := map[string]any{"input": map[string]any{"name": name}}

	// First creation.
	status, result := ts.graphqlQuery(t, createQuery, input, token)
	require.Equal(t, http.StatusOK, status)
	requireNoErrors(t, result)

	// Second creation with same name.
	status, result = ts.graphqlQuery(t, createQuery, input, token)
	assert.Equal(t, http.StatusOK, status)
	assert.Equal(t, "ALREADY_EXISTS", gqlErrorCode(t, result))
}

// ---------------------------------------------------------------------------
// Scenario 9: Link and unlink entry from topic.
// ---------------------------------------------------------------------------

func TestE2E_LinkUnlinkEntry_UpdatesCount(t *testing.T) {
	ts := setupTestServer(t)
	token := createTestUserAndGetToken(t, ts)

	// Create topic.
	createTopicQuery := `mutation($input: CreateTopicInput!) {
		createTopic(input: $input) { topic { id } }
	}`
	status, result := ts.graphqlQuery(t, createTopicQuery, map[string]any{
		"input": map[string]any{"name": "LinkTest " + uuid.New().String()[:8]},
	}, token)
	require.Equal(t, http.StatusOK, status)
	requireNoErrors(t, result)
	topicID := gqlPayload(t, result, "createTopic")["topic"].(map[string]any)["id"].(string)

	// Create entry.
	createEntryQuery := `mutation($input: CreateEntryCustomInput!) {
		createEntryCustom(input: $input) { entry { id } }
	}`
	status, result = ts.graphqlQuery(t, createEntryQuery, map[string]any{
		"input": map[string]any{
			"text":   "linked-" + uuid.New().String()[:8],
			"senses": []any{map[string]any{"definition": "test"}},
		},
	}, token)
	require.Equal(t, http.StatusOK, status)
	requireNoErrors(t, result)
	entryID := gqlPayload(t, result, "createEntryCustom")["entry"].(map[string]any)["id"].(string)

	// Link.
	linkQuery := `mutation($input: LinkEntryInput!) { linkEntryToTopic(input: $input) { success } }`
	status, result = ts.graphqlQuery(t, linkQuery, map[string]any{
		"input": map[string]any{"topicId": topicID, "entryId": entryID},
	}, token)
	assert.Equal(t, http.StatusOK, status)
	requireNoErrors(t, result)

	// Link again — should be idempotent.
	status, result = ts.graphqlQuery(t, linkQuery, map[string]any{
		"input": map[string]any{"topicId": topicID, "entryId": entryID},
	}, token)
	assert.Equal(t, http.StatusOK, status)
	requireNoErrors(t, result)

	// Unlink.
	unlinkQuery := `mutation($input: UnlinkEntryInput!) { unlinkEntryFromTopic(input: $input) { success } }`
	status, result = ts.graphqlQuery(t, unlinkQuery, map[string]any{
		"input": map[string]any{"topicId": topicID, "entryId": entryID},
	}, token)
	assert.Equal(t, http.StatusOK, status)
	requireNoErrors(t, result)

	// Verify count is 0.
	listQuery := `query { topics { id entryCount } }`
	status, result = ts.graphqlQuery(t, listQuery, nil, token)
	assert.Equal(t, http.StatusOK, status)
	requireNoErrors(t, result)
	for _, tp := range gqlData(t, result)["topics"].([]any) {
		tMap := tp.(map[string]any)
		if tMap["id"] == topicID {
			assert.Equal(t, float64(0), tMap["entryCount"])
		}
	}
}

// ---------------------------------------------------------------------------
// Scenario 9: Update topic.
// ---------------------------------------------------------------------------

func TestE2E_UpdateTopic(t *testing.T) {
	ts := setupTestServer(t)
	token := createTestUserAndGetToken(t, ts)

	name := "OldName " + uuid.New().String()[:8]
	createQuery := `mutation($input: CreateTopicInput!) {
		createTopic(input: $input) { topic { id } }
	}`
	status, result := ts.graphqlQuery(t, createQuery, map[string]any{
		"input": map[string]any{"name": name, "description": "old desc"},
	}, token)
	require.Equal(t, http.StatusOK, status)
	requireNoErrors(t, result)
	topicID := gqlPayload(t, result, "createTopic")["topic"].(map[string]any)["id"].(string)

	// Update name and description.
	updateQuery := `mutation($input: UpdateTopicInput!) {
		updateTopic(input: $input) { topic { id name description } }
	}`
	status, result = ts.graphqlQuery(t, updateQuery, map[string]any{
		"input": map[string]any{
			"topicId":     topicID,
			"name":        "NewName " + uuid.New().String()[:8],
			"description": "new desc",
		},
	}, token)
	assert.Equal(t, http.StatusOK, status)
	requireNoErrors(t, result)
	updated := gqlPayload(t, result, "updateTopic")["topic"].(map[string]any)
	assert.Equal(t, "new desc", updated["description"])
}

// ---------------------------------------------------------------------------
// Scenario 9: Batch link entries.
// ---------------------------------------------------------------------------

func TestE2E_BatchLinkEntries(t *testing.T) {
	ts := setupTestServer(t)
	token := createTestUserAndGetToken(t, ts)

	// Create topic.
	createTopicQuery := `mutation($input: CreateTopicInput!) {
		createTopic(input: $input) { topic { id } }
	}`
	status, result := ts.graphqlQuery(t, createTopicQuery, map[string]any{
		"input": map[string]any{"name": "Batch " + uuid.New().String()[:8]},
	}, token)
	require.Equal(t, http.StatusOK, status)
	requireNoErrors(t, result)
	topicID := gqlPayload(t, result, "createTopic")["topic"].(map[string]any)["id"].(string)

	// Create 3 entries.
	entryIDs := make([]any, 3)
	createEntryQuery := `mutation($input: CreateEntryCustomInput!) {
		createEntryCustom(input: $input) { entry { id } }
	}`
	for i := 0; i < 3; i++ {
		status, result = ts.graphqlQuery(t, createEntryQuery, map[string]any{
			"input": map[string]any{
				"text":   "batch-" + uuid.New().String()[:8],
				"senses": []any{map[string]any{"definition": "def"}},
			},
		}, token)
		require.Equal(t, http.StatusOK, status)
		requireNoErrors(t, result)
		entryIDs[i] = gqlPayload(t, result, "createEntryCustom")["entry"].(map[string]any)["id"].(string)
	}

	// Batch link.
	batchQuery := `mutation($input: BatchLinkEntriesInput!) {
		batchLinkEntriesToTopic(input: $input) { linked skipped }
	}`
	status, result = ts.graphqlQuery(t, batchQuery, map[string]any{
		"input": map[string]any{"topicId": topicID, "entryIds": entryIDs},
	}, token)
	assert.Equal(t, http.StatusOK, status)
	requireNoErrors(t, result)

	payload := gqlPayload(t, result, "batchLinkEntriesToTopic")
	assert.Equal(t, float64(3), payload["linked"])
	assert.Equal(t, float64(0), payload["skipped"])
}

// ---------------------------------------------------------------------------
// Scenario 10: Inbox lifecycle — create, list, get, delete.
// ---------------------------------------------------------------------------

func TestE2E_InboxLifecycle(t *testing.T) {
	ts := setupTestServer(t)
	token := createTestUserAndGetToken(t, ts)

	// Create inbox item.
	createQuery := `mutation($input: CreateInboxItemInput!) {
		createInboxItem(input: $input) { item { id text context createdAt } }
	}`
	itemText := "Remember this word " + uuid.New().String()[:8]
	status, result := ts.graphqlQuery(t, createQuery, map[string]any{
		"input": map[string]any{
			"text":    itemText,
			"context": "heard in a podcast",
		},
	}, token)
	assert.Equal(t, http.StatusOK, status)
	requireNoErrors(t, result)

	item := gqlPayload(t, result, "createInboxItem")["item"].(map[string]any)
	itemID := item["id"].(string)
	assert.Equal(t, itemText, item["text"])
	assert.Equal(t, "heard in a podcast", item["context"])

	// List inbox — should contain the item.
	listQuery := `query { inboxItems(limit: 50) { items { id text } totalCount } }`
	status, result = ts.graphqlQuery(t, listQuery, nil, token)
	assert.Equal(t, http.StatusOK, status)
	requireNoErrors(t, result)
	listData := gqlPayload(t, result, "inboxItems")
	assert.GreaterOrEqual(t, listData["totalCount"].(float64), float64(1))

	// Get single item.
	getQuery := `query($id: UUID!) { inboxItem(id: $id) { id text context } }`
	status, result = ts.graphqlQuery(t, getQuery, map[string]any{"id": itemID}, token)
	assert.Equal(t, http.StatusOK, status)
	requireNoErrors(t, result)
	assert.Equal(t, itemText, gqlPayload(t, result, "inboxItem")["text"])

	// Delete item.
	deleteQuery := `mutation($id: UUID!) { deleteInboxItem(id: $id) { itemId } }`
	status, result = ts.graphqlQuery(t, deleteQuery, map[string]any{"id": itemID}, token)
	assert.Equal(t, http.StatusOK, status)
	requireNoErrors(t, result)

	// Get should now return NOT_FOUND.
	status, result = ts.graphqlQuery(t, getQuery, map[string]any{"id": itemID}, token)
	assert.Equal(t, http.StatusOK, status)
	assert.Equal(t, "NOT_FOUND", gqlErrorCode(t, result))
}

// ---------------------------------------------------------------------------
// Scenario 10: Clear inbox deletes all items.
// ---------------------------------------------------------------------------

func TestE2E_ClearInbox_DeletesAllItems(t *testing.T) {
	ts := setupTestServer(t)
	token := createTestUserAndGetToken(t, ts)

	// Create 3 inbox items.
	createQuery := `mutation($input: CreateInboxItemInput!) {
		createInboxItem(input: $input) { item { id } }
	}`
	for i := 0; i < 3; i++ {
		status, result := ts.graphqlQuery(t, createQuery, map[string]any{
			"input": map[string]any{"text": "clear-" + uuid.New().String()[:8]},
		}, token)
		require.Equal(t, http.StatusOK, status)
		requireNoErrors(t, result)
	}

	// Verify we have items.
	listQuery := `query { inboxItems(limit: 50) { totalCount } }`
	status, result := ts.graphqlQuery(t, listQuery, nil, token)
	assert.Equal(t, http.StatusOK, status)
	requireNoErrors(t, result)
	assert.GreaterOrEqual(t, gqlPayload(t, result, "inboxItems")["totalCount"].(float64), float64(3))

	// Clear inbox.
	clearQuery := `mutation { clearInbox { deletedCount } }`
	status, result = ts.graphqlQuery(t, clearQuery, nil, token)
	assert.Equal(t, http.StatusOK, status)
	requireNoErrors(t, result)
	assert.GreaterOrEqual(t, gqlPayload(t, result, "clearInbox")["deletedCount"].(float64), float64(3))

	// Verify inbox is empty.
	status, result = ts.graphqlQuery(t, listQuery, nil, token)
	assert.Equal(t, http.StatusOK, status)
	requireNoErrors(t, result)
	assert.Equal(t, float64(0), gqlPayload(t, result, "inboxItems")["totalCount"])
}
