package http_test

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestCreateWord tests creating a word.
func TestCreateWord(t *testing.T) {
	app := setupTestApp(t)
	defer app.teardown(t)

	query := `
		mutation {
			createWord(input: {
				text: "hello"
				createCard: false
				senses: [{
					definition: "a greeting"
					partOfSpeech: NOUN
					sourceSlug: "user"
				}]
			}) {
				id
				text
				senses {
					id
					definition
					partOfSpeech
				}
			}
		}
	`

	resp := app.executeGraphQL(t, query, nil)
	require.Empty(t, resp.Errors, "Expected no errors, got: %v", resp.Errors)

	// Verify the word was created
	id := extractString(t, resp.Data, "createWord", "id")
	assert.NotEmpty(t, id, "Word ID should not be empty")

	text := extractString(t, resp.Data, "createWord", "text")
	assert.Equal(t, "hello", text)

	// Verify senses
	senses := extractArray(t, resp.Data, "createWord", "senses")
	assert.Len(t, senses, 1, "Should have one sense")

	sense := senses[0].(map[string]interface{})
	assert.Equal(t, "a greeting", sense["definition"])
	assert.Equal(t, "NOUN", sense["partOfSpeech"])
}

// TestCreateWordWithCard tests creating a word with a card.
func TestCreateWordWithCard(t *testing.T) {
	app := setupTestApp(t)
	defer app.teardown(t)

	query := `
		mutation {
			createWord(input: {
				text: "hello"
				createCard: true
				senses: [{
					definition: "a greeting"
					partOfSpeech: NOUN
					sourceSlug: "user"
				}]
			}) {
				id
				text
				card {
					id
					entryId
					status
				}
			}
		}
	`

	resp := app.executeGraphQL(t, query, nil)
	require.Empty(t, resp.Errors)

	// Verify card was created
	cardData := extractObject(t, resp.Data, "createWord", "card")
	assert.NotNil(t, cardData, "Card should be created")
	assert.NotEmpty(t, cardData["id"], "Card should have an ID")
}

// TestCreateWordWithMultipleSenses tests creating a word with multiple senses.
func TestCreateWordWithMultipleSenses(t *testing.T) {
	app := setupTestApp(t)
	defer app.teardown(t)

	query := `
		mutation {
			createWord(input: {
				text: "bank"
				createCard: false
				senses: [
					{
						definition: "financial institution"
						partOfSpeech: NOUN
						sourceSlug: "user"
					},
					{
						definition: "river edge"
						partOfSpeech: NOUN
						sourceSlug: "user"
					}
				]
			}) {
				id
				text
				senses {
					id
					definition
					partOfSpeech
				}
			}
		}
	`

	resp := app.executeGraphQL(t, query, nil)
	require.Empty(t, resp.Errors)

	senses := extractArray(t, resp.Data, "createWord", "senses")
	assert.Len(t, senses, 2, "Should have two senses")
}

// TestCreateWordDuplicate tests creating a duplicate word (should fail).
func TestCreateWordDuplicate(t *testing.T) {
	app := setupTestApp(t)
	defer app.teardown(t)

	// Create first word
	query1 := `
		mutation {
			createWord(input: {
				text: "hello"
				createCard: false
				senses: [{
					definition: "a greeting"
					partOfSpeech: NOUN
					sourceSlug: "user"
				}]
			}) {
				id
			}
		}
	`

	app.executeGraphQL(t, query1, nil)

	// Try to create duplicate
	query2 := `
		mutation {
			createWord(input: {
				text: "hello"
				createCard: false
				senses: [{
					definition: "another definition"
					partOfSpeech: VERB
					sourceSlug: "user"
				}]
			}) {
				id
			}
		}
	`

	resp := app.executeGraphQLWithError(t, query2, nil)
	require.NotEmpty(t, resp.Errors, "Expected error for duplicate word")
	assert.Contains(t, resp.Errors[0].Message, "already exists", "Error should mention duplicate")
}

// TestUpdateWord tests updating a word.
func TestUpdateWord(t *testing.T) {
	app := setupTestApp(t)
	defer app.teardown(t)

	// Create a word first
	createQuery := `
		mutation {
			createWord(input: {
				text: "hello"
				createCard: false
				senses: [{
					definition: "a greeting"
					partOfSpeech: NOUN
					sourceSlug: "user"
				}]
			}) {
				id
			}
		}
	`

	createResp := app.executeGraphQL(t, createQuery, nil)
	require.Empty(t, createResp.Errors)
	wordID := extractString(t, createResp.Data, "createWord", "id")

	// Update the word
	updateQuery := `
		mutation($id: UUID!, $input: UpdateWordInput!) {
			updateWord(id: $id, input: $input) {
				id
				text
				senses {
					definition
				}
			}
		}
	`

	updateResp := app.executeGraphQL(t, updateQuery, map[string]interface{}{
		"id": wordID,
		"input": map[string]interface{}{
			"senses": []map[string]interface{}{
				{
					"definition":   "updated definition",
					"partOfSpeech": "NOUN",
					"sourceSlug":   "user",
				},
			},
		},
	})

	require.Empty(t, updateResp.Errors)
	senses := extractArray(t, updateResp.Data, "updateWord", "senses")
	assert.Len(t, senses, 1, "Should have one sense after update")

	sense := senses[0].(map[string]interface{})
	assert.Equal(t, "updated definition", sense["definition"])
}

