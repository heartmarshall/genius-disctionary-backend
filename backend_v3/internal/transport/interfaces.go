package transport

import (
	"context"
	"net/http"

	"github.com/vektah/gqlparser/v2/gqlerror"
)

// ErrorHandlerInterface определяет интерфейс для обработки ошибок.
// Позволяет легко мокировать обработчик ошибок в тестах.
// Возвращает *gqlerror.Error для типобезопасности, который также реализует error.
type ErrorHandlerInterface interface {
	HandleError(ctx context.Context, err error) *gqlerror.Error
}

// MiddlewareFactory создает HTTP middleware.
type MiddlewareFactory interface {
	RequestID() func(http.Handler) http.Handler
	Logging(logger interface{}) func(http.Handler) http.Handler
	Recovery(logger interface{}) func(http.Handler) http.Handler
	Timeout(timeout interface{}) func(http.Handler) http.Handler
	CORS(origins, methods, headers []string) func(http.Handler) http.Handler
	SecurityHeaders() func(http.Handler) http.Handler
}
