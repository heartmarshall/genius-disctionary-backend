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
// Registration tests
// ---------------------------------------------------------------------------

func TestE2E_Auth_Register_Success(t *testing.T) {
	ts := setupTestServer(t)

	resp := restRequest(t, ts, "POST", "/auth/register", "", map[string]string{
		"email":    "reg-success@example.com",
		"username": "regsuccess",
		"password": "securepassword123",
	})
	defer resp.Body.Close()

	assert.Equal(t, http.StatusCreated, resp.StatusCode)

	var body map[string]any
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&body))

	// Verify response contains tokens and user info.
	assert.NotEmpty(t, body["accessToken"])
	assert.NotEmpty(t, body["refreshToken"])

	user, ok := body["user"].(map[string]any)
	require.True(t, ok, "expected user object in response")
	assert.NotEmpty(t, user["id"])
	assert.Equal(t, "reg-success@example.com", user["email"])
	assert.Equal(t, "regsuccess", user["username"])
	assert.Equal(t, "user", user["role"])

	// Verify the access token works with GraphQL `me` query.
	accessToken := body["accessToken"].(string)
	status, result := ts.graphqlQuery(t, `query { me { id email username } }`, nil, accessToken)
	assert.Equal(t, http.StatusOK, status)
	requireNoErrors(t, result)

	me := gqlPayload(t, result, "me")
	assert.Equal(t, "reg-success@example.com", me["email"])
	assert.Equal(t, "regsuccess", me["username"])
}

func TestE2E_Auth_Register_DuplicateEmail(t *testing.T) {
	ts := setupTestServer(t)

	body := map[string]string{
		"email":    "dup@example.com",
		"username": "dupuser1",
		"password": "securepassword123",
	}

	// First registration should succeed.
	resp := restRequest(t, ts, "POST", "/auth/register", "", body)
	resp.Body.Close()
	require.Equal(t, http.StatusCreated, resp.StatusCode)

	// Second registration with same email should fail with 409.
	body["username"] = "dupuser2" // different username, same email
	resp2 := restRequest(t, ts, "POST", "/auth/register", "", body)
	defer resp2.Body.Close()

	assert.Equal(t, http.StatusConflict, resp2.StatusCode)
}

func TestE2E_Auth_Register_InvalidInput(t *testing.T) {
	ts := setupTestServer(t)

	cases := []struct {
		name string
		body map[string]string
	}{
		{
			name: "missing email",
			body: map[string]string{
				"email":    "",
				"username": "testuser",
				"password": "securepassword123",
			},
		},
		{
			name: "short password",
			body: map[string]string{
				"email":    "short@example.com",
				"username": "testuser",
				"password": "short",
			},
		},
		{
			name: "missing username",
			body: map[string]string{
				"email":    "nouser@example.com",
				"username": "",
				"password": "securepassword123",
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			resp := restRequest(t, ts, "POST", "/auth/register", "", tc.body)
			defer resp.Body.Close()

			assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
		})
	}
}

// ---------------------------------------------------------------------------
// Login tests
// ---------------------------------------------------------------------------

func TestE2E_Auth_LoginPassword_Success(t *testing.T) {
	ts := setupTestServer(t)

	// Register first.
	resp := restRequest(t, ts, "POST", "/auth/register", "", map[string]string{
		"email":    "login-success@example.com",
		"username": "loginsuccess",
		"password": "securepassword123",
	})
	resp.Body.Close()
	require.Equal(t, http.StatusCreated, resp.StatusCode)

	// Login.
	resp2 := restRequest(t, ts, "POST", "/auth/login/password", "", map[string]string{
		"email":    "login-success@example.com",
		"password": "securepassword123",
	})
	defer resp2.Body.Close()

	assert.Equal(t, http.StatusOK, resp2.StatusCode)

	var body map[string]any
	require.NoError(t, json.NewDecoder(resp2.Body).Decode(&body))
	assert.NotEmpty(t, body["accessToken"])
	assert.NotEmpty(t, body["refreshToken"])

	user, ok := body["user"].(map[string]any)
	require.True(t, ok)
	assert.Equal(t, "login-success@example.com", user["email"])
}

