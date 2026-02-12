package topics

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/heartmarshall/my-english/internal/database"
	"github.com/heartmarshall/my-english/internal/database/testutil"
	"github.com/heartmarshall/my-english/internal/model"
	"github.com/jackc/pgx/v5"
	pgxmock "github.com/pashagolub/pgxmock/v2"
)

func TestTopicRepository_GetByID(t *testing.T) {
	topicID := uuid.New()
	now := time.Now()
	desc := "Topic description"

	tests := []struct {
		name    string
		id      uuid.UUID
		setup   func(mock pgxmock.PgxPoolIface)
		wantErr bool
		check   func(t *testing.T, result *model.Topic)
	}{
		{
			name: "found",
			id:   topicID,
			setup: func(mock pgxmock.PgxPoolIface) {
				rows := pgxmock.NewRows([]string{"id", "name", "description", "created_at", "updated_at"}).
					AddRow(topicID, "IT", &desc, now, now)
				mock.ExpectQuery(`SELECT`).
					WithArgs(pgxmock.AnyArg()).
					WillReturnRows(rows)
			},
			wantErr: false,
			check: func(t *testing.T, result *model.Topic) {
				if result.ID != topicID {
					t.Errorf("GetByID() id = %v, want %v", result.ID, topicID)
				}
				if result.Name != "IT" {
					t.Errorf("GetByID() name = %q, want %q", result.Name, "IT")
				}
			},
		},
		{
			name: "not found",
			id:   topicID,
			setup: func(mock pgxmock.PgxPoolIface) {
				mock.ExpectQuery(`SELECT`).
					WithArgs(pgxmock.AnyArg()).
					WillReturnError(pgx.ErrNoRows)
			},
			wantErr: true,
		},
		{
			name:    "zero uuid returns validation error",
			id:      uuid.Nil,
			setup:   func(mock pgxmock.PgxPoolIface) {},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			querier, mock := testutil.NewMockQuerier(t)
			repo := NewTopicRepository(querier)
			tt.setup(mock)

			ctx := context.Background()
			result, err := repo.GetByID(ctx, tt.id)

			if (err != nil) != tt.wantErr {
				t.Errorf("GetByID() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr {
				if result == nil {
					t.Error("GetByID() returned nil result")
					return
				}
				if tt.check != nil {
					tt.check(t, result)
				}
			}

			if tt.name == "zero uuid returns validation error" && err != nil && !errors.Is(err, database.ErrInvalidInput) {
				t.Errorf("GetByID() expected ErrInvalidInput, got %v", err)
			}

			testutil.ExpectationsWereMet(t, mock)
		})
	}
}

func TestTopicRepository_Create(t *testing.T) {
	topicID := uuid.New()
	now := time.Now()
	desc := "Some topic"

	tests := []struct {
		name    string
		topic   *model.Topic
		setup   func(mock pgxmock.PgxPoolIface)
		wantErr bool
		check   func(t *testing.T, result *model.Topic)
	}{
		{
			name: "successful creation",
			topic: &model.Topic{
				Name:        "Basics",
				Description: &desc,
			},
			setup: func(mock pgxmock.PgxPoolIface) {
				rows := pgxmock.NewRows([]string{"id", "name", "description", "created_at", "updated_at"}).
					AddRow(topicID, "Basics", &desc, now, now)
				mock.ExpectQuery(`INSERT INTO topics`).
					WithArgs(pgxmock.AnyArg(), pgxmock.AnyArg()).
					WillReturnRows(rows)
			},
			wantErr: false,
			check: func(t *testing.T, result *model.Topic) {
				if result.Name != "Basics" {
					t.Errorf("Create() name = %q, want %q", result.Name, "Basics")
				}
				if result.Description == nil || *result.Description != desc {
					t.Errorf("Create() description = %v, want %q", result.Description, desc)
				}
			},
		},
		{
			name: "successful creation without description",
			topic: &model.Topic{
				Name: "Verbs",
			},
			setup: func(mock pgxmock.PgxPoolIface) {
				rows := pgxmock.NewRows([]string{"id", "name", "description", "created_at", "updated_at"}).
					AddRow(topicID, "Verbs", nil, now, now)
				mock.ExpectQuery(`INSERT INTO topics`).
					WithArgs(pgxmock.AnyArg(), pgxmock.AnyArg()).
					WillReturnRows(rows)
			},
			wantErr: false,
			check: func(t *testing.T, result *model.Topic) {
				if result.Description != nil {
					t.Errorf("Create() description = %v, want nil", result.Description)
				}
			},
		},
		{
			name:    "nil topic returns validation error",
			topic:   nil,
			setup:   func(mock pgxmock.PgxPoolIface) {},
			wantErr: true,
		},
		{
			name: "empty name returns validation error",
			topic: &model.Topic{
				Name: "",
			},
			setup:   func(mock pgxmock.PgxPoolIface) {},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			querier, mock := testutil.NewMockQuerier(t)
			repo := NewTopicRepository(querier)
			tt.setup(mock)

			ctx := context.Background()
			result, err := repo.Create(ctx, tt.topic)

			if (err != nil) != tt.wantErr {
				t.Errorf("Create() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr {
				if result == nil {
					t.Error("Create() returned nil result")
					return
				}
				if tt.check != nil {
					tt.check(t, result)
				}
			}

			if (tt.name == "nil topic returns validation error" || tt.name == "empty name returns validation error") && err != nil && !errors.Is(err, database.ErrInvalidInput) {
				t.Errorf("Create() expected ErrInvalidInput, got %v", err)
			}

			testutil.ExpectationsWereMet(t, mock)
		})
	}
}

