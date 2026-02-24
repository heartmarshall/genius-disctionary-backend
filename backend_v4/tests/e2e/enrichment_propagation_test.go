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

// TestE2E_EnrichmentPropagation verifies the FK SET NULL behavior when
// ref_senses are replaced during LLM enrichment.
//
// Scenario:
//  1. Seed a ref_entry with Wiktionary senses (definition, POS).
//  2. User creates an entry from catalog → user senses link via ref_sense_id.
//  3. Verify user sees Wiktionary definitions via COALESCE.
//  4. Simulate LLM enrichment: DELETE old ref_senses, INSERT new ones.
//  5. Query user entry again.
//  6. Verify: user senses now have ref_sense_id = NULL (FK SET NULL)
//     and definitions are NULL (no local definition, ref link broken).
//
// This documents KNOWN BEHAVIOR: users who added words BEFORE enrichment
// need to re-sync their entries to see updated definitions.
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
	// Step 4: Simulate LLM enrichment — delete old ref_senses, insert new.
	// Old ref_senses are deleted; FK ON DELETE SET NULL nullifies user links.
	// -----------------------------------------------------------------------
	_, err = ts.Pool.Exec(ctx,
		`DELETE FROM ref_senses WHERE ref_entry_id = $1`,
		refEntryID,
	)
	require.NoError(t, err, "delete old ref_senses")

	// Insert replacement LLM-enriched senses (new IDs).
	newSenseID := uuid.New()
	_, err = ts.Pool.Exec(ctx,
		`INSERT INTO ref_senses (id, ref_entry_id, definition, part_of_speech, source_slug, position, created_at)
		 VALUES ($1, $2, $3, $4, $5, $6, $7)`,
		newSenseID, refEntryID, "articulate and persuasive in expression (LLM-enriched)", "adjective", "llm", 0, now,
	)
	require.NoError(t, err, "insert LLM ref_sense")

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
	// Step 6: Verify FK SET NULL behavior.
	// User senses now have ref_sense_id = NULL. Since user has no local
	// definition (only ref link), COALESCE returns NULL.
	// -----------------------------------------------------------------------
	for i, s := range senses {
		senseMap := s.(map[string]any)
		// definition is nil because: COALESCE(user.definition=NULL, ref.definition=NULL)
		// ref.definition is NULL because LEFT JOIN on ref_sense_id=NULL yields no match.
		assert.Nil(t, senseMap["definition"],
			"sense %d: definition should be nil after ref_senses replaced (FK SET NULL)", i)
	}

	// Verify in DB: ref_sense_id is now NULL on all user senses.
	var nullCount int
	err = ts.Pool.QueryRow(ctx,
		`SELECT count(*) FROM senses WHERE entry_id = $1 AND ref_sense_id IS NULL`,
		entryID,
	).Scan(&nullCount)
	require.NoError(t, err)
	assert.Equal(t, 2, nullCount,
		"all user senses should have ref_sense_id = NULL after replacement")

	// Verify new LLM ref_senses exist (available for re-sync).
	var newRefCount int
	err = ts.Pool.QueryRow(ctx,
		`SELECT count(*) FROM ref_senses WHERE ref_entry_id = $1 AND source_slug = 'llm'`,
		refEntryID,
	).Scan(&newRefCount)
	require.NoError(t, err)
	assert.Equal(t, 1, newRefCount, "new LLM ref_sense should exist")
}
