package middleware

import (
	"log/slog"
	"net/http"
	"runtime/debug"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/heartmarshall/my-english/internal/transport"
	ctx "github.com/heartmarshall/my-english/pkg/context"
)

// HTTP header names
const (
	headerRequestID                     = "X-Request-ID"
	headerOrigin                        = "Origin"
	headerAccessControlAllowOrigin      = "Access-Control-Allow-Origin"
	headerAccessControlAllowMethods     = "Access-Control-Allow-Methods"
	headerAccessControlAllowHeaders     = "Access-Control-Allow-Headers"
	headerAccessControlAllowCredentials = "Access-Control-Allow-Credentials"
	headerAccessControlMaxAge           = "Access-Control-Max-Age"
	headerContentType                   = "Content-Type"
	headerXContentTypeOptions           = "X-Content-Type-Options"
	headerXFrameOptions                 = "X-Frame-Options"
	headerXXSSProtection                = "X-XSS-Protection"
	headerStrictTransportSecurity       = "Strict-Transport-Security"
	headerContentSecurityPolicy         = "Content-Security-Policy"
)

// Security header values
const (
	contentTypeJSON              = "application/json"
	xContentTypeOptionsNoSniff   = "nosniff"
	xFrameOptionsDeny            = "DENY"
	xXSSProtectionBlock          = "1; mode=block"
	contentSecurityPolicyDefault = "default-src 'self'"
)

// CORS constants
const (
	// DefaultCORSMaxAge is the default max age for CORS preflight requests (1 hour in seconds)
	DefaultCORSMaxAge = 3600
	// CORSWildcard represents the wildcard origin
	CORSWildcard = "*"
	// CORSCredentialsTrue is the string value for enabling CORS credentials
	CORSCredentialsTrue = "true"
)

// HSTS (HTTP Strict Transport Security) constants
const (
	// HSTSMaxAgeOneYear is the max age for HSTS header (1 year in seconds)
	HSTSMaxAgeOneYear = 31536000
	// HSTSIncludeSubDomains is the HSTS directive to include subdomains
	HSTSIncludeSubDomains = "includeSubDomains"
)

// Default CORS values
var (
	// DefaultCORSMethods are the default allowed HTTP methods for CORS
	DefaultCORSMethods = []string{"GET", "POST", "OPTIONS"}
	// DefaultCORSHeaders are the default allowed headers for CORS
	DefaultCORSHeaders = []string{"Content-Type", "Authorization", "X-Request-ID"}
)

// RequestIDMiddleware добавляет уникальный request ID к каждому запросу.
func RequestIDMiddleware() func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Проверяем, есть ли уже request ID в заголовке (для distributed tracing)
			requestID := r.Header.Get(headerRequestID)
			if requestID == "" {
				requestID = uuid.New().String()
			}

			// Добавляем в контекст
			ctx := transport.WithRequestID(r.Context(), requestID)
			r = r.WithContext(ctx)

			// Добавляем в заголовок ответа
			w.Header().Set(headerRequestID, requestID)

			next.ServeHTTP(w, r)
		})
	}
}

// LoggingMiddleware логирует HTTP запросы.
// Если logger равен nil, используется slog.Default().
func LoggingMiddleware(logger *slog.Logger) func(http.Handler) http.Handler {
	// Валидация: используем default logger, если передан nil
	if logger == nil {
		logger = slog.Default()
	}

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()

			// Wrap response writer to capture status code
			wrapped := &responseWriter{ResponseWriter: w, statusCode: http.StatusOK}

			// Добавляем логгер в контекст
			requestCtx := ctx.WithLogger(r.Context(), logger)
			r = r.WithContext(requestCtx)

			next.ServeHTTP(wrapped, r)

			duration := time.Since(start)

			attrs := []any{
				slog.String("method", r.Method),
				slog.String("path", r.URL.Path),
				slog.String("query", r.URL.RawQuery),
				slog.Int("status", wrapped.statusCode),
				slog.Duration("duration", duration),
				slog.String("remote_addr", r.RemoteAddr),
				slog.String("user_agent", r.UserAgent()),
			}

			// Добавляем request ID, если есть
			if requestID := transport.GetRequestID(r.Context()); requestID != "" {
				attrs = append(attrs, slog.String("request_id", requestID))
			}

			// Логируем с соответствующим уровнем
			if wrapped.statusCode >= 500 {
				logger.Error("http request", attrs...)
			} else if wrapped.statusCode >= 400 {
				logger.Warn("http request", attrs...)
			} else {
				logger.Info("http request", attrs...)
			}
		})
	}
}

