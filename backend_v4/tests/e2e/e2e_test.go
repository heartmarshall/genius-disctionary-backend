//go:build e2e

package e2e_test

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestE2E_LiveEndpoint verifies the /live liveness probe returns 200 OK.
func TestE2E_LiveEndpoint(t *testing.T) {
	ts := setupTestServer(t)

	resp, err := ts.Client.Get(ts.URL + "/live")
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusOK, resp.StatusCode)

	var body map[string]any
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&body))
	assert.Equal(t, "ok", body["status"])
}

// TestE2E_ReadyEndpoint verifies the /ready readiness probe returns 200 OK
// when the database is reachable.
func TestE2E_ReadyEndpoint(t *testing.T) {
	ts := setupTestServer(t)

	resp, err := ts.Client.Get(ts.URL + "/ready")
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusOK, resp.StatusCode)

	var body map[string]any
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&body))
	assert.Equal(t, "ok", body["status"])
}

// TestE2E_HealthEndpoint verifies the /health endpoint returns 200 with
// version and database component status.
func TestE2E_HealthEndpoint(t *testing.T) {
	ts := setupTestServer(t)

	resp, err := ts.Client.Get(ts.URL + "/health")
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusOK, resp.StatusCode)

	var body map[string]any
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&body))
	assert.Equal(t, "ok", body["status"])
	assert.NotEmpty(t, body["version"])

	components, ok := body["components"].(map[string]any)
	require.True(t, ok, "expected components object")

	db, ok := components["database"].(map[string]any)
	require.True(t, ok, "expected database component")
	assert.Equal(t, "ok", db["status"])
}

// TestE2E_GraphQL_Unauthenticated verifies that the searchCatalog query
// works without authentication (anonymous endpoint).
func TestE2E_GraphQL_Unauthenticated(t *testing.T) {
	ts := setupTestServer(t)

	query := `query { searchCatalog(query: "test", limit: 5) { id text } }`

	status, result := ts.graphqlQuery(t, query, nil, "")
	assert.Equal(t, http.StatusOK, status)

	// Should have data (possibly empty array) and no errors.
	data, ok := result["data"].(map[string]any)
	require.True(t, ok, "expected data in response")

	catalog, ok := data["searchCatalog"].([]any)
	require.True(t, ok, "expected searchCatalog array")
	// The catalog is empty in a fresh DB — that is fine.
	_ = catalog
	assert.Nil(t, result["errors"], "expected no errors for anonymous searchCatalog")
}

// TestE2E_GraphQL_AuthRequired verifies that a mutation requiring
// authentication returns UNAUTHENTICATED when no token is provided.
func TestE2E_GraphQL_AuthRequired(t *testing.T) {
	ts := setupTestServer(t)

	query := `mutation { createTopic(input: {name: "Test"}) { topic { id name } } }`

	status, result := ts.graphqlQuery(t, query, nil, "")
	assert.Equal(t, http.StatusOK, status)

	errors, ok := result["errors"].([]any)
	require.True(t, ok, "expected errors array")
	require.NotEmpty(t, errors)

	firstErr, ok := errors[0].(map[string]any)
	require.True(t, ok)

	extensions, ok := firstErr["extensions"].(map[string]any)
	require.True(t, ok, "expected extensions in error")
	assert.Equal(t, "UNAUTHENTICATED", extensions["code"])
}

// TestE2E_GraphQL_CRUD_Topic verifies the full create-read-delete cycle
// for topics through the GraphQL API.
func TestE2E_GraphQL_CRUD_Topic(t *testing.T) {
	ts := setupTestServer(t)
	token := createTestUserAndGetToken(t, ts)

	// 1. Create topic.
	createQuery := `mutation($input: CreateTopicInput!) { createTopic(input: $input) { topic { id name description } } }`
	createVars := map[string]any{
		"input": map[string]any{
			"name":        "E2E Test Topic",
			"description": "Integration test description",
		},
	}

	status, result := ts.graphqlQuery(t, createQuery, createVars, token)
	assert.Equal(t, http.StatusOK, status)
	assert.Nil(t, result["errors"], "create topic should not return errors")

	data := result["data"].(map[string]any)
	createPayload := data["createTopic"].(map[string]any)
	topic := createPayload["topic"].(map[string]any)

	topicID, ok := topic["id"].(string)
	require.True(t, ok, "expected topic id string")
	assert.Equal(t, "E2E Test Topic", topic["name"])
	assert.Equal(t, "Integration test description", topic["description"])

	// 2. List topics — verify it exists.
	listQuery := `query { topics { id name } }`
	status, result = ts.graphqlQuery(t, listQuery, nil, token)
	assert.Equal(t, http.StatusOK, status)
	assert.Nil(t, result["errors"])

	data = result["data"].(map[string]any)
	topics := data["topics"].([]any)
	require.NotEmpty(t, topics, "expected at least one topic after creation")

	found := false
	for _, tp := range topics {
		tMap := tp.(map[string]any)
		if tMap["id"] == topicID {
			found = true
			break
		}
	}
	assert.True(t, found, "created topic should appear in topics list")

	// 3. Delete topic.
	deleteQuery := `mutation($id: UUID!) { deleteTopic(id: $id) { topicId } }`
	deleteVars := map[string]any{"id": topicID}

	status, result = ts.graphqlQuery(t, deleteQuery, deleteVars, token)
	assert.Equal(t, http.StatusOK, status)
	assert.Nil(t, result["errors"], "delete topic should not return errors")

	// 4. List topics — verify it is gone.
	status, result = ts.graphqlQuery(t, listQuery, nil, token)
	assert.Equal(t, http.StatusOK, status)

	data = result["data"].(map[string]any)
	topics = data["topics"].([]any)
	for _, tp := range topics {
		tMap := tp.(map[string]any)
		assert.NotEqual(t, topicID, tMap["id"], "deleted topic should not appear in list")
	}
}

