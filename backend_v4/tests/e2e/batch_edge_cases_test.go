//go:build e2e

package e2e_test

import (
	"net/http"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/heartmarshall/myenglish-backend/internal/adapter/postgres/testhelper"
)

// ---------------------------------------------------------------------------
// Scenario 14: Batch delete entries.
// ---------------------------------------------------------------------------

func TestE2E_BatchDeleteEntries(t *testing.T) {
	ts := setupTestServer(t)
	token := createTestUserAndGetToken(t, ts)

	// Create 3 entries.
	entryIDs := make([]any, 3)
	createQuery := `mutation($input: CreateEntryCustomInput!) {
		createEntryCustom(input: $input) { entry { id } }
	}`
	for i := 0; i < 3; i++ {
		status, result := ts.graphqlQuery(t, createQuery, map[string]any{
			"input": map[string]any{
				"text":   "batchdel-" + uuid.New().String()[:8],
				"senses": []any{map[string]any{"definition": "def"}},
			},
		}, token)
		require.Equal(t, http.StatusOK, status)
		requireNoErrors(t, result)
		entryIDs[i] = gqlPayload(t, result, "createEntryCustom")["entry"].(map[string]any)["id"].(string)
	}

	// Batch delete all 3.
	batchDeleteQuery := `mutation($ids: [UUID!]!) {
		batchDeleteEntries(ids: $ids) { deletedCount errors { id message } }
	}`
	status, result := ts.graphqlQuery(t, batchDeleteQuery, map[string]any{"ids": entryIDs}, token)
	assert.Equal(t, http.StatusOK, status)
	requireNoErrors(t, result)

	payload := gqlPayload(t, result, "batchDeleteEntries")
	assert.Equal(t, float64(3), payload["deletedCount"])
	assert.Empty(t, payload["errors"].([]any))

	// Verify all 3 are in trash.
	trashQuery := `query { deletedEntries(limit: 100) { entries { id } totalCount } }`
	status, result = ts.graphqlQuery(t, trashQuery, nil, token)
	assert.Equal(t, http.StatusOK, status)
	requireNoErrors(t, result)

	trashEntries := gqlPayload(t, result, "deletedEntries")["entries"].([]any)
	trashIDs := make(map[string]bool)
	for _, e := range trashEntries {
		trashIDs[e.(map[string]any)["id"].(string)] = true
	}
	for _, id := range entryIDs {
		assert.True(t, trashIDs[id.(string)], "entry %s should be in trash", id)
	}
}

// ---------------------------------------------------------------------------
// Scenario 15: Batch create cards — partial success.
// ---------------------------------------------------------------------------

func TestE2E_BatchCreateCards_PartialSuccess(t *testing.T) {
	ts := setupTestServer(t)
	token, userID := createTestUserWithID(t, ts)

	// Entry A: has senses, no card → should succeed.
	ref := testhelper.SeedRefEntry(t, ts.Pool, "batchcard-a-"+uuid.New().String()[:8])
	entryA := testhelper.SeedEntry(t, ts.Pool, userID, ref.ID)

	// Entry B: already has a card → should be skipped.
	ref2 := testhelper.SeedRefEntry(t, ts.Pool, "batchcard-b-"+uuid.New().String()[:8])
	entryB := testhelper.SeedEntryWithCard(t, ts.Pool, userID, ref2.ID)

	batchQuery := `mutation($ids: [UUID!]!) {
		batchCreateCards(entryIds: $ids) { createdCount skippedExisting skippedNoSenses errors { entryId message } }
	}`
	status, result := ts.graphqlQuery(t, batchQuery, map[string]any{
		"ids": []any{entryA.ID.String(), entryB.ID.String()},
	}, token)
	assert.Equal(t, http.StatusOK, status)
	requireNoErrors(t, result)

	payload := gqlPayload(t, result, "batchCreateCards")
	assert.Equal(t, float64(1), payload["createdCount"], "should create 1 card (entry A)")
	assert.Equal(t, float64(1), payload["skippedExisting"], "should skip 1 (entry B already has card)")

	// Verify entry A now has a card.
	getQuery := `query($id: UUID!) { dictionaryEntry(id: $id) { card { id status } } }`
	status, result = ts.graphqlQuery(t, getQuery, map[string]any{"id": entryA.ID.String()}, token)
	assert.Equal(t, http.StatusOK, status)
	requireNoErrors(t, result)
	assert.NotNil(t, gqlPayload(t, result, "dictionaryEntry")["card"])
}

// ---------------------------------------------------------------------------
// Scenario 18: Create card requires senses — entry with no senses fails.
// ---------------------------------------------------------------------------

