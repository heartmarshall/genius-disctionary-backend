// Package base предоставляет базовый generic репозиторий для CRUD операций.
package base

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/Masterminds/squirrel"
	"github.com/georgysavva/scany/v2/pgxscan"
	"github.com/google/uuid"
	"github.com/heartmarshall/my-english/internal/database"
)

// ============================================================================
// CONSTANTS
// ============================================================================

const (
	// DefaultQueryTimeout — таймаут по умолчанию для запросов к БД.
	// Используется только если контекст не имеет своего deadline.
	DefaultQueryTimeout = 30 * time.Second

	// MaxBatchSize — максимальный размер батча для операций вставки.
	// PostgreSQL имеет лимит на количество параметров (~65535),
	// поэтому ограничиваем размер батча для безопасности.
	MaxBatchSize = 1000

	// DefaultBatchSize — размер батча по умолчанию.
	DefaultBatchSize = 100
)

// ============================================================================
// SQL BUILDER
// ============================================================================

// psql — squirrel builder с PostgreSQL placeholder format ($1, $2, ...).
// Используем private переменную чтобы избежать случайного изменения.
var psql = squirrel.StatementBuilder.PlaceholderFormat(squirrel.Dollar)

// Builder возвращает squirrel builder для построения SQL запросов.
// Предпочтительно использовать методы репозитория (SelectBuilder, etc.).
func Builder() squirrel.StatementBuilderType {
	return psql
}

// ============================================================================
// HELPER FUNCTIONS
// ============================================================================

// UUIDsToAny конвертирует слайс uuid.UUID в слайс any.
// Используется для передачи в squirrel.Eq{column: values}.
func UUIDsToAny(ids []uuid.UUID) []any {
	result := make([]any, len(ids))
	for i, id := range ids {
		result[i] = id
	}
	return result
}

// IsZeroUUID проверяет, является ли UUID нулевым.
func IsZeroUUID(id uuid.UUID) bool {
	return id == uuid.UUID{}
}

// ValidateUUID проверяет, что UUID не нулевой.
// Возвращает ErrInvalidInput если UUID пустой.
func ValidateUUID(id uuid.UUID, fieldName string) error {
	if IsZeroUUID(id) {
		return fmt.Errorf("%w: %s is required", database.ErrInvalidInput, fieldName)
	}
	return nil
}

// ValidateString проверяет, что строка не пустая.
// Возвращает ErrInvalidInput если строка пустая.
//
// Внимание: проверяет только на пустую строку, не на whitespace.
// Для проверки на whitespace используйте strings.TrimSpace перед вызовом.
func ValidateString(s string, fieldName string) error {
	if s == "" {
		return fmt.Errorf("%w: %s is required", database.ErrInvalidInput, fieldName)
	}
	return nil
}

// ValidateStringMaxLength проверяет, что строка не превышает максимальную длину.
func ValidateStringMaxLength(s string, fieldName string, maxLength int) error {
	if len(s) > maxLength {
		return fmt.Errorf("%w: %s exceeds maximum length of %d characters", database.ErrInvalidInput, fieldName, maxLength)
	}
	return nil
}

// ============================================================================
// BASE REPOSITORY
// ============================================================================

// Base предоставляет общие CRUD операции для репозиториев.
// T — тип модели, с которой работает репозиторий.
//
// Все методы:
//   - Используют таймаут (DefaultQueryTimeout или из контекста)
//   - Оборачивают ошибки через database.WrapDBError
//   - Возвращают ErrNotFound при отсутствии записи
type Base[T any] struct {
	querier database.Querier
	table   string
	columns []string
}

// Config содержит конфигурацию для создания базового репозитория.
type Config struct {
	Table   string
	Columns []string
}

// Validate проверяет валидность конфигурации.
func (c Config) Validate() error {
	if c.Table == "" {
		return errors.New("table name is required")
	}
	if len(c.Columns) == 0 {
		return errors.New("columns are required")
	}
	return nil
}

