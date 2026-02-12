package context

import (
	"context"
	"log/slog"
)

type ctxKey string

const loggerKey ctxKey = "logger"

// WithLogger добавляет логгер в контекст.
func WithLogger(ctx context.Context, logger *slog.Logger) context.Context {
	return context.WithValue(ctx, loggerKey, logger)
}

// LoggerFromContext извлекает логгер из контекста.
// Если логгер не найден, возвращает default логгер.
func LoggerFromContext(ctx context.Context) *slog.Logger {
	if logger, ok := ctx.Value(loggerKey).(*slog.Logger); ok {
		return logger
	}
	return slog.Default()
}

// L — короткий алиас для LoggerFromContext.
func L(ctx context.Context) *slog.Logger {
	return LoggerFromContext(ctx)
}
