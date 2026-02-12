package inbox

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/heartmarshall/my-english/internal/database/testutil"
	"github.com/heartmarshall/my-english/internal/model"
	"github.com/jackc/pgx/v5"
	pgxmock "github.com/pashagolub/pgxmock/v2"
)

func TestInboxRepository_GetByID(t *testing.T) {
	itemID := uuid.New()
	now := time.Now()

	tests := []struct {
		name    string
		id      uuid.UUID
		setup   func(mock pgxmock.PgxPoolIface)
		wantErr bool
	}{
		{
			name: "found",
			id:   itemID,
			setup: func(mock pgxmock.PgxPoolIface) {
				rows := pgxmock.NewRows([]string{"id", "text", "context", "created_at"}).
					AddRow(itemID, "hello", nil, now)
				mock.ExpectQuery(`SELECT`).
					WithArgs(pgxmock.AnyArg()).
					WillReturnRows(rows)
			},
			wantErr: false,
		},
		{
			name: "not found",
			id:   itemID,
			setup: func(mock pgxmock.PgxPoolIface) {
				mock.ExpectQuery(`SELECT`).
					WithArgs(pgxmock.AnyArg()).
					WillReturnError(pgx.ErrNoRows)
			},
			wantErr: true,
		},
		{
			name:    "zero uuid",
			id:      uuid.UUID{},
			setup:   func(mock pgxmock.PgxPoolIface) {},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			querier, mock := testutil.NewMockQuerier(t)
			repo := NewInboxRepository(querier)
			tt.setup(mock)

			ctx := context.Background()
			result, err := repo.GetByID(ctx, tt.id)

			if (err != nil) != tt.wantErr {
				t.Errorf("GetByID() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr && result == nil {
				t.Error("GetByID() returned nil result")
			}

			testutil.ExpectationsWereMet(t, mock)
		})
	}
}

