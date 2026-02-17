package middleware

import (
	"bytes"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/heartmarshall/myenglish-backend/pkg/ctxutil"
)

func TestLogger_Success(t *testing.T) {
	var buf bytes.Buffer
	logger := slog.New(slog.NewJSONHandler(&buf, &slog.HandlerOptions{Level: slog.LevelInfo}))

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	wrapped := Logger(logger)(handler)

	req := httptest.NewRequest(http.MethodGet, "/test-path", nil)
	rec := httptest.NewRecorder()

	wrapped.ServeHTTP(rec, req)

	logOutput := buf.String()
	if !strings.Contains(logOutput, "http.request") {
		t.Errorf("expected log message %q, got %q", "http.request", logOutput)
	}
	if !strings.Contains(logOutput, "GET") {
		t.Errorf("expected log to contain method GET, got %q", logOutput)
	}
	if !strings.Contains(logOutput, "/test-path") {
		t.Errorf("expected log to contain path /test-path, got %q", logOutput)
	}
	if !strings.Contains(logOutput, `"status":200`) {
		t.Errorf("expected log to contain status 200, got %q", logOutput)
	}
	if !strings.Contains(logOutput, "duration") {
		t.Errorf("expected log to contain duration, got %q", logOutput)
	}
	// INFO level for 200
	if !strings.Contains(logOutput, "INFO") {
		t.Errorf("expected INFO level for status 200, got %q", logOutput)
	}
}

func TestLogger_ServerError(t *testing.T) {
	var buf bytes.Buffer
	logger := slog.New(slog.NewJSONHandler(&buf, &slog.HandlerOptions{Level: slog.LevelInfo}))

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	})

	wrapped := Logger(logger)(handler)

	req := httptest.NewRequest(http.MethodPost, "/error", nil)
	rec := httptest.NewRecorder()

	wrapped.ServeHTTP(rec, req)

	logOutput := buf.String()
	if !strings.Contains(logOutput, "ERROR") {
		t.Errorf("expected ERROR level for status 500, got %q", logOutput)
	}
	if !strings.Contains(logOutput, `"status":500`) {
		t.Errorf("expected log to contain status 500, got %q", logOutput)
	}
}

func TestLogger_IncludesRequestID(t *testing.T) {
	var buf bytes.Buffer
	logger := slog.New(slog.NewJSONHandler(&buf, &slog.HandlerOptions{Level: slog.LevelInfo}))

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	wrapped := Logger(logger)(handler)

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	ctx := ctxutil.WithRequestID(req.Context(), "test-request-id-123")
	req = req.WithContext(ctx)
	rec := httptest.NewRecorder()

	wrapped.ServeHTTP(rec, req)

	logOutput := buf.String()
	if !strings.Contains(logOutput, "test-request-id-123") {
		t.Errorf("expected log to contain request_id %q, got %q", "test-request-id-123", logOutput)
	}
}
