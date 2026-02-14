package testhelper

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/heartmarshall/myenglish-backend/internal/domain"
)

// uniqueSuffix returns a short unique string for generating non-conflicting test data.
func uniqueSuffix() string {
	return uuid.New().String()[:8]
}

// SeedUser creates a user and user_settings with default values.
// Returns a filled domain.User.
func SeedUser(t *testing.T, pool *pgxpool.Pool) domain.User {
	t.Helper()
	ctx := context.Background()

	suffix := uniqueSuffix()
	now := time.Now().UTC().Truncate(time.Microsecond)
	user := domain.User{
		ID:            uuid.New(),
		Email:         "testuser-" + suffix + "@example.com",
		Name:          "Test User " + suffix,
		OAuthProvider: domain.OAuthProviderGoogle,
		OAuthID:       "oauth-" + suffix,
		CreatedAt:     now,
		UpdatedAt:     now,
	}

	_, err := pool.Exec(ctx,
		`INSERT INTO users (id, email, name, oauth_provider, oauth_id, created_at, updated_at)
		 VALUES ($1, $2, $3, $4, $5, $6, $7)`,
		user.ID, user.Email, user.Name, string(user.OAuthProvider), user.OAuthID, user.CreatedAt, user.UpdatedAt,
	)
	if err != nil {
		t.Fatalf("testhelper: SeedUser insert user: %v", err)
	}

	settings := domain.DefaultUserSettings(user.ID)
	settings.UpdatedAt = now

	_, err = pool.Exec(ctx,
		`INSERT INTO user_settings (user_id, new_cards_per_day, reviews_per_day, max_interval_days, timezone, updated_at)
		 VALUES ($1, $2, $3, $4, $5, $6)`,
		settings.UserID, settings.NewCardsPerDay, settings.ReviewsPerDay, settings.MaxIntervalDays, settings.Timezone, settings.UpdatedAt,
	)
	if err != nil {
		t.Fatalf("testhelper: SeedUser insert user_settings: %v", err)
	}

	return user
}