// NewBase создаёт базовый репозиторий.
//
// Параметры:
//   - q: Querier для выполнения запросов (может быть Pool или Tx)
//   - cfg: конфигурация с именем таблицы и списком колонок
//
// Возвращает ошибку, если конфигурация невалидна.
func NewBase[T any](q database.Querier, cfg Config) (*Base[T], error) {
	if err := cfg.Validate(); err != nil {
		return nil, fmt.Errorf("invalid repository config: %w", err)
	}
	return &Base[T]{
		querier: q,
		table:   cfg.Table,
		columns: cfg.Columns,
	}, nil
}

// MustNewBase создаёт базовый репозиторий или паникует при ошибке.
// Используйте только при инициализации приложения, не в runtime коде.
func MustNewBase[T any](q database.Querier, cfg Config) *Base[T] {
	b, err := NewBase[T](q, cfg)
	if err != nil {
		panic(fmt.Sprintf("failed to create base repository: %v", err))
	}
	return b
}

// Q возвращает Querier для прямого доступа к БД.
// Используйте для кастомных запросов, которые не покрываются базовыми методами.
func (r *Base[T]) Q() database.Querier {
	return r.querier
}

// Table возвращает имя таблицы.
func (r *Base[T]) Table() string {
	return r.table
}

// Columns возвращает список колонок.
func (r *Base[T]) Columns() []string {
	return r.columns
}

// ============================================================================
// QUERY BUILDERS
// ============================================================================

// SelectBuilder создаёт новый SelectBuilder для текущей таблицы.
// Автоматически добавляет все колонки репозитория.
func (r *Base[T]) SelectBuilder() squirrel.SelectBuilder {
	return psql.Select(r.columns...).From(r.table)
}

// InsertBuilder создаёт новый InsertBuilder для текущей таблицы.
func (r *Base[T]) InsertBuilder() squirrel.InsertBuilder {
	return psql.Insert(r.table)
}

// UpdateBuilder создаёт новый UpdateBuilder для текущей таблицы.
func (r *Base[T]) UpdateBuilder() squirrel.UpdateBuilder {
	return psql.Update(r.table)
}

// DeleteBuilder создаёт новый DeleteBuilder для текущей таблицы.
func (r *Base[T]) DeleteBuilder() squirrel.DeleteBuilder {
	return psql.Delete(r.table)
}

// ============================================================================
// CONTEXT HELPERS
// ============================================================================

// withTimeout возвращает контекст с таймаутом.
// Если контекст уже имеет deadline раньше DefaultQueryTimeout, используется он.
// Всегда возвращает cancel функцию, которую нужно вызвать для освобождения ресурсов.
func (r *Base[T]) withTimeout(ctx context.Context) (context.Context, context.CancelFunc) {
	// Проверяем, есть ли уже deadline в контексте
	if deadline, ok := ctx.Deadline(); ok {
		// Если deadline контекста ближе - используем его
		remaining := time.Until(deadline)
		if remaining < DefaultQueryTimeout {
			// Используем минимальный таймаут 1 секунда для безопасности
			if remaining < time.Second {
				remaining = time.Second
			}
			return context.WithTimeout(ctx, remaining)
		}
	}
	return context.WithTimeout(ctx, DefaultQueryTimeout)
}

// ============================================================================
// READ OPERATIONS
// ============================================================================

// GetByID получает запись по ID.
//
// Возвращает:
//   - *T: найденную запись
//   - ErrNotFound: если запись не найдена
//   - ErrInvalidInput: если idColumn пуст или id == nil
func (r *Base[T]) GetByID(ctx context.Context, idColumn string, id any) (*T, error) {
	if idColumn == "" {
		return nil, fmt.Errorf("%w: id column is required", database.ErrInvalidInput)
	}
	if id == nil {
		return nil, fmt.Errorf("%w: id is required", database.ErrInvalidInput)
	}

	query := r.SelectBuilder().Where(squirrel.Eq{idColumn: id})
	return r.GetOne(ctx, query)
}