// TestDeleteWord tests deleting a word.
func TestDeleteWord(t *testing.T) {
	app := setupTestApp(t)
	defer app.teardown(t)

	// Create a word first
	createQuery := `
		mutation {
			createWord(input: {
				text: "hello"
				createCard: false
				senses: [{
					definition: "a greeting"
					partOfSpeech: NOUN
					sourceSlug: "user"
				}]
			}) {
				id
			}
		}
	`

	createResp := app.executeGraphQL(t, createQuery, nil)
	require.Empty(t, createResp.Errors)
	wordID := extractString(t, createResp.Data, "createWord", "id")

	// Delete the word
	deleteQuery := `
		mutation($id: UUID!) {
			deleteWord(id: $id)
		}
	`

	deleteResp := app.executeGraphQL(t, deleteQuery, map[string]interface{}{
		"id": wordID,
	})

	require.Empty(t, deleteResp.Errors)
	deleted := extractBool(t, deleteResp.Data, "deleteWord")
	assert.True(t, deleted, "Word should be deleted")

	// Verify word is gone
	queryQuery := `
		query($id: UUID!) {
			dictionaryEntry(id: $id) {
				id
			}
		}
	`

	queryResp := app.executeGraphQL(t, queryQuery, map[string]interface{}{
		"id": wordID,
	})

	// Should return null or error
	// The response might have null data or an error
	if len(queryResp.Errors) == 0 {
		// Check if entry is null
		var data map[string]interface{}
		err := json.Unmarshal(queryResp.Data, &data)
		require.NoError(t, err)
		entry := data["dictionaryEntry"]
		assert.Nil(t, entry, "Entry should be null after deletion")
	}
}

// TestAddToInbox tests adding an item to inbox.
func TestAddToInbox(t *testing.T) {
	app := setupTestApp(t)
	defer app.teardown(t)

	query := `
		mutation {
			addToInbox(text: "new word to learn", context: "from a book") {
				id
				text
				context
			}
		}
	`

	resp := app.executeGraphQL(t, query, nil)
	require.Empty(t, resp.Errors)

	id := extractString(t, resp.Data, "addToInbox", "id")
	assert.NotEmpty(t, id, "Inbox item should have an ID")

	text := extractString(t, resp.Data, "addToInbox", "text")
	assert.Equal(t, "new word to learn", text)

	context := extractString(t, resp.Data, "addToInbox", "context")
	assert.Equal(t, "from a book", context)
}

// TestDeleteInboxItem tests deleting an inbox item.
func TestDeleteInboxItem(t *testing.T) {
	app := setupTestApp(t)
	defer app.teardown(t)

	// Add to inbox first
	addQuery := `
		mutation {
			addToInbox(text: "test item") {
				id
			}
		}
	`

	addResp := app.executeGraphQL(t, addQuery, nil)
	require.Empty(t, addResp.Errors)
	itemID := extractString(t, addResp.Data, "addToInbox", "id")

	// Delete the item
	deleteQuery := `
		mutation($id: UUID!) {
			deleteInboxItem(id: $id)
		}
	`

	deleteResp := app.executeGraphQL(t, deleteQuery, map[string]interface{}{
		"id": itemID,
	})

	require.Empty(t, deleteResp.Errors)
	deleted := extractBool(t, deleteResp.Data, "deleteInboxItem")
	assert.True(t, deleted, "Inbox item should be deleted")
}

// TestReviewCard tests reviewing a card.
func TestReviewCard(t *testing.T) {
	app := setupTestApp(t)
	defer app.teardown(t)

	// Create a word with a card
	createQuery := `
		mutation {
			createWord(input: {
				text: "hello"
				createCard: true
				senses: [{
					definition: "a greeting"
					partOfSpeech: NOUN
					sourceSlug: "user"
				}]
			}) {
				id
				card {
					id
				}
			}
		}
	`

	createResp := app.executeGraphQL(t, createQuery, nil)
	require.Empty(t, createResp.Errors)

	// Get card ID (need to extract from card field)
	cardData := extractObject(t, createResp.Data, "createWord", "card")
	require.NotNil(t, cardData, "Card should exist")
	cardID := cardData["id"].(string)

	// Review the card
	reviewQuery := `
		mutation($cardId: UUID!, $grade: ReviewGrade!) {
			reviewCard(cardId: $cardId, grade: $grade) {
				entry {
					id
				}
				nextReviewAt
			}
		}
	`

	reviewResp := app.executeGraphQL(t, reviewQuery, map[string]interface{}{
		"cardId": cardID,
		"grade":  "GOOD",
	})

	require.Empty(t, reviewResp.Errors)
	nextReviewAt := extractString(t, reviewResp.Data, "reviewCard", "nextReviewAt")
	assert.NotEmpty(t, nextReviewAt, "nextReviewAt should be set")
}

// TestReviewCardInvalidGrade tests reviewing with invalid grade.
func TestReviewCardInvalidGrade(t *testing.T) {
	app := setupTestApp(t)
	defer app.teardown(t)

	// Create a word with a card
	createQuery := `
		mutation {
			createWord(input: {
				text: "hello"
				createCard: true
				senses: [{
					definition: "a greeting"
					partOfSpeech: NOUN
					sourceSlug: "user"
				}]
			}) {
				id
				card {
					id
				}
			}
		}
	`

	createResp := app.executeGraphQL(t, createQuery, nil)
	require.Empty(t, createResp.Errors)

	cardData := extractObject(t, createResp.Data, "createWord", "card")
	require.NotNil(t, cardData, "Card should exist")
	cardID := cardData["id"].(string)

	// Try to review with invalid grade (this should be caught by GraphQL validation)
	reviewQuery := `
		mutation($cardId: UUID!) {
			reviewCard(cardId: $cardId, grade: INVALID) {
				entry {
					id
				}
			}
		}
	`

	// This should fail at GraphQL parsing/validation level
	resp := app.executeGraphQLWithError(t, reviewQuery, map[string]interface{}{
		"cardId": cardID,
	})

	require.NotEmpty(t, resp.Errors, "Expected error for invalid grade")
}
