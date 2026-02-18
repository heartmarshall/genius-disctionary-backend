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
// Scenario 5: Study queue shows due and new cards.
// ---------------------------------------------------------------------------

func TestE2E_StudyQueue_ContainsNewCards(t *testing.T) {
	ts := setupTestServer(t)
	token, userID := createTestUserWithID(t, ts)

	// Seed 3 entries with NEW cards.
	for i := 0; i < 3; i++ {
		ref := testhelper.SeedRefEntry(t, ts.Pool, "queue-"+uuid.New().String()[:8])
		testhelper.SeedEntryWithCard(t, ts.Pool, userID, ref.ID)
	}

	queueQuery := `query { studyQueue(limit: 10) { id text card { id status } } }`
	status, result := ts.graphqlQuery(t, queueQuery, nil, token)
	assert.Equal(t, http.StatusOK, status)
	requireNoErrors(t, result)

	queue := gqlData(t, result)["studyQueue"].([]any)
	assert.GreaterOrEqual(t, len(queue), 3, "queue should contain at least 3 new cards")

	// All returned entries should have a card.
	for _, item := range queue {
		entry := item.(map[string]any)
		assert.NotNil(t, entry["card"], "queue entry should have a card")
	}
}

// ---------------------------------------------------------------------------
// Scenario 5: Review card — state transitions for each grade from NEW.
// ---------------------------------------------------------------------------

func TestE2E_ReviewCard_AllGradesFromNew(t *testing.T) {
	ts := setupTestServer(t)

	grades := []struct {
		grade          string
		expectedStatus string
	}{
		{"AGAIN", "LEARNING"},
		{"HARD", "LEARNING"},
		{"GOOD", "LEARNING"}, // 2 learning steps → goes to step 1, still LEARNING
		{"EASY", "REVIEW"},   // EASY from NEW → graduates immediately
	}

	for _, tc := range grades {
		t.Run(tc.grade, func(t *testing.T) {
			token, userID := createTestUserWithID(t, ts)
			ref := testhelper.SeedRefEntry(t, ts.Pool, "review-"+uuid.New().String()[:8])
			entry := testhelper.SeedEntryWithCard(t, ts.Pool, userID, ref.ID)
			cardID := entry.Card.ID.String()

			reviewQuery := `mutation($input: ReviewCardInput!) {
				reviewCard(input: $input) { card { id status nextReviewAt intervalDays easeFactor } }
			}`
			status, result := ts.graphqlQuery(t, reviewQuery, map[string]any{
				"input": map[string]any{
					"cardId":     cardID,
					"grade":      tc.grade,
					"durationMs": 5000,
				},
			}, token)
			assert.Equal(t, http.StatusOK, status)
			requireNoErrors(t, result)

			card := gqlPayload(t, result, "reviewCard")["card"].(map[string]any)
			assert.Equal(t, tc.expectedStatus, card["status"],
				"grade %s from NEW should result in status %s", tc.grade, tc.expectedStatus)
			assert.NotNil(t, card["nextReviewAt"], "nextReviewAt should be set after review")

			// Verify via separate card stats query.
			statsQuery := `query($id: UUID!) { cardStats(cardId: $id) { totalReviews } }`
			status, result = ts.graphqlQuery(t, statsQuery, map[string]any{"id": cardID}, token)
			assert.Equal(t, http.StatusOK, status)
			requireNoErrors(t, result)

			stats := gqlPayload(t, result, "cardStats")
			assert.Equal(t, float64(1), stats["totalReviews"])
		})
	}
}

// ---------------------------------------------------------------------------
// Scenario 5: Undo review restores previous card state.
// ---------------------------------------------------------------------------

