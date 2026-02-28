//go:build e2e

package e2e_test

import (
	"context"
	"net/http"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/heartmarshall/myenglish-backend/internal/adapter/postgres/testhelper"
)

// ---------------------------------------------------------------------------
// Scenario 1: Create entry from catalog — copies senses, translations,
// pronunciations, and optionally creates a card atomically.
// ---------------------------------------------------------------------------

func TestE2E_CreateEntryFromCatalog_CopiesSensesAndCreatesCard(t *testing.T) {
	ts := setupTestServer(t)
	token, userID := createTestUserWithID(t, ts)

	// Seed a reference entry with 2 senses, translations, pronunciations.
	refEntry := testhelper.SeedRefEntry(t, ts.Pool, "eloquent-"+uuid.New().String()[:8])

	senseIDs := make([]any, len(refEntry.Senses))
	for i, s := range refEntry.Senses {
		senseIDs[i] = s.ID.String()
	}

	// Create entry from catalog with card.
	createQuery := `mutation($input: CreateEntryFromCatalogInput!) {
		createEntryFromCatalog(input: $input) {
			entry { id text senses { id definition partOfSpeech translations { id text } } card { id state } }
		}
	}`
	createVars := map[string]any{
		"input": map[string]any{
			"refEntryId": refEntry.ID.String(),
			"senseIds":   senseIDs,
			"createCard": true,
			"notes":      "test notes",
		},
	}

	status, result := ts.graphqlQuery(t, createQuery, createVars, token)
	assert.Equal(t, http.StatusOK, status)
	requireNoErrors(t, result)

	entry := gqlPayload(t, result, "createEntryFromCatalog")["entry"].(map[string]any)
	entryID := entry["id"].(string)
	assert.Equal(t, refEntry.Text, entry["text"])

	// Senses should be copied from ref.
	senses := entry["senses"].([]any)
	assert.Len(t, senses, len(refEntry.Senses), "all selected senses should be copied")

	// Card should be created.
	card := entry["card"].(map[string]any)
	assert.Equal(t, "NEW", card["state"])

	// Verify via separate query — don't trust the mutation response alone.
	getQuery := `query($id: UUID!) {
		dictionaryEntry(id: $id) {
			id text notes
			senses { id definition translations { id text } examples { id sentence } }
			pronunciations { id transcription }
			card { id state stability difficulty }
		}
	}`
	status, result = ts.graphqlQuery(t, getQuery, map[string]any{"id": entryID}, token)
	assert.Equal(t, http.StatusOK, status)
	requireNoErrors(t, result)

	fetched := gqlPayload(t, result, "dictionaryEntry")
	assert.Equal(t, refEntry.Text, fetched["text"])
	assert.Equal(t, "test notes", fetched["notes"])

	fetchedSenses := fetched["senses"].([]any)
	assert.Len(t, fetchedSenses, len(refEntry.Senses))

	// Each sense should have translations copied.
	for _, s := range fetchedSenses {
		sMap := s.(map[string]any)
		translations := sMap["translations"].([]any)
		assert.NotEmpty(t, translations, "each sense should have translations")
	}

	// Pronunciations should be linked.
	pronunciations := fetched["pronunciations"].([]any)
	assert.NotEmpty(t, pronunciations, "pronunciations should be linked from ref")

	// Verify entry appears in dictionary list.
	dictQuery := `query($input: DictionaryFilterInput!) {
		dictionary(input: $input) { edges { node { id text } } totalCount }
	}`
	status, result = ts.graphqlQuery(t, dictQuery, map[string]any{"input": map[string]any{"limit": 100}}, token)
	assert.Equal(t, http.StatusOK, status)
	requireNoErrors(t, result)

	dictData := gqlPayload(t, result, "dictionary")
	edges := dictData["edges"].([]any)
	found := false
	for _, e := range edges {
		node := e.(map[string]any)["node"].(map[string]any)
		if node["id"] == entryID {
			found = true
			break
		}
	}
	assert.True(t, found, "entry should appear in dictionary list")

	// Verify card exists in DB.
	var cardCount int
	err := ts.Pool.QueryRow(context.Background(),
		`SELECT count(*) FROM cards WHERE user_id = $1 AND entry_id = $2`,
		userID, entryID,
	).Scan(&cardCount)
	require.NoError(t, err)
	assert.Equal(t, 1, cardCount, "exactly one card should exist for this entry")
}