func TestE2E_Auth_LoginPassword_WrongPassword(t *testing.T) {
	ts := setupTestServer(t)

	// Register first.
	resp := restRequest(t, ts, "POST", "/auth/register", "", map[string]string{
		"email":    "wrongpw@example.com",
		"username": "wrongpwuser",
		"password": "securepassword123",
	})
	resp.Body.Close()
	require.Equal(t, http.StatusCreated, resp.StatusCode)

	// Login with wrong password.
	resp2 := restRequest(t, ts, "POST", "/auth/login/password", "", map[string]string{
		"email":    "wrongpw@example.com",
		"password": "wrongpassword999",
	})
	defer resp2.Body.Close()

	assert.Equal(t, http.StatusUnauthorized, resp2.StatusCode)
}

func TestE2E_Auth_LoginPassword_NonExistentEmail(t *testing.T) {
	ts := setupTestServer(t)

	resp := restRequest(t, ts, "POST", "/auth/login/password", "", map[string]string{
		"email":    "nonexistent@example.com",
		"password": "securepassword123",
	})
	defer resp.Body.Close()

	assert.Equal(t, http.StatusUnauthorized, resp.StatusCode)
}

// ---------------------------------------------------------------------------
// Refresh tests
// ---------------------------------------------------------------------------

func TestE2E_Auth_Refresh_Success(t *testing.T) {
	ts := setupTestServer(t)

	// Register to get tokens.
	resp := restRequest(t, ts, "POST", "/auth/register", "", map[string]string{
		"email":    "refresh-ok@example.com",
		"username": "refreshok",
		"password": "securepassword123",
	})
	defer resp.Body.Close()
	require.Equal(t, http.StatusCreated, resp.StatusCode)

	var regBody map[string]any
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&regBody))
	oldRefresh := regBody["refreshToken"].(string)

	// Refresh.
	resp2 := restRequest(t, ts, "POST", "/auth/refresh", "", map[string]string{
		"refreshToken": oldRefresh,
	})
	defer resp2.Body.Close()

	assert.Equal(t, http.StatusOK, resp2.StatusCode)

	var refreshBody map[string]any
	require.NoError(t, json.NewDecoder(resp2.Body).Decode(&refreshBody))
	newAccess := refreshBody["accessToken"].(string)
	newRefresh := refreshBody["refreshToken"].(string)

	assert.NotEmpty(t, newAccess)
	assert.NotEmpty(t, newRefresh)
	// Refresh token must be rotated (different from old).
	assert.NotEqual(t, oldRefresh, newRefresh)
}

func TestE2E_Auth_Refresh_OldTokenInvalidated(t *testing.T) {
	ts := setupTestServer(t)

	// Register.
	resp := restRequest(t, ts, "POST", "/auth/register", "", map[string]string{
		"email":    "refresh-old@example.com",
		"username": "refreshold",
		"password": "securepassword123",
	})
	defer resp.Body.Close()
	require.Equal(t, http.StatusCreated, resp.StatusCode)

	var regBody map[string]any
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&regBody))
	oldRefresh := regBody["refreshToken"].(string)

	// Refresh once — rotates the token.
	resp2 := restRequest(t, ts, "POST", "/auth/refresh", "", map[string]string{
		"refreshToken": oldRefresh,
	})
	resp2.Body.Close()
	require.Equal(t, http.StatusOK, resp2.StatusCode)

	// Try using the OLD refresh token again — should fail.
	resp3 := restRequest(t, ts, "POST", "/auth/refresh", "", map[string]string{
		"refreshToken": oldRefresh,
	})
	defer resp3.Body.Close()

	assert.Equal(t, http.StatusUnauthorized, resp3.StatusCode)
}

func TestE2E_Auth_Refresh_InvalidToken(t *testing.T) {
	ts := setupTestServer(t)

	resp := restRequest(t, ts, "POST", "/auth/refresh", "", map[string]string{
		"refreshToken": "completely-garbage-token",
	})
	defer resp.Body.Close()

	assert.Equal(t, http.StatusUnauthorized, resp.StatusCode)
}

