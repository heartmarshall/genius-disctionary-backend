package study

import (
	"errors"
	"testing"

	"github.com/google/uuid"
	"github.com/heartmarshall/myenglish-backend/internal/domain"
)

func TestGetQueueInput_Validate(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		input   GetQueueInput
		wantErr bool
	}{
		{name: "valid zero (means default)", input: GetQueueInput{Limit: 0}, wantErr: false},
		{name: "valid 1", input: GetQueueInput{Limit: 1}, wantErr: false},
		{name: "valid 200", input: GetQueueInput{Limit: 200}, wantErr: false},
		{name: "invalid negative", input: GetQueueInput{Limit: -1}, wantErr: true},
		{name: "invalid 201", input: GetQueueInput{Limit: 201}, wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.input.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
			if tt.wantErr && !errors.Is(err, domain.ErrValidation) {
				t.Errorf("expected ErrValidation, got %v", err)
			}
		})
	}
}

func TestReviewCardInput_Validate(t *testing.T) {
	t.Parallel()

	validID := uuid.New()

	tests := []struct {
		name    string
		input   ReviewCardInput
		wantErr bool
	}{
		{
			name:    "valid minimal",
			input:   ReviewCardInput{CardID: validID, Grade: domain.ReviewGradeGood},
			wantErr: false,
		},
		{
			name:    "valid with duration",
			input:   ReviewCardInput{CardID: validID, Grade: domain.ReviewGradeEasy, DurationMs: ptr(5000)},
			wantErr: false,
		},
		{
			name:    "valid duration zero",
			input:   ReviewCardInput{CardID: validID, Grade: domain.ReviewGradeAgain, DurationMs: ptr(0)},
			wantErr: false,
		},
		{
			name:    "valid duration max",
			input:   ReviewCardInput{CardID: validID, Grade: domain.ReviewGradeHard, DurationMs: ptr(600_000)},
			wantErr: false,
		},
		{
			name:    "invalid nil card ID",
			input:   ReviewCardInput{CardID: uuid.Nil, Grade: domain.ReviewGradeGood},
			wantErr: true,
		},
		{
			name:    "invalid grade",
			input:   ReviewCardInput{CardID: validID, Grade: "INVALID"},
			wantErr: true,
		},
		{
			name:    "invalid negative duration",
			input:   ReviewCardInput{CardID: validID, Grade: domain.ReviewGradeGood, DurationMs: ptr(-1)},
			wantErr: true,
		},
		{
			name:    "invalid duration exceeds 10 min",
			input:   ReviewCardInput{CardID: validID, Grade: domain.ReviewGradeGood, DurationMs: ptr(600_001)},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.input.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestUndoReviewInput_Validate(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		input   UndoReviewInput
		wantErr bool
	}{
		{name: "valid", input: UndoReviewInput{CardID: uuid.New()}, wantErr: false},
		{name: "invalid nil", input: UndoReviewInput{CardID: uuid.Nil}, wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.input.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestCreateCardInput_Validate(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		input   CreateCardInput
		wantErr bool
	}{
		{name: "valid", input: CreateCardInput{EntryID: uuid.New()}, wantErr: false},
		{name: "invalid nil", input: CreateCardInput{EntryID: uuid.Nil}, wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.input.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestDeleteCardInput_Validate(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		input   DeleteCardInput
		wantErr bool
	}{
		{name: "valid", input: DeleteCardInput{CardID: uuid.New()}, wantErr: false},
		{name: "invalid nil", input: DeleteCardInput{CardID: uuid.Nil}, wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.input.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestGetCardHistoryInput_Validate(t *testing.T) {
	t.Parallel()

	validID := uuid.New()

	tests := []struct {
		name    string
		input   GetCardHistoryInput
		wantErr bool
	}{
		{name: "valid all zeros", input: GetCardHistoryInput{CardID: validID, Limit: 0, Offset: 0}, wantErr: false},
		{name: "valid max limit", input: GetCardHistoryInput{CardID: validID, Limit: 200, Offset: 0}, wantErr: false},
		{name: "valid with offset", input: GetCardHistoryInput{CardID: validID, Limit: 50, Offset: 100}, wantErr: false},
		{name: "invalid nil card ID", input: GetCardHistoryInput{CardID: uuid.Nil, Limit: 50, Offset: 0}, wantErr: true},
		{name: "invalid limit too high", input: GetCardHistoryInput{CardID: validID, Limit: 201, Offset: 0}, wantErr: true},
		{name: "invalid negative limit", input: GetCardHistoryInput{CardID: validID, Limit: -1, Offset: 0}, wantErr: true},
		{name: "invalid negative offset", input: GetCardHistoryInput{CardID: validID, Limit: 50, Offset: -1}, wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.input.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestBatchCreateCardsInput_Validate(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		count   int
		wantErr bool
	}{
		{name: "valid 1 entry", count: 1, wantErr: false},
		{name: "valid 100 entries", count: 100, wantErr: false},
		{name: "invalid empty", count: 0, wantErr: true},
		{name: "invalid 101 entries", count: 101, wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ids := make([]uuid.UUID, tt.count)
			for i := range ids {
				ids[i] = uuid.New()
			}
			input := BatchCreateCardsInput{EntryIDs: ids}
			err := input.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestFinishSessionInput_Validate(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		input   FinishSessionInput
		wantErr bool
	}{
		{name: "valid", input: FinishSessionInput{SessionID: uuid.New()}, wantErr: false},
		{name: "invalid nil", input: FinishSessionInput{SessionID: uuid.Nil}, wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.input.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