// ---------------------------------------------------------------------------
// Scenario 2: Create custom entry — appears in dictionary with senses.
// ---------------------------------------------------------------------------

func TestE2E_CreateCustomEntry_AppearsInDictionary(t *testing.T) {
	ts := setupTestServer(t)
	token := createTestUserAndGetToken(t, ts)

	text := "serendipity-" + uuid.New().String()[:8]

	createQuery := `mutation($input: CreateEntryCustomInput!) {
		createEntryCustom(input: $input) {
			entry { id text senses { id definition partOfSpeech translations { id text } examples { id sentence } } card { id state } }
		}
	}`
	createVars := map[string]any{
		"input": map[string]any{
			"text": text,
			"senses": []any{
				map[string]any{
					"definition":   "A happy accident",
					"partOfSpeech": "NOUN",
					"translations": []any{"счастливая случайность"},
					"examples": []any{
						map[string]any{"sentence": "It was pure serendipity.", "translation": "Это была чистая случайность."},
					},
				},
			},
			"notes":      "custom test notes",
			"createCard": true,
		},
	}

	status, result := ts.graphqlQuery(t, createQuery, createVars, token)
	assert.Equal(t, http.StatusOK, status)
	requireNoErrors(t, result)

	entry := gqlPayload(t, result, "createEntryCustom")["entry"].(map[string]any)
	entryID := entry["id"].(string)
	assert.Equal(t, text, entry["text"])

	senses := entry["senses"].([]any)
	require.Len(t, senses, 1)

	firstSense := senses[0].(map[string]any)
	assert.Equal(t, "A happy accident", firstSense["definition"])
	assert.Equal(t, "NOUN", firstSense["partOfSpeech"])

	translations := firstSense["translations"].([]any)
	require.Len(t, translations, 1)
	assert.Equal(t, "счастливая случайность", translations[0].(map[string]any)["text"])

	examples := firstSense["examples"].([]any)
	require.Len(t, examples, 1)
	assert.Equal(t, "It was pure serendipity.", examples[0].(map[string]any)["sentence"])

	// Card should be created.
	require.NotNil(t, entry["card"])
	assert.Equal(t, "NEW", entry["card"].(map[string]any)["state"])

	// Read back via separate query.
	getQuery := `query($id: UUID!) { dictionaryEntry(id: $id) { id text notes } }`
	status, result = ts.graphqlQuery(t, getQuery, map[string]any{"id": entryID}, token)
	assert.Equal(t, http.StatusOK, status)
	requireNoErrors(t, result)

	fetched := gqlPayload(t, result, "dictionaryEntry")
	assert.Equal(t, text, fetched["text"])
	assert.Equal(t, "custom test notes", fetched["notes"])
}

// ---------------------------------------------------------------------------
// Scenario 3: Duplicate entry prevention.
// ---------------------------------------------------------------------------

func TestE2E_DuplicateEntry_ReturnsAlreadyExists(t *testing.T) {
	ts := setupTestServer(t)
	token := createTestUserAndGetToken(t, ts)

	text := "duplicate-" + uuid.New().String()[:8]

	createQuery := `mutation($input: CreateEntryCustomInput!) {
		createEntryCustom(input: $input) { entry { id } }
	}`
	input := map[string]any{
		"input": map[string]any{
			"text": text,
			"senses": []any{
				map[string]any{"definition": "First meaning"},
			},
		},
	}

	// First creation should succeed.
	status, result := ts.graphqlQuery(t, createQuery, input, token)
	assert.Equal(t, http.StatusOK, status)
	requireNoErrors(t, result)

	// Second creation with same text should fail.
	status, result = ts.graphqlQuery(t, createQuery, input, token)
	assert.Equal(t, http.StatusOK, status)
	assert.Equal(t, "ALREADY_EXISTS", gqlErrorCode(t, result))
}