// GetOne выполняет запрос и возвращает одну запись.
//
// Возвращает:
//   - *T: найденную запись
//   - ErrNotFound: если запись не найдена
//   - ErrTimeout: если превышен таймаут запроса
func (r *Base[T]) GetOne(ctx context.Context, query squirrel.SelectBuilder) (*T, error) {
	// Проверяем контекст перед выполнением запроса
	if err := ctx.Err(); err != nil {
		return nil, database.WrapDBError(err)
	}

	sql, args, err := query.ToSql()
	if err != nil {
		return nil, database.WrapDBError(err)
	}

	ctx, cancel := r.withTimeout(ctx)
	defer cancel()

	var dest T
	if err := pgxscan.Get(ctx, r.querier, &dest, sql, args...); err != nil {
		if pgxscan.NotFound(err) {
			return nil, database.ErrNotFound
		}
		return nil, database.WrapDBError(err)
	}
	return &dest, nil
}

// List выполняет запрос и возвращает список записей.
// Возвращает пустой слайс, если записей нет (не ошибка).
//
// Внимание: для больших выборок (>10000 записей) рассмотрите использование
// пагинации или курсоров для оптимизации памяти.
func (r *Base[T]) List(ctx context.Context, query squirrel.SelectBuilder) ([]T, error) {
	// Проверяем контекст перед выполнением запроса
	if err := ctx.Err(); err != nil {
		return nil, database.WrapDBError(err)
	}

	sql, args, err := query.ToSql()
	if err != nil {
		return nil, database.WrapDBError(err)
	}

	ctx, cancel := r.withTimeout(ctx)
	defer cancel()

	var dest []T
	if err := pgxscan.Select(ctx, r.querier, &dest, sql, args...); err != nil {
		return nil, database.WrapDBError(err)
	}

	// Гарантируем non-nil слайс для консистентности API
	if dest == nil {
		return []T{}, nil
	}
	return dest, nil
}

// ListByIDs получает записи по списку ID.
// Возвращает пустой слайс, если ids пуст или записи не найдены.
func (r *Base[T]) ListByIDs(ctx context.Context, idColumn string, ids []any) ([]T, error) {
	if len(ids) == 0 {
		return []T{}, nil
	}
	if idColumn == "" {
		return nil, fmt.Errorf("%w: id column is required", database.ErrInvalidInput)
	}

	query := r.SelectBuilder().Where(squirrel.Eq{idColumn: ids})
	return r.List(ctx, query)
}

// ListByUUIDs получает записи по списку UUID.
// Convenience wrapper над ListByIDs для типизированных UUID.
func (r *Base[T]) ListByUUIDs(ctx context.Context, idColumn string, ids []uuid.UUID) ([]T, error) {
	return r.ListByIDs(ctx, idColumn, UUIDsToAny(ids))
}

// FindOneBy находит одну запись по значению колонки.
//
// Возвращает:
//   - *T: найденную запись
//   - ErrNotFound: если запись не найдена
//   - ErrInvalidInput: если column пуст
func (r *Base[T]) FindOneBy(ctx context.Context, column string, value any) (*T, error) {
	if column == "" {
		return nil, fmt.Errorf("%w: column is required", database.ErrInvalidInput)
	}
	query := r.SelectBuilder().Where(squirrel.Eq{column: value}).Limit(1)
	return r.GetOne(ctx, query)
}

// ============================================================================
// COUNT OPERATIONS
// ============================================================================

// Count подсчитывает записи используя подзапрос.
//
// Внимание: метод использует подзапрос, что может быть неэффективно для сложных запросов.
// Для простых запросов предпочтительнее CountBy или создание COUNT запроса вручную.
//
// Производительность:
//   - Для простых запросов: используйте CountBy
//   - Для запросов с JOIN: рассмотрите оптимизацию через EXPLAIN ANALYZE
func (r *Base[T]) Count(ctx context.Context, query squirrel.SelectBuilder) (int64, error) {
	// Проверяем контекст перед выполнением
	if err := ctx.Err(); err != nil {
		return 0, database.WrapDBError(err)
	}

	fullSQL, args, err := query.ToSql()
	if err != nil {
		return 0, database.WrapDBError(err)
	}

	// Используем подзапрос для подсчета
	// ВАЖНО: не используем fmt.Sprintf для безопасности от SQL injection
	// squirrel уже экранировал параметры в fullSQL
	countSQL := "SELECT COUNT(*) FROM (" + fullSQL + ") AS count_subquery"

	ctx, cancel := r.withTimeout(ctx)
	defer cancel()

	var count int64
	if err := r.querier.QueryRow(ctx, countSQL, args...).Scan(&count); err != nil {
		return 0, database.WrapDBError(err)
	}
	return count, nil
}