// TestE2E_RequestID_InResponse verifies that every response from the
// middleware stack includes an X-Request-Id header.
func TestE2E_RequestID_InResponse(t *testing.T) {
	ts := setupTestServer(t)

	query := `query { searchCatalog(query: "x", limit: 1) { id } }`

	jsonBody, err := json.Marshal(map[string]any{"query": query})
	require.NoError(t, err)

	req, err := http.NewRequest(http.MethodPost, ts.URL+"/query", io.NopCloser(
		bytes.NewReader(jsonBody),
	))
	require.NoError(t, err)
	req.Header.Set("Content-Type", "application/json")

	resp, err := ts.Client.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()

	requestID := resp.Header.Get("X-Request-Id")
	assert.NotEmpty(t, requestID, "response should include X-Request-Id header")

	// The value should be a valid UUID.
	_, err = uuid.Parse(requestID)
	assert.NoError(t, err, "X-Request-Id should be a valid UUID")
}

// TestE2E_CORS_Preflight verifies that an OPTIONS preflight request to
// /query returns the appropriate Access-Control-Allow-* headers.
func TestE2E_CORS_Preflight(t *testing.T) {
	ts := setupTestServer(t)

	req, err := http.NewRequest(http.MethodOptions, ts.URL+"/query", nil)
	require.NoError(t, err)
	req.Header.Set("Origin", "https://example.com")
	req.Header.Set("Access-Control-Request-Method", "POST")
	req.Header.Set("Access-Control-Request-Headers", "Authorization,Content-Type")

	resp, err := ts.Client.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusNoContent, resp.StatusCode)
	assert.NotEmpty(t, resp.Header.Get("Access-Control-Allow-Origin"))
	assert.NotEmpty(t, resp.Header.Get("Access-Control-Allow-Methods"))
	assert.NotEmpty(t, resp.Header.Get("Access-Control-Allow-Headers"))
}

// TestE2E_GraphQL_NotFound verifies that querying a non-existent dictionary
// entry returns a NOT_FOUND error code.
func TestE2E_GraphQL_NotFound(t *testing.T) {
	ts := setupTestServer(t)
	token := createTestUserAndGetToken(t, ts)

	query := `query($id: UUID!) { dictionaryEntry(id: $id) { id text } }`
	vars := map[string]any{"id": uuid.New().String()}

	status, result := ts.graphqlQuery(t, query, vars, token)
	assert.Equal(t, http.StatusOK, status)

	errors, ok := result["errors"].([]any)
	require.True(t, ok, "expected errors array")
	require.NotEmpty(t, errors)

	firstErr := errors[0].(map[string]any)
	extensions, ok := firstErr["extensions"].(map[string]any)
	require.True(t, ok, "expected extensions in error")
	assert.Equal(t, "NOT_FOUND", extensions["code"])
}

// TestE2E_GraphQL_ValidationError verifies that an empty topic name
// returns a VALIDATION error code.
func TestE2E_GraphQL_ValidationError(t *testing.T) {
	ts := setupTestServer(t)
	token := createTestUserAndGetToken(t, ts)

	query := `mutation { createTopic(input: {name: ""}) { topic { id } } }`

	status, result := ts.graphqlQuery(t, query, nil, token)
	assert.Equal(t, http.StatusOK, status)

	errors, ok := result["errors"].([]any)
	require.True(t, ok, "expected errors array")
	require.NotEmpty(t, errors)

	firstErr := errors[0].(map[string]any)
	extensions, ok := firstErr["extensions"].(map[string]any)
	require.True(t, ok, "expected extensions in error")
	assert.Equal(t, "VALIDATION", extensions["code"])
}
