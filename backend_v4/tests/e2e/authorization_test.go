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
// Scenario 7: User A's entries are invisible to User B.
// ---------------------------------------------------------------------------

func TestE2E_UserCannotSeeOtherUserEntries(t *testing.T) {
	ts := setupTestServer(t)
	tokenA, userA := createTestUserWithID(t, ts)
	tokenB := createTestUserAndGetToken(t, ts)

	// User A creates an entry.
	ref := testhelper.SeedRefEntry(t, ts.Pool, "private-"+uuid.New().String()[:8])
	entry := testhelper.SeedEntry(t, ts.Pool, userA, ref.ID)

	// User B tries to read it → NOT_FOUND.
	getQuery := `query($id: UUID!) { dictionaryEntry(id: $id) { id text } }`
	status, result := ts.graphqlQuery(t, getQuery, map[string]any{"id": entry.ID.String()}, tokenB)
	assert.Equal(t, http.StatusOK, status)
	assert.Equal(t, "NOT_FOUND", gqlErrorCode(t, result))

	// User A can see it.
	status, result = ts.graphqlQuery(t, getQuery, map[string]any{"id": entry.ID.String()}, tokenA)
	assert.Equal(t, http.StatusOK, status)
	requireNoErrors(t, result)
	assert.Equal(t, ref.Text, gqlPayload(t, result, "dictionaryEntry")["text"])
}

// ---------------------------------------------------------------------------
// Scenario 7: User A cannot delete User B's entry.
// ---------------------------------------------------------------------------

func TestE2E_UserCannotDeleteOtherUserEntry(t *testing.T) {
	ts := setupTestServer(t)
	_, userA := createTestUserWithID(t, ts)
	tokenB := createTestUserAndGetToken(t, ts)

	// User A creates an entry via DB seed.
	ref := testhelper.SeedRefEntry(t, ts.Pool, "protected-"+uuid.New().String()[:8])
	entry := testhelper.SeedEntry(t, ts.Pool, userA, ref.ID)

	// User B tries to delete it.
	deleteQuery := `mutation($id: UUID!) { deleteEntry(id: $id) { entryId } }`
	status, result := ts.graphqlQuery(t, deleteQuery, map[string]any{"id": entry.ID.String()}, tokenB)
	assert.Equal(t, http.StatusOK, status)

	// Should get NOT_FOUND (service filters by user_id, so B can't even find it).
	assert.Equal(t, "NOT_FOUND", gqlErrorCode(t, result))
}

// ---------------------------------------------------------------------------
// Scenario 7: User B cannot see User A's topics.
// ---------------------------------------------------------------------------

func TestE2E_UserCannotSeeOtherUserTopics(t *testing.T) {
	ts := setupTestServer(t)
	tokenA := createTestUserAndGetToken(t, ts)
	tokenB := createTestUserAndGetToken(t, ts)

	// User A creates a topic.
	createQuery := `mutation($input: CreateTopicInput!) {
		createTopic(input: $input) { topic { id name } }
	}`
	status, result := ts.graphqlQuery(t, createQuery, map[string]any{
		"input": map[string]any{"name": "User A Topic " + uuid.New().String()[:8]},
	}, tokenA)
	require.Equal(t, http.StatusOK, status)
	requireNoErrors(t, result)
	topicID := gqlPayload(t, result, "createTopic")["topic"].(map[string]any)["id"].(string)

	// User B lists topics — should not see User A's topic.
	listQuery := `query { topics { id name } }`
	status, result = ts.graphqlQuery(t, listQuery, nil, tokenB)
	assert.Equal(t, http.StatusOK, status)
	requireNoErrors(t, result)

	topics := gqlData(t, result)["topics"].([]any)
	for _, tp := range topics {
		assert.NotEqual(t, topicID, tp.(map[string]any)["id"],
			"User B should not see User A's topic")
	}
}

// ---------------------------------------------------------------------------
// Scenario 7: User B cannot access User A's inbox items.
// ---------------------------------------------------------------------------

func TestE2E_UserCannotAccessOtherUserInboxItem(t *testing.T) {
	ts := setupTestServer(t)
	tokenA := createTestUserAndGetToken(t, ts)
	tokenB := createTestUserAndGetToken(t, ts)

	// User A creates an inbox item.
	createQuery := `mutation($input: CreateInboxItemInput!) {
		createInboxItem(input: $input) { item { id text } }
	}`
	status, result := ts.graphqlQuery(t, createQuery, map[string]any{
		"input": map[string]any{
			"text":    "User A's private note " + uuid.New().String()[:8],
			"context": "private context",
		},
	}, tokenA)
	require.Equal(t, http.StatusOK, status)
	requireNoErrors(t, result)
	itemID := gqlPayload(t, result, "createInboxItem")["item"].(map[string]any)["id"].(string)

	// User B tries to read it → NOT_FOUND.
	getQuery := `query($id: UUID!) { inboxItem(id: $id) { id text } }`
	status, result = ts.graphqlQuery(t, getQuery, map[string]any{"id": itemID}, tokenB)
	assert.Equal(t, http.StatusOK, status)
	assert.Equal(t, "NOT_FOUND", gqlErrorCode(t, result))
}

// ---------------------------------------------------------------------------
// Scenario 7: User B cannot review User A's card.
// ---------------------------------------------------------------------------

func TestE2E_UserCannotReviewOtherUserCard(t *testing.T) {
	ts := setupTestServer(t)
	_, userA := createTestUserWithID(t, ts)
	tokenB := createTestUserAndGetToken(t, ts)

	// User A has an entry with a card.
	ref := testhelper.SeedRefEntry(t, ts.Pool, "stolen-"+uuid.New().String()[:8])
	entry := testhelper.SeedEntryWithCard(t, ts.Pool, userA, ref.ID)

	// User B tries to review it.
	reviewQuery := `mutation($input: ReviewCardInput!) {
		reviewCard(input: $input) { card { id status } }
	}`
	status, result := ts.graphqlQuery(t, reviewQuery, map[string]any{
		"input": map[string]any{"cardId": entry.Card.ID.String(), "grade": "GOOD"},
	}, tokenB)
	assert.Equal(t, http.StatusOK, status)

	errorCode := gqlErrorCode(t, result)
	assert.Contains(t, []string{"NOT_FOUND", "FORBIDDEN"}, errorCode,
		"should get NOT_FOUND or FORBIDDEN for another user's card")
}

// ---------------------------------------------------------------------------
// Scenario 7: User B cannot modify User A's entry content.
// ---------------------------------------------------------------------------

func TestE2E_UserCannotAddSenseToOtherUserEntry(t *testing.T) {
	ts := setupTestServer(t)
	_, userA := createTestUserWithID(t, ts)
	tokenB := createTestUserAndGetToken(t, ts)

	ref := testhelper.SeedRefEntry(t, ts.Pool, "authcontent-"+uuid.New().String()[:8])
	entry := testhelper.SeedEntry(t, ts.Pool, userA, ref.ID)

	addSenseQuery := `mutation($input: AddSenseInput!) {
		addSense(input: $input) { sense { id } }
	}`
	status, result := ts.graphqlQuery(t, addSenseQuery, map[string]any{
		"input": map[string]any{
			"entryId":    entry.ID.String(),
			"definition": "injected sense",
		},
	}, tokenB)
	assert.Equal(t, http.StatusOK, status)

	errorCode := gqlErrorCode(t, result)
	assert.Contains(t, []string{"NOT_FOUND", "FORBIDDEN"}, errorCode,
		"should not allow adding sense to another user's entry")
}
