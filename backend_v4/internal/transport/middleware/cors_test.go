package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/heartmarshall/myenglish-backend/internal/config"
)

func TestCORS_Preflight(t *testing.T) {
	cfg := config.CORSConfig{
		AllowedOrigins:   "https://example.com",
		AllowedMethods:   "GET,POST,OPTIONS",
		AllowedHeaders:   "Authorization,Content-Type",
		AllowCredentials: true,
		MaxAge:           86400,
	}

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Error("handler should not be called for preflight")
	})

	wrapped := CORS(cfg)(handler)

	req := httptest.NewRequest(http.MethodOptions, "/graphql", nil)
	req.Header.Set("Origin", "https://example.com")
	rec := httptest.NewRecorder()

	wrapped.ServeHTTP(rec, req)

	if rec.Code != http.StatusNoContent {
		t.Errorf("expected status %d, got %d", http.StatusNoContent, rec.Code)
	}

	if got := rec.Header().Get("Access-Control-Allow-Origin"); got != "https://example.com" {
		t.Errorf("expected Access-Control-Allow-Origin %q, got %q", "https://example.com", got)
	}
	if got := rec.Header().Get("Access-Control-Allow-Methods"); got != "GET,POST,OPTIONS" {
		t.Errorf("expected Access-Control-Allow-Methods %q, got %q", "GET,POST,OPTIONS", got)
	}
	if got := rec.Header().Get("Access-Control-Allow-Headers"); got != "Authorization,Content-Type" {
		t.Errorf("expected Access-Control-Allow-Headers %q, got %q", "Authorization,Content-Type", got)
	}
	if got := rec.Header().Get("Access-Control-Allow-Credentials"); got != "true" {
		t.Errorf("expected Access-Control-Allow-Credentials %q, got %q", "true", got)
	}
	if got := rec.Header().Get("Access-Control-Max-Age"); got != "86400" {
		t.Errorf("expected Access-Control-Max-Age %q, got %q", "86400", got)
	}
}

func TestCORS_AllowedOrigin(t *testing.T) {
	cfg := config.CORSConfig{
		AllowedOrigins:   "https://example.com,https://other.com",
		AllowedMethods:   "GET,POST",
		AllowedHeaders:   "Authorization",
		AllowCredentials: true,
		MaxAge:           3600,
	}

	called := false
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
		w.WriteHeader(http.StatusOK)
	})

	wrapped := CORS(cfg)(handler)

	req := httptest.NewRequest(http.MethodGet, "/api", nil)
	req.Header.Set("Origin", "https://example.com")
	rec := httptest.NewRecorder()

	wrapped.ServeHTTP(rec, req)

	if !called {
		t.Error("expected handler to be called")
	}
	if got := rec.Header().Get("Access-Control-Allow-Origin"); got != "https://example.com" {
		t.Errorf("expected Access-Control-Allow-Origin %q, got %q", "https://example.com", got)
	}
	if got := rec.Header().Get("Access-Control-Allow-Credentials"); got != "true" {
		t.Errorf("expected Access-Control-Allow-Credentials %q, got %q", "true", got)
	}
}

func TestCORS_DisallowedOrigin(t *testing.T) {
	cfg := config.CORSConfig{
		AllowedOrigins:   "https://example.com",
		AllowedMethods:   "GET,POST",
		AllowedHeaders:   "Authorization",
		AllowCredentials: true,
		MaxAge:           3600,
	}

	called := false
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
		w.WriteHeader(http.StatusOK)
	})

	wrapped := CORS(cfg)(handler)

	req := httptest.NewRequest(http.MethodGet, "/api", nil)
	req.Header.Set("Origin", "https://evil.com")
	rec := httptest.NewRecorder()

	wrapped.ServeHTTP(rec, req)

	if !called {
		t.Error("expected handler to be called")
	}
	if got := rec.Header().Get("Access-Control-Allow-Origin"); got != "" {
		t.Errorf("expected no Access-Control-Allow-Origin header, got %q", got)
	}
}

func TestCORS_Wildcard(t *testing.T) {
	cfg := config.CORSConfig{
		AllowedOrigins:   "*",
		AllowedMethods:   "GET,POST",
		AllowedHeaders:   "Authorization",
		AllowCredentials: false,
		MaxAge:           3600,
	}

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	wrapped := CORS(cfg)(handler)

	req := httptest.NewRequest(http.MethodGet, "/api", nil)
	req.Header.Set("Origin", "https://any-origin.com")
	rec := httptest.NewRecorder()

	wrapped.ServeHTTP(rec, req)

	if got := rec.Header().Get("Access-Control-Allow-Origin"); got != "https://any-origin.com" {
		t.Errorf("expected Access-Control-Allow-Origin %q, got %q", "https://any-origin.com", got)
	}
	// AllowCredentials is false, so header should not be set.
	if got := rec.Header().Get("Access-Control-Allow-Credentials"); got != "" {
		t.Errorf("expected no Access-Control-Allow-Credentials header, got %q", got)
	}
}