// SeedRefEntry creates a ref_entry with 2 ref_senses (each having 2 ref_translations and
// 2 ref_examples) and 2 ref_pronunciations. Returns a fully populated domain.RefEntry.
func SeedRefEntry(t *testing.T, pool *pgxpool.Pool, text string) domain.RefEntry {
	t.Helper()
	ctx := context.Background()

	suffix := uniqueSuffix()
	now := time.Now().UTC().Truncate(time.Microsecond)

	refEntry := domain.RefEntry{
		ID:             uuid.New(),
		Text:           text,
		TextNormalized: domain.NormalizeText(text),
		CreatedAt:      now,
	}

	_, err := pool.Exec(ctx,
		`INSERT INTO ref_entries (id, text, text_normalized, created_at)
		 VALUES ($1, $2, $3, $4)`,
		refEntry.ID, refEntry.Text, refEntry.TextNormalized, refEntry.CreatedAt,
	)
	if err != nil {
		t.Fatalf("testhelper: SeedRefEntry insert ref_entry: %v", err)
	}

	// Create 2 senses.
	posNoun := domain.PartOfSpeechNoun
	posVerb := domain.PartOfSpeechVerb
	cefrB1 := "B1"
	cefrB2 := "B2"

	senseConfigs := []struct {
		pos  *domain.PartOfSpeech
		cefr *string
	}{
		{pos: &posNoun, cefr: &cefrB1},
		{pos: &posVerb, cefr: &cefrB2},
	}

	refEntry.Senses = make([]domain.RefSense, 2)
	for i, cfg := range senseConfigs {
		sense := domain.RefSense{
			ID:           uuid.New(),
			RefEntryID:   refEntry.ID,
			Definition:   "Definition " + suffix + "-" + string(rune('A'+i)),
			PartOfSpeech: cfg.pos,
			CEFRLevel:    cfg.cefr,
			SourceSlug:   "test-source",
			Position:     i,
			CreatedAt:    now,
		}

		_, err := pool.Exec(ctx,
			`INSERT INTO ref_senses (id, ref_entry_id, definition, part_of_speech, cefr_level, source_slug, position, created_at)
			 VALUES ($1, $2, $3, $4, $5, $6, $7, $8)`,
			sense.ID, sense.RefEntryID, sense.Definition, (*string)(sense.PartOfSpeech), sense.CEFRLevel, sense.SourceSlug, sense.Position, sense.CreatedAt,
		)
		if err != nil {
			t.Fatalf("testhelper: SeedRefEntry insert ref_sense[%d]: %v", i, err)
		}

		// 2 translations per sense.
		sense.Translations = make([]domain.RefTranslation, 2)
		for j := 0; j < 2; j++ {
			tr := domain.RefTranslation{
				ID:         uuid.New(),
				RefSenseID: sense.ID,
				Text:       "Translation " + suffix + "-" + string(rune('A'+i)) + string(rune('1'+j)),
				SourceSlug: "test-source",
				Position:   j,
			}

			_, err := pool.Exec(ctx,
				`INSERT INTO ref_translations (id, ref_sense_id, text, source_slug, position)
				 VALUES ($1, $2, $3, $4, $5)`,
				tr.ID, tr.RefSenseID, tr.Text, tr.SourceSlug, tr.Position,
			)
			if err != nil {
				t.Fatalf("testhelper: SeedRefEntry insert ref_translation[%d][%d]: %v", i, j, err)
			}
			sense.Translations[j] = tr
		}

		// 2 examples per sense.
		sense.Examples = make([]domain.RefExample, 2)
		for j := 0; j < 2; j++ {
			exTranslation := "Example translation " + suffix + "-" + string(rune('A'+i)) + string(rune('1'+j))
			ex := domain.RefExample{
				ID:          uuid.New(),
				RefSenseID:  sense.ID,
				Sentence:    "Example sentence " + suffix + "-" + string(rune('A'+i)) + string(rune('1'+j)),
				Translation: &exTranslation,
				SourceSlug:  "test-source",
				Position:    j,
			}

			_, err := pool.Exec(ctx,
				`INSERT INTO ref_examples (id, ref_sense_id, sentence, translation, source_slug, position)
				 VALUES ($1, $2, $3, $4, $5, $6)`,
				ex.ID, ex.RefSenseID, ex.Sentence, ex.Translation, ex.SourceSlug, ex.Position,
			)
			if err != nil {
				t.Fatalf("testhelper: SeedRefEntry insert ref_example[%d][%d]: %v", i, j, err)
			}
			sense.Examples[j] = ex
		}

		refEntry.Senses[i] = sense
	}

	// Create 2 pronunciations.
	refEntry.Pronunciations = make([]domain.RefPronunciation, 2)
	regions := []string{"us", "uk"}
	for i, region := range regions {
		transcription := "/" + text + "-" + region + "/"
		audioURL := "https://example.com/audio/" + suffix + "-" + region + ".mp3"
		r := region
		pron := domain.RefPronunciation{
			ID:            uuid.New(),
			RefEntryID:    refEntry.ID,
			Transcription: &transcription,
			AudioURL:      &audioURL,
			Region:        &r,
			SourceSlug:    "test-source",
		}

		_, err := pool.Exec(ctx,
			`INSERT INTO ref_pronunciations (id, ref_entry_id, transcription, audio_url, region, source_slug)
			 VALUES ($1, $2, $3, $4, $5, $6)`,
			pron.ID, pron.RefEntryID, pron.Transcription, pron.AudioURL, pron.Region, pron.SourceSlug,
		)
		if err != nil {
			t.Fatalf("testhelper: SeedRefEntry insert ref_pronunciation[%d]: %v", i, err)
		}
		refEntry.Pronunciations[i] = pron
	}

	return refEntry
}

