package http_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestDataLoaderSenses tests that DataLoader batches sense queries.
func TestDataLoaderSenses(t *testing.T) {
	app := setupTestApp(t)
	defer app.teardown(t)

	// Create multiple words with senses
	wordIDs := make([]string, 3)
	for i := 0; i < 3; i++ {
		text := "word" + string(rune('0'+i))
		createQuery := `
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

		resp := app.executeGraphQL(t, createQuery, map[string]interface{}{
			"text":       text,
			"definition": "definition for " + text,
		})
		require.Empty(t, resp.Errors)
		wordIDs[i] = extractString(t, resp.Data, "createWord", "id")
	}

	// Query all words with their senses in a single request
	// This should trigger DataLoader batching
	query := `
		query {
			dictionary {
				id
				text
				senses {
					id
					definition
				}
			}
		}
	`

	resp := app.executeGraphQL(t, query, nil)
	require.Empty(t, resp.Errors)

	entries := extractArray(t, resp.Data, "dictionary")
	assert.Len(t, entries, 3, "Should have 3 entries")

	// Verify each entry has senses loaded
	for _, entry := range entries {
		entryMap := entry.(map[string]interface{})
		senses := entryMap["senses"].([]interface{})
		assert.Len(t, senses, 1, "Each entry should have one sense")
	}
}

// TestDataLoaderImages tests that DataLoader batches image queries.
func TestDataLoaderImages(t *testing.T) {
	app := setupTestApp(t)
	defer app.teardown(t)

	// Create a word with images
	createQuery := `
		mutation {
			createWord(input: {
				text: "test"
				createCard: false
				senses: [{
					definition: "test definition"
					partOfSpeech: NOUN
					sourceSlug: "user"
				}]
				images: [{
					url: "https://example.com/image1.jpg"
					sourceSlug: "test"
				}]
			}) {
				id
			}
		}
	`

	createResp := app.executeGraphQL(t, createQuery, nil)
	require.Empty(t, createResp.Errors)
	wordID := extractString(t, createResp.Data, "createWord", "id")

	// Query the word with images
	query := `
		query($id: UUID!) {
			dictionaryEntry(id: $id) {
				id
				images {
					id
					url
				}
			}
		}
	`

	resp := app.executeGraphQL(t, query, map[string]interface{}{
		"id": wordID,
	})

	require.Empty(t, resp.Errors)
	images := extractArray(t, resp.Data, "dictionaryEntry", "images")
	assert.Len(t, images, 1, "Should have one image")
}

// TestDataLoaderPronunciations tests that DataLoader batches pronunciation queries.
func TestDataLoaderPronunciations(t *testing.T) {
	app := setupTestApp(t)
	defer app.teardown(t)

	// Create a word with pronunciations
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
				pronunciations: [{
					transcription: "/həˈloʊ/"
					audioUrl: "https://example.com/hello.mp3"
					sourceSlug: "test"
				}]
			}) {
				id
			}
		}
	`

	createResp := app.executeGraphQL(t, createQuery, nil)
	require.Empty(t, createResp.Errors)
	wordID := extractString(t, createResp.Data, "createWord", "id")

	// Query the word with pronunciations
	query := `
		query($id: UUID!) {
			dictionaryEntry(id: $id) {
				id
				pronunciations {
					id
					transcription
				}
			}
		}
	`

	resp := app.executeGraphQL(t, query, map[string]interface{}{
		"id": wordID,
	})

	require.Empty(t, resp.Errors)
	pronunciations := extractArray(t, resp.Data, "dictionaryEntry", "pronunciations")
	assert.Len(t, pronunciations, 1, "Should have one pronunciation")
}

// TestDataLoaderExamples tests that DataLoader batches example queries.
func TestDataLoaderExamples(t *testing.T) {
	app := setupTestApp(t)
	defer app.teardown(t)

	// Create a word with senses and examples
	createQuery := `
		mutation {
			createWord(input: {
				text: "hello"
				createCard: false
				senses: [{
					definition: "a greeting"
					partOfSpeech: NOUN
					sourceSlug: "user"
					examples: [{
						sentence: "Hello, world!"
						translation: "Привет, мир!"
						sourceSlug: "user"
					}]
				}]
			}) {
				id
				senses {
					id
					examples {
						id
						sentence
					}
				}
			}
		}
	`

	resp := app.executeGraphQL(t, createQuery, nil)
	require.Empty(t, resp.Errors)

	senses := extractArray(t, resp.Data, "createWord", "senses")
	require.Len(t, senses, 1, "Should have one sense")

	sense := senses[0].(map[string]interface{})
	examples := sense["examples"].([]interface{})
	assert.Len(t, examples, 1, "Should have one example")
}

// TestDataLoaderTranslations tests that DataLoader batches translation queries.
func TestDataLoaderTranslations(t *testing.T) {
	app := setupTestApp(t)
	defer app.teardown(t)

	// Create a word with senses and translations
	createQuery := `
		mutation {
			createWord(input: {
				text: "hello"
				createCard: false
				senses: [{
					definition: "a greeting"
					partOfSpeech: NOUN
					sourceSlug: "user"
					translations: [{
						text: "привет"
						sourceSlug: "user"
					}]
				}]
			}) {
				id
				senses {
					id
					translations {
						id
						text
					}
				}
			}
		}
	`

	resp := app.executeGraphQL(t, createQuery, nil)
	require.Empty(t, resp.Errors)

	senses := extractArray(t, resp.Data, "createWord", "senses")
	require.Len(t, senses, 1)

	sense := senses[0].(map[string]interface{})
	translations := sense["translations"].([]interface{})
	assert.Len(t, translations, 1, "Should have one translation")
}

// TestDataLoaderCard tests that DataLoader loads cards correctly.
func TestDataLoaderCard(t *testing.T) {
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
					entryId
				}
			}
		}
	`

	resp := app.executeGraphQL(t, createQuery, nil)
	require.Empty(t, resp.Errors)

	// Verify card is loaded
	cardData := extractObject(t, resp.Data, "createWord", "card")
	require.NotNil(t, cardData, "Card should be loaded")
	assert.NotEmpty(t, cardData["id"], "Card should have an ID")
}
