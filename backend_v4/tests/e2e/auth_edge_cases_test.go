//go:build e2e

package e2e_test

import (
	"encoding/json"
	"net/http"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ---------------------------------------------------------------------------
// Helper: sendRaw sends a raw string body to the given REST endpoint.
// Unlike restRequest, it does NOT json.Marshal the body, allowing callers
// to send malformed JSON or other non-standard payloads.
// ---------------------------------------------------------------------------

func sendRaw(t *testing.T, ts *testServer, method, path string, bodyStr string) *http.Response {
	t.Helper()

	req, err := http.NewRequest(method, ts.URL+path, strings.NewReader(bodyStr))
	require.NoError(t, err)
	req.Header.Set("Content-Type", "application/json")

	resp, err := ts.Client.Do(req)
	require.NoError(t, err)
	return resp
}

// decodeBody reads and decodes the JSON response body into a map.
func decodeBody(t *testing.T, resp *http.Response) map[string]any {
	t.Helper()

	var body map[string]any
	err := json.NewDecoder(resp.Body).Decode(&body)
	require.NoError(t, err, "response body should be valid JSON")
	return body
}

// ---------------------------------------------------------------------------
// Register edge cases
// ---------------------------------------------------------------------------

// TestE2E_Auth_Register_MalformedJSON verifies that sending syntactically
// invalid JSON to the register endpoint returns 400, not 500.
func TestE2E_Auth_Register_MalformedJSON(t *testing.T) {
	ts := setupTestServer(t)

	resp := sendRaw(t, ts, "POST", "/auth/register", `{invalid json`)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusBadRequest, resp.StatusCode)

	body := decodeBody(t, resp)
	assert.Contains(t, body["error"], "invalid request body")
}

// TestE2E_Auth_Register_EmptyBody verifies that sending an empty body
// to the register endpoint returns 400. The JSON decoder treats an empty
// reader as an error (EOF), so the handler returns 400 before validation.
func TestE2E_Auth_Register_EmptyBody(t *testing.T) {
	ts := setupTestServer(t)

	resp := sendRaw(t, ts, "POST", "/auth/register", "")
	defer resp.Body.Close()

	assert.Equal(t, http.StatusBadRequest, resp.StatusCode)

	body := decodeBody(t, resp)
	assert.NotEmpty(t, body["error"])
}

// ---------------------------------------------------------------------------
// Login edge cases
// ---------------------------------------------------------------------------

// TestE2E_Auth_LoginPassword_MalformedJSON verifies that sending
// syntactically invalid JSON to the login endpoint returns 400.
func TestE2E_Auth_LoginPassword_MalformedJSON(t *testing.T) {
	ts := setupTestServer(t)

	resp := sendRaw(t, ts, "POST", "/auth/login/password", `{not valid`)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusBadRequest, resp.StatusCode)

	body := decodeBody(t, resp)
	assert.Contains(t, body["error"], "invalid request body")
}

// TestE2E_Auth_LoginPassword_EmptyBody verifies that sending an empty body
// to the login endpoint returns 401. An empty body decodes to zero-value
// fields, and the service-layer validation rejects the empty email/password
// with ErrValidation, which the handler maps to a non-200 status.
func TestE2E_Auth_LoginPassword_EmptyBody(t *testing.T) {
	ts := setupTestServer(t)

	// Empty string causes json.Decoder.Decode to fail with EOF → 400.
	resp := sendRaw(t, ts, "POST", "/auth/login/password", "")
	defer resp.Body.Close()

	// The handler's json.Decode fails on empty body → 400 "invalid request body".
	assert.Equal(t, http.StatusBadRequest, resp.StatusCode)

	body := decodeBody(t, resp)
	assert.NotEmpty(t, body["error"])
}

// ---------------------------------------------------------------------------
// Refresh edge cases
// ---------------------------------------------------------------------------

// TestE2E_Auth_Refresh_MalformedJSON verifies that sending syntactically
// invalid JSON to the refresh endpoint returns 400.
func TestE2E_Auth_Refresh_MalformedJSON(t *testing.T) {
	ts := setupTestServer(t)

	resp := sendRaw(t, ts, "POST", "/auth/refresh", `{"refreshToken":}`)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusBadRequest, resp.StatusCode)

	body := decodeBody(t, resp)
	assert.Contains(t, body["error"], "invalid request body")
}

// TestE2E_Auth_Refresh_EmptyBody verifies that sending an empty body to
// the refresh endpoint returns 401. An empty body causes json.Decode to
// fail with EOF, so the handler returns 400.
func TestE2E_Auth_Refresh_EmptyBody(t *testing.T) {
	ts := setupTestServer(t)

	resp := sendRaw(t, ts, "POST", "/auth/refresh", "")
	defer resp.Body.Close()

	// Empty body → json.Decode EOF → 400.
	assert.Equal(t, http.StatusBadRequest, resp.StatusCode)

	body := decodeBody(t, resp)
	assert.NotEmpty(t, body["error"])
}

// ---------------------------------------------------------------------------
// Unicode and encoding edge cases
// ---------------------------------------------------------------------------

