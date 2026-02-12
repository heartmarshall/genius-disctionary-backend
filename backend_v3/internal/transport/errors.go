package transport

import (
	"context"
	"errors"
	"log/slog"
	"runtime/debug"

	"github.com/99designs/gqlgen/graphql"
	"github.com/heartmarshall/my-english/internal/database"
	"github.com/heartmarshall/my-english/internal/service/types"
	"github.com/vektah/gqlparser/v2/gqlerror"
)

// ErrorCode — коды ошибок для GraphQL.
type ErrorCode string

const (
	CodeNotFound           ErrorCode = "NOT_FOUND"
	CodeAlreadyExists      ErrorCode = "ALREADY_EXISTS"
	CodeInvalidInput       ErrorCode = "INVALID_INPUT"
	CodeInternal           ErrorCode = "INTERNAL_ERROR"
	CodeTimeout            ErrorCode = "TIMEOUT"
	CodeServiceUnavailable ErrorCode = "SERVICE_UNAVAILABLE"
)

// ErrorHandler обрабатывает ошибки и преобразует их в GraphQL ошибки.
type ErrorHandler struct {
	logger  *slog.Logger
	devMode bool
}

// NewErrorHandler создает новый обработчик ошибок.
func NewErrorHandler(logger *slog.Logger, devMode bool) *ErrorHandler {
	return &ErrorHandler{
		logger:  logger,
		devMode: devMode,
	}
}

// HandleError преобразует service ошибки в GraphQL ошибки.
// Возвращает *gqlerror.Error для типобезопасности и прямой совместимости с GraphQL.
func (h *ErrorHandler) HandleError(ctx context.Context, err error) *gqlerror.Error {
	if err == nil {
		return nil
	}

	path := graphql.GetPath(ctx)
	extensions := make(map[string]interface{})

	// 1. Ошибка валидации
	if types.IsValidationError(err) {
		var validationErr *types.ValidationError
		if errors.As(err, &validationErr) {
			extensions["code"] = CodeInvalidInput
			if validationErr.Field != "" {
				extensions["field"] = validationErr.Field
			}
			return &gqlerror.Error{
				Message:    validationErr.Message,
				Path:       path,
				Extensions: extensions,
			}
		}
		return &gqlerror.Error{
			Message:    err.Error(),
			Path:       path,
			Extensions: map[string]interface{}{"code": CodeInvalidInput},
		}
	}

	// 2. Not Found
	if errors.Is(err, types.ErrNotFound) || database.IsNotFoundError(err) {
		return &gqlerror.Error{
			Message:    "Entity not found",
			Path:       path,
			Extensions: map[string]interface{}{"code": CodeNotFound},
		}
	}

	// 3. Already Exists / Duplicate
	if errors.Is(err, types.ErrAlreadyExists) || database.IsDuplicateError(err) {
		return &gqlerror.Error{
			Message:    "Entity already exists",
			Path:       path,
			Extensions: map[string]interface{}{"code": CodeAlreadyExists},
		}
	}

	// 4. Invalid Input (общий)
	if errors.Is(err, types.ErrInvalidInput) || database.IsConstraintError(err) {
		msg := "Invalid input data"
		if h.devMode {
			msg = err.Error()
		}
		return &gqlerror.Error{
			Message:    msg,
			Path:       path,
			Extensions: map[string]interface{}{"code": CodeInvalidInput},
		}
	}

	// 5. Timeout / Context canceled
	if database.IsTimeoutError(err) || errors.Is(err, context.DeadlineExceeded) || errors.Is(err, context.Canceled) {
		return &gqlerror.Error{
			Message:    "Request timeout",
			Path:       path,
			Extensions: map[string]interface{}{"code": CodeTimeout},
		}
	}

	// 6. Connection errors
	if database.IsConnectionError(err) {
		return &gqlerror.Error{
			Message:    "Service temporarily unavailable",
			Path:       path,
			Extensions: map[string]interface{}{"code": CodeServiceUnavailable},
		}
	}

	// 7. Internal Error - логируем и скрываем детали в продакшене
	h.logInternalError(ctx, err, path)

	msg := "Internal server error"
	if h.devMode {
		msg = err.Error()
		extensions["error"] = err.Error()
		if stack := debug.Stack(); len(stack) > 0 {
			extensions["stack"] = string(stack)
		}
	}

	extensions["code"] = CodeInternal
	return &gqlerror.Error{
		Message:    msg,
		Path:       path,
		Extensions: extensions,
	}
}

// logInternalError логирует внутренние ошибки с контекстом.
func (h *ErrorHandler) logInternalError(ctx context.Context, err error, path interface{}) {
	attrs := []any{
		slog.String("error", err.Error()),
		slog.Any("path", path),
	}

	// Добавляем request ID, если есть
	if requestID := GetRequestID(ctx); requestID != "" {
		attrs = append(attrs, slog.String("request_id", requestID))
	}

	// Добавляем stack trace в dev режиме
	if h.devMode {
		attrs = append(attrs, slog.String("stack", string(debug.Stack())))
	}

	h.logger.Error("internal error", attrs...)
}

// HandleError - функция-обертка для обратной совместимости.
// Использует глобальный обработчик ошибок (будет установлен через SetDefaultErrorHandler).
var defaultErrorHandler *ErrorHandler

// SetDefaultErrorHandler устанавливает обработчик ошибок по умолчанию.
func SetDefaultErrorHandler(handler *ErrorHandler) {
	defaultErrorHandler = handler
}

// HandleError преобразует service ошибки в GraphQL ошибки (legacy функция).
// Использует defaultErrorHandler, если он установлен, иначе создает временный обработчик.
// Возвращает *gqlerror.Error, который реализует интерфейс error, поэтому может использоваться
// везде, где ожидается error.
func HandleError(ctx context.Context, err error) *gqlerror.Error {
	if err == nil {
		return nil
	}

	if defaultErrorHandler != nil {
		return defaultErrorHandler.HandleError(ctx, err)
	}

	// Fallback для обратной совместимости
	handler := NewErrorHandler(slog.Default(), true)
	return handler.HandleError(ctx, err)
}
