package rest

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
)

type dbPingerMock struct {
	err error
}

func (m *dbPingerMock) Ping(_ context.Context) error {
	return m.err
}

func TestLive_Always200(t *testing.T) {
	t.Parallel()

	h := NewHealthHandler(&dbPingerMock{}, "test-version")

	req := httptest.NewRequest(http.MethodGet, "/live", nil)
	rec := httptest.NewRecorder()

	h.Live(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", rec.Code)
	}

	var resp HealthResponse
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if resp.Status != "ok" {
		t.Errorf("expected status 'ok', got %q", resp.Status)
	}

	if resp.Timestamp.IsZero() {
		t.Error("expected non-zero timestamp")
	}
}

func TestReady_DBUp(t *testing.T) {
	t.Parallel()

	h := NewHealthHandler(&dbPingerMock{err: nil}, "test-version")

	req := httptest.NewRequest(http.MethodGet, "/ready", nil)
	rec := httptest.NewRecorder()

	h.Ready(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", rec.Code)
	}

	var resp HealthResponse
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if resp.Status != "ok" {
		t.Errorf("expected status 'ok', got %q", resp.Status)
	}
}

func TestReady_DBDown(t *testing.T) {
	t.Parallel()

	h := NewHealthHandler(&dbPingerMock{err: errors.New("connection refused")}, "test-version")

	req := httptest.NewRequest(http.MethodGet, "/ready", nil)
	rec := httptest.NewRecorder()

	h.Ready(rec, req)

	if rec.Code != http.StatusServiceUnavailable {
		t.Fatalf("expected status 503, got %d", rec.Code)
	}

	var resp HealthResponse
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if resp.Status != "down" {
		t.Errorf("expected status 'down', got %q", resp.Status)
	}
}

func TestHealth_AllOK(t *testing.T) {
	t.Parallel()

	h := NewHealthHandler(&dbPingerMock{err: nil}, "v1.0.0")

	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	rec := httptest.NewRecorder()

	h.Health(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", rec.Code)
	}

	var resp HealthResponse
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if resp.Status != "ok" {
		t.Errorf("expected status 'ok', got %q", resp.Status)
	}

	if resp.Version != "v1.0.0" {
		t.Errorf("expected version 'v1.0.0', got %q", resp.Version)
	}

	dbComp, ok := resp.Components["database"]
	if !ok {
		t.Fatal("expected 'database' component in response")
	}

	if dbComp.Status != "ok" {
		t.Errorf("expected database status 'ok', got %q", dbComp.Status)
	}
}

func TestHealth_DBDown(t *testing.T) {
	t.Parallel()

	h := NewHealthHandler(&dbPingerMock{err: errors.New("connection refused")}, "v1.0.0")

	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	rec := httptest.NewRecorder()

	h.Health(rec, req)

	if rec.Code != http.StatusServiceUnavailable {
		t.Fatalf("expected status 503, got %d", rec.Code)
	}

	var resp HealthResponse
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if resp.Status != "down" {
		t.Errorf("expected status 'down', got %q", resp.Status)
	}

	dbComp, ok := resp.Components["database"]
	if !ok {
		t.Fatal("expected 'database' component in response")
	}

	if dbComp.Status != "down" {
		t.Errorf("expected database status 'down', got %q", dbComp.Status)
	}
}

func TestHealth_IncludesLatency(t *testing.T) {
	t.Parallel()

	h := NewHealthHandler(&dbPingerMock{err: nil}, "v1.0.0")

	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	rec := httptest.NewRecorder()

	h.Health(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", rec.Code)
	}

	var resp HealthResponse
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	dbComp, ok := resp.Components["database"]
	if !ok {
		t.Fatal("expected 'database' component in response")
	}

	if dbComp.Latency == "" {
		t.Error("expected non-empty latency for database component")
	}
}