func TestE2E_CreateCard_RequiresSenses(t *testing.T) {
	ts := setupTestServer(t)
	token := createTestUserAndGetToken(t, ts)

	// Create entry, then delete all its senses to leave it senseless.
	text := "senseless-" + uuid.New().String()[:8]
	createQuery := `mutation($input: CreateEntryCustomInput!) {
		createEntryCustom(input: $input) { entry { id senses { id } } }
	}`
	status, result := ts.graphqlQuery(t, createQuery, map[string]any{
		"input": map[string]any{
			"text":   text,
			"senses": []any{map[string]any{"definition": "will be deleted"}},
		},
	}, token)
	require.Equal(t, http.StatusOK, status)
	requireNoErrors(t, result)
	entry := gqlPayload(t, result, "createEntryCustom")["entry"].(map[string]any)
	entryID := entry["id"].(string)
	senseID := entry["senses"].([]any)[0].(map[string]any)["id"].(string)

	// Delete the sense.
	deleteSenseQuery := `mutation($id: UUID!) { deleteSense(id: $id) { senseId } }`
	status, result = ts.graphqlQuery(t, deleteSenseQuery, map[string]any{"id": senseID}, token)
	require.Equal(t, http.StatusOK, status)
	requireNoErrors(t, result)

	// Try to create card → should fail with VALIDATION.
	createCardQuery := `mutation($id: UUID!) { createCard(entryId: $id) { card { id } } }`
	status, result = ts.graphqlQuery(t, createCardQuery, map[string]any{"id": entryID}, token)
	assert.Equal(t, http.StatusOK, status)
	assert.Equal(t, "VALIDATION", gqlErrorCode(t, result))
}

// ---------------------------------------------------------------------------
// Scenario 18: Duplicate card creation → ALREADY_EXISTS.
// ---------------------------------------------------------------------------

func TestE2E_CreateCard_Duplicate_ReturnsAlreadyExists(t *testing.T) {
	ts := setupTestServer(t)
	token := createTestUserAndGetToken(t, ts)

	// Create entry with card.
	text := "dupcard-" + uuid.New().String()[:8]
	createQuery := `mutation($input: CreateEntryCustomInput!) {
		createEntryCustom(input: $input) { entry { id card { id } } }
	}`
	status, result := ts.graphqlQuery(t, createQuery, map[string]any{
		"input": map[string]any{
			"text":       text,
			"senses":     []any{map[string]any{"definition": "has card"}},
			"createCard": true,
		},
	}, token)
	require.Equal(t, http.StatusOK, status)
	requireNoErrors(t, result)
	entryID := gqlPayload(t, result, "createEntryCustom")["entry"].(map[string]any)["id"].(string)

	// Try to create another card → should fail.
	createCardQuery := `mutation($id: UUID!) { createCard(entryId: $id) { card { id } } }`
	status, result = ts.graphqlQuery(t, createCardQuery, map[string]any{"id": entryID}, token)
	assert.Equal(t, http.StatusOK, status)
	assert.Equal(t, "ALREADY_EXISTS", gqlErrorCode(t, result))
}

// ---------------------------------------------------------------------------
// Scenario 15: Batch create cards with entry that has no senses.
// ---------------------------------------------------------------------------

func TestE2E_BatchCreateCards_SkipsEntriesWithNoSenses(t *testing.T) {
	ts := setupTestServer(t)
	token := createTestUserAndGetToken(t, ts)

	// Create 2 entries: one with senses, one without (delete its sense after).
	createQuery := `mutation($input: CreateEntryCustomInput!) {
		createEntryCustom(input: $input) { entry { id senses { id } } }
	}`

	// Entry with senses.
	status, result := ts.graphqlQuery(t, createQuery, map[string]any{
		"input": map[string]any{
			"text":   "withsense-" + uuid.New().String()[:8],
			"senses": []any{map[string]any{"definition": "has sense"}},
		},
	}, token)
	require.Equal(t, http.StatusOK, status)
	requireNoErrors(t, result)
	entryWithSense := gqlPayload(t, result, "createEntryCustom")["entry"].(map[string]any)["id"].(string)

	// Entry without senses (create then delete sense).
	status, result = ts.graphqlQuery(t, createQuery, map[string]any{
		"input": map[string]any{
			"text":   "nosense-" + uuid.New().String()[:8],
			"senses": []any{map[string]any{"definition": "temp"}},
		},
	}, token)
	require.Equal(t, http.StatusOK, status)
	requireNoErrors(t, result)
	entry2 := gqlPayload(t, result, "createEntryCustom")["entry"].(map[string]any)
	entryNoSense := entry2["id"].(string)
	tmpSenseID := entry2["senses"].([]any)[0].(map[string]any)["id"].(string)

	deleteSenseQuery := `mutation($id: UUID!) { deleteSense(id: $id) { senseId } }`
	_, result = ts.graphqlQuery(t, deleteSenseQuery, map[string]any{"id": tmpSenseID}, token)
	requireNoErrors(t, result)

	// Batch create cards.
	batchQuery := `mutation($ids: [UUID!]!) {
		batchCreateCards(entryIds: $ids) { createdCount skippedExisting skippedNoSenses errors { entryId message } }
	}`
	status, result = ts.graphqlQuery(t, batchQuery, map[string]any{
		"ids": []any{entryWithSense, entryNoSense},
	}, token)
	assert.Equal(t, http.StatusOK, status)
	requireNoErrors(t, result)

	payload := gqlPayload(t, result, "batchCreateCards")
	assert.Equal(t, float64(1), payload["createdCount"])
	// The senseless entry should be in skippedNoSenses or errors.
	totalSkippedOrError := payload["skippedNoSenses"].(float64) + float64(len(payload["errors"].([]any)))
	assert.GreaterOrEqual(t, totalSkippedOrError, float64(1), "entry without senses should be skipped or error")
}