// TestE2E_Auth_Register_UnicodeUsername verifies that registering with
// a non-ASCII (Unicode) name as the username works correctly, as long as
// the username meets the length constraints.
func TestE2E_Auth_Register_UnicodeUsername(t *testing.T) {
	ts := setupTestServer(t)

	reqBody := map[string]string{
		"email":    "unicode-user@example.com",
		"username": "Тест Юзер",
		"password": "securepassword123",
	}

	resp := restRequest(t, ts, "POST", "/auth/register", "", reqBody)
	defer resp.Body.Close()

	require.Equal(t, http.StatusCreated, resp.StatusCode)

	body := decodeBody(t, resp)
	assert.NotEmpty(t, body["accessToken"])
	assert.NotEmpty(t, body["refreshToken"])

	user, ok := body["user"].(map[string]any)
	require.True(t, ok, "expected user object in response")
	assert.Equal(t, "unicode-user@example.com", user["email"])
	assert.Equal(t, "Тест Юзер", user["username"])

	// Verify the token works with a GraphQL query.
	accessToken := body["accessToken"].(string)
	status, result := ts.graphqlQuery(t, `query { me { id email username } }`, nil, accessToken)
	assert.Equal(t, http.StatusOK, status)
	requireNoErrors(t, result)

	me := gqlPayload(t, result, "me")
	assert.Equal(t, "Тест Юзер", me["username"])
}

// ---------------------------------------------------------------------------
// Email case sensitivity
// ---------------------------------------------------------------------------

// TestE2E_Auth_Register_EmailCaseSensitivity verifies that email
// normalization prevents duplicate accounts from different casings.
// Emails are lowercased before storage, so "Test@Example.COM" and
// "test@example.com" are treated as the same email.
func TestE2E_Auth_Register_EmailCaseSensitivity(t *testing.T) {
	ts := setupTestServer(t)

	// Register with mixed-case email.
	resp1 := restRequest(t, ts, "POST", "/auth/register", "", map[string]string{
		"email":    "Test@Example.COM",
		"username": "caseuser1",
		"password": "securepassword123",
	})
	defer resp1.Body.Close()
	require.Equal(t, http.StatusCreated, resp1.StatusCode)

	// Verify the stored email is lowercase.
	body1 := decodeBody(t, resp1)
	user1 := body1["user"].(map[string]any)
	assert.Equal(t, "test@example.com", user1["email"], "email should be normalized to lowercase")

	// Register with the same email in lowercase — should conflict.
	resp2 := restRequest(t, ts, "POST", "/auth/register", "", map[string]string{
		"email":    "test@example.com",
		"username": "caseuser2",
		"password": "securepassword123",
	})
	defer resp2.Body.Close()

	assert.Equal(t, http.StatusConflict, resp2.StatusCode,
		"same email with different casing should be treated as duplicate")
}

// ---------------------------------------------------------------------------
// Password boundary conditions
// ---------------------------------------------------------------------------

// TestE2E_Auth_Register_LongPassword verifies that a password at the
// bcrypt maximum (72 bytes) is accepted. The input validation allows
// passwords up to 72 characters.
func TestE2E_Auth_Register_LongPassword(t *testing.T) {
	ts := setupTestServer(t)

	// 72-character password — exactly at the bcrypt byte limit.
	password72 := strings.Repeat("A", 72)

	resp := restRequest(t, ts, "POST", "/auth/register", "", map[string]string{
		"email":    "long-pw@example.com",
		"username": "longpwuser",
		"password": password72,
	})
	defer resp.Body.Close()

	require.Equal(t, http.StatusCreated, resp.StatusCode)

	body := decodeBody(t, resp)
	assert.NotEmpty(t, body["accessToken"])
	assert.NotEmpty(t, body["refreshToken"])

	// Verify the user can log in with the same 72-char password.
	resp2 := restRequest(t, ts, "POST", "/auth/login/password", "", map[string]string{
		"email":    "long-pw@example.com",
		"password": password72,
	})
	defer resp2.Body.Close()

	assert.Equal(t, http.StatusOK, resp2.StatusCode)
}

// ---------------------------------------------------------------------------
// Email validation boundary
// ---------------------------------------------------------------------------

// TestE2E_Auth_Register_TooLongEmail verifies that an email exceeding
// 254 characters is rejected with 400. The RegisterInput.Validate method
// checks len(email) > 254.
func TestE2E_Auth_Register_TooLongEmail(t *testing.T) {
	ts := setupTestServer(t)

	// Build an email that exceeds 254 characters: local@domain.
	// 250 chars of local part + "@x.com" = 256 total.
	longLocal := strings.Repeat("a", 250)
	longEmail := longLocal + "@x.com"
	require.Greater(t, len(longEmail), 254, "test email must exceed 254 chars")

	resp := restRequest(t, ts, "POST", "/auth/register", "", map[string]string{
		"email":    longEmail,
		"username": "longemailuser",
		"password": "securepassword123",
	})
	defer resp.Body.Close()

	assert.Equal(t, http.StatusBadRequest, resp.StatusCode)

	body := decodeBody(t, resp)
	errorMsg, ok := body["error"].(string)
	require.True(t, ok, "expected error string in response")
	assert.Contains(t, errorMsg, "email")
}
