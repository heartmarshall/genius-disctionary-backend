package middleware

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/google/uuid"
	"github.com/heartmarshall/myenglish-backend/pkg/ctxutil"
)

//go:generate moq -out token_validator_mock_test.go -pkg middleware . tokenValidator

func TestAuth_ValidToken(t *testing.T) {
	userID := uuid.New()
	validator := &tokenValidatorMock{
		ValidateTokenFunc: func(ctx context.Context, token string) (uuid.UUID, string, error) {
			if token == "valid-token" {
				return userID, "user", nil
			}
			return uuid.Nil, "", errors.New("invalid token")
		},
	}

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotUserID, ok := ctxutil.UserIDFromCtx(r.Context())
		if !ok {
			t.Error("expected userID in context")
			return
		}
		if gotUserID != userID {
			t.Errorf("expected userID %v, got %v", userID, gotUserID)
		}
		w.WriteHeader(http.StatusOK)
	})

	middleware := Auth(validator)
	wrappedHandler := middleware(handler)

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Authorization", "Bearer valid-token")
	rec := httptest.NewRecorder()

	wrappedHandler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d", http.StatusOK, rec.Code)
	}
}

func TestAuth_InvalidToken(t *testing.T) {
	validator := &tokenValidatorMock{
		ValidateTokenFunc: func(ctx context.Context, token string) (uuid.UUID, string, error) {
			return uuid.Nil, "", errors.New("invalid token")
		},
	}

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Error("handler should not be called for invalid token")
	})

	middleware := Auth(validator)
	wrappedHandler := middleware(handler)

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Authorization", "Bearer invalid-token")
	rec := httptest.NewRecorder()

	wrappedHandler.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Errorf("expected status %d, got %d", http.StatusUnauthorized, rec.Code)
	}
}

func TestAuth_NoAuthHeader(t *testing.T) {
	validator := &tokenValidatorMock{
		ValidateTokenFunc: func(ctx context.Context, token string) (uuid.UUID, string, error) {
			t.Error("ValidateToken should not be called when no header present")
			return uuid.Nil, "", errors.New("should not be called")
		},
	}

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, ok := ctxutil.UserIDFromCtx(r.Context())
		if ok {
			t.Error("expected no userID in context for anonymous request")
		}
		w.WriteHeader(http.StatusOK)
	})

	middleware := Auth(validator)
	wrappedHandler := middleware(handler)

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()

	wrappedHandler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d", http.StatusOK, rec.Code)
	}

	if len(validator.ValidateTokenCalls()) > 0 {
		t.Error("ValidateToken should not be called for anonymous request")
	}
}

func TestAuth_NonBearerToken(t *testing.T) {
	validator := &tokenValidatorMock{
		ValidateTokenFunc: func(ctx context.Context, token string) (uuid.UUID, string, error) {
			t.Error("ValidateToken should not be called for non-Bearer token")
			return uuid.Nil, "", errors.New("should not be called")
		},
	}

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, ok := ctxutil.UserIDFromCtx(r.Context())
		if ok {
			t.Error("expected no userID in context for non-Bearer auth")
		}
		w.WriteHeader(http.StatusOK)
	})

	middleware := Auth(validator)
	wrappedHandler := middleware(handler)

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Authorization", "Basic dXNlcjpwYXNz")
	rec := httptest.NewRecorder()

	wrappedHandler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d", http.StatusOK, rec.Code)
	}

	if len(validator.ValidateTokenCalls()) > 0 {
		t.Error("ValidateToken should not be called for non-Bearer token")
	}
}

func TestAuth_EmptyBearerToken(t *testing.T) {
	validator := &tokenValidatorMock{
		ValidateTokenFunc: func(ctx context.Context, token string) (uuid.UUID, string, error) {
			t.Error("ValidateToken should not be called for empty Bearer token")
			return uuid.Nil, "", errors.New("should not be called")
		},
	}

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, ok := ctxutil.UserIDFromCtx(r.Context())
		if ok {
			t.Error("expected no userID in context for empty Bearer token")
		}
		w.WriteHeader(http.StatusOK)
	})

	middleware := Auth(validator)
	wrappedHandler := middleware(handler)

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Authorization", "Bearer ")
	rec := httptest.NewRecorder()

	wrappedHandler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d", http.StatusOK, rec.Code)
	}

	if len(validator.ValidateTokenCalls()) > 0 {
		t.Error("ValidateToken should not be called for empty Bearer token")
	}
}

func TestExtractBearerToken_Cases(t *testing.T) {
	cases := []struct {
		name   string
		header string
		want   string
	}{
		{"empty header", "", ""},
		{"bearer with token", "Bearer valid-token", "valid-token"},
		{"bearer lowercase", "bearer valid-token", "valid-token"},
		{"bearer mixed case", "BEARER valid-token", "valid-token"},
		{"basic auth", "Basic dXNlcjpwYXNz", ""},
		{"bearer no space", "Bearertoken", ""},
		{"bearer empty token", "Bearer ", ""},
		{"just bearer", "Bearer", ""},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/", nil)
			if tc.header != "" {
				req.Header.Set("Authorization", tc.header)
			}
			got := extractBearerToken(req)
			if got != tc.want {
				t.Errorf("extractBearerToken(%q) = %q, want %q", tc.header, got, tc.want)
			}
		})
	}
}