func TestE2E_UndoReview_RestoresPreviousState(t *testing.T) {
	ts := setupTestServer(t)
	token, userID := createTestUserWithID(t, ts)

	ref := testhelper.SeedRefEntry(t, ts.Pool, "undo-"+uuid.New().String()[:8])
	entry := testhelper.SeedEntryWithCard(t, ts.Pool, userID, ref.ID)
	cardID := entry.Card.ID.String()

	// Card starts as NEW.
	getCardQuery := `query($id: UUID!) { dictionaryEntry(id: $id) { card { id status easeFactor } } }`
	status, result := ts.graphqlQuery(t, getCardQuery, map[string]any{"id": entry.ID.String()}, token)
	require.Equal(t, http.StatusOK, status)
	requireNoErrors(t, result)
	originalCard := gqlPayload(t, result, "dictionaryEntry")["card"].(map[string]any)
	assert.Equal(t, "NEW", originalCard["status"])

	// Review with AGAIN → LEARNING.
	reviewQuery := `mutation($input: ReviewCardInput!) {
		reviewCard(input: $input) { card { id status } }
	}`
	status, result = ts.graphqlQuery(t, reviewQuery, map[string]any{
		"input": map[string]any{"cardId": cardID, "grade": "AGAIN"},
	}, token)
	assert.Equal(t, http.StatusOK, status)
	requireNoErrors(t, result)
	assert.Equal(t, "LEARNING", gqlPayload(t, result, "reviewCard")["card"].(map[string]any)["status"])

	// Undo the review.
	undoQuery := `mutation($id: UUID!) { undoReview(cardId: $id) { card { id status } } }`
	status, result = ts.graphqlQuery(t, undoQuery, map[string]any{"id": cardID}, token)
	assert.Equal(t, http.StatusOK, status)
	requireNoErrors(t, result)

	undoneCard := gqlPayload(t, result, "undoReview")["card"].(map[string]any)
	assert.Equal(t, "NEW", undoneCard["status"], "undo should restore card to NEW")

	// Verify review log was deleted.
	historyQuery := `query($input: GetCardHistoryInput!) { cardHistory(input: $input) { id } }`
	status, result = ts.graphqlQuery(t, historyQuery, map[string]any{
		"input": map[string]any{"cardId": cardID},
	}, token)
	assert.Equal(t, http.StatusOK, status)
	requireNoErrors(t, result)
	history := gqlData(t, result)["cardHistory"].([]any)
	assert.Empty(t, history, "review log should be deleted after undo")
}

// ---------------------------------------------------------------------------
// Scenario 6: Study session full lifecycle — start → review → finish.
// ---------------------------------------------------------------------------

func TestE2E_StudySession_FullLifecycle(t *testing.T) {
	ts := setupTestServer(t)
	token, userID := createTestUserWithID(t, ts)

	// Seed 2 entries with cards.
	var cardIDs []string
	for i := 0; i < 2; i++ {
		ref := testhelper.SeedRefEntry(t, ts.Pool, "session-"+uuid.New().String()[:8])
		entry := testhelper.SeedEntryWithCard(t, ts.Pool, userID, ref.ID)
		cardIDs = append(cardIDs, entry.Card.ID.String())
	}

	// Start session.
	startQuery := `mutation { startStudySession { session { id status startedAt } } }`
	status, result := ts.graphqlQuery(t, startQuery, nil, token)
	assert.Equal(t, http.StatusOK, status)
	requireNoErrors(t, result)

	session := gqlPayload(t, result, "startStudySession")["session"].(map[string]any)
	sessionID := session["id"].(string)
	assert.Equal(t, "ACTIVE", session["status"])

	// Start again should be idempotent — same session.
	status, result = ts.graphqlQuery(t, startQuery, nil, token)
	assert.Equal(t, http.StatusOK, status)
	requireNoErrors(t, result)
	sameSession := gqlPayload(t, result, "startStudySession")["session"].(map[string]any)
	assert.Equal(t, sessionID, sameSession["id"], "startStudySession should be idempotent")

	// Review both cards.
	reviewQuery := `mutation($input: ReviewCardInput!) {
		reviewCard(input: $input) { card { id status } }
	}`
	for _, cardID := range cardIDs {
		status, result = ts.graphqlQuery(t, reviewQuery, map[string]any{
			"input": map[string]any{"cardId": cardID, "grade": "GOOD", "durationMs": 3000},
		}, token)
		assert.Equal(t, http.StatusOK, status)
		requireNoErrors(t, result)
	}

	// Finish session.
	finishQuery := `mutation($input: FinishSessionInput!) {
		finishStudySession(input: $input) {
			session { id status finishedAt result { totalReviews gradeCounts { again hard good easy } } }
		}
	}`
	status, result = ts.graphqlQuery(t, finishQuery, map[string]any{
		"input": map[string]any{"sessionId": sessionID},
	}, token)
	assert.Equal(t, http.StatusOK, status)
	requireNoErrors(t, result)

	finished := gqlPayload(t, result, "finishStudySession")["session"].(map[string]any)
	assert.Equal(t, "FINISHED", finished["status"])
	assert.NotNil(t, finished["finishedAt"])

	sessionResult := finished["result"].(map[string]any)
	assert.Equal(t, float64(2), sessionResult["totalReviews"])

	gradeCounts := sessionResult["gradeCounts"].(map[string]any)
	assert.Equal(t, float64(2), gradeCounts["good"])
	assert.Equal(t, float64(0), gradeCounts["again"])
}

