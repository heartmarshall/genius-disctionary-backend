package http_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestDictionaryQuery tests the dictionary query.
func TestDictionaryQuery(t *testing.T) {
	app := setupTestApp(t)
	defer app.teardown(t)

	query := `
		query {
			dictionary {
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

	// Initially, dictionary should be empty
	arr := extractArray(t, resp.Data, "dictionary")
	assert.Empty(t, arr, "Dictionary should be empty initially")
}

// TestDictionaryEntryQuery tests querying a specific dictionary entry.
func TestDictionaryEntryQuery(t *testing.T) {
	app := setupTestApp(t)
	defer app.teardown(t)

	// First, create a word
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
				text
			}
		}
	`

	createResp := app.executeGraphQL(t, createQuery, nil)
	require.Empty(t, createResp.Errors)
	entryID := extractString(t, createResp.Data, "createWord", "id")

	// Now query the entry
	query := `
		query($id: UUID!) {
			dictionaryEntry(id: $id) {
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

	resp := app.executeGraphQL(t, query, map[string]interface{}{
		"id": entryID,
	})

	require.Empty(t, resp.Errors)
	assert.Equal(t, entryID, extractString(t, resp.Data, "dictionaryEntry", "id"))
	assert.Equal(t, "hello", extractString(t, resp.Data, "dictionaryEntry", "text"))
}

// TestDictionaryQueryWithFilter tests dictionary query with filters.
func TestDictionaryQueryWithFilter(t *testing.T) {
	app := setupTestApp(t)
	defer app.teardown(t)

	// Create multiple words
	words := []string{"hello", "world", "test"}
	for _, word := range words {
		query := `
			mutation($text: String!, $definition: String!) {
				createWord(input: {
					text: $text
					createCard: false
					senses: [{
						definition: $definition
						partOfSpeech: NOUN
						sourceSlug: "user"
					}]
				}) {
					id
				}
			}
		`
		app.executeGraphQL(t, query, map[string]interface{}{
			"text":       word,
			"definition": "definition for " + word,
		})
	}

	// Query with text filter
	query := `
		query($filter: WordFilter) {
			dictionary(filter: $filter) {
				text
			}
		}
	`

	resp := app.executeGraphQL(t, query, map[string]interface{}{
		"filter": map[string]interface{}{
			"search": "hello",
		},
	})

	require.Empty(t, resp.Errors)
	arr := extractArray(t, resp.Data, "dictionary")
	assert.Len(t, arr, 1, "Should find exactly one word")
}

// TestInboxItemsQuery tests the inbox items query.
func TestInboxItemsQuery(t *testing.T) {
	app := setupTestApp(t)
	defer app.teardown(t)

	// Initially, inbox should be empty
	query := `
		query {
			inboxItems {
				id
				text
				context
			}
		}
	`

	resp := app.executeGraphQL(t, query, nil)
	require.Empty(t, resp.Errors)
	arr := extractArray(t, resp.Data, "inboxItems")
	assert.Empty(t, arr, "Inbox should be empty initially")
}

// TestStudyQueueQuery tests the study queue query.
func TestStudyQueueQuery(t *testing.T) {
	app := setupTestApp(t)
	defer app.teardown(t)

	// Initially, study queue should be empty
	query := `
		query {
			studyQueue {
				id
				text
			}
		}
	`

	resp := app.executeGraphQL(t, query, nil)
	require.Empty(t, resp.Errors)
	arr := extractArray(t, resp.Data, "studyQueue")
	assert.Empty(t, arr, "Study queue should be empty initially")
}

// TestDashboardStatsQuery tests the dashboard stats query.
func TestDashboardStatsQuery(t *testing.T) {
	app := setupTestApp(t)
	defer app.teardown(t)

	query := `
		query {
			dashboardStats {
				totalWords
				totalCards
				newCards
				learningCards
				reviewCards
				masteredCards
				dueToday
			}
		}
	`

	resp := app.executeGraphQL(t, query, nil)
	require.Empty(t, resp.Errors)

	// Initially, all stats should be zero
	assert.Equal(t, 0, extractInt(t, resp.Data, "dashboardStats", "totalWords"))
	assert.Equal(t, 0, extractInt(t, resp.Data, "dashboardStats", "totalCards"))
	assert.Equal(t, 0, extractInt(t, resp.Data, "dashboardStats", "newCards"))
	assert.Equal(t, 0, extractInt(t, resp.Data, "dashboardStats", "learningCards"))
	assert.Equal(t, 0, extractInt(t, resp.Data, "dashboardStats", "reviewCards"))
	assert.Equal(t, 0, extractInt(t, resp.Data, "dashboardStats", "masteredCards"))
	assert.Equal(t, 0, extractInt(t, resp.Data, "dashboardStats", "dueToday"))
}

// TestDashboardStatsWithData tests dashboard stats after creating words and cards.
func TestDashboardStatsWithData(t *testing.T) {
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
			}
		}
	`

	app.executeGraphQL(t, createQuery, nil)

	// Query stats
	query := `
		query {
			dashboardStats {
				totalWords
				totalCards
				newCards
			}
		}
	`

	resp := app.executeGraphQL(t, query, nil)
	require.Empty(t, resp.Errors)

	assert.Equal(t, 1, extractInt(t, resp.Data, "dashboardStats", "totalWords"))
	assert.Equal(t, 1, extractInt(t, resp.Data, "dashboardStats", "totalCards"))
	assert.Equal(t, 1, extractInt(t, resp.Data, "dashboardStats", "newCards"))
}