func TestTopicRepository_Update(t *testing.T) {
	topicID := uuid.New()
	now := time.Now()
	desc := "Updated description"

	tests := []struct {
		name    string
		id      uuid.UUID
		topic   *model.Topic
		setup   func(mock pgxmock.PgxPoolIface)
		wantErr bool
	}{
		{
			name: "successful update",
			id:   topicID,
			topic: &model.Topic{
				Name:        "IT_Dev",
				Description: &desc,
			},
			setup: func(mock pgxmock.PgxPoolIface) {
				rows := pgxmock.NewRows([]string{"id", "name", "description", "created_at", "updated_at"}).
					AddRow(topicID, "IT_Dev", &desc, now, now)
				mock.ExpectQuery(`UPDATE topics`).
					WithArgs(pgxmock.AnyArg(), pgxmock.AnyArg(), pgxmock.AnyArg()).
					WillReturnRows(rows)
			},
			wantErr: false,
		},
		{
			name: "not found",
			id:   topicID,
			topic: &model.Topic{
				Name: "IT_Dev",
			},
			setup: func(mock pgxmock.PgxPoolIface) {
				mock.ExpectQuery(`UPDATE topics`).
					WithArgs(pgxmock.AnyArg(), pgxmock.AnyArg(), pgxmock.AnyArg()).
					WillReturnError(pgx.ErrNoRows)
			},
			wantErr: true,
		},
		{
			name: "zero uuid returns validation error",
			id:   uuid.Nil,
			topic: &model.Topic{
				Name: "IT",
			},
			setup:   func(mock pgxmock.PgxPoolIface) {},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			querier, mock := testutil.NewMockQuerier(t)
			repo := NewTopicRepository(querier)
			tt.setup(mock)

			ctx := context.Background()
			result, err := repo.Update(ctx, tt.id, tt.topic)

			if (err != nil) != tt.wantErr {
				t.Errorf("Update() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr && result == nil {
				t.Error("Update() returned nil result")
			}

			if tt.wantErr && tt.name == "not found" && err != database.ErrNotFound {
				t.Errorf("Update() expected ErrNotFound, got %v", err)
			}

			if tt.name == "zero uuid returns validation error" && err != nil && !errors.Is(err, database.ErrInvalidInput) {
				t.Errorf("Update() expected ErrInvalidInput, got %v", err)
			}

			testutil.ExpectationsWereMet(t, mock)
		})
	}
}

func TestTopicRepository_Delete(t *testing.T) {
	topicID := uuid.New()

	tests := []struct {
		name    string
		id      uuid.UUID
		setup   func(mock pgxmock.PgxPoolIface)
		wantErr bool
	}{
		{
			name: "successful delete",
			id:   topicID,
			setup: func(mock pgxmock.PgxPoolIface) {
				mock.ExpectExec(`DELETE FROM topics`).
					WithArgs(pgxmock.AnyArg()).
					WillReturnResult(pgxmock.NewResult("DELETE", 1))
			},
			wantErr: false,
		},
		{
			name: "not found",
			id:   topicID,
			setup: func(mock pgxmock.PgxPoolIface) {
				mock.ExpectExec(`DELETE FROM topics`).
					WithArgs(pgxmock.AnyArg()).
					WillReturnResult(pgxmock.NewResult("DELETE", 0))
			},
			wantErr: true,
		},
		{
			name:    "zero uuid returns validation error",
			id:      uuid.Nil,
			setup:   func(mock pgxmock.PgxPoolIface) {},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			querier, mock := testutil.NewMockQuerier(t)
			repo := NewTopicRepository(querier)
			tt.setup(mock)

			ctx := context.Background()
			err := repo.Delete(ctx, tt.id)

			if (err != nil) != tt.wantErr {
				t.Errorf("Delete() error = %v, wantErr %v", err, tt.wantErr)
			}

			if tt.wantErr && tt.name == "not found" && err != database.ErrNotFound {
				t.Errorf("Delete() expected ErrNotFound, got %v", err)
			}

			if tt.name == "zero uuid returns validation error" && err != nil && !errors.Is(err, database.ErrInvalidInput) {
				t.Errorf("Delete() expected ErrInvalidInput, got %v", err)
			}

			testutil.ExpectationsWereMet(t, mock)
		})
	}
}

