//go:build e2e

package e2e_test

import (
	"net/http"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ---------------------------------------------------------------------------
// Scenario 8: Full content enrichment cycle — add sense, translation,
// example, then update and delete.
// ---------------------------------------------------------------------------

func TestE2E_ContentEnrichment_AddSenseTranslationExample(t *testing.T) {
	ts := setupTestServer(t)
	token := createTestUserAndGetToken(t, ts)

	// Create a minimal custom entry.
	text := "enrich-" + uuid.New().String()[:8]
	createQuery := `mutation($input: CreateEntryCustomInput!) {
		createEntryCustom(input: $input) { entry { id } }
	}`
	status, result := ts.graphqlQuery(t, createQuery, map[string]any{
		"input": map[string]any{
			"text":   text,
			"senses": []any{map[string]any{"definition": "initial sense"}},
		},
	}, token)
	require.Equal(t, http.StatusOK, status)
	requireNoErrors(t, result)
	entryID := gqlPayload(t, result, "createEntryCustom")["entry"].(map[string]any)["id"].(string)

	// Add a second sense.
	addSenseQuery := `mutation($input: AddSenseInput!) {
		addSense(input: $input) { sense { id definition partOfSpeech position } }
	}`
	status, result = ts.graphqlQuery(t, addSenseQuery, map[string]any{
		"input": map[string]any{
			"entryId":      entryID,
			"definition":   "second meaning",
			"partOfSpeech": "VERB",
			"translations": []any{"второе значение"},
		},
	}, token)
	assert.Equal(t, http.StatusOK, status)
	requireNoErrors(t, result)
	newSense := gqlPayload(t, result, "addSense")["sense"].(map[string]any)
	senseID := newSense["id"].(string)
	assert.Equal(t, "second meaning", newSense["definition"])
	assert.Equal(t, "VERB", newSense["partOfSpeech"])

	// Add a translation to the new sense.
	addTransQuery := `mutation($input: AddTranslationInput!) {
		addTranslation(input: $input) { translation { id text position } }
	}`
	status, result = ts.graphqlQuery(t, addTransQuery, map[string]any{
		"input": map[string]any{
			"senseId": senseID,
			"text":    "дополнительный перевод",
		},
	}, token)
	assert.Equal(t, http.StatusOK, status)
	requireNoErrors(t, result)
	translation := gqlPayload(t, result, "addTranslation")["translation"].(map[string]any)
	assert.Equal(t, "дополнительный перевод", translation["text"])

	// Add an example to the sense.
	addExampleQuery := `mutation($input: AddExampleInput!) {
		addExample(input: $input) { example { id sentence translation position } }
	}`
	status, result = ts.graphqlQuery(t, addExampleQuery, map[string]any{
		"input": map[string]any{
			"senseId":     senseID,
			"sentence":    "She enriched the text with examples.",
			"translation": "Она обогатила текст примерами.",
		},
	}, token)
	assert.Equal(t, http.StatusOK, status)
	requireNoErrors(t, result)
	example := gqlPayload(t, result, "addExample")["example"].(map[string]any)
	assert.Equal(t, "She enriched the text with examples.", example["sentence"])

	// Verify full entry structure via query.
	getQuery := `query($id: UUID!) {
		dictionaryEntry(id: $id) {
			senses {
				id definition partOfSpeech
				translations { id text }
				examples { id sentence translation }
			}
		}
	}`
	status, result = ts.graphqlQuery(t, getQuery, map[string]any{"id": entryID}, token)
	assert.Equal(t, http.StatusOK, status)
	requireNoErrors(t, result)

	senses := gqlPayload(t, result, "dictionaryEntry")["senses"].([]any)
	assert.Len(t, senses, 2, "entry should have 2 senses")

	// Find the new sense and verify its children.
	var found bool
	for _, s := range senses {
		sMap := s.(map[string]any)
		if sMap["id"] == senseID {
			found = true
			translations := sMap["translations"].([]any)
			assert.GreaterOrEqual(t, len(translations), 2, "sense should have at least 2 translations (1 inline + 1 added)")
			examples := sMap["examples"].([]any)
			assert.Len(t, examples, 1, "sense should have 1 example")
			break
		}
	}
	assert.True(t, found, "added sense should appear in entry")
}

// ---------------------------------------------------------------------------
// Scenario 8: Update sense definition.
// ---------------------------------------------------------------------------

func TestE2E_UpdateSense_ChangesDefinition(t *testing.T) {
	ts := setupTestServer(t)
	token := createTestUserAndGetToken(t, ts)

	// Create entry with a sense.
	text := "updatesense-" + uuid.New().String()[:8]
	createQuery := `mutation($input: CreateEntryCustomInput!) {
		createEntryCustom(input: $input) {
			entry { id senses { id definition } }
		}
	}`
	status, result := ts.graphqlQuery(t, createQuery, map[string]any{
		"input": map[string]any{
			"text":   text,
			"senses": []any{map[string]any{"definition": "original"}},
		},
	}, token)
	require.Equal(t, http.StatusOK, status)
	requireNoErrors(t, result)
	entry := gqlPayload(t, result, "createEntryCustom")["entry"].(map[string]any)
	senseID := entry["senses"].([]any)[0].(map[string]any)["id"].(string)

	// Update the sense.
	updateQuery := `mutation($input: UpdateSenseInput!) {
		updateSense(input: $input) { sense { id definition partOfSpeech } }
	}`
	status, result = ts.graphqlQuery(t, updateQuery, map[string]any{
		"input": map[string]any{
			"senseId":      senseID,
			"definition":   "updated meaning",
			"partOfSpeech": "ADJECTIVE",
		},
	}, token)
	assert.Equal(t, http.StatusOK, status)
	requireNoErrors(t, result)
	updated := gqlPayload(t, result, "updateSense")["sense"].(map[string]any)
	assert.Equal(t, "updated meaning", updated["definition"])
	assert.Equal(t, "ADJECTIVE", updated["partOfSpeech"])
}

// ---------------------------------------------------------------------------
// Scenario 8: Delete sense removes it from entry.
// ---------------------------------------------------------------------------

func TestE2E_DeleteSense_RemovesFromEntry(t *testing.T) {
	ts := setupTestServer(t)
	token := createTestUserAndGetToken(t, ts)

	text := "delsense-" + uuid.New().String()[:8]
	createQuery := `mutation($input: CreateEntryCustomInput!) {
		createEntryCustom(input: $input) {
			entry { id senses { id } }
		}
	}`
	status, result := ts.graphqlQuery(t, createQuery, map[string]any{
		"input": map[string]any{
			"text": text,
			"senses": []any{
				map[string]any{"definition": "sense one"},
				map[string]any{"definition": "sense two"},
			},
		},
	}, token)
	require.Equal(t, http.StatusOK, status)
	requireNoErrors(t, result)
	entry := gqlPayload(t, result, "createEntryCustom")["entry"].(map[string]any)
	entryID := entry["id"].(string)
	senses := entry["senses"].([]any)
	require.Len(t, senses, 2)
	senseToDelete := senses[0].(map[string]any)["id"].(string)

	// Delete first sense.
	deleteQuery := `mutation($id: UUID!) { deleteSense(id: $id) { senseId } }`
	status, result = ts.graphqlQuery(t, deleteQuery, map[string]any{"id": senseToDelete}, token)
	assert.Equal(t, http.StatusOK, status)
	requireNoErrors(t, result)

	// Verify entry now has 1 sense.
	getQuery := `query($id: UUID!) { dictionaryEntry(id: $id) { senses { id } } }`
	status, result = ts.graphqlQuery(t, getQuery, map[string]any{"id": entryID}, token)
	assert.Equal(t, http.StatusOK, status)
	requireNoErrors(t, result)
	remaining := gqlPayload(t, result, "dictionaryEntry")["senses"].([]any)
	assert.Len(t, remaining, 1)
	assert.NotEqual(t, senseToDelete, remaining[0].(map[string]any)["id"])
}

// ---------------------------------------------------------------------------
// Scenario 8: Reorder senses.
// ---------------------------------------------------------------------------

func TestE2E_ReorderSenses_ChangesPositions(t *testing.T) {
	ts := setupTestServer(t)
	token := createTestUserAndGetToken(t, ts)

	text := "reorder-" + uuid.New().String()[:8]
	createQuery := `mutation($input: CreateEntryCustomInput!) {
		createEntryCustom(input: $input) {
			entry { id senses { id position } }
		}
	}`
	status, result := ts.graphqlQuery(t, createQuery, map[string]any{
		"input": map[string]any{
			"text": text,
			"senses": []any{
				map[string]any{"definition": "first"},
				map[string]any{"definition": "second"},
			},
		},
	}, token)
	require.Equal(t, http.StatusOK, status)
	requireNoErrors(t, result)
	entry := gqlPayload(t, result, "createEntryCustom")["entry"].(map[string]any)
	entryID := entry["id"].(string)
	senses := entry["senses"].([]any)
	sense0 := senses[0].(map[string]any)["id"].(string)
	sense1 := senses[1].(map[string]any)["id"].(string)

	// Reorder: swap positions.
	reorderQuery := `mutation($input: ReorderSensesInput!) {
		reorderSenses(input: $input) { success }
	}`
	status, result = ts.graphqlQuery(t, reorderQuery, map[string]any{
		"input": map[string]any{
			"entryId": entryID,
			"items": []any{
				map[string]any{"id": sense0, "position": 1},
				map[string]any{"id": sense1, "position": 0},
			},
		},
	}, token)
	assert.Equal(t, http.StatusOK, status)
	requireNoErrors(t, result)

	// Verify new order.
	getQuery := `query($id: UUID!) { dictionaryEntry(id: $id) { senses { id position } } }`
	status, result = ts.graphqlQuery(t, getQuery, map[string]any{"id": entryID}, token)
	assert.Equal(t, http.StatusOK, status)
	requireNoErrors(t, result)

	reorderedSenses := gqlPayload(t, result, "dictionaryEntry")["senses"].([]any)
	for _, s := range reorderedSenses {
		sMap := s.(map[string]any)
		if sMap["id"] == sense0 {
			assert.Equal(t, float64(1), sMap["position"])
		}
		if sMap["id"] == sense1 {
			assert.Equal(t, float64(0), sMap["position"])
		}
	}
}

// ---------------------------------------------------------------------------
// Scenario 8: Update and delete translation.
// ---------------------------------------------------------------------------

func TestE2E_TranslationUpdateAndDelete(t *testing.T) {
	ts := setupTestServer(t)
	token := createTestUserAndGetToken(t, ts)

	text := "transud-" + uuid.New().String()[:8]
	createQuery := `mutation($input: CreateEntryCustomInput!) {
		createEntryCustom(input: $input) {
			entry { id senses { id translations { id text } } }
		}
	}`
	status, result := ts.graphqlQuery(t, createQuery, map[string]any{
		"input": map[string]any{
			"text": text,
			"senses": []any{
				map[string]any{"definition": "def", "translations": []any{"original translation"}},
			},
		},
	}, token)
	require.Equal(t, http.StatusOK, status)
	requireNoErrors(t, result)
	senses := gqlPayload(t, result, "createEntryCustom")["entry"].(map[string]any)["senses"].([]any)
	translationID := senses[0].(map[string]any)["translations"].([]any)[0].(map[string]any)["id"].(string)

	// Update translation.
	updateQuery := `mutation($input: UpdateTranslationInput!) {
		updateTranslation(input: $input) { translation { id text } }
	}`
	status, result = ts.graphqlQuery(t, updateQuery, map[string]any{
		"input": map[string]any{
			"translationId": translationID,
			"text":          "updated translation",
		},
	}, token)
	assert.Equal(t, http.StatusOK, status)
	requireNoErrors(t, result)
	assert.Equal(t, "updated translation",
		gqlPayload(t, result, "updateTranslation")["translation"].(map[string]any)["text"])

	// Delete translation.
	deleteQuery := `mutation($id: UUID!) { deleteTranslation(id: $id) { translationId } }`
	status, result = ts.graphqlQuery(t, deleteQuery, map[string]any{"id": translationID}, token)
	assert.Equal(t, http.StatusOK, status)
	requireNoErrors(t, result)
}

// ---------------------------------------------------------------------------
// Scenario 8: Update and delete example.
// ---------------------------------------------------------------------------

func TestE2E_ExampleUpdateAndDelete(t *testing.T) {
	ts := setupTestServer(t)
	token := createTestUserAndGetToken(t, ts)

	text := "exud-" + uuid.New().String()[:8]
	createQuery := `mutation($input: CreateEntryCustomInput!) {
		createEntryCustom(input: $input) {
			entry { id senses { id } }
		}
	}`
	status, result := ts.graphqlQuery(t, createQuery, map[string]any{
		"input": map[string]any{
			"text":   text,
			"senses": []any{map[string]any{"definition": "def"}},
		},
	}, token)
	require.Equal(t, http.StatusOK, status)
	requireNoErrors(t, result)
	senseID := gqlPayload(t, result, "createEntryCustom")["entry"].(map[string]any)["senses"].([]any)[0].(map[string]any)["id"].(string)

	// Add example.
	addQuery := `mutation($input: AddExampleInput!) {
		addExample(input: $input) { example { id sentence translation } }
	}`
	status, result = ts.graphqlQuery(t, addQuery, map[string]any{
		"input": map[string]any{
			"senseId":     senseID,
			"sentence":    "Original sentence.",
			"translation": "Оригинальное предложение.",
		},
	}, token)
	require.Equal(t, http.StatusOK, status)
	requireNoErrors(t, result)
	exampleID := gqlPayload(t, result, "addExample")["example"].(map[string]any)["id"].(string)

	// Update example.
	updateQuery := `mutation($input: UpdateExampleInput!) {
		updateExample(input: $input) { example { id sentence } }
	}`
	status, result = ts.graphqlQuery(t, updateQuery, map[string]any{
		"input": map[string]any{
			"exampleId": exampleID,
			"sentence":  "Updated sentence.",
		},
	}, token)
	assert.Equal(t, http.StatusOK, status)
	requireNoErrors(t, result)
	assert.Equal(t, "Updated sentence.",
		gqlPayload(t, result, "updateExample")["example"].(map[string]any)["sentence"])

	// Delete example.
	deleteQuery := `mutation($id: UUID!) { deleteExample(id: $id) { exampleId } }`
	status, result = ts.graphqlQuery(t, deleteQuery, map[string]any{"id": exampleID}, token)
	assert.Equal(t, http.StatusOK, status)
	requireNoErrors(t, result)
}
