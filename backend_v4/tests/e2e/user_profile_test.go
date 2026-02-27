//go:build e2e

package e2e_test

import (
	"encoding/json"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ---------------------------------------------------------------------------
// Test 1: me query returns full user profile with all fields.
// ---------------------------------------------------------------------------

func TestE2E_Me_ReturnsFullProfile(t *testing.T) {
	ts := setupTestServer(t)
	token := createTestUserAndGetToken(t, ts)

	meQuery := `query {
		me {
			id email username name role createdAt
			settings { newCardsPerDay reviewsPerDay maxIntervalDays timezone }
		}
	}`

	status, result := ts.graphqlQuery(t, meQuery, nil, token)
	assert.Equal(t, http.StatusOK, status)
	requireNoErrors(t, result)

	me := gqlPayload(t, result, "me")

	// Core user fields.
	assert.NotEmpty(t, me["id"], "id must be present")
	assert.NotEmpty(t, me["email"], "email must be present")
	assert.NotEmpty(t, me["username"], "username must be present")
	assert.Equal(t, "Test User", me["name"], "name should match seeded value")
	assert.Equal(t, "user", me["role"], "default role should be 'user'")
	assert.NotEmpty(t, me["createdAt"], "createdAt must be present")

	// Settings sub-object.
	settings, ok := me["settings"].(map[string]any)
	require.True(t, ok, "expected settings object")
	assert.Equal(t, float64(20), settings["newCardsPerDay"])
	assert.Equal(t, float64(200), settings["reviewsPerDay"])
	assert.Equal(t, float64(365), settings["maxIntervalDays"])
	assert.Equal(t, "UTC", settings["timezone"])
}

// ---------------------------------------------------------------------------
// Test 2: me query without token returns UNAUTHENTICATED.
// ---------------------------------------------------------------------------

func TestE2E_Me_Unauthorized(t *testing.T) {
	ts := setupTestServer(t)

	meQuery := `query { me { id email } }`

	status, result := ts.graphqlQuery(t, meQuery, nil, "")
	assert.Equal(t, http.StatusOK, status)
	assert.Equal(t, "UNAUTHENTICATED", gqlErrorCode(t, result))
}

// ---------------------------------------------------------------------------
// Test 3: updateProfile mutation updates name and avatarUrl.
// ---------------------------------------------------------------------------

func TestE2E_UpdateProfile_Success(t *testing.T) {
	ts := setupTestServer(t)
	token := createTestUserAndGetToken(t, ts)

	updateQuery := `mutation($input: UpdateProfileInput!) {
		updateProfile(input: $input) {
			user { id name avatarUrl }
		}
	}`
	status, result := ts.graphqlQuery(t, updateQuery, map[string]any{
		"input": map[string]any{
			"name":      "Updated Name",
			"avatarUrl": "https://example.com/avatar.png",
		},
	}, token)
	assert.Equal(t, http.StatusOK, status)
	requireNoErrors(t, result)

	user := gqlPayload(t, result, "updateProfile")["user"].(map[string]any)
	assert.Equal(t, "Updated Name", user["name"])
	assert.Equal(t, "https://example.com/avatar.png", user["avatarUrl"])

	// Verify via separate me query.
	meQuery := `query { me { name avatarUrl } }`
	status, result = ts.graphqlQuery(t, meQuery, nil, token)
	assert.Equal(t, http.StatusOK, status)
	requireNoErrors(t, result)

	me := gqlPayload(t, result, "me")
	assert.Equal(t, "Updated Name", me["name"])
	assert.Equal(t, "https://example.com/avatar.png", me["avatarUrl"])
}

// ---------------------------------------------------------------------------
// Test 4: updateProfile with name only leaves avatarUrl null.
// ---------------------------------------------------------------------------

func TestE2E_UpdateProfile_ClearsAvatar(t *testing.T) {
	ts := setupTestServer(t)
	token := createTestUserAndGetToken(t, ts)

	// First set an avatar.
	updateQuery := `mutation($input: UpdateProfileInput!) {
		updateProfile(input: $input) {
			user { id name avatarUrl }
		}
	}`
	status, result := ts.graphqlQuery(t, updateQuery, map[string]any{
		"input": map[string]any{
			"name":      "With Avatar",
			"avatarUrl": "https://example.com/pic.png",
		},
	}, token)
	require.Equal(t, http.StatusOK, status)
	requireNoErrors(t, result)

	// Now update with name only (no avatarUrl).
	status, result = ts.graphqlQuery(t, updateQuery, map[string]any{
		"input": map[string]any{
			"name": "Without Avatar",
		},
	}, token)
	assert.Equal(t, http.StatusOK, status)
	requireNoErrors(t, result)

	user := gqlPayload(t, result, "updateProfile")["user"].(map[string]any)
	assert.Equal(t, "Without Avatar", user["name"])
	assert.Nil(t, user["avatarUrl"], "avatarUrl should be null when not provided in input")
}

// ---------------------------------------------------------------------------
// Test 5: updateProfile without token returns UNAUTHENTICATED.
// ---------------------------------------------------------------------------

func TestE2E_UpdateProfile_Unauthorized(t *testing.T) {
	ts := setupTestServer(t)

	updateQuery := `mutation($input: UpdateProfileInput!) {
		updateProfile(input: $input) {
			user { id name }
		}
	}`
	status, result := ts.graphqlQuery(t, updateQuery, map[string]any{
		"input": map[string]any{
			"name": "Hacker",
		},
	}, "")
	assert.Equal(t, http.StatusOK, status)
	assert.Equal(t, "UNAUTHENTICATED", gqlErrorCode(t, result))
}

// ---------------------------------------------------------------------------
// Test 6: updateProfile with empty name returns VALIDATION error.
// ---------------------------------------------------------------------------

func TestE2E_UpdateProfile_ValidationError(t *testing.T) {
	ts := setupTestServer(t)
	token := createTestUserAndGetToken(t, ts)

	updateQuery := `mutation($input: UpdateProfileInput!) {
		updateProfile(input: $input) {
			user { id name }
		}
	}`
	status, result := ts.graphqlQuery(t, updateQuery, map[string]any{
		"input": map[string]any{
			"name": "",
		},
	}, token)
	assert.Equal(t, http.StatusOK, status)
	assert.Equal(t, "VALIDATION", gqlErrorCode(t, result))
}

// ---------------------------------------------------------------------------
// Test 7: updateSettings with invalid timezone returns VALIDATION error.
// ---------------------------------------------------------------------------

func TestE2E_UpdateSettings_InvalidTimezone(t *testing.T) {
	ts := setupTestServer(t)
	token := createTestUserAndGetToken(t, ts)

	updateQuery := `mutation($input: UpdateSettingsInput!) {
		updateSettings(input: $input) { settings { timezone } }
	}`
	status, result := ts.graphqlQuery(t, updateQuery, map[string]any{
		"input": map[string]any{
			"timezone": "Invalid/Zone",
		},
	}, token)
	assert.Equal(t, http.StatusOK, status)
	assert.Equal(t, "VALIDATION", gqlErrorCode(t, result))
}

// ---------------------------------------------------------------------------
// Test 8: updateSettings boundary value validation.
// ---------------------------------------------------------------------------

func TestE2E_UpdateSettings_BoundaryValues(t *testing.T) {
	ts := setupTestServer(t)
	token := createTestUserAndGetToken(t, ts)

	updateQuery := `mutation($input: UpdateSettingsInput!) {
		updateSettings(input: $input) {
			settings { newCardsPerDay reviewsPerDay maxIntervalDays timezone }
		}
	}`

	// Sub-test: min and max valid boundaries should succeed.
	t.Run("valid_min_max", func(t *testing.T) {
		status, result := ts.graphqlQuery(t, updateQuery, map[string]any{
			"input": map[string]any{
				"newCardsPerDay": 1,
				"reviewsPerDay": 9999,
			},
		}, token)
		assert.Equal(t, http.StatusOK, status)
		requireNoErrors(t, result)

		settings := gqlPayload(t, result, "updateSettings")["settings"].(map[string]any)
		assert.Equal(t, float64(1), settings["newCardsPerDay"])
		assert.Equal(t, float64(9999), settings["reviewsPerDay"])
	})

	// Sub-test: newCardsPerDay = 0 (below min) should fail.
	t.Run("below_min_new_cards", func(t *testing.T) {
		status, result := ts.graphqlQuery(t, updateQuery, map[string]any{
			"input": map[string]any{
				"newCardsPerDay": 0,
			},
		}, token)
		assert.Equal(t, http.StatusOK, status)
		assert.Equal(t, "VALIDATION", gqlErrorCode(t, result))
	})

	// Sub-test: newCardsPerDay = 1000 (above max of 999) should fail.
	t.Run("above_max_new_cards", func(t *testing.T) {
		status, result := ts.graphqlQuery(t, updateQuery, map[string]any{
			"input": map[string]any{
				"newCardsPerDay": 1000,
			},
		}, token)
		assert.Equal(t, http.StatusOK, status)
		assert.Equal(t, "VALIDATION", gqlErrorCode(t, result))
	})
}

// ---------------------------------------------------------------------------
// Test 9: updateSettings without token returns UNAUTHENTICATED.
// ---------------------------------------------------------------------------

func TestE2E_UpdateSettings_Unauthorized(t *testing.T) {
	ts := setupTestServer(t)

	updateQuery := `mutation($input: UpdateSettingsInput!) {
		updateSettings(input: $input) { settings { newCardsPerDay } }
	}`
	status, result := ts.graphqlQuery(t, updateQuery, map[string]any{
		"input": map[string]any{
			"newCardsPerDay": 10,
		},
	}, "")
	assert.Equal(t, http.StatusOK, status)
	assert.Equal(t, "UNAUTHENTICATED", gqlErrorCode(t, result))
}

// ---------------------------------------------------------------------------
// Test 10: freshly registered user has expected default settings.
// ---------------------------------------------------------------------------

func TestE2E_UserSettings_DefaultValues(t *testing.T) {
	ts := setupTestServer(t)

	// Register a new user via the REST endpoint.
	resp := restRequest(t, ts, "POST", "/auth/register", "", map[string]string{
		"email":    "defaults@example.com",
		"username": "defaultsuser",
		"password": "securepassword123",
	})
	defer resp.Body.Close()
	require.Equal(t, http.StatusCreated, resp.StatusCode)

	var regBody map[string]any
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&regBody))
	accessToken, ok := regBody["accessToken"].(string)
	require.True(t, ok, "expected accessToken in register response")

	// Query the user's settings via GraphQL.
	meQuery := `query {
		me {
			settings { newCardsPerDay reviewsPerDay maxIntervalDays timezone }
		}
	}`
	status, result := ts.graphqlQuery(t, meQuery, nil, accessToken)
	assert.Equal(t, http.StatusOK, status)
	requireNoErrors(t, result)

	settings, ok := gqlPayload(t, result, "me")["settings"].(map[string]any)
	require.True(t, ok, "expected settings object")

	assert.Equal(t, float64(20), settings["newCardsPerDay"], "default newCardsPerDay should be 20")
	assert.Equal(t, float64(200), settings["reviewsPerDay"], "default reviewsPerDay should be 200")
	assert.Equal(t, float64(365), settings["maxIntervalDays"], "default maxIntervalDays should be 365")
	assert.Equal(t, "UTC", settings["timezone"], "default timezone should be UTC")
}
