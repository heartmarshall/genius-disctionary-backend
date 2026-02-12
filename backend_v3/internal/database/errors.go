// Package database предоставляет базовые компоненты для работы с PostgreSQL.
package database

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

// ============================================================================
// SENTINEL ERRORS
// ============================================================================

// Кастомные ошибки слоя данных.
// Позволяют бизнес-логике не зависеть от sql/pgx пакетов.
var (
	// ErrNotFound возвращается, когда запись не найдена.
	ErrNotFound = errors.New("record not found")

	// ErrDuplicate возвращается при попытке создать дубликат
	// (нарушение UNIQUE constraint).
	ErrDuplicate = errors.New("record already exists")

	// ErrInvalidInput возвращается при невалидных входных данных.
	ErrInvalidInput = errors.New("invalid input")

	// ErrTimeout возвращается при превышении таймаута запроса.
	ErrTimeout = errors.New("query timeout")

	// ErrConstraintViolation возвращается при нарушении constraint (кроме UNIQUE).
	ErrConstraintViolation = errors.New("constraint violation")

	// ErrForeignKeyViolation возвращается при нарушении внешнего ключа.
	ErrForeignKeyViolation = errors.New("foreign key violation")

	// ErrConnection возвращается при ошибке соединения с БД.
	ErrConnection = errors.New("database connection error")
)

// ============================================================================
// POSTGRESQL ERROR CODES
// ============================================================================

// PostgreSQL error codes
// https://www.postgresql.org/docs/current/errcodes-appendix.html
const (
	// Class 23 — Integrity Constraint Violation
	pgUniqueViolation     = "23505"
	pgForeignKeyViolation = "23503"
	pgNotNullViolation    = "23502"
	pgCheckViolation      = "23514"
	pgExclusionViolation  = "23P01"

	// Class 08 — Connection Exception
	pgConnectionException   = "08000"
	pgConnectionDoesNotExit = "08003"
	pgConnectionFailure     = "08006"

	// Class 53 — Insufficient Resources
	pgDiskFull        = "53100"
	pgOutOfMemory     = "53200"
	pgTooManyConnects = "53300"

	// Class 57 — Operator Intervention
	pgQueryCanceled = "57014"
)

// ============================================================================
// ERROR CHECKING FUNCTIONS
// ============================================================================

// IsNotFoundError проверяет, является ли ошибка "запись не найдена".
// Проверяет как кастомный ErrNotFound, так и оригинальный pgx.ErrNoRows.
func IsNotFoundError(err error) bool {
	if err == nil {
		return false
	}
	return errors.Is(err, ErrNotFound) || errors.Is(err, pgx.ErrNoRows)
}

// IsDuplicateError проверяет, является ли ошибка нарушением UNIQUE constraint.
func IsDuplicateError(err error) bool {
	if err == nil {
		return false
	}

	// Проверяем wrapped ошибку
	if errors.Is(err, ErrDuplicate) {
		return true
	}

	// Проверяем pgconn.PgError
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) {
		return pgErr.Code == pgUniqueViolation
	}

	return false
}

// IsConstraintError проверяет, является ли ошибка нарушением constraint.
func IsConstraintError(err error) bool {
	if err == nil {
		return false
	}

	if errors.Is(err, ErrConstraintViolation) ||
		errors.Is(err, ErrDuplicate) ||
		errors.Is(err, ErrForeignKeyViolation) {
		return true
	}

	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) {
		switch pgErr.Code {
		case pgUniqueViolation, pgForeignKeyViolation, pgNotNullViolation,
			pgCheckViolation, pgExclusionViolation:
			return true
		}
	}

	return false
}

// IsForeignKeyError проверяет, является ли ошибка нарушением FK constraint.
func IsForeignKeyError(err error) bool {
	if err == nil {
		return false
	}

	if errors.Is(err, ErrForeignKeyViolation) {
		return true
	}

	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) {
		return pgErr.Code == pgForeignKeyViolation
	}

	return false
}

// IsTimeoutError проверяет, является ли ошибка таймаутом.
func IsTimeoutError(err error) bool {
	if err == nil {
		return false
	}

	if errors.Is(err, ErrTimeout) || errors.Is(err, context.DeadlineExceeded) {
		return true
	}

	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) {
		return pgErr.Code == pgQueryCanceled
	}

	return false
}