// SeedEntry creates a user entry linked to a ref_entry, with senses (linked to ref),
// translations, examples, and pronunciations (M2M). Does NOT create a card.
// Returns a filled domain.Entry.
func SeedEntry(t *testing.T, pool *pgxpool.Pool, userID uuid.UUID, refEntryID uuid.UUID) domain.Entry {
	t.Helper()
	ctx := context.Background()

	// First fetch the ref_entry to build linked senses.
	refEntry := fetchRefEntry(t, pool, refEntryID)

	now := time.Now().UTC().Truncate(time.Microsecond)
	entry := domain.Entry{
		ID:             uuid.New(),
		UserID:         userID,
		RefEntryID:     &refEntryID,
		Text:           refEntry.Text,
		TextNormalized: refEntry.TextNormalized,
		CreatedAt:      now,
		UpdatedAt:      now,
	}

	_, err := pool.Exec(ctx,
		`INSERT INTO entries (id, user_id, ref_entry_id, text, text_normalized, created_at, updated_at)
		 VALUES ($1, $2, $3, $4, $5, $6, $7)`,
		entry.ID, entry.UserID, entry.RefEntryID, entry.Text, entry.TextNormalized, entry.CreatedAt, entry.UpdatedAt,
	)
	if err != nil {
		t.Fatalf("testhelper: SeedEntry insert entry: %v", err)
	}

	// Create senses linked to ref_senses.
	entry.Senses = make([]domain.Sense, len(refEntry.Senses))
	for i, rs := range refEntry.Senses {
		rsID := rs.ID
		sense := domain.Sense{
			ID:         uuid.New(),
			EntryID:    entry.ID,
			RefSenseID: &rsID,
			SourceSlug: rs.SourceSlug,
			Position:   rs.Position,
			CreatedAt:  now,
		}

		_, err := pool.Exec(ctx,
			`INSERT INTO senses (id, entry_id, ref_sense_id, source_slug, position, created_at)
			 VALUES ($1, $2, $3, $4, $5, $6)`,
			sense.ID, sense.EntryID, sense.RefSenseID, sense.SourceSlug, sense.Position, sense.CreatedAt,
		)
		if err != nil {
			t.Fatalf("testhelper: SeedEntry insert sense[%d]: %v", i, err)
		}

		// Create translations linked to ref_translations.
		sense.Translations = make([]domain.Translation, len(rs.Translations))
		for j, rt := range rs.Translations {
			rtID := rt.ID
			tr := domain.Translation{
				ID:               uuid.New(),
				SenseID:          sense.ID,
				RefTranslationID: &rtID,
				SourceSlug:       rt.SourceSlug,
				Position:         rt.Position,
			}

			_, err := pool.Exec(ctx,
				`INSERT INTO translations (id, sense_id, ref_translation_id, source_slug, position)
				 VALUES ($1, $2, $3, $4, $5)`,
				tr.ID, tr.SenseID, tr.RefTranslationID, tr.SourceSlug, tr.Position,
			)
			if err != nil {
				t.Fatalf("testhelper: SeedEntry insert translation[%d][%d]: %v", i, j, err)
			}
			sense.Translations[j] = tr
		}

		// Create examples linked to ref_examples.
		sense.Examples = make([]domain.Example, len(rs.Examples))
		for j, re := range rs.Examples {
			reID := re.ID
			ex := domain.Example{
				ID:           uuid.New(),
				SenseID:      sense.ID,
				RefExampleID: &reID,
				SourceSlug:   re.SourceSlug,
				Position:     re.Position,
				CreatedAt:    now,
			}

			_, err := pool.Exec(ctx,
				`INSERT INTO examples (id, sense_id, ref_example_id, source_slug, position, created_at)
				 VALUES ($1, $2, $3, $4, $5, $6)`,
				ex.ID, ex.SenseID, ex.RefExampleID, ex.SourceSlug, ex.Position, ex.CreatedAt,
			)
			if err != nil {
				t.Fatalf("testhelper: SeedEntry insert example[%d][%d]: %v", i, j, err)
			}
			sense.Examples[j] = ex
		}

		entry.Senses[i] = sense
	}

	// Link pronunciations via M2M table.
	entry.Pronunciations = make([]domain.RefPronunciation, len(refEntry.Pronunciations))
	for i, p := range refEntry.Pronunciations {
		_, err := pool.Exec(ctx,
			`INSERT INTO entry_pronunciations (entry_id, ref_pronunciation_id) VALUES ($1, $2)`,
			entry.ID, p.ID,
		)
		if err != nil {
			t.Fatalf("testhelper: SeedEntry insert entry_pronunciation[%d]: %v", i, err)
		}
		entry.Pronunciations[i] = p
	}

	return entry
}

// SeedEntryWithCard creates an entry (same as SeedEntry) plus a card with status NEW.
// Returns a filled domain.Entry with Card populated.
func SeedEntryWithCard(t *testing.T, pool *pgxpool.Pool, userID uuid.UUID, refEntryID uuid.UUID) domain.Entry {
	t.Helper()
	ctx := context.Background()

	entry := SeedEntry(t, pool, userID, refEntryID)

	now := time.Now().UTC().Truncate(time.Microsecond)
	card := domain.Card{
		ID:           uuid.New(),
		UserID:       userID,
		EntryID:      entry.ID,
		Status:       domain.LearningStatusNew,
		LearningStep: 0,
		IntervalDays: 0,
		EaseFactor:   2.5,
		CreatedAt:    now,
		UpdatedAt:    now,
	}

	_, err := pool.Exec(ctx,
		`INSERT INTO cards (id, user_id, entry_id, status, learning_step, interval_days, ease_factor, created_at, updated_at)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)`,
		card.ID, card.UserID, card.EntryID, string(card.Status), card.LearningStep, card.IntervalDays, card.EaseFactor, card.CreatedAt, card.UpdatedAt,
	)
	if err != nil {
		t.Fatalf("testhelper: SeedEntryWithCard insert card: %v", err)
	}

	entry.Card = &card
	return entry
}