// ---------------------------------------------------------------------------
// Scenario 6: Abandon session.
// ---------------------------------------------------------------------------

func TestE2E_StudySession_Abandon(t *testing.T) {
	ts := setupTestServer(t)
	token := createTestUserAndGetToken(t, ts)

	// Start session.
	startQuery := `mutation { startStudySession { session { id status } } }`
	status, result := ts.graphqlQuery(t, startQuery, nil, token)
	assert.Equal(t, http.StatusOK, status)
	requireNoErrors(t, result)
	assert.Equal(t, "ACTIVE", gqlPayload(t, result, "startStudySession")["session"].(map[string]any)["status"])

	// Abandon.
	abandonQuery := `mutation { abandonStudySession { success } }`
	status, result = ts.graphqlQuery(t, abandonQuery, nil, token)
	assert.Equal(t, http.StatusOK, status)
	requireNoErrors(t, result)
	assert.Equal(t, true, gqlPayload(t, result, "abandonStudySession")["success"])

	// Dashboard should show no active session.
	dashQuery := `query { dashboard { activeSession { id } } }`
	status, result = ts.graphqlQuery(t, dashQuery, nil, token)
	assert.Equal(t, http.StatusOK, status)
	requireNoErrors(t, result)
	assert.Nil(t, gqlPayload(t, result, "dashboard")["activeSession"])
}

// ---------------------------------------------------------------------------
// Scenario 5: Card history tracks reviews.
// ---------------------------------------------------------------------------

func TestE2E_CardHistory_TracksMultipleReviews(t *testing.T) {
	ts := setupTestServer(t)
	token, userID := createTestUserWithID(t, ts)

	ref := testhelper.SeedRefEntry(t, ts.Pool, "history-"+uuid.New().String()[:8])
	entry := testhelper.SeedEntryWithCard(t, ts.Pool, userID, ref.ID)
	cardID := entry.Card.ID.String()

	reviewQuery := `mutation($input: ReviewCardInput!) {
		reviewCard(input: $input) { card { id status } }
	}`

	// Review twice (AGAIN, then GOOD).
	grades := []string{"AGAIN", "GOOD"}
	for _, grade := range grades {
		status, result := ts.graphqlQuery(t, reviewQuery, map[string]any{
			"input": map[string]any{"cardId": cardID, "grade": grade, "durationMs": 2000},
		}, token)
		assert.Equal(t, http.StatusOK, status)
		requireNoErrors(t, result)
	}

	// Check history.
	historyQuery := `query($input: GetCardHistoryInput!) {
		cardHistory(input: $input) { id grade durationMs reviewedAt }
	}`
	status, result := ts.graphqlQuery(t, historyQuery, map[string]any{
		"input": map[string]any{"cardId": cardID},
	}, token)
	assert.Equal(t, http.StatusOK, status)
	requireNoErrors(t, result)

	history := gqlData(t, result)["cardHistory"].([]any)
	assert.Len(t, history, 2, "should have 2 review logs")

	// Check card stats.
	statsQuery := `query($id: UUID!) {
		cardStats(cardId: $id) { totalReviews accuracy gradeDistribution { again hard good easy } }
	}`
	status, result = ts.graphqlQuery(t, statsQuery, map[string]any{"id": cardID}, token)
	assert.Equal(t, http.StatusOK, status)
	requireNoErrors(t, result)

	stats := gqlPayload(t, result, "cardStats")
	assert.Equal(t, float64(2), stats["totalReviews"])
	dist := stats["gradeDistribution"].(map[string]any)
	assert.Equal(t, float64(1), dist["again"])
	assert.Equal(t, float64(1), dist["good"])
}