// ---------------------------------------------------------------------------
// Scenario 3 (variant): Case-insensitive duplicate detection.
// ---------------------------------------------------------------------------

func TestE2E_DuplicateEntry_CaseInsensitive(t *testing.T) {
	ts := setupTestServer(t)
	token := createTestUserAndGetToken(t, ts)

	base := "CamelCase-" + uuid.New().String()[:8]

	createQuery := `mutation($input: CreateEntryCustomInput!) {
		createEntryCustom(input: $input) { entry { id text textNormalized } }
	}`

	// Create with mixed case.
	status, result := ts.graphqlQuery(t, createQuery, map[string]any{
		"input": map[string]any{
			"text":   base,
			"senses": []any{map[string]any{"definition": "def"}},
		},
	}, token)
	assert.Equal(t, http.StatusOK, status)
	requireNoErrors(t, result)

	// Try to create with lowercase — should be duplicate.
	_, result = ts.graphqlQuery(t, createQuery, map[string]any{
		"input": map[string]any{
			"text":   "  " + base + "  ", // extra spaces too
			"senses": []any{map[string]any{"definition": "def"}},
		},
	}, token)
	assert.Equal(t, "ALREADY_EXISTS", gqlErrorCode(t, result))
}

// ---------------------------------------------------------------------------
// Scenario 4: Soft delete → trash → restore → back in dictionary.
// ---------------------------------------------------------------------------

func TestE2E_SoftDeleteAndRestore_Lifecycle(t *testing.T) {
	ts := setupTestServer(t)
	token := createTestUserAndGetToken(t, ts)

	text := "ephemeral-" + uuid.New().String()[:8]

	// Create an entry.
	createQuery := `mutation($input: CreateEntryCustomInput!) {
		createEntryCustom(input: $input) { entry { id } }
	}`
	status, result := ts.graphqlQuery(t, createQuery, map[string]any{
		"input": map[string]any{
			"text":   text,
			"senses": []any{map[string]any{"definition": "short-lived"}},
		},
	}, token)
	assert.Equal(t, http.StatusOK, status)
	requireNoErrors(t, result)
	entryID := gqlPayload(t, result, "createEntryCustom")["entry"].(map[string]any)["id"].(string)

	// Delete the entry (soft delete).
	deleteQuery := `mutation($id: UUID!) { deleteEntry(id: $id) { entryId } }`
	status, result = ts.graphqlQuery(t, deleteQuery, map[string]any{"id": entryID}, token)
	assert.Equal(t, http.StatusOK, status)
	requireNoErrors(t, result)

	// Entry should NOT appear in active dictionary.
	dictQuery := `query($input: DictionaryFilterInput!) {
		dictionary(input: $input) { edges { node { id } } totalCount }
	}`
	status, result = ts.graphqlQuery(t, dictQuery, map[string]any{
		"input": map[string]any{"search": text, "limit": 10},
	}, token)
	assert.Equal(t, http.StatusOK, status)
	requireNoErrors(t, result)
	dictData := gqlPayload(t, result, "dictionary")
	for _, e := range dictData["edges"].([]any) {
		node := e.(map[string]any)["node"].(map[string]any)
		assert.NotEqual(t, entryID, node["id"], "deleted entry should not appear in dictionary")
	}

	// Entry SHOULD appear in deleted entries (trash).
	trashQuery := `query { deletedEntries(limit: 100) { entries { id text } totalCount } }`
	status, result = ts.graphqlQuery(t, trashQuery, nil, token)
	assert.Equal(t, http.StatusOK, status)
	requireNoErrors(t, result)
	trashData := gqlPayload(t, result, "deletedEntries")
	trashEntries := trashData["entries"].([]any)
	foundInTrash := false
	for _, e := range trashEntries {
		if e.(map[string]any)["id"] == entryID {
			foundInTrash = true
			break
		}
	}
	assert.True(t, foundInTrash, "deleted entry should appear in trash")

	// Restore the entry.
	restoreQuery := `mutation($id: UUID!) { restoreEntry(id: $id) { entry { id text } } }`
	status, result = ts.graphqlQuery(t, restoreQuery, map[string]any{"id": entryID}, token)
	assert.Equal(t, http.StatusOK, status)
	requireNoErrors(t, result)
	restored := gqlPayload(t, result, "restoreEntry")["entry"].(map[string]any)
	assert.Equal(t, text, restored["text"])

	// Entry should be back in active dictionary.
	getQuery := `query($id: UUID!) { dictionaryEntry(id: $id) { id text deletedAt } }`
	status, result = ts.graphqlQuery(t, getQuery, map[string]any{"id": entryID}, token)
	assert.Equal(t, http.StatusOK, status)
	requireNoErrors(t, result)
	fetched := gqlPayload(t, result, "dictionaryEntry")
	assert.Equal(t, text, fetched["text"])
	assert.Nil(t, fetched["deletedAt"], "restored entry should not have deletedAt")
}