func TestTopicRepository_ListAll(t *testing.T) {
	topicID := uuid.New()
	now := time.Now()

	tests := []struct {
		name    string
		setup   func(mock pgxmock.PgxPoolIface)
		wantLen int
		wantErr bool
	}{
		{
			name: "returns topics ordered by name",
			setup: func(mock pgxmock.PgxPoolIface) {
				rows := pgxmock.NewRows([]string{"id", "name", "description", "created_at", "updated_at"}).
					AddRow(topicID, "Basics", nil, now, now)
				mock.ExpectQuery(`SELECT`).
					WithArgs().
					WillReturnRows(rows)
			},
			wantLen: 1,
			wantErr: false,
		},
		{
			name: "returns empty slice when no topics",
			setup: func(mock pgxmock.PgxPoolIface) {
				rows := pgxmock.NewRows([]string{"id", "name", "description", "created_at", "updated_at"})
				mock.ExpectQuery(`SELECT`).
					WithArgs().
					WillReturnRows(rows)
			},
			wantLen: 0,
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			querier, mock := testutil.NewMockQuerier(t)
			repo := NewTopicRepository(querier)
			tt.setup(mock)

			ctx := context.Background()
			result, err := repo.ListAll(ctx)

			if (err != nil) != tt.wantErr {
				t.Errorf("ListAll() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if len(result) != tt.wantLen {
				t.Errorf("ListAll() returned %d topics, want %d", len(result), tt.wantLen)
			}

			testutil.ExpectationsWereMet(t, mock)
		})
	}
}

func TestTopicRepository_BindToEntry(t *testing.T) {
	entryID := uuid.New()
	topicID := uuid.New()

	tests := []struct {
		name     string
		entryID  uuid.UUID
		topicID  uuid.UUID
		setup    func(mock pgxmock.PgxPoolIface)
		wantErr  bool
		checkErr func(t *testing.T, err error)
	}{
		{
			name:    "successful bind",
			entryID: entryID,
			topicID: topicID,
			setup: func(mock pgxmock.PgxPoolIface) {
				mock.ExpectExec(`INSERT INTO dictionary_entry_topics`).
					WithArgs(pgxmock.AnyArg(), pgxmock.AnyArg()).
					WillReturnResult(pgxmock.NewResult("INSERT", 1))
			},
			wantErr: false,
		},
		{
			name:     "zero entry_id returns validation error",
			entryID:  uuid.Nil,
			topicID:  topicID,
			setup:    func(mock pgxmock.PgxPoolIface) {},
			wantErr:  true,
			checkErr: func(t *testing.T, err error) { _ = errors.Is(err, database.ErrInvalidInput) },
		},
		{
			name:     "zero topic_id returns validation error",
			entryID:  entryID,
			topicID:  uuid.Nil,
			setup:    func(mock pgxmock.PgxPoolIface) {},
			wantErr:  true,
			checkErr: func(t *testing.T, err error) { _ = errors.Is(err, database.ErrInvalidInput) },
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			querier, mock := testutil.NewMockQuerier(t)
			repo := NewTopicRepository(querier)
			tt.setup(mock)

			ctx := context.Background()
			err := repo.BindToEntry(ctx, tt.entryID, tt.topicID)

			if (err != nil) != tt.wantErr {
				t.Errorf("BindToEntry() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if tt.wantErr && tt.checkErr != nil && err != nil {
				if !errors.Is(err, database.ErrInvalidInput) {
					t.Errorf("BindToEntry() expected ErrInvalidInput, got %v", err)
				}
			}

			testutil.ExpectationsWereMet(t, mock)
		})
	}
}