// ---------------------------------------------------------------------------
// Scenario 9 (edge): Create custom entry with topicId — entry auto-linked.
// ---------------------------------------------------------------------------

func TestE2E_CreateCustomEntry_WithTopicId(t *testing.T) {
	ts := setupTestServer(t)
	token := createTestUserAndGetToken(t, ts)

	// Create topic first.
	createTopicQuery := `mutation($input: CreateTopicInput!) {
		createTopic(input: $input) { topic { id } }
	}`
	status, result := ts.graphqlQuery(t, createTopicQuery, map[string]any{
		"input": map[string]any{"name": "AutoLink " + uuid.New().String()[:8]},
	}, token)
	require.Equal(t, http.StatusOK, status)
	requireNoErrors(t, result)
	topicID := gqlPayload(t, result, "createTopic")["topic"].(map[string]any)["id"].(string)

	// Create entry with topicId.
	createEntryQuery := `mutation($input: CreateEntryCustomInput!) {
		createEntryCustom(input: $input) { entry { id topics { id } } }
	}`
	status, result = ts.graphqlQuery(t, createEntryQuery, map[string]any{
		"input": map[string]any{
			"text":    "autolinked-" + uuid.New().String()[:8],
			"senses":  []any{map[string]any{"definition": "auto linked"}},
			"topicId": topicID,
		},
	}, token)
	assert.Equal(t, http.StatusOK, status)
	requireNoErrors(t, result)

	entry := gqlPayload(t, result, "createEntryCustom")["entry"].(map[string]any)
	entryTopics := entry["topics"].([]any)
	require.Len(t, entryTopics, 1)
	assert.Equal(t, topicID, entryTopics[0].(map[string]any)["id"])
}

// ---------------------------------------------------------------------------
// Scenario: Dictionary filtering by hasCard and search.
// ---------------------------------------------------------------------------

func TestE2E_Dictionary_Filtering(t *testing.T) {
	ts := setupTestServer(t)
	token := createTestUserAndGetToken(t, ts)

	prefix := "filter-" + uuid.New().String()[:8]

	// Create entry without card.
	createQuery := `mutation($input: CreateEntryCustomInput!) {
		createEntryCustom(input: $input) { entry { id } }
	}`
	status, result := ts.graphqlQuery(t, createQuery, map[string]any{
		"input": map[string]any{
			"text":       prefix + "-nocard",
			"senses":     []any{map[string]any{"definition": "no card entry"}},
			"createCard": false,
		},
	}, token)
	require.Equal(t, http.StatusOK, status)
	requireNoErrors(t, result)

	// Create entry with card.
	status, result = ts.graphqlQuery(t, createQuery, map[string]any{
		"input": map[string]any{
			"text":       prefix + "-withcard",
			"senses":     []any{map[string]any{"definition": "has card entry"}},
			"createCard": true,
		},
	}, token)
	require.Equal(t, http.StatusOK, status)
	requireNoErrors(t, result)

	dictQuery := `query($input: DictionaryFilterInput!) {
		dictionary(input: $input) { edges { node { id text card { id } } } totalCount }
	}`

	// Filter by hasCard=true.
	status, result = ts.graphqlQuery(t, dictQuery, map[string]any{
		"input": map[string]any{"search": prefix, "hasCard": true, "limit": 10},
	}, token)
	assert.Equal(t, http.StatusOK, status)
	requireNoErrors(t, result)
	edges := gqlPayload(t, result, "dictionary")["edges"].([]any)
	for _, e := range edges {
		node := e.(map[string]any)["node"].(map[string]any)
		assert.NotNil(t, node["card"], "hasCard=true should only return entries with cards")
	}
	assert.Equal(t, float64(1), gqlPayload(t, result, "dictionary")["totalCount"])

	// Filter by hasCard=false.
	status, result = ts.graphqlQuery(t, dictQuery, map[string]any{
		"input": map[string]any{"search": prefix, "hasCard": false, "limit": 10},
	}, token)
	assert.Equal(t, http.StatusOK, status)
	requireNoErrors(t, result)
	edges = gqlPayload(t, result, "dictionary")["edges"].([]any)
	for _, e := range edges {
		node := e.(map[string]any)["node"].(map[string]any)
		assert.Nil(t, node["card"], "hasCard=false should only return entries without cards")
	}
	assert.Equal(t, float64(1), gqlPayload(t, result, "dictionary")["totalCount"])
}