// CountBy выполняет COUNT(*) с условием WHERE по одной колонке.
func (r *Base[T]) CountBy(ctx context.Context, column string, value any) (int64, error) {
	if column == "" {
		return 0, fmt.Errorf("%w: column is required", database.ErrInvalidInput)
	}

	query := psql.Select("COUNT(*)").From(r.table).Where(squirrel.Eq{column: value})
	sql, args, err := query.ToSql()
	if err != nil {
		return 0, database.WrapDBError(err)
	}

	ctx, cancel := r.withTimeout(ctx)
	defer cancel()

	var count int64
	if err := r.querier.QueryRow(ctx, sql, args...).Scan(&count); err != nil {
		return 0, database.WrapDBError(err)
	}
	return count, nil
}

// CountAll подсчитывает все записи в таблице.
func (r *Base[T]) CountAll(ctx context.Context) (int64, error) {
	query := psql.Select("COUNT(*)").From(r.table)
	sql, args, err := query.ToSql()
	if err != nil {
		return 0, database.WrapDBError(err)
	}

	ctx, cancel := r.withTimeout(ctx)
	defer cancel()

	var count int64
	if err := r.querier.QueryRow(ctx, sql, args...).Scan(&count); err != nil {
		return 0, database.WrapDBError(err)
	}
	return count, nil
}

// ============================================================================
// EXISTENCE CHECK
// ============================================================================

// Exists проверяет существование записи по условию.
// Оптимизирован для минимального количества данных (SELECT 1 ... LIMIT 1).
//
// Производительность:
//   - Использует SELECT 1 вместо SELECT * для экономии памяти
//   - LIMIT 1 останавливает сканирование после первой найденной записи
//   - Рекомендуется использовать индекс на проверяемой колонке
func (r *Base[T]) Exists(ctx context.Context, column string, value any) (bool, error) {
	// Проверяем контекст перед выполнением
	if err := ctx.Err(); err != nil {
		return false, database.WrapDBError(err)
	}

	if column == "" {
		return false, fmt.Errorf("%w: column is required", database.ErrInvalidInput)
	}

	query := psql.Select("1").From(r.table).Where(squirrel.Eq{column: value}).Limit(1)
	sql, args, err := query.ToSql()
	if err != nil {
		return false, database.WrapDBError(err)
	}

	ctx, cancel := r.withTimeout(ctx)
	defer cancel()

	var exists int
	err = r.querier.QueryRow(ctx, sql, args...).Scan(&exists)
	if err != nil {
		if database.IsNotFoundError(err) {
			return false, nil
		}
		return false, database.WrapDBError(err)
	}
	return true, nil
}

// ============================================================================
// WRITE OPERATIONS
// ============================================================================

// Create создаёт новую запись и возвращает её.
// Псевдоним для InsertReturning для семантической ясности.
func (r *Base[T]) Create(ctx context.Context, insert squirrel.InsertBuilder) (*T, error) {
	return r.InsertReturning(ctx, insert)
}

// InsertReturning выполняет INSERT с RETURNING * и возвращает созданную запись.
func (r *Base[T]) InsertReturning(ctx context.Context, insert squirrel.InsertBuilder) (*T, error) {
	query := insert.Suffix("RETURNING *")
	sql, args, err := query.ToSql()
	if err != nil {
		return nil, database.WrapDBError(err)
	}

	ctx, cancel := r.withTimeout(ctx)
	defer cancel()

	var dest T
	if err := pgxscan.Get(ctx, r.querier, &dest, sql, args...); err != nil {
		return nil, database.WrapDBError(err)
	}
	return &dest, nil
}

