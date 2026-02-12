package http_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestErrorHandlingNotFound tests error handling for not found resources.
func TestErrorHandlingNotFound(t *testing.T) {
	app := setupTestApp(t)
	defer app.teardown(t)

	// Try to query a non-existent entry (using a valid but non-existent UUID)
	query := `
		query($id: UUID!) {
			dictionaryEntry(id: $id) {
				id
			}
		}
	`

	resp := app.executeGraphQLWithError(t, query, map[string]interface{}{
		"id": "123e4567-e89b-12d3-a456-426614174000",
	})

	require.NotEmpty(t, resp.Errors, "Expected error for non-existent entry")
	assert.Contains(t, resp.Errors[0].Message, "not found", "Error should indicate not found")
}

// TestErrorHandlingInvalidInput tests error handling for invalid input.
func TestErrorHandlingInvalidInput(t *testing.T) {
	app := setupTestApp(t)
	defer app.teardown(t)

	// Try to create a word with empty text
	query := `
		mutation {
			createWord(input: {
				text: ""
				createCard: false
				senses: [{
					definition: "test"
					partOfSpeech: NOUN
					sourceSlug: "user"
				}]
			}) {
				id
			}
		}
	`

	resp := app.executeGraphQLWithError(t, query, nil)
	require.NotEmpty(t, resp.Errors, "Expected error for invalid input")
}

// TestErrorHandlingUpdateNonExistent tests updating a non-existent word.
func TestErrorHandlingUpdateNonExistent(t *testing.T) {
	app := setupTestApp(t)
	defer app.teardown(t)

	query := `
		mutation($id: UUID!, $input: UpdateWordInput!) {
			updateWord(id: $id, input: $input) {
				id
			}
		}
	`

	resp := app.executeGraphQLWithError(t, query, map[string]interface{}{
		"id": "00000000-0000-0000-0000-000000000000",
		"input": map[string]interface{}{
			"senses": []map[string]interface{}{},
		},
	})

	require.NotEmpty(t, resp.Errors, "Expected error for non-existent word")
}

// TestErrorHandlingDeleteNonExistent tests deleting a non-existent word.
func TestErrorHandlingDeleteNonExistent(t *testing.T) {
	app := setupTestApp(t)
	defer app.teardown(t)

	query := `
		mutation($id: UUID!) {
			deleteWord(id: $id)
		}
	`

	resp := app.executeGraphQLWithError(t, query, map[string]interface{}{
		"id": "00000000-0000-0000-0000-000000000000",
	})

	require.NotEmpty(t, resp.Errors, "Expected error for non-existent word")
}

// TestErrorHandlingInvalidUUID tests error handling for invalid UUID format.
func TestErrorHandlingInvalidUUID(t *testing.T) {
	app := setupTestApp(t)
	defer app.teardown(t)

	// GraphQL will validate UUID format, so this should fail at parsing
	query := `
		query {
			dictionaryEntry(id: "invalid-uuid") {
				id
			}
		}
	`

	resp := app.executeGraphQLWithError(t, query, nil)
	// This might fail at GraphQL validation or parsing level
	assert.NotEmpty(t, resp.Errors, "Expected error for invalid UUID")
}
