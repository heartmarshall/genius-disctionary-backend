package middleware

import (
	"bytes"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestRecovery_NoPanic(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(&bytes.Buffer{}, nil))
	called := false

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
		w.WriteHeader(http.StatusOK)
	})

	wrapped := Recovery(logger)(handler)

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()

	wrapped.ServeHTTP(rec, req)

	if !called {
		t.Error("expected handler to be called")
	}
	if rec.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d", http.StatusOK, rec.Code)
	}
}

func TestRecovery_Panic(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(&bytes.Buffer{}, nil))

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		panic("something went wrong")
	})

	wrapped := Recovery(logger)(handler)

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	rec := httptest.NewRecorder()

	wrapped.ServeHTTP(rec, req)

	if rec.Code != http.StatusInternalServerError {
		t.Errorf("expected status %d, got %d", http.StatusInternalServerError, rec.Code)
	}

	body := strings.TrimSpace(rec.Body.String())
	if body != "internal server error" {
		t.Errorf("expected body %q, got %q", "internal server error", body)
	}
}

func TestRecovery_PanicStringError(t *testing.T) {
	var buf bytes.Buffer
	logger := slog.New(slog.NewTextHandler(&buf, &slog.HandlerOptions{Level: slog.LevelError}))

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		panic("string error message")
	})

	wrapped := Recovery(logger)(handler)

	req := httptest.NewRequest(http.MethodGet, "/error-path", nil)
	rec := httptest.NewRecorder()

	wrapped.ServeHTTP(rec, req)

	if rec.Code != http.StatusInternalServerError {
		t.Errorf("expected status %d, got %d", http.StatusInternalServerError, rec.Code)
	}

	logOutput := buf.String()
	if !strings.Contains(logOutput, "panic recovered") {
		t.Errorf("expected log to contain %q, got %q", "panic recovered", logOutput)
	}
	if !strings.Contains(logOutput, "string error message") {
		t.Errorf("expected log to contain panic value, got %q", logOutput)
	}
}