// ---------------------------------------------------------------------------
// Logout tests
// ---------------------------------------------------------------------------

func TestE2E_Auth_Logout_Success(t *testing.T) {
	ts := setupTestServer(t)

	// Register.
	resp := restRequest(t, ts, "POST", "/auth/register", "", map[string]string{
		"email":    "logout-ok@example.com",
		"username": "logoutok",
		"password": "securepassword123",
	})
	defer resp.Body.Close()
	require.Equal(t, http.StatusCreated, resp.StatusCode)

	var regBody map[string]any
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&regBody))
	accessToken := regBody["accessToken"].(string)
	refreshToken := regBody["refreshToken"].(string)

	// Logout with access token.
	resp2 := restRequest(t, ts, "POST", "/auth/logout", accessToken, nil)
	defer resp2.Body.Close()

	assert.Equal(t, http.StatusOK, resp2.StatusCode)

	// Try to refresh after logout — should fail.
	resp3 := restRequest(t, ts, "POST", "/auth/refresh", "", map[string]string{
		"refreshToken": refreshToken,
	})
	defer resp3.Body.Close()

	assert.Equal(t, http.StatusUnauthorized, resp3.StatusCode)
}

func TestE2E_Auth_Logout_NoToken(t *testing.T) {
	ts := setupTestServer(t)

	resp := restRequest(t, ts, "POST", "/auth/logout", "", nil)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusUnauthorized, resp.StatusCode)
}

// ---------------------------------------------------------------------------
// Full lifecycle test
// ---------------------------------------------------------------------------

func TestE2E_Auth_FullLifecycle(t *testing.T) {
	ts := setupTestServer(t)

	// 1. Register.
	resp := restRequest(t, ts, "POST", "/auth/register", "", map[string]string{
		"email":    "lifecycle@example.com",
		"username": "lifecycle",
		"password": "securepassword123",
	})
	defer resp.Body.Close()
	require.Equal(t, http.StatusCreated, resp.StatusCode)

	var regBody map[string]any
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&regBody))
	regAccess := regBody["accessToken"].(string)

	// 2. Login.
	resp2 := restRequest(t, ts, "POST", "/auth/login/password", "", map[string]string{
		"email":    "lifecycle@example.com",
		"password": "securepassword123",
	})
	defer resp2.Body.Close()
	require.Equal(t, http.StatusOK, resp2.StatusCode)

	var loginBody map[string]any
	require.NoError(t, json.NewDecoder(resp2.Body).Decode(&loginBody))
	loginRefresh := loginBody["refreshToken"].(string)

	// 3. Refresh.
	resp3 := restRequest(t, ts, "POST", "/auth/refresh", "", map[string]string{
		"refreshToken": loginRefresh,
	})
	defer resp3.Body.Close()
	require.Equal(t, http.StatusOK, resp3.StatusCode)

	var refreshBody map[string]any
	require.NoError(t, json.NewDecoder(resp3.Body).Decode(&refreshBody))
	newAccess := refreshBody["accessToken"].(string)
	newRefresh := refreshBody["refreshToken"].(string)

	// 4. Use new access token with GraphQL.
	status, result := ts.graphqlQuery(t, `query { me { id email } }`, nil, newAccess)
	assert.Equal(t, http.StatusOK, status)
	requireNoErrors(t, result)
	me := gqlPayload(t, result, "me")
	assert.Equal(t, "lifecycle@example.com", me["email"])

	// 5. Logout with new access token.
	resp4 := restRequest(t, ts, "POST", "/auth/logout", newAccess, nil)
	defer resp4.Body.Close()
	require.Equal(t, http.StatusOK, resp4.StatusCode)

	// 6. Refresh with the new refresh token should fail (logged out).
	resp5 := restRequest(t, ts, "POST", "/auth/refresh", "", map[string]string{
		"refreshToken": newRefresh,
	})
	defer resp5.Body.Close()
	assert.Equal(t, http.StatusUnauthorized, resp5.StatusCode)

	// Verify the registration access token also can't be used to check
	// that the user is truly logged out (original session is separate,
	// but the reg access token itself is still a valid JWT until expiry).
	_ = regAccess
}