// BatchInsertReturning выполняет множественную вставку и возвращает созданные записи.
//
// Параметры:
//   - columns: список колонок для вставки
//   - items: слайс элементов для вставки
//   - valuesFunc: функция для извлечения значений из элемента
//
// Для больших батчей (>MaxBatchSize) автоматически разбивает на чанки.
// Для очень больших объемов (>10000) рассмотрите использование COPY.
//
// Производительность:
//   - Малые батчи (<100): один запрос
//   - Средние батчи (100-1000): один запрос с множественными VALUES
//   - Большие батчи (>1000): автоматическое разбиение на чанки
func (r *Base[T]) BatchInsertReturning(ctx context.Context, columns []string, items []T, valuesFunc func(T) []any) ([]T, error) {
	// Проверяем контекст перед выполнением
	if err := ctx.Err(); err != nil {
		return nil, database.WrapDBError(err)
	}

	if len(items) == 0 {
		return []T{}, nil
	}
	if len(columns) == 0 {
		return nil, fmt.Errorf("%w: columns are required", database.ErrInvalidInput)
	}
	if valuesFunc == nil {
		return nil, fmt.Errorf("%w: valuesFunc is required", database.ErrInvalidInput)
	}

	// Для больших батчей разбиваем на чанки
	if len(items) > MaxBatchSize {
		return r.batchInsertChunked(ctx, columns, items, valuesFunc)
	}

	return r.batchInsertSingle(ctx, columns, items, valuesFunc)
}

// batchInsertChunked выполняет вставку больших батчей по частям.
// Проверяет контекст между чанками для возможности отмены.
func (r *Base[T]) batchInsertChunked(ctx context.Context, columns []string, items []T, valuesFunc func(T) []any) ([]T, error) {
	allResults := make([]T, 0, len(items))

	for i := 0; i < len(items); i += MaxBatchSize {
		// Проверяем контекст перед каждым чанком
		if err := ctx.Err(); err != nil {
			return nil, database.WrapDBError(err)
		}

		end := min(i+MaxBatchSize, len(items))
		chunk := items[i:end]

		results, err := r.batchInsertSingle(ctx, columns, chunk, valuesFunc)
		if err != nil {
			return nil, err
		}

		allResults = append(allResults, results...)
	}

	return allResults, nil
}

// batchInsertSingle выполняет вставку одного батча.
func (r *Base[T]) batchInsertSingle(ctx context.Context, columns []string, items []T, valuesFunc func(T) []any) ([]T, error) {
	insert := r.InsertBuilder().Columns(columns...)
	for _, item := range items {
		insert = insert.Values(valuesFunc(item)...)
	}

	query := insert.Suffix("RETURNING *")
	sql, args, err := query.ToSql()
	if err != nil {
		return nil, database.WrapDBError(err)
	}

	ctx, cancel := r.withTimeout(ctx)
	defer cancel()

	var dest []T
	if err := pgxscan.Select(ctx, r.querier, &dest, sql, args...); err != nil {
		return nil, database.WrapDBError(err)
	}

	if dest == nil {
		return []T{}, nil
	}
	return dest, nil
}

// Update выполняет UPDATE с RETURNING * и возвращает обновлённую запись.
//
// Возвращает:
//   - *T: обновлённую запись
//   - ErrNotFound: если запись не найдена
//   - ErrTimeout: если превышен таймаут запроса
//
// Внимание: метод не проверяет, были ли обновлены строки.
// Если нужно гарантировать обновление, используйте UpdateWithCheck.
func (r *Base[T]) Update(ctx context.Context, update squirrel.UpdateBuilder) (*T, error) {
	// Проверяем контекст перед выполнением запроса
	if err := ctx.Err(); err != nil {
		return nil, database.WrapDBError(err)
	}

	query := update.Suffix("RETURNING *")
	sql, args, err := query.ToSql()
	if err != nil {
		return nil, database.WrapDBError(err)
	}

	ctx, cancel := r.withTimeout(ctx)
	defer cancel()

	var dest T
	if err := pgxscan.Get(ctx, r.querier, &dest, sql, args...); err != nil {
		if pgxscan.NotFound(err) {
			return nil, database.ErrNotFound
		}
		return nil, database.WrapDBError(err)
	}
	return &dest, nil
}