// SeedEntryCustom creates a user entry with custom senses (no ref links).
// Senses have filled definition and part_of_speech. Returns a filled domain.Entry.
func SeedEntryCustom(t *testing.T, pool *pgxpool.Pool, userID uuid.UUID) domain.Entry {
	t.Helper()
	ctx := context.Background()

	suffix := uniqueSuffix()
	now := time.Now().UTC().Truncate(time.Microsecond)
	text := "custom-" + suffix

	entry := domain.Entry{
		ID:             uuid.New(),
		UserID:         userID,
		Text:           text,
		TextNormalized: domain.NormalizeText(text),
		CreatedAt:      now,
		UpdatedAt:      now,
	}

	_, err := pool.Exec(ctx,
		`INSERT INTO entries (id, user_id, text, text_normalized, created_at, updated_at)
		 VALUES ($1, $2, $3, $4, $5, $6)`,
		entry.ID, entry.UserID, entry.Text, entry.TextNormalized, entry.CreatedAt, entry.UpdatedAt,
	)
	if err != nil {
		t.Fatalf("testhelper: SeedEntryCustom insert entry: %v", err)
	}

	// Create 2 custom senses with definitions and POS.
	posNoun := domain.PartOfSpeechNoun
	posVerb := domain.PartOfSpeechVerb
	defA := "Custom definition A " + suffix
	defB := "Custom definition B " + suffix

	senseConfigs := []struct {
		def *string
		pos *domain.PartOfSpeech
	}{
		{def: &defA, pos: &posNoun},
		{def: &defB, pos: &posVerb},
	}

	entry.Senses = make([]domain.Sense, len(senseConfigs))
	for i, cfg := range senseConfigs {
		sense := domain.Sense{
			ID:           uuid.New(),
			EntryID:      entry.ID,
			Definition:   cfg.def,
			PartOfSpeech: cfg.pos,
			SourceSlug:   "user",
			Position:     i,
			CreatedAt:    now,
		}

		_, err := pool.Exec(ctx,
			`INSERT INTO senses (id, entry_id, definition, part_of_speech, source_slug, position, created_at)
			 VALUES ($1, $2, $3, $4, $5, $6, $7)`,
			sense.ID, sense.EntryID, sense.Definition, (*string)(sense.PartOfSpeech), sense.SourceSlug, sense.Position, sense.CreatedAt,
		)
		if err != nil {
			t.Fatalf("testhelper: SeedEntryCustom insert sense[%d]: %v", i, err)
		}

		// Add 1 custom translation per sense.
		trText := "Custom translation " + suffix + "-" + string(rune('A'+i))
		tr := domain.Translation{
			ID:         uuid.New(),
			SenseID:    sense.ID,
			Text:       &trText,
			SourceSlug: "user",
			Position:   0,
		}

		_, err = pool.Exec(ctx,
			`INSERT INTO translations (id, sense_id, text, source_slug, position)
			 VALUES ($1, $2, $3, $4, $5)`,
			tr.ID, tr.SenseID, tr.Text, tr.SourceSlug, tr.Position,
		)
		if err != nil {
			t.Fatalf("testhelper: SeedEntryCustom insert translation[%d]: %v", i, err)
		}
		sense.Translations = []domain.Translation{tr}

		// Add 1 custom example per sense.
		exSentence := "Custom example " + suffix + "-" + string(rune('A'+i))
		ex := domain.Example{
			ID:         uuid.New(),
			SenseID:    sense.ID,
			Sentence:   &exSentence,
			SourceSlug: "user",
			Position:   0,
			CreatedAt:  now,
		}

		_, err = pool.Exec(ctx,
			`INSERT INTO examples (id, sense_id, sentence, source_slug, position, created_at)
			 VALUES ($1, $2, $3, $4, $5, $6)`,
			ex.ID, ex.SenseID, ex.Sentence, ex.SourceSlug, ex.Position, ex.CreatedAt,
		)
		if err != nil {
			t.Fatalf("testhelper: SeedEntryCustom insert example[%d]: %v", i, err)
		}
		sense.Examples = []domain.Example{ex}

		entry.Senses[i] = sense
	}

	return entry
}

