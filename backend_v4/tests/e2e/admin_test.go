//go:build e2e

package e2e_test

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestE2E_Admin_RegularUserForbidden verifies that regular users get 403 on
// admin REST endpoints.
func TestE2E_Admin_RegularUserForbidden(t *testing.T) {
	ts := setupTestServer(t)
	token := createTestUserAndGetToken(t, ts)

	endpoints := []struct {
		method string
		path   string
	}{
		{"GET", "/admin/users"},
		{"GET", "/admin/enrichment/stats"},
		{"GET", "/admin/enrichment/queue"},
		{"POST", "/admin/enrichment/retry"},
		{"POST", "/admin/enrichment/reset-processing"},
	}

	for _, ep := range endpoints {
		t.Run(ep.method+" "+ep.path, func(t *testing.T) {
			req, err := http.NewRequest(ep.method, ts.URL+ep.path, nil)
			require.NoError(t, err)
			req.Header.Set("Authorization", "Bearer "+token)

			resp, err := ts.Client.Do(req)
			require.NoError(t, err)
			defer resp.Body.Close()

			assert.Equal(t, http.StatusForbidden, resp.StatusCode)
		})
	}
}

// TestE2E_Admin_ListUsers verifies admin can list users via REST.
func TestE2E_Admin_ListUsers(t *testing.T) {
	ts := setupTestServer(t)
	_, userID := createTestUserWithID(t, ts)

	// Promote to admin directly in DB.
	promoteToAdmin(t, ts, userID)
	adminToken := generateAdminToken(t, ts, userID)

	// Create a second user so there's more than one.
	createTestUserAndGetToken(t, ts)

	req, err := http.NewRequest("GET", ts.URL+"/admin/users?limit=50&offset=0", nil)
	require.NoError(t, err)
	req.Header.Set("Authorization", "Bearer "+adminToken)

	resp, err := ts.Client.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusOK, resp.StatusCode)

	var body map[string]any
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&body))

	total, ok := body["total"].(float64)
	require.True(t, ok)
	assert.GreaterOrEqual(t, int(total), 2)

	users, ok := body["users"].([]any)
	require.True(t, ok)
	assert.GreaterOrEqual(t, len(users), 2)
}

// TestE2E_Admin_SetUserRole verifies admin can change a user's role via REST.
func TestE2E_Admin_SetUserRole(t *testing.T) {
	ts := setupTestServer(t)
	_, adminID := createTestUserWithID(t, ts)
	promoteToAdmin(t, ts, adminID)
	adminToken := generateAdminToken(t, ts, adminID)

	// Create target user.
	_, targetID := createTestUserWithID(t, ts)

	// Promote target to admin.
	body, _ := json.Marshal(map[string]string{"role": "admin"})
	req, err := http.NewRequest("PUT", ts.URL+"/admin/users/"+targetID.String()+"/role", bytes.NewReader(body))
	require.NoError(t, err)
	req.Header.Set("Authorization", "Bearer "+adminToken)
	req.Header.Set("Content-Type", "application/json")

	resp, err := ts.Client.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusOK, resp.StatusCode)

	var user map[string]any
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&user))
	assert.Equal(t, "admin", user["Role"])

	// Verify in DB.
	var role string
	err = ts.Pool.QueryRow(context.Background(),
		"SELECT role FROM users WHERE id = $1", targetID,
	).Scan(&role)
	require.NoError(t, err)
	assert.Equal(t, "admin", role)
}

// TestE2E_Admin_CannotDemoteSelf verifies admin cannot demote themselves.
func TestE2E_Admin_CannotDemoteSelf(t *testing.T) {
	ts := setupTestServer(t)
	_, adminID := createTestUserWithID(t, ts)
	promoteToAdmin(t, ts, adminID)
	adminToken := generateAdminToken(t, ts, adminID)

	body, _ := json.Marshal(map[string]string{"role": "user"})
	req, err := http.NewRequest("PUT", ts.URL+"/admin/users/"+adminID.String()+"/role", bytes.NewReader(body))
	require.NoError(t, err)
	req.Header.Set("Authorization", "Bearer "+adminToken)
	req.Header.Set("Content-Type", "application/json")

	resp, err := ts.Client.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
}

// TestE2E_Admin_EnrichmentStats verifies admin can view enrichment queue stats.
func TestE2E_Admin_EnrichmentStats(t *testing.T) {
	ts := setupTestServer(t)
	_, adminID := createTestUserWithID(t, ts)
	promoteToAdmin(t, ts, adminID)
	adminToken := generateAdminToken(t, ts, adminID)

	req, err := http.NewRequest("GET", ts.URL+"/admin/enrichment/stats", nil)
	require.NoError(t, err)
	req.Header.Set("Authorization", "Bearer "+adminToken)

	resp, err := ts.Client.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusOK, resp.StatusCode)

	var stats map[string]any
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&stats))
	assert.Contains(t, stats, "Pending")
	assert.Contains(t, stats, "Total")
}