func TestTopicRepository_UnbindFromEntry(t *testing.T) {
	entryID := uuid.New()
	topicID := uuid.New()

	tests := []struct {
		name     string
		entryID  uuid.UUID
		topicID  uuid.UUID
		setup    func(mock pgxmock.PgxPoolIface)
		wantErr  bool
		checkErr func(t *testing.T, err error)
	}{
		{
			name:    "successful unbind",
			entryID: entryID,
			topicID: topicID,
			setup: func(mock pgxmock.PgxPoolIface) {
				mock.ExpectExec(`DELETE FROM dictionary_entry_topics`).
					WithArgs(pgxmock.AnyArg(), pgxmock.AnyArg()).
					WillReturnResult(pgxmock.NewResult("DELETE", 1))
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			querier, mock := testutil.NewMockQuerier(t)
			repo := NewTopicRepository(querier)
			tt.setup(mock)

			ctx := context.Background()
			err := repo.UnbindFromEntry(ctx, tt.entryID, tt.topicID)

			if (err != nil) != tt.wantErr {
				t.Errorf("UnbindFromEntry() error = %v, wantErr %v", err, tt.wantErr)
			}

			testutil.ExpectationsWereMet(t, mock)
		})
	}
}

func TestTopicRepository_UnbindAllFromEntry(t *testing.T) {
	entryID := uuid.New()

	tests := []struct {
		name    string
		entryID uuid.UUID
		setup   func(mock pgxmock.PgxPoolIface)
		wantErr bool
	}{
		{
			name:    "successful unbind all",
			entryID: entryID,
			setup: func(mock pgxmock.PgxPoolIface) {
				mock.ExpectExec(`DELETE FROM dictionary_entry_topics`).
					WithArgs(pgxmock.AnyArg()).
					WillReturnResult(pgxmock.NewResult("DELETE", 3))
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			querier, mock := testutil.NewMockQuerier(t)
			repo := NewTopicRepository(querier)
			tt.setup(mock)

			ctx := context.Background()
			err := repo.UnbindAllFromEntry(ctx, tt.entryID)

			if (err != nil) != tt.wantErr {
				t.Errorf("UnbindAllFromEntry() error = %v, wantErr %v", err, tt.wantErr)
			}

			testutil.ExpectationsWereMet(t, mock)
		})
	}
}

func TestTopicRepository_ListByEntryIDs(t *testing.T) {
	entryID := uuid.New()
	topicID := uuid.New()
	now := time.Now()

	tests := []struct {
		name     string
		entryIDs []uuid.UUID
		setup    func(mock pgxmock.PgxPoolIface)
		wantLen  int
		wantErr  bool
	}{
		{
			name:     "empty entryIDs returns nil without query",
			entryIDs: nil,
			setup:    func(mock pgxmock.PgxPoolIface) {},
			wantLen:  0,
			wantErr:  false,
		},
		{
			name:     "empty slice returns nil without query",
			entryIDs: []uuid.UUID{},
			setup:    func(mock pgxmock.PgxPoolIface) {},
			wantLen:  0,
			wantErr:  false,
		},
		{
			name:     "returns topics for entry IDs",
			entryIDs: []uuid.UUID{entryID},
			setup: func(mock pgxmock.PgxPoolIface) {
				rows := pgxmock.NewRows([]string{"id", "name", "description", "created_at", "updated_at", "entry_id"}).
					AddRow(topicID, "IT", nil, now, now, entryID)
				mock.ExpectQuery(`SELECT`).
					WithArgs(pgxmock.AnyArg()).
					WillReturnRows(rows)
			},
			wantLen: 1,
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			querier, mock := testutil.NewMockQuerier(t)
			repo := NewTopicRepository(querier)
			tt.setup(mock)

			ctx := context.Background()
			result, err := repo.ListByEntryIDs(ctx, tt.entryIDs)

			if (err != nil) != tt.wantErr {
				t.Errorf("ListByEntryIDs() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if tt.wantErr {
				return
			}

			if tt.entryIDs != nil && len(tt.entryIDs) == 0 && len(result) != 0 {
				t.Errorf("ListByEntryIDs() with empty slice should return empty slice, got %v", result)
			}

			if len(tt.entryIDs) > 0 && len(result) != tt.wantLen {
				t.Errorf("ListByEntryIDs() returned %d items, want %d", len(result), tt.wantLen)
			}

			if tt.wantLen == 1 && len(result) == 1 {
				if result[0].EntryID != entryID {
					t.Errorf("ListByEntryIDs() entry_id = %v, want %v", result[0].EntryID, entryID)
				}
				if result[0].ID != topicID {
					t.Errorf("ListByEntryIDs() topic id = %v, want %v", result[0].ID, topicID)
				}
			}

			testutil.ExpectationsWereMet(t, mock)
		})
	}
}