// fetchRefEntry loads a RefEntry with its senses, translations, examples, and pronunciations.
// Used internally by SeedEntry to build linked user entries.
func fetchRefEntry(t *testing.T, pool *pgxpool.Pool, refEntryID uuid.UUID) domain.RefEntry {
	t.Helper()
	ctx := context.Background()

	var re domain.RefEntry
	err := pool.QueryRow(ctx,
		`SELECT id, text, text_normalized, created_at FROM ref_entries WHERE id = $1`,
		refEntryID,
	).Scan(&re.ID, &re.Text, &re.TextNormalized, &re.CreatedAt)
	if err != nil {
		t.Fatalf("testhelper: fetchRefEntry select ref_entry: %v", err)
	}

	// Fetch senses.
	senseRows, err := pool.Query(ctx,
		`SELECT id, ref_entry_id, definition, part_of_speech, cefr_level, source_slug, position, created_at
		 FROM ref_senses WHERE ref_entry_id = $1 ORDER BY position`,
		refEntryID,
	)
	if err != nil {
		t.Fatalf("testhelper: fetchRefEntry select ref_senses: %v", err)
	}
	defer senseRows.Close()

	for senseRows.Next() {
		var s domain.RefSense
		var pos *string
		if err := senseRows.Scan(&s.ID, &s.RefEntryID, &s.Definition, &pos, &s.CEFRLevel, &s.SourceSlug, &s.Position, &s.CreatedAt); err != nil {
			t.Fatalf("testhelper: fetchRefEntry scan ref_sense: %v", err)
		}
		if pos != nil {
			p := domain.PartOfSpeech(*pos)
			s.PartOfSpeech = &p
		}

		// Fetch translations for this sense.
		trRows, err := pool.Query(ctx,
			`SELECT id, ref_sense_id, text, source_slug, position
			 FROM ref_translations WHERE ref_sense_id = $1 ORDER BY position`,
			s.ID,
		)
		if err != nil {
			t.Fatalf("testhelper: fetchRefEntry select ref_translations: %v", err)
		}
		for trRows.Next() {
			var tr domain.RefTranslation
			if err := trRows.Scan(&tr.ID, &tr.RefSenseID, &tr.Text, &tr.SourceSlug, &tr.Position); err != nil {
				t.Fatalf("testhelper: fetchRefEntry scan ref_translation: %v", err)
			}
			s.Translations = append(s.Translations, tr)
		}
		trRows.Close()

		// Fetch examples for this sense.
		exRows, err := pool.Query(ctx,
			`SELECT id, ref_sense_id, sentence, translation, source_slug, position
			 FROM ref_examples WHERE ref_sense_id = $1 ORDER BY position`,
			s.ID,
		)
		if err != nil {
			t.Fatalf("testhelper: fetchRefEntry select ref_examples: %v", err)
		}
		for exRows.Next() {
			var ex domain.RefExample
			if err := exRows.Scan(&ex.ID, &ex.RefSenseID, &ex.Sentence, &ex.Translation, &ex.SourceSlug, &ex.Position); err != nil {
				t.Fatalf("testhelper: fetchRefEntry scan ref_example: %v", err)
			}
			s.Examples = append(s.Examples, ex)
		}
		exRows.Close()

		re.Senses = append(re.Senses, s)
	}

	// Fetch pronunciations.
	pronRows, err := pool.Query(ctx,
		`SELECT id, ref_entry_id, transcription, audio_url, region, source_slug
		 FROM ref_pronunciations WHERE ref_entry_id = $1`,
		refEntryID,
	)
	if err != nil {
		t.Fatalf("testhelper: fetchRefEntry select ref_pronunciations: %v", err)
	}
	defer pronRows.Close()

	for pronRows.Next() {
		var p domain.RefPronunciation
		if err := pronRows.Scan(&p.ID, &p.RefEntryID, &p.Transcription, &p.AudioURL, &p.Region, &p.SourceSlug); err != nil {
			t.Fatalf("testhelper: fetchRefEntry scan ref_pronunciation: %v", err)
		}
		re.Pronunciations = append(re.Pronunciations, p)
	}

	return re
}