// IsConnectionError проверяет, является ли ошибка проблемой соединения.
func IsConnectionError(err error) bool {
	if err == nil {
		return false
	}

	if errors.Is(err, ErrConnection) {
		return true
	}

	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) {
		switch pgErr.Code {
		case pgConnectionException, pgConnectionDoesNotExit, pgConnectionFailure,
			pgTooManyConnects:
			return true
		}
	}

	return false
}

// IsRetryableError проверяет, можно ли повторить операцию.
// Возвращает true для временных ошибок (таймауты, проблемы соединения).
func IsRetryableError(err error) bool {
	if err == nil {
		return false
	}

	return IsTimeoutError(err) || IsConnectionError(err)
}

// ============================================================================
// ERROR WRAPPING
// ============================================================================

// DBError представляет структурированную ошибку базы данных.
type DBError struct {
	// Err — базовая ошибка (sentinel)
	Err error

	// Op — операция, при которой произошла ошибка
	Op string

	// Table — таблица, к которой относится ошибка
	Table string

	// Detail — дополнительные детали (из PostgreSQL)
	Detail string

	// Cause — оригинальная ошибка
	Cause error
}

// Error реализует интерфейс error.
func (e *DBError) Error() string {
	if e.Op != "" && e.Table != "" {
		return fmt.Sprintf("%s %s: %v", e.Op, e.Table, e.Err)
	}
	if e.Op != "" {
		return fmt.Sprintf("%s: %v", e.Op, e.Err)
	}
	if e.Detail != "" {
		return fmt.Sprintf("%v: %s", e.Err, e.Detail)
	}
	return e.Err.Error()
}

// Unwrap возвращает базовую ошибку для errors.Is/As.
func (e *DBError) Unwrap() error {
	return e.Err
}

// WrapDBError оборачивает ошибку базы данных в domain-specific ошибку.
// Преобразует специфичные ошибки PostgreSQL в понятные доменные ошибки.
func WrapDBError(err error) error {
	if err == nil {
		return nil
	}

	// Проверяем контекстные ошибки
	if errors.Is(err, context.DeadlineExceeded) {
		return &DBError{
			Err:   ErrTimeout,
			Cause: err,
		}
	}
	if errors.Is(err, context.Canceled) {
		return &DBError{
			Err:    ErrTimeout,
			Detail: "operation canceled",
			Cause:  err,
		}
	}

	// Проверяем специфичные ошибки PostgreSQL
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) {
		return wrapPgError(pgErr)
	}

	// Возвращаем ошибку как есть, если не можем её обработать
	return err
}

// wrapPgError преобразует pgconn.PgError в DBError.
func wrapPgError(pgErr *pgconn.PgError) error {
	switch pgErr.Code {
	case pgUniqueViolation:
		return &DBError{
			Err:    ErrDuplicate,
			Detail: pgErr.Detail,
			Cause:  pgErr,
		}

	case pgForeignKeyViolation:
		return &DBError{
			Err:    ErrForeignKeyViolation,
			Detail: pgErr.Detail,
			Cause:  pgErr,
		}

	case pgNotNullViolation, pgCheckViolation, pgExclusionViolation:
		return &DBError{
			Err:    ErrConstraintViolation,
			Detail: pgErr.Detail,
			Cause:  pgErr,
		}

	case pgConnectionException, pgConnectionDoesNotExit, pgConnectionFailure:
		return &DBError{
			Err:    ErrConnection,
			Detail: pgErr.Message,
			Cause:  pgErr,
		}

	case pgQueryCanceled:
		return &DBError{
			Err:    ErrTimeout,
			Detail: "query was canceled",
			Cause:  pgErr,
		}

	default:
		// Для неизвестных ошибок возвращаем оригинал
		return pgErr
	}
}

// WrapWithOp оборачивает ошибку с указанием операции.
// Создаёт shallow copy, чтобы не мутировать оригинальный *DBError.
func WrapWithOp(err error, op string) error {
	if err == nil {
		return nil
	}

	var dbErr *DBError
	if errors.As(err, &dbErr) {
		copied := *dbErr
		copied.Op = op
		return &copied
	}

	return &DBError{
		Err:   err,
		Op:    op,
		Cause: err,
	}
}

// WrapWithTable оборачивает ошибку с указанием таблицы.
// Создаёт shallow copy, чтобы не мутировать оригинальный *DBError.
func WrapWithTable(err error, table string) error {
	if err == nil {
		return nil
	}

	var dbErr *DBError
	if errors.As(err, &dbErr) {
		copied := *dbErr
		copied.Table = table
		return &copied
	}

	return &DBError{
		Err:   err,
		Table: table,
		Cause: err,
	}
}
