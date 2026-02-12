package testutil

import (
	"testing"
	"time"

	"github.com/heartmarshall/my-english/internal/database"
	pgxmock "github.com/pashagolub/pgxmock/v2"
)

// MockClock — мок для database.Clock.
type MockClock struct {
	Time time.Time
}

// Now возвращает заранее заданное время.
func (m *MockClock) Now() time.Time {
	return m.Time
}

// FixedTime возвращает фиксированное время для тестов.
func FixedTime() time.Time {
	return time.Date(2025, 1, 15, 10, 30, 0, 0, time.UTC)
}

// NewMockClock создаёт MockClock с фиксированным временем.
func NewMockClock() *MockClock {
	return &MockClock{Time: FixedTime()}
}

// Compile-time проверка.
var _ database.Clock = (*MockClock)(nil)

// NewMockQuerier создаёт мок Querier для тестов.
func NewMockQuerier(t *testing.T) (database.Querier, pgxmock.PgxPoolIface) {
	t.Helper()

	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatalf("failed to create pgxmock: %v", err)
	}

	t.Cleanup(func() {
		mock.Close()
	})

	return mock, mock
}

// ExpectationsWereMet проверяет, что все ожидания pgxmock выполнены.
func ExpectationsWereMet(t *testing.T, mock pgxmock.PgxPoolIface) {
	t.Helper()

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unfulfilled expectations: %v", err)
	}
}