// Delete удаляет запись по ID.
//
// Возвращает:
//   - nil: если запись успешно удалена
//   - ErrNotFound: если запись не найдена
//   - ErrInvalidInput: если idColumn пуст или id == nil
//   - ErrTimeout: если превышен таймаут запроса
func (r *Base[T]) Delete(ctx context.Context, idColumn string, id any) error {
	// Проверяем контекст перед выполнением
	if err := ctx.Err(); err != nil {
		return database.WrapDBError(err)
	}

	if idColumn == "" {
		return fmt.Errorf("%w: id column is required", database.ErrInvalidInput)
	}
	if id == nil {
		return fmt.Errorf("%w: id is required", database.ErrInvalidInput)
	}

	ctx, cancel := r.withTimeout(ctx)
	defer cancel()

	del := r.DeleteBuilder().Where(squirrel.Eq{idColumn: id})
	sql, args, err := del.ToSql()
	if err != nil {
		return database.WrapDBError(err)
	}

	tag, err := r.querier.Exec(ctx, sql, args...)
	if err != nil {
		return database.WrapDBError(err)
	}

	if tag.RowsAffected() == 0 {
		return database.ErrNotFound
	}
	return nil
}

// DeleteWhere удаляет записи по произвольному условию.
// Возвращает количество удалённых записей.
func (r *Base[T]) DeleteWhere(ctx context.Context, pred squirrel.Sqlizer) (int64, error) {
	ctx, cancel := r.withTimeout(ctx)
	defer cancel()

	del := r.DeleteBuilder().Where(pred)
	sql, args, err := del.ToSql()
	if err != nil {
		return 0, database.WrapDBError(err)
	}

	tag, err := r.querier.Exec(ctx, sql, args...)
	if err != nil {
		return 0, database.WrapDBError(err)
	}

	return tag.RowsAffected(), nil
}

// ============================================================================
// RAW QUERY EXECUTION
// ============================================================================

// ExecRaw выполняет произвольный SQL запрос.
// Используйте для сложных запросов, которые не покрываются builder'ами.
//
// Внимание: используйте с осторожностью! Убедитесь, что SQL запрос безопасен от SQL injection.
// Предпочтительно использовать методы репозитория или squirrel builders.
func (r *Base[T]) ExecRaw(ctx context.Context, sql string, args ...any) (int64, error) {
	// Проверяем контекст перед выполнением
	if err := ctx.Err(); err != nil {
		return 0, database.WrapDBError(err)
	}

	ctx, cancel := r.withTimeout(ctx)
	defer cancel()

	tag, err := r.querier.Exec(ctx, sql, args...)
	if err != nil {
		return 0, database.WrapDBError(err)
	}
	return tag.RowsAffected(), nil
}

// QueryRaw выполняет произвольный SELECT запрос и возвращает результаты.
//
// Внимание: используйте с осторожностью! Убедитесь, что SQL запрос безопасен от SQL injection.
// Предпочтительно использовать методы репозитория или squirrel builders.
func (r *Base[T]) QueryRaw(ctx context.Context, dest any, sql string, args ...any) error {
	// Проверяем контекст перед выполнением
	if err := ctx.Err(); err != nil {
		return database.WrapDBError(err)
	}

	ctx, cancel := r.withTimeout(ctx)
	defer cancel()

	if err := pgxscan.Select(ctx, r.querier, dest, sql, args...); err != nil {
		return database.WrapDBError(err)
	}
	return nil
}

// QueryRowRaw выполняет произвольный SELECT запрос и возвращает одну строку.
//
// Внимание: используйте с осторожностью! Убедитесь, что SQL запрос безопасен от SQL injection.
// Предпочтительно использовать методы репозитория или squirrel builders.
func (r *Base[T]) QueryRowRaw(ctx context.Context, dest any, sql string, args ...any) error {
	// Проверяем контекст перед выполнением
	if err := ctx.Err(); err != nil {
		return database.WrapDBError(err)
	}

	ctx, cancel := r.withTimeout(ctx)
	defer cancel()

	if err := pgxscan.Get(ctx, r.querier, dest, sql, args...); err != nil {
		if pgxscan.NotFound(err) {
			return database.ErrNotFound
		}
		return database.WrapDBError(err)
	}
	return nil
}
