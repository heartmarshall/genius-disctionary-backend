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
// Scenario 11: Dashboard reflects card statuses correctly.
// ---------------------------------------------------------------------------

func TestE2E_Dashboard_ReflectsCardStatuses(t *testing.T) {
	ts := setupTestServer(t)
	token, userID := createTestUserWithID(t, ts)

	// Seed 3 entries with NEW cards.
	for i := 0; i < 3; i++ {
		ref := testhelper.SeedRefEntry(t, ts.Pool, "dash-"+uuid.New().String()[:8])
		testhelper.SeedEntryWithCard(t, ts.Pool, userID, ref.ID)
	}

	dashQuery := `query {
		dashboard {
			dueCount newCount reviewedToday streak
			statusCounts { new learning review mastered }
			overdueCount activeSession { id }
		}
	}`
	status, result := ts.graphqlQuery(t, dashQuery, nil, token)
	assert.Equal(t, http.StatusOK, status)
	requireNoErrors(t, result)

	dash := gqlPayload(t, result, "dashboard")
	assert.GreaterOrEqual(t, dash["newCount"].(float64), float64(3), "should have at least 3 new cards")
	assert.Equal(t, float64(0), dash["reviewedToday"], "no reviews yet")

	statusCounts := dash["statusCounts"].(map[string]any)
	assert.GreaterOrEqual(t, statusCounts["new"].(float64), float64(3))
	assert.Nil(t, dash["activeSession"], "no active session initially")
}

// ---------------------------------------------------------------------------
// Scenario 11: Dashboard tracks reviews today after a review.
// ---------------------------------------------------------------------------

func TestE2E_Dashboard_TracksReviewsToday(t *testing.T) {
	ts := setupTestServer(t)
	token, userID := createTestUserWithID(t, ts)

	// Seed an entry with card.
	ref := testhelper.SeedRefEntry(t, ts.Pool, "dashrev-"+uuid.New().String()[:8])
	entry := testhelper.SeedEntryWithCard(t, ts.Pool, userID, ref.ID)
	cardID := entry.Card.ID.String()

	// Review the card.
	reviewQuery := `mutation($input: ReviewCardInput!) {
		reviewCard(input: $input) { card { id } }
	}`
	status, result := ts.graphqlQuery(t, reviewQuery, map[string]any{
		"input": map[string]any{"cardId": cardID, "grade": "GOOD"},
	}, token)
	require.Equal(t, http.StatusOK, status)
	requireNoErrors(t, result)

	// Check dashboard.
	dashQuery := `query { dashboard { reviewedToday } }`
	status, result = ts.graphqlQuery(t, dashQuery, nil, token)
	assert.Equal(t, http.StatusOK, status)
	requireNoErrors(t, result)
	assert.GreaterOrEqual(t, gqlPayload(t, result, "dashboard")["reviewedToday"].(float64), float64(1))
}

// ---------------------------------------------------------------------------
// Scenario 11: Dashboard shows active session.
// ---------------------------------------------------------------------------

func TestE2E_Dashboard_ShowsActiveSession(t *testing.T) {
	ts := setupTestServer(t)
	token := createTestUserAndGetToken(t, ts)

	// Start a session.
	startQuery := `mutation { startStudySession { session { id } } }`
	status, result := ts.graphqlQuery(t, startQuery, nil, token)
	require.Equal(t, http.StatusOK, status)
	requireNoErrors(t, result)
	sessionID := gqlPayload(t, result, "startStudySession")["session"].(map[string]any)["id"].(string)

	// Dashboard should show it.
	dashQuery := `query { dashboard { activeSession { id status } } }`
	status, result = ts.graphqlQuery(t, dashQuery, nil, token)
	assert.Equal(t, http.StatusOK, status)
	requireNoErrors(t, result)

	activeSession := gqlPayload(t, result, "dashboard")["activeSession"]
	require.NotNil(t, activeSession)
	assert.Equal(t, sessionID, activeSession.(map[string]any)["id"])
	assert.Equal(t, "ACTIVE", activeSession.(map[string]any)["status"])
}

// ---------------------------------------------------------------------------
// Scenario 13: Update user settings and verify persistence.
// ---------------------------------------------------------------------------

func TestE2E_UpdateSettings_PersistsChanges(t *testing.T) {
	ts := setupTestServer(t)
	token := createTestUserAndGetToken(t, ts)

	// Read current settings.
	meQuery := `query { me { id email settings { newCardsPerDay reviewsPerDay maxIntervalDays timezone } } }`
	status, result := ts.graphqlQuery(t, meQuery, nil, token)
	assert.Equal(t, http.StatusOK, status)
	requireNoErrors(t, result)

	me := gqlPayload(t, result, "me")
	assert.NotEmpty(t, me["email"])
	settings := me["settings"].(map[string]any)
	assert.Equal(t, float64(20), settings["newCardsPerDay"])
	assert.Equal(t, float64(200), settings["reviewsPerDay"])
	assert.Equal(t, "UTC", settings["timezone"])

	// Update settings.
	updateQuery := `mutation($input: UpdateSettingsInput!) {
		updateSettings(input: $input) { settings { newCardsPerDay reviewsPerDay maxIntervalDays timezone } }
	}`
	status, result = ts.graphqlQuery(t, updateQuery, map[string]any{
		"input": map[string]any{
			"newCardsPerDay":  10,
			"reviewsPerDay":  50,
			"maxIntervalDays": 180,
			"timezone":       "Europe/Moscow",
		},
	}, token)
	assert.Equal(t, http.StatusOK, status)
	requireNoErrors(t, result)

	updated := gqlPayload(t, result, "updateSettings")["settings"].(map[string]any)
	assert.Equal(t, float64(10), updated["newCardsPerDay"])
	assert.Equal(t, float64(50), updated["reviewsPerDay"])
	assert.Equal(t, float64(180), updated["maxIntervalDays"])
	assert.Equal(t, "Europe/Moscow", updated["timezone"])

	// Verify via separate me query.
	status, result = ts.graphqlQuery(t, meQuery, nil, token)
	assert.Equal(t, http.StatusOK, status)
	requireNoErrors(t, result)
	verifiedSettings := gqlPayload(t, result, "me")["settings"].(map[string]any)
	assert.Equal(t, float64(10), verifiedSettings["newCardsPerDay"])
	assert.Equal(t, "Europe/Moscow", verifiedSettings["timezone"])
}

// ---------------------------------------------------------------------------
// Scenario 13: Partial settings update (only change one field).
// ---------------------------------------------------------------------------

func TestE2E_UpdateSettings_PartialUpdate(t *testing.T) {
	ts := setupTestServer(t)
	token := createTestUserAndGetToken(t, ts)

	// Update only newCardsPerDay.
	updateQuery := `mutation($input: UpdateSettingsInput!) {
		updateSettings(input: $input) { settings { newCardsPerDay reviewsPerDay timezone } }
	}`
	status, result := ts.graphqlQuery(t, updateQuery, map[string]any{
		"input": map[string]any{"newCardsPerDay": 5},
	}, token)
	assert.Equal(t, http.StatusOK, status)
	requireNoErrors(t, result)

	settings := gqlPayload(t, result, "updateSettings")["settings"].(map[string]any)
	assert.Equal(t, float64(5), settings["newCardsPerDay"])
	// Other fields should remain at defaults.
	assert.Equal(t, float64(200), settings["reviewsPerDay"])
	assert.Equal(t, "UTC", settings["timezone"])
}
