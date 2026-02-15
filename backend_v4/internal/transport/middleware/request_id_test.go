package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/google/uuid"
	"github.com/heartmarshall/myenglish-backend/pkg/ctxutil"
)

func TestRequestID_ReuseIncoming(t *testing.T) {
	incomingID := uuid.New().String()

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotID := ctxutil.RequestIDFromCtx(r.Context())
		if gotID == "" {
			t.Error("expected requestID in context")
			return
		}
		if gotID != incomingID {
			t.Errorf("expected requestID %s, got %s", incomingID, gotID)
		}
		w.WriteHeader(http.StatusOK)
	})

	wrappedHandler := RequestID(handler)

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("X-Request-Id", incomingID)
	rec := httptest.NewRecorder()

	wrappedHandler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d", http.StatusOK, rec.Code)
	}

	gotHeader := rec.Header().Get("X-Request-Id")
	if gotHeader != incomingID {
		t.Errorf("expected X-Request-Id header %s, got %s", incomingID, gotHeader)
	}
}

func TestRequestID_GenerateNew(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotID := ctxutil.RequestIDFromCtx(r.Context())
		if gotID == "" {
			t.Error("expected non-empty requestID")
			return
		}
		// Validate it's a valid UUID
		if _, err := uuid.Parse(gotID); err != nil {
			t.Errorf("expected valid UUID, got %s: %v", gotID, err)
		}
		w.WriteHeader(http.StatusOK)
	})

	wrappedHandler := RequestID(handler)

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()

	wrappedHandler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d", http.StatusOK, rec.Code)
	}

	gotHeader := rec.Header().Get("X-Request-Id")
	if gotHeader == "" {
		t.Error("expected X-Request-Id header to be set")
	}

	// Validate it's a valid UUID
	if _, err := uuid.Parse(gotHeader); err != nil {
		t.Errorf("expected valid UUID in header, got %s: %v", gotHeader, err)
	}
}