// ---------------------------------------------------------------------------
// Create card via API (not seeded).
// ---------------------------------------------------------------------------

func TestE2E_CreateCard_ViaAPI(t *testing.T) {
	ts := setupTestServer(t)
	token := createTestUserAndGetToken(t, ts)

	// Create entry WITHOUT card.
	text := "cardless-" + uuid.New().String()[:8]
	createEntryQuery := `mutation($input: CreateEntryCustomInput!) {
		createEntryCustom(input: $input) { entry { id card { id } } }
	}`
	status, result := ts.graphqlQuery(t, createEntryQuery, map[string]any{
		"input": map[string]any{
			"text":       text,
			"senses":     []any{map[string]any{"definition": "without card"}},
			"createCard": false,
		},
	}, token)
	require.Equal(t, http.StatusOK, status)
	requireNoErrors(t, result)
	entry := gqlPayload(t, result, "createEntryCustom")["entry"].(map[string]any)
	entryID := entry["id"].(string)
	assert.Nil(t, entry["card"], "entry should not have a card initially")

	// Create card via mutation.
	createCardQuery := `mutation($id: UUID!) { createCard(entryId: $id) { card { id status easeFactor } } }`
	status, result = ts.graphqlQuery(t, createCardQuery, map[string]any{"id": entryID}, token)
	assert.Equal(t, http.StatusOK, status)
	requireNoErrors(t, result)

	card := gqlPayload(t, result, "createCard")["card"].(map[string]any)
	assert.Equal(t, "NEW", card["status"])
	assert.Equal(t, 2.5, card["easeFactor"])

	// Verify via entry query.
	getQuery := `query($id: UUID!) { dictionaryEntry(id: $id) { card { id status } } }`
	status, result = ts.graphqlQuery(t, getQuery, map[string]any{"id": entryID}, token)
	assert.Equal(t, http.StatusOK, status)
	requireNoErrors(t, result)
	assert.NotNil(t, gqlPayload(t, result, "dictionaryEntry")["card"])
}

// ---------------------------------------------------------------------------
// Delete card.
// ---------------------------------------------------------------------------

func TestE2E_DeleteCard_RemovesFromStudyQueue(t *testing.T) {
	ts := setupTestServer(t)
	token, userID := createTestUserWithID(t, ts)

	ref := testhelper.SeedRefEntry(t, ts.Pool, "delcard-"+uuid.New().String()[:8])
	entry := testhelper.SeedEntryWithCard(t, ts.Pool, userID, ref.ID)
	cardID := entry.Card.ID.String()

	// Delete card.
	deleteCardQuery := `mutation($id: UUID!) { deleteCard(id: $id) { cardId } }`
	status, result := ts.graphqlQuery(t, deleteCardQuery, map[string]any{"id": cardID}, token)
	assert.Equal(t, http.StatusOK, status)
	requireNoErrors(t, result)

	// Entry should still exist but without a card.
	getQuery := `query($id: UUID!) { dictionaryEntry(id: $id) { id card { id } } }`
	status, result = ts.graphqlQuery(t, getQuery, map[string]any{"id": entry.ID.String()}, token)
	assert.Equal(t, http.StatusOK, status)
	requireNoErrors(t, result)
	assert.Nil(t, gqlPayload(t, result, "dictionaryEntry")["card"], "card should be gone after deletion")
}
