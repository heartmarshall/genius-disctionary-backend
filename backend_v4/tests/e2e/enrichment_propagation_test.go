//go:build e2e

package e2e_test

import (
	"context"
	"net/http"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestE2E_EnrichmentPropagation verifies the upsert behavior when ref_senses
// are enriched by the LLM pipeline.
//
// Scenario:
//  1. Seed a ref_entry with Wiktionary senses.
//  2. User creates an entry from catalog → user senses link via ref_sense_id.
//  3. Verify user sees Wiktionary definitions via COALESCE.
//  4. Simulate LLM enrichment via ReplaceEntryContent (upsert strategy):
//     - Existing senses are updated in-place (UUIDs preserved).
//     - New senses are inserted.
//  5. Query user entry again.
//  6. Verify: user senses still have ref_sense_id NOT NULL and see LLM-enriched definitions.
func TestE2E_EnrichmentPropagation(t *testing.T) {
	ts := setupTestServer(t)
	token, _ := createTestUserWithID(t, ts)
	ctx := context.Background()

	// -----------------------------------------------------------------------
	// Step 1: Seed ref_entry + ref_senses directly in DB.
	// -----------------------------------------------------------------------
	refEntryID := uuid.New()
	refSense1ID := uuid.New()
	refSense2ID := uuid.New()
	now := time.Now().UTC().Truncate(time.Microsecond)

	_, err := ts.Pool.Exec(ctx,
		`INSERT INTO ref_entries (id, text, text_normalized, is_core_lexicon, created_at)
		 VALUES ($1, $2, $3, $4, $5)`,
		refEntryID, "eloquent", "eloquent", true, now,
	)
	require.NoError(t, err, "seed ref_entry")

	_, err = ts.Pool.Exec(ctx,
		`INSERT INTO ref_senses (id, ref_entry_id, definition, part_of_speech, source_slug, position, created_at)
		 VALUES ($1, $2, $3, $4, $5, $6, $7)`,
		refSense1ID, refEntryID, "fluent or persuasive in speaking", "adjective", "wiktionary", 0, now,
	)
	require.NoError(t, err, "seed ref_sense 1")

	_, err = ts.Pool.Exec(ctx,
		`INSERT INTO ref_senses (id, ref_entry_id, definition, part_of_speech, source_slug, position, created_at)
		 VALUES ($1, $2, $3, $4, $5, $6, $7)`,
		refSense2ID, refEntryID, "expressing strong emotion", "adjective", "wiktionary", 1, now,
	)
	require.NoError(t, err, "seed ref_sense 2")

	// -----------------------------------------------------------------------
	// Step 2: Create user entry from catalog via GraphQL.
	// -----------------------------------------------------------------------
	createMutation := `mutation($input: CreateEntryFromCatalogInput!) {
		createEntryFromCatalog(input: $input) {
			entry {
				id
				text
				senses {
					id
					definition
					partOfSpeech
				}
			}
		}
	}`
	createVars := map[string]any{
		"input": map[string]any{
			"refEntryId": refEntryID.String(),
			"senseIds":   []string{},
		},
	}

	status, result := ts.graphqlQuery(t, createMutation, createVars, token)
	assert.Equal(t, http.StatusOK, status)
	requireNoErrors(t, result)

	payload := gqlPayload(t, result, "createEntryFromCatalog")
	entryData := payload["entry"].(map[string]any)
	entryID := entryData["id"].(string)
	assert.Equal(t, "eloquent", entryData["text"])

	// -----------------------------------------------------------------------
	// Step 3: Verify user sees Wiktionary definitions via COALESCE.
	// -----------------------------------------------------------------------
	senses := entryData["senses"].([]any)
	require.Len(t, senses, 2, "expected 2 senses from catalog")

	sense0 := senses[0].(map[string]any)
	sense1 := senses[1].(map[string]any)
	assert.Equal(t, "fluent or persuasive in speaking", sense0["definition"])
	assert.Equal(t, "expressing strong emotion", sense1["definition"])

	// Verify ref_sense_id is set in DB.
	var refSenseLinked int
	err = ts.Pool.QueryRow(ctx,
		`SELECT count(*) FROM senses WHERE entry_id = $1 AND ref_sense_id IS NOT NULL`,
		entryID,
	).Scan(&refSenseLinked)
	require.NoError(t, err)
	assert.Equal(t, 2, refSenseLinked, "both senses should link to ref_senses")

	// -----------------------------------------------------------------------
	// Step 4: Simulate LLM enrichment — upsert ref_senses in-place.
	// UUIDs are preserved, so user FK links remain valid.
	// -----------------------------------------------------------------------

	// UPDATE sense at position 0: enriched definition, same UUID stays.
	_, err = ts.Pool.Exec(ctx,
		`UPDATE ref_senses SET definition = $1, source_slug = 'llm'
		 WHERE id = $2`,
		"articulate and persuasive in expression (LLM-enriched)", refSense1ID,
	)
	require.NoError(t, err, "upsert ref_sense 1")

	// UPDATE sense at position 1: enriched definition, same UUID stays.
	_, err = ts.Pool.Exec(ctx,
		`UPDATE ref_senses SET definition = $1, source_slug = 'llm'
		 WHERE id = $2`,
		"vividly expressing strong feelings (LLM-enriched)", refSense2ID,
	)
	require.NoError(t, err, "upsert ref_sense 2")

	// INSERT new sense at position 2 (didn't exist before).
	newSenseID := uuid.New()
	_, err = ts.Pool.Exec(ctx,
		`INSERT INTO ref_senses (id, ref_entry_id, definition, part_of_speech, source_slug, position, created_at)
		 VALUES ($1, $2, $3, $4, $5, $6, $7)`,
		newSenseID, refEntryID, "powerfully moving (LLM-enriched)", "adjective", "llm", 2, now,
	)
	require.NoError(t, err, "insert new LLM ref_sense")

	// -----------------------------------------------------------------------
	// Step 5: Query user entry again.
	// -----------------------------------------------------------------------
	getQuery := `query($id: UUID!) {
		dictionaryEntry(id: $id) {
			id
			text
			senses {
				id
				definition
				partOfSpeech
			}
		}
	}`
	getVars := map[string]any{"id": entryID}

	status, result = ts.graphqlQuery(t, getQuery, getVars, token)
	assert.Equal(t, http.StatusOK, status)
	requireNoErrors(t, result)

	entryData = gqlPayload(t, result, "dictionaryEntry")
	senses = entryData["senses"].([]any)
	require.Len(t, senses, 2, "user still has 2 senses (rows not deleted)")

	// -----------------------------------------------------------------------
	// Step 6: Verify upsert behavior — FK preserved, LLM definitions visible.
	// -----------------------------------------------------------------------

	// User senses should now show LLM-enriched definitions via COALESCE.
	sense0 = senses[0].(map[string]any)
	sense1 = senses[1].(map[string]any)
	assert.Equal(t, "articulate and persuasive in expression (LLM-enriched)",
		sense0["definition"], "sense 0: should show LLM-enriched definition")
	assert.Equal(t, "vividly expressing strong feelings (LLM-enriched)",
		sense1["definition"], "sense 1: should show LLM-enriched definition")

	// Verify in DB: ref_sense_id is still NOT NULL on all user senses.
	var linkedCount int
	err = ts.Pool.QueryRow(ctx,
		`SELECT count(*) FROM senses WHERE entry_id = $1 AND ref_sense_id IS NOT NULL`,
		entryID,
	).Scan(&linkedCount)
	require.NoError(t, err)
	assert.Equal(t, 2, linkedCount,
		"all user senses should still have ref_sense_id NOT NULL after upsert")

	// Verify new LLM ref_senses exist (3 total: 2 updated + 1 new).
	var newRefCount int
	err = ts.Pool.QueryRow(ctx,
		`SELECT count(*) FROM ref_senses WHERE ref_entry_id = $1`,
		refEntryID,
	).Scan(&newRefCount)
	require.NoError(t, err)
	assert.Equal(t, 3, newRefCount, "should have 3 ref_senses (2 updated + 1 new)")
}

// TestE2E_EnrichmentPropagation_ExcessDeletion verifies that when LLM enrichment
// reduces the number of senses, excess old senses are deleted and FK SET NULL fires.
func TestE2E_EnrichmentPropagation_ExcessDeletion(t *testing.T) {
	ts := setupTestServer(t)
	token, _ := createTestUserWithID(t, ts)
	ctx := context.Background()

	// Seed ref_entry + 3 ref_senses.
	refEntryID := uuid.New()
	refSenseIDs := [3]uuid.UUID{uuid.New(), uuid.New(), uuid.New()}
	now := time.Now().UTC().Truncate(time.Microsecond)

	_, err := ts.Pool.Exec(ctx,
		`INSERT INTO ref_entries (id, text, text_normalized, is_core_lexicon, created_at)
		 VALUES ($1, $2, $3, $4, $5)`,
		refEntryID, "bright", "bright", true, now,
	)
	require.NoError(t, err)

	for i, id := range refSenseIDs {
		_, err = ts.Pool.Exec(ctx,
			`INSERT INTO ref_senses (id, ref_entry_id, definition, part_of_speech, source_slug, position, created_at)
			 VALUES ($1, $2, $3, $4, $5, $6, $7)`,
			id, refEntryID, []string{"shining with light", "intelligent", "vivid in color"}[i],
			"adjective", "wiktionary", i, now,
		)
		require.NoError(t, err)
	}

	// Create user entry from catalog.
	createMutation := `mutation($input: CreateEntryFromCatalogInput!) {
		createEntryFromCatalog(input: $input) {
			entry { id senses { id definition } }
		}
	}`
	status, result := ts.graphqlQuery(t, createMutation, map[string]any{
		"input": map[string]any{"refEntryId": refEntryID.String(), "senseIds": []string{}},
	}, token)
	assert.Equal(t, http.StatusOK, status)
	requireNoErrors(t, result)

	payload := gqlPayload(t, result, "createEntryFromCatalog")
	entryID := payload["entry"].(map[string]any)["id"].(string)

	// Verify 3 linked senses.
	var linkedBefore int
	err = ts.Pool.QueryRow(ctx,
		`SELECT count(*) FROM senses WHERE entry_id = $1 AND ref_sense_id IS NOT NULL`,
		entryID,
	).Scan(&linkedBefore)
	require.NoError(t, err)
	assert.Equal(t, 3, linkedBefore)

	// Simulate upsert that reduces senses from 3 to 1.
	// Sense at position 0: UPDATE in-place.
	_, err = ts.Pool.Exec(ctx,
		`UPDATE ref_senses SET definition = $1, source_slug = 'llm' WHERE id = $2`,
		"emitting or reflecting light (LLM)", refSenseIDs[0],
	)
	require.NoError(t, err)

	// Delete senses at positions 1 and 2 (excess).
	_, err = ts.Pool.Exec(ctx,
		`DELETE FROM ref_senses WHERE id = ANY($1)`,
		[]uuid.UUID{refSenseIDs[1], refSenseIDs[2]},
	)
	require.NoError(t, err)

	// Verify results.
	var linkedAfter, nullAfter int
	err = ts.Pool.QueryRow(ctx,
		`SELECT count(*) FROM senses WHERE entry_id = $1 AND ref_sense_id IS NOT NULL`,
		entryID,
	).Scan(&linkedAfter)
	require.NoError(t, err)
	assert.Equal(t, 1, linkedAfter, "1 sense should still be linked after upsert")

	err = ts.Pool.QueryRow(ctx,
		`SELECT count(*) FROM senses WHERE entry_id = $1 AND ref_sense_id IS NULL`,
		entryID,
	).Scan(&nullAfter)
	require.NoError(t, err)
	assert.Equal(t, 2, nullAfter, "2 senses should have ref_sense_id = NULL (excess deleted)")

	// The linked sense should show the LLM-enriched definition.
	getQuery := `query($id: UUID!) {
		dictionaryEntry(id: $id) { senses { definition } }
	}`
	status, result = ts.graphqlQuery(t, getQuery, map[string]any{"id": entryID}, token)
	assert.Equal(t, http.StatusOK, status)
	requireNoErrors(t, result)

	entryData := gqlPayload(t, result, "dictionaryEntry")
	senses := entryData["senses"].([]any)
	require.Len(t, senses, 3, "user still has 3 senses (rows not deleted from senses table)")

	// At least one sense should show the enriched definition.
	var foundEnriched bool
	for _, s := range senses {
		def := s.(map[string]any)["definition"]
		if def == "emitting or reflecting light (LLM)" {
			foundEnriched = true
		}
	}
	assert.True(t, foundEnriched, "should find the LLM-enriched definition in user senses")
}
