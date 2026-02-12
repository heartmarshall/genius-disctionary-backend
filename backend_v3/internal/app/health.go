package app

import (
	"context"
	"encoding/json"
	"net/http"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

// HealthStatus — статус компонента.
type HealthStatus string

const (
	HealthStatusUp   HealthStatus = "up"
	HealthStatusDown HealthStatus = "down"
)

// HealthResponse — ответ health check endpoint.
type HealthResponse struct {
	Status     HealthStatus               `json:"status"`
	Components map[string]ComponentHealth `json:"components"`
	Timestamp  time.Time                  `json:"timestamp"`
}

// ComponentHealth — здоровье отдельного компонента.
type ComponentHealth struct {
	Status  HealthStatus `json:"status"`
	Latency string       `json:"latency,omitempty"`
	Error   string       `json:"error,omitempty"`
}

// HealthChecker проверяет здоровье приложения.
type HealthChecker struct {
	pool *pgxpool.Pool
}

// NewHealthChecker создаёт новый HealthChecker.
func NewHealthChecker(pool *pgxpool.Pool) *HealthChecker {
	return &HealthChecker{pool: pool}
}

// Handler возвращает HTTP handler для health check.
func (h *HealthChecker) Handler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
		defer cancel()

		response := h.Check(ctx)

		w.Header().Set("Content-Type", "application/json")

		if response.Status == HealthStatusDown {
			w.WriteHeader(http.StatusServiceUnavailable)
		} else {
			w.WriteHeader(http.StatusOK)
		}

		json.NewEncoder(w).Encode(response)
	}
}

// LivenessHandler — простой liveness probe (для Kubernetes).
func (h *HealthChecker) LivenessHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	}
}

// ReadinessHandler — readiness probe с проверкой БД.
func (h *HealthChecker) ReadinessHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx, cancel := context.WithTimeout(r.Context(), 3*time.Second)
		defer cancel()

		if err := h.pool.Ping(ctx); err != nil {
			w.WriteHeader(http.StatusServiceUnavailable)
			w.Write([]byte("NOT READY"))
			return
		}

		w.WriteHeader(http.StatusOK)
		w.Write([]byte("READY"))
	}
}

// Check выполняет полную проверку здоровья.
func (h *HealthChecker) Check(ctx context.Context) HealthResponse {
	components := make(map[string]ComponentHealth)

	// Проверка БД
	dbHealth := h.checkDatabase(ctx)
	components["database"] = dbHealth

	// Определяем общий статус
	overallStatus := HealthStatusUp
	for _, comp := range components {
		if comp.Status == HealthStatusDown {
			overallStatus = HealthStatusDown
			break
		}
	}

	return HealthResponse{
		Status:     overallStatus,
		Components: components,
		Timestamp:  time.Now().UTC(),
	}
}

// checkDatabase проверяет соединение с БД.
func (h *HealthChecker) checkDatabase(ctx context.Context) ComponentHealth {
	start := time.Now()

	err := h.pool.Ping(ctx)
	latency := time.Since(start)

	if err != nil {
		return ComponentHealth{
			Status:  HealthStatusDown,
			Latency: latency.String(),
			Error:   err.Error(),
		}
	}

	return ComponentHealth{
		Status:  HealthStatusUp,
		Latency: latency.String(),
	}
}
