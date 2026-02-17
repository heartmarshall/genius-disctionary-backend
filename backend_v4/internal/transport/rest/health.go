package rest

import (
	"context"
	"encoding/json"
	"net/http"
	"time"
)

// dbPinger defines the minimal interface for DB health checks.
type dbPinger interface {
	Ping(ctx context.Context) error
}

// HealthHandler serves health check endpoints.
type HealthHandler struct {
	db      dbPinger
	version string
}

// NewHealthHandler creates a HealthHandler.
func NewHealthHandler(db dbPinger, version string) *HealthHandler {
	return &HealthHandler{db: db, version: version}
}

// HealthResponse is the JSON response for /health and /ready.
type HealthResponse struct {
	Status     string                `json:"status"`
	Version    string                `json:"version,omitempty"`
	Components map[string]CompStatus `json:"components,omitempty"`
	Timestamp  time.Time             `json:"timestamp"`
}

// CompStatus is the status of an individual component.
type CompStatus struct {
	Status  string `json:"status"`
	Latency string `json:"latency,omitempty"`
}

// Live is the liveness probe. Always returns 200.
func (h *HealthHandler) Live(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, HealthResponse{
		Status:    "ok",
		Timestamp: time.Now(),
	})
}

// Ready is the readiness probe. Pings DB: 200 if OK, 503 if not.
func (h *HealthHandler) Ready(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 3*time.Second)
	defer cancel()

	if err := h.db.Ping(ctx); err != nil {
		writeJSON(w, http.StatusServiceUnavailable, HealthResponse{
			Status:    "down",
			Timestamp: time.Now(),
		})
		return
	}

	writeJSON(w, http.StatusOK, HealthResponse{
		Status:    "ok",
		Timestamp: time.Now(),
	})
}

// Health is the full health check. Pings DB with latency measurement and includes version.
func (h *HealthHandler) Health(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 3*time.Second)
	defer cancel()

	components := make(map[string]CompStatus)
	overallStatus := "ok"

	start := time.Now()
	err := h.db.Ping(ctx)
	latency := time.Since(start)

	if err != nil {
		components["database"] = CompStatus{Status: "down"}
		overallStatus = "down"
	} else {
		components["database"] = CompStatus{
			Status:  "ok",
			Latency: latency.String(),
		}
	}

	status := http.StatusOK
	if overallStatus != "ok" {
		status = http.StatusServiceUnavailable
	}

	writeJSON(w, status, HealthResponse{
		Status:     overallStatus,
		Version:    h.version,
		Components: components,
		Timestamp:  time.Now(),
	})
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(v) //nolint:errcheck
}