// TestE2E_Admin_RetryFailed verifies admin can retry failed enrichment items.
func TestE2E_Admin_RetryFailed(t *testing.T) {
	ts := setupTestServer(t)
	_, adminID := createTestUserWithID(t, ts)
	promoteToAdmin(t, ts, adminID)
	adminToken := generateAdminToken(t, ts, adminID)

	req, err := http.NewRequest("POST", ts.URL+"/admin/enrichment/retry", nil)
	require.NoError(t, err)
	req.Header.Set("Authorization", "Bearer "+adminToken)

	resp, err := ts.Client.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusOK, resp.StatusCode)

	var result map[string]any
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&result))
	assert.Contains(t, result, "retried")
}

// TestE2E_Admin_ResetProcessing verifies admin can reset stuck processing items.
func TestE2E_Admin_ResetProcessing(t *testing.T) {
	ts := setupTestServer(t)
	_, adminID := createTestUserWithID(t, ts)
	promoteToAdmin(t, ts, adminID)
	adminToken := generateAdminToken(t, ts, adminID)

	req, err := http.NewRequest("POST", ts.URL+"/admin/enrichment/reset-processing", nil)
	require.NoError(t, err)
	req.Header.Set("Authorization", "Bearer "+adminToken)

	resp, err := ts.Client.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusOK, resp.StatusCode)

	var result map[string]any
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&result))
	assert.Contains(t, result, "reset")
}

// TestE2E_Admin_GraphQL_AdminUsers verifies the adminUsers GraphQL query.
func TestE2E_Admin_GraphQL_AdminUsers(t *testing.T) {
	ts := setupTestServer(t)
	_, adminID := createTestUserWithID(t, ts)
	promoteToAdmin(t, ts, adminID)
	adminToken := generateAdminToken(t, ts, adminID)

	// Create another user.
	createTestUserAndGetToken(t, ts)

	query := `query { adminUsers(limit: 50) { users { id email role } total } }`
	status, result := ts.graphqlQuery(t, query, nil, adminToken)
	assert.Equal(t, http.StatusOK, status)
	requireNoErrors(t, result)

	payload := gqlPayload(t, result, "adminUsers")
	total := payload["total"].(float64)
	assert.GreaterOrEqual(t, int(total), 2)

	users := payload["users"].([]any)
	assert.GreaterOrEqual(t, len(users), 2)
}

// TestE2E_Admin_GraphQL_AdminSetUserRole verifies the adminSetUserRole mutation.
func TestE2E_Admin_GraphQL_AdminSetUserRole(t *testing.T) {
	ts := setupTestServer(t)
	_, adminID := createTestUserWithID(t, ts)
	promoteToAdmin(t, ts, adminID)
	adminToken := generateAdminToken(t, ts, adminID)

	_, targetID := createTestUserWithID(t, ts)

	query := `mutation($userId: UUID!, $role: String!) { adminSetUserRole(userId: $userId, role: $role) { id role } }`
	vars := map[string]any{
		"userId": targetID.String(),
		"role":   "admin",
	}

	status, result := ts.graphqlQuery(t, query, vars, adminToken)
	assert.Equal(t, http.StatusOK, status)
	requireNoErrors(t, result)

	user := gqlPayload(t, result, "adminSetUserRole")
	assert.Equal(t, "admin", user["role"])
}

// TestE2E_Admin_GraphQL_RegularUserForbidden verifies non-admin gets FORBIDDEN
// error on admin GraphQL queries.
func TestE2E_Admin_GraphQL_RegularUserForbidden(t *testing.T) {
	ts := setupTestServer(t)
	token := createTestUserAndGetToken(t, ts)

	query := `query { adminUsers(limit: 10) { total } }`
	status, result := ts.graphqlQuery(t, query, nil, token)
	assert.Equal(t, http.StatusOK, status)

	code := gqlErrorCode(t, result)
	assert.Equal(t, "FORBIDDEN", code)
}

// ---------------------------------------------------------------------------
// Admin test helpers
// ---------------------------------------------------------------------------

// promoteToAdmin promotes a user to admin directly in the database.
func promoteToAdmin(t *testing.T, ts *testServer, userID uuid.UUID) {
	t.Helper()
	_, err := ts.Pool.Exec(context.Background(),
		"UPDATE users SET role = 'admin', updated_at = now() WHERE id = $1",
		userID,
	)
	require.NoError(t, err, "promote user to admin")
}

// generateAdminToken generates a JWT with admin role for the given user.
func generateAdminToken(t *testing.T, ts *testServer, userID uuid.UUID) string {
	t.Helper()
	tok, err := ts.jwt.GenerateAccessToken(userID, "admin")
	if err != nil {
		t.Fatalf("generate admin token: %v", err)
	}
	return tok
}

// restRequest is a helper to make REST requests with auth and JSON body.
func restRequest(t *testing.T, ts *testServer, method, path, token string, body any) *http.Response {
	t.Helper()

	var reqBody *bytes.Reader
	if body != nil {
		jsonBody, err := json.Marshal(body)
		require.NoError(t, err)
		reqBody = bytes.NewReader(jsonBody)
	} else {
		reqBody = bytes.NewReader(nil)
	}

	req, err := http.NewRequest(method, ts.URL+path, reqBody)
	require.NoError(t, err)
	req.Header.Set("Authorization", "Bearer "+token)
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	resp, err := ts.Client.Do(req)
	require.NoError(t, err)
	return resp
}

// suppress unused import warning
var _ = fmt.Sprintf