// RecoveryMiddleware обрабатывает паники.
// Если logger равен nil, используется slog.Default().
func RecoveryMiddleware(logger *slog.Logger) func(http.Handler) http.Handler {
	// Валидация: используем default logger, если передан nil
	if logger == nil {
		logger = slog.Default()
	}

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			defer func() {
				if err := recover(); err != nil {
					stack := debug.Stack()

					attrs := []any{
						slog.Any("error", err),
						slog.String("path", r.URL.Path),
						slog.String("method", r.Method),
						slog.String("stack", string(stack)),
					}

					// Добавляем request ID, если есть
					if requestID := transport.GetRequestID(r.Context()); requestID != "" {
						attrs = append(attrs, slog.String("request_id", requestID))
					}

					logger.Error("panic recovered", attrs...)

					// Отправляем ответ только если еще не отправлен
					if !wrappedResponseWriter(w).written {
						w.Header().Set(headerContentType, contentTypeJSON)
						w.WriteHeader(http.StatusInternalServerError)
						w.Write([]byte(`{"error":"Internal Server Error"}`))
					}
				}
			}()
			next.ServeHTTP(w, r)
		})
	}
}

// TimeoutMiddleware добавляет таймаут к контексту запроса.
// Использует http.TimeoutHandler для правильной обработки таймаутов HTTP запросов.
// TimeoutHandler создает контекст с таймаутом и передает его в следующий обработчик,
// что позволяет downstream сервисам и репозиториям корректно обрабатывать таймауты.
//
// Валидация: если timeout <= 0, возвращается no-op middleware (таймаут не применяется).
// Рекомендуется использовать timeout >= 1 секунда для production.
func TimeoutMiddleware(timeout time.Duration) func(http.Handler) http.Handler {
	// Валидация: проверяем корректность таймаута
	if timeout <= 0 {
		// Логируем предупреждение, если таймаут некорректен
		slog.Default().Warn("TimeoutMiddleware: invalid timeout, middleware disabled",
			slog.Duration("timeout", timeout))
		// Возвращаем no-op middleware, если таймаут некорректен
		return func(next http.Handler) http.Handler {
			return next
		}
	}

	// Валидация: предупреждаем о слишком маленьких таймаутах
	const minRecommendedTimeout = time.Second
	if timeout < minRecommendedTimeout {
		slog.Default().Warn("TimeoutMiddleware: timeout is very short",
			slog.Duration("timeout", timeout),
			slog.Duration("recommended_min", minRecommendedTimeout))
	}

	return func(next http.Handler) http.Handler {
		// http.TimeoutHandler создает контекст с таймаутом и правильно обрабатывает
		// ситуацию, когда обработчик не успевает ответить в срок (возвращает 503).
		return http.TimeoutHandler(next, timeout, "Request timeout")
	}
}