func TestInboxRepository_ListAll(t *testing.T) {
	itemID := uuid.New()
	now := time.Now()

	tests := []struct {
		name    string
		setup   func(mock pgxmock.PgxPoolIface)
		wantLen int
		wantErr bool
	}{
		{
			name: "returns items",
			setup: func(mock pgxmock.PgxPoolIface) {
				rows := pgxmock.NewRows([]string{"id", "text", "context", "created_at"}).
					AddRow(itemID, "hello", nil, now)
				mock.ExpectQuery(`SELECT`).
					WithArgs().
					WillReturnRows(rows)
			},
			wantLen: 1,
			wantErr: false,
		},
		{
			name: "returns empty",
			setup: func(mock pgxmock.PgxPoolIface) {
				rows := pgxmock.NewRows([]string{"id", "text", "context", "created_at"})
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
			repo := NewInboxRepository(querier)
			tt.setup(mock)

			ctx := context.Background()
			result, err := repo.ListAll(ctx)

			if (err != nil) != tt.wantErr {
				t.Errorf("ListAll() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if len(result) != tt.wantLen {
				t.Errorf("ListAll() returned %d items, want %d", len(result), tt.wantLen)
			}

			testutil.ExpectationsWereMet(t, mock)
		})
	}
}

func TestInboxRepository_ListPaginated(t *testing.T) {
	itemID := uuid.New()
	now := time.Now()

	tests := []struct {
		name    string
		limit   int
		offset  int
		setup   func(mock pgxmock.PgxPoolIface)
		wantLen int
		wantErr bool
	}{
		{
			name:   "returns items",
			limit:  10,
			offset: 0,
			setup: func(mock pgxmock.PgxPoolIface) {
				rows := pgxmock.NewRows([]string{"id", "text", "context", "created_at"}).
					AddRow(itemID, "hello", nil, now)
				mock.ExpectQuery(`SELECT`).
					WithArgs().
					WillReturnRows(rows)
			},
			wantLen: 1,
			wantErr: false,
		},
		{
			name:   "returns empty",
			limit:  10,
			offset: 0,
			setup: func(mock pgxmock.PgxPoolIface) {
				rows := pgxmock.NewRows([]string{"id", "text", "context", "created_at"})
				mock.ExpectQuery(`SELECT`).
					WithArgs().
					WillReturnRows(rows)
			},
			wantLen: 0,
			wantErr: false,
		},
		{
			name:   "default limit when zero",
			limit:  0,
			offset: 0,
			setup: func(mock pgxmock.PgxPoolIface) {
				rows := pgxmock.NewRows([]string{"id", "text", "context", "created_at"})
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
			repo := NewInboxRepository(querier)
			tt.setup(mock)

			ctx := context.Background()
			result, err := repo.ListPaginated(ctx, tt.limit, tt.offset)

			if (err != nil) != tt.wantErr {
				t.Errorf("ListPaginated() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if len(result) != tt.wantLen {
				t.Errorf("ListPaginated() returned %d items, want %d", len(result), tt.wantLen)
			}

			testutil.ExpectationsWereMet(t, mock)
		})
	}
}

func TestInboxRepository_Count(t *testing.T) {
	tests := []struct {
		name    string
		setup   func(mock pgxmock.PgxPoolIface)
		want    int64
		wantErr bool
	}{
		{
			name: "returns count",
			setup: func(mock pgxmock.PgxPoolIface) {
				rows := pgxmock.NewRows([]string{"count"}).AddRow(int64(5))
				mock.ExpectQuery(`SELECT COUNT`).
					WithArgs().
					WillReturnRows(rows)
			},
			want:    5,
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			querier, mock := testutil.NewMockQuerier(t)
			repo := NewInboxRepository(querier)
			tt.setup(mock)

			ctx := context.Background()
			got, err := repo.Count(ctx)

			if (err != nil) != tt.wantErr {
				t.Errorf("Count() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if got != tt.want {
				t.Errorf("Count() = %v, want %v", got, tt.want)
			}

			testutil.ExpectationsWereMet(t, mock)
		})
	}
}

func TestInboxRepository_Create(t *testing.T) {
	itemID := uuid.New()
	now := time.Now()

	tests := []struct {
		name    string
		item    *model.InboxItem
		setup   func(mock pgxmock.PgxPoolIface)
		wantErr bool
	}{
		{
			name: "successful creation",
			item: &model.InboxItem{
				Text: "hello",
			},
			setup: func(mock pgxmock.PgxPoolIface) {
				rows := pgxmock.NewRows([]string{"id", "text", "context", "created_at"}).
					AddRow(itemID, "hello", nil, now)
				mock.ExpectQuery(`INSERT INTO inbox_items`).
					WithArgs(pgxmock.AnyArg(), pgxmock.AnyArg()).
					WillReturnRows(rows)
			},
			wantErr: false,
		},
		{
			name:    "nil input",
			item:    nil,
			setup:   func(mock pgxmock.PgxPoolIface) {},
			wantErr: true,
		},
		{
			name: "empty text",
			item: &model.InboxItem{
				Text: "",
			},
			setup:   func(mock pgxmock.PgxPoolIface) {},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			querier, mock := testutil.NewMockQuerier(t)
			repo := NewInboxRepository(querier)
			tt.setup(mock)

			ctx := context.Background()
			result, err := repo.Create(ctx, tt.item)

			if (err != nil) != tt.wantErr {
				t.Errorf("Create() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr && result == nil {
				t.Error("Create() returned nil result")
			}

			testutil.ExpectationsWereMet(t, mock)
		})
	}
}

func TestInboxRepository_Delete(t *testing.T) {
	itemID := uuid.New()

	tests := []struct {
		name    string
		id      uuid.UUID
		setup   func(mock pgxmock.PgxPoolIface)
		wantErr bool
	}{
		{
			name: "successful delete",
			id:   itemID,
			setup: func(mock pgxmock.PgxPoolIface) {
				mock.ExpectExec(`DELETE FROM inbox_items`).
					WithArgs(pgxmock.AnyArg()).
					WillReturnResult(pgxmock.NewResult("DELETE", 1))
			},
			wantErr: false,
		},
		{
			name: "not found",
			id:   itemID,
			setup: func(mock pgxmock.PgxPoolIface) {
				mock.ExpectExec(`DELETE FROM inbox_items`).
					WithArgs(pgxmock.AnyArg()).
					WillReturnResult(pgxmock.NewResult("DELETE", 0))
			},
			wantErr: true,
		},
		{
			name:    "zero uuid",
			id:      uuid.UUID{},
			setup:   func(mock pgxmock.PgxPoolIface) {},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			querier, mock := testutil.NewMockQuerier(t)
			repo := NewInboxRepository(querier)
			tt.setup(mock)

			ctx := context.Background()
			err := repo.Delete(ctx, tt.id)

			if (err != nil) != tt.wantErr {
				t.Errorf("Delete() error = %v, wantErr %v", err, tt.wantErr)
			}

			testutil.ExpectationsWereMet(t, mock)
		})
	}
}

func TestInboxRepository_DeleteAll(t *testing.T) {
	tests := []struct {
		name    string
		setup   func(mock pgxmock.PgxPoolIface)
		want    int64
		wantErr bool
	}{
		{
			name: "deletes all items",
			setup: func(mock pgxmock.PgxPoolIface) {
				mock.ExpectExec(`DELETE FROM inbox_items`).
					WillReturnResult(pgxmock.NewResult("DELETE", 3))
			},
			want:    3,
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			querier, mock := testutil.NewMockQuerier(t)
			repo := NewInboxRepository(querier)
			tt.setup(mock)

			ctx := context.Background()
			got, err := repo.DeleteAll(ctx)

			if (err != nil) != tt.wantErr {
				t.Errorf("DeleteAll() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if got != tt.want {
				t.Errorf("DeleteAll() = %v, want %v", got, tt.want)
			}

			testutil.ExpectationsWereMet(t, mock)
		})
	}
}