// ---------------------------------------------------------------------------
// Scenario 4 (variant): Restore fails when duplicate active entry exists.
// ---------------------------------------------------------------------------

func TestE2E_RestoreConflict_WhenDuplicateActiveExists(t *testing.T) {
	ts := setupTestServer(t)
	token := createTestUserAndGetToken(t, ts)

	text := "conflict-" + uuid.New().String()[:8]

	createQuery := `mutation($input: CreateEntryCustomInput!) {
		createEntryCustom(input: $input) { entry { id } }
	}`
	input := func(txt string) map[string]any {
		return map[string]any{
			"input": map[string]any{
				"text":   txt,
				"senses": []any{map[string]any{"definition": "def"}},
			},
		}
	}

	// Create entry A.
	status, result := ts.graphqlQuery(t, createQuery, input(text), token)
	require.Equal(t, http.StatusOK, status)
	requireNoErrors(t, result)
	entryA := gqlPayload(t, result, "createEntryCustom")["entry"].(map[string]any)["id"].(string)

	// Delete entry A.
	deleteQuery := `mutation($id: UUID!) { deleteEntry(id: $id) { entryId } }`
	_, result = ts.graphqlQuery(t, deleteQuery, map[string]any{"id": entryA}, token)
	requireNoErrors(t, result)

	// Create entry B with the same text (now that A is soft-deleted, this works).
	status, result = ts.graphqlQuery(t, createQuery, input(text), token)
	require.Equal(t, http.StatusOK, status)
	requireNoErrors(t, result)

	// Try to restore A — should fail because B occupies the same text_normalized.
	restoreQuery := `mutation($id: UUID!) { restoreEntry(id: $id) { entry { id } } }`
	status, result = ts.graphqlQuery(t, restoreQuery, map[string]any{"id": entryA}, token)
	assert.Equal(t, http.StatusOK, status)
	assert.Equal(t, "ALREADY_EXISTS", gqlErrorCode(t, result))
}

// ---------------------------------------------------------------------------
// Update entry notes.
// ---------------------------------------------------------------------------

func TestE2E_UpdateEntryNotes(t *testing.T) {
	ts := setupTestServer(t)
	token := createTestUserAndGetToken(t, ts)

	text := "noted-" + uuid.New().String()[:8]

	// Create entry.
	createQuery := `mutation($input: CreateEntryCustomInput!) {
		createEntryCustom(input: $input) { entry { id } }
	}`
	status, result := ts.graphqlQuery(t, createQuery, map[string]any{
		"input": map[string]any{
			"text":   text,
			"senses": []any{map[string]any{"definition": "test"}},
		},
	}, token)
	require.Equal(t, http.StatusOK, status)
	requireNoErrors(t, result)
	entryID := gqlPayload(t, result, "createEntryCustom")["entry"].(map[string]any)["id"].(string)

	// Update notes.
	updateQuery := `mutation($input: UpdateEntryNotesInput!) {
		updateEntryNotes(input: $input) { entry { id notes } }
	}`
	status, result = ts.graphqlQuery(t, updateQuery, map[string]any{
		"input": map[string]any{
			"entryId": entryID,
			"notes":   "These are my updated notes",
		},
	}, token)
	assert.Equal(t, http.StatusOK, status)
	requireNoErrors(t, result)

	// Verify via separate query.
	getQuery := `query($id: UUID!) { dictionaryEntry(id: $id) { notes } }`
	status, result = ts.graphqlQuery(t, getQuery, map[string]any{"id": entryID}, token)
	assert.Equal(t, http.StatusOK, status)
	requireNoErrors(t, result)
	assert.Equal(t, "These are my updated notes", gqlPayload(t, result, "dictionaryEntry")["notes"])
}