// CORSMiddleware добавляет CORS заголовки.
// Валидация входных параметров:
//   - allowedOrigins: если пусто, используется ["*"] (не рекомендуется для production)
//   - allowedMethods: если пусто, используется ["GET", "POST", "OPTIONS"]
//   - allowedHeaders: если пусто, используется ["Content-Type", "Authorization", "X-Request-ID"]
//
// ВАЖНО: Использование "*" в allowedOrigins с credentials=true запрещено спецификацией CORS.
// Если используется "*", credentials будут установлены в false.
func CORSMiddleware(allowedOrigins []string, allowedMethods []string, allowedHeaders []string) func(http.Handler) http.Handler {
	// Валидация и установка значений по умолчанию
	if len(allowedOrigins) == 0 {
		allowedOrigins = []string{CORSWildcard}
		slog.Default().Warn("CORSMiddleware: no origins specified, using wildcard (not recommended for production)")
	}
	if len(allowedMethods) == 0 {
		allowedMethods = DefaultCORSMethods
	}
	if len(allowedHeaders) == 0 {
		allowedHeaders = DefaultCORSHeaders
	}

	// Валидация: проверяем наличие wildcard в origins
	hasWildcard := false
	for _, origin := range allowedOrigins {
		if origin == CORSWildcard {
			hasWildcard = true
			break
		}
	}

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			origin := r.Header.Get(headerOrigin)

			// Проверяем origin
			allowed := false
			for _, allowedOrigin := range allowedOrigins {
				if allowedOrigin == CORSWildcard || allowedOrigin == origin {
					allowed = true
					break
				}
			}

			if allowed {
				// Валидация: если используется wildcard, нельзя устанавливать credentials=true
				// Это запрещено спецификацией CORS (RFC 6454)
				if hasWildcard {
					// Если wildcard в списке, всегда используем "*" и НЕ устанавливаем credentials
					w.Header().Set(headerAccessControlAllowOrigin, CORSWildcard)
					// Явно НЕ устанавливаем Access-Control-Allow-Credentials для wildcard
				} else {
					// Если только конкретные origins, используем origin из запроса и можем установить credentials
					if origin != "" {
						w.Header().Set(headerAccessControlAllowOrigin, origin)
						w.Header().Set(headerAccessControlAllowCredentials, CORSCredentialsTrue)
					}
				}

				w.Header().Set(headerAccessControlAllowMethods, strings.Join(allowedMethods, ", "))
				w.Header().Set(headerAccessControlAllowHeaders, strings.Join(allowedHeaders, ", "))
				w.Header().Set(headerAccessControlMaxAge, formatCORSMaxAge(DefaultCORSMaxAge))
			}

			// Обрабатываем preflight запросы
			if r.Method == "OPTIONS" {
				w.WriteHeader(http.StatusNoContent)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

// SecurityHeadersMiddleware добавляет заголовки безопасности.
func SecurityHeadersMiddleware() func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Защита от XSS
			w.Header().Set(headerXContentTypeOptions, xContentTypeOptionsNoSniff)
			w.Header().Set(headerXFrameOptions, xFrameOptionsDeny)
			w.Header().Set(headerXXSSProtection, xXSSProtectionBlock)

			// Strict Transport Security (только для HTTPS)
			if r.TLS != nil {
				hstsValue := formatHSTSMaxAge(HSTSMaxAgeOneYear, HSTSIncludeSubDomains)
				w.Header().Set(headerStrictTransportSecurity, hstsValue)
			}

			// Content Security Policy
			// Для GraphQL Playground используем более мягкий CSP, чтобы разрешить внешние скрипты
			if r.URL.Path == "/" {
				// CSP для GraphQL Playground: разрешаем внешние скрипты и стили
				csp := "default-src 'self'; script-src 'self' 'unsafe-inline' 'unsafe-eval' https://cdn.jsdelivr.net; style-src 'self' 'unsafe-inline' https://cdn.jsdelivr.net; img-src 'self' data: https:; font-src 'self' data: https://cdn.jsdelivr.net;"
				w.Header().Set(headerContentSecurityPolicy, csp)
			} else {
				// Строгий CSP для всех остальных путей
				w.Header().Set(headerContentSecurityPolicy, contentSecurityPolicyDefault)
			}

			next.ServeHTTP(w, r)
		})
	}
}

// formatCORSMaxAge formats the CORS max age value as a string.
func formatCORSMaxAge(seconds int) string {
	return "max-age=" + strconv.Itoa(seconds)
}

// formatHSTSMaxAge formats the HSTS max age value with optional directives.
func formatHSTSMaxAge(seconds int, directives ...string) string {
	parts := []string{"max-age=" + strconv.Itoa(seconds)}
	parts = append(parts, directives...)
	return strings.Join(parts, "; ")
}

type responseWriter struct {
	http.ResponseWriter
	statusCode int
	written    bool
}

func (rw *responseWriter) WriteHeader(code int) {
	if !rw.written {
		rw.statusCode = code
		rw.written = true
		rw.ResponseWriter.WriteHeader(code)
	}
}

func (rw *responseWriter) Write(b []byte) (int, error) {
	if !rw.written {
		rw.WriteHeader(http.StatusOK)
	}
	return rw.ResponseWriter.Write(b)
}

// wrappedResponseWriter пытается извлечь responseWriter из http.ResponseWriter.
func wrappedResponseWriter(w http.ResponseWriter) *responseWriter {
	if rw, ok := w.(*responseWriter); ok {
		return rw
	}
	return &responseWriter{ResponseWriter: w, statusCode: http.StatusOK, written: false}
}