// ---------------------------------------------------------------------------
// Scenario 12: Import entries — deduplication and correct counts.
// ---------------------------------------------------------------------------

func TestE2E_ImportEntries_DeduplicatesAndCreates(t *testing.T) {
	ts := setupTestServer(t)
	token := createTestUserAndGetToken(t, ts)

	suffix := uuid.New().String()[:8]

	importQuery := `mutation($input: ImportEntriesInput!) {
		importEntries(input: $input) { importedCount skippedCount errors { index text message } }
	}`
	importVars := map[string]any{
		"input": map[string]any{
			"items": []any{
				map[string]any{"text": "import-a-" + suffix, "translations": []any{"перевод А"}},
				map[string]any{"text": "import-b-" + suffix, "translations": []any{"перевод Б"}},
				map[string]any{"text": "import-a-" + suffix, "translations": []any{"дубликат"}}, // duplicate
			},
		},
	}

	status, result := ts.graphqlQuery(t, importQuery, importVars, token)
	assert.Equal(t, http.StatusOK, status)
	requireNoErrors(t, result)

	payload := gqlPayload(t, result, "importEntries")
	// 2 unique items imported, 1 skipped as duplicate within the batch.
	assert.Equal(t, float64(2), payload["importedCount"])
	assert.Equal(t, float64(1), payload["skippedCount"])

	// Verify both entries exist in dictionary.
	dictQuery := `query($input: DictionaryFilterInput!) {
		dictionary(input: $input) { totalCount }
	}`
	status, result = ts.graphqlQuery(t, dictQuery, map[string]any{
		"input": map[string]any{"search": "import-", "limit": 10},
	}, token)
	assert.Equal(t, http.StatusOK, status)
	requireNoErrors(t, result)

	totalCount := gqlPayload(t, result, "dictionary")["totalCount"].(float64)
	assert.GreaterOrEqual(t, totalCount, float64(2))
}

// ---------------------------------------------------------------------------
// Scenario 12: Export entries — returns complete data.
// ---------------------------------------------------------------------------

func TestE2E_ExportEntries_ReturnsCompleteData(t *testing.T) {
	ts := setupTestServer(t)
	token, userID := createTestUserWithID(t, ts)

	// Seed entry with card via DB so we have known data.
	refEntry := testhelper.SeedRefEntry(t, ts.Pool, "export-"+uuid.New().String()[:8])
	testhelper.SeedEntryWithCard(t, ts.Pool, userID, refEntry.ID)

	exportQuery := `query {
		exportEntries {
			items { text notes senses { definition translations } cardStatus }
			exportedAt
		}
	}`
	status, result := ts.graphqlQuery(t, exportQuery, nil, token)
	assert.Equal(t, http.StatusOK, status)
	requireNoErrors(t, result)

	exportData := gqlPayload(t, result, "exportEntries")
	items := exportData["items"].([]any)
	assert.NotEmpty(t, items, "export should include seeded entry")

	assert.NotNil(t, exportData["exportedAt"], "exportedAt should be set")

	// Find our seeded entry.
	found := false
	for _, item := range items {
		iMap := item.(map[string]any)
		if iMap["text"] == refEntry.Text {
			found = true
			assert.Equal(t, "NEW", iMap["cardStatus"])
			senses := iMap["senses"].([]any)
			assert.NotEmpty(t, senses)
			break
		}
	}
	assert.True(t, found, "exported items should include seeded entry")
}
