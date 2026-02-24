package rest

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"

	"github.com/heartmarshall/myenglish-backend/internal/domain"
	"github.com/heartmarshall/myenglish-backend/pkg/ctxutil"
)

type enrichmentService interface {
	GetStats(ctx context.Context) (domain.EnrichmentQueueStats, error)
	List(ctx context.Context, status string, limit, offset int) ([]domain.EnrichmentQueueItem, error)
}

// AdminHandler serves admin REST endpoints.
type AdminHandler struct {
	enrichment enrichmentService
	log        *slog.Logger
}

// NewAdminHandler creates an AdminHandler.
func NewAdminHandler(enrichment enrichmentService, logger *slog.Logger) *AdminHandler {
	return &AdminHandler{
		enrichment: enrichment,
		log:        logger.With("handler", "admin"),
	}
}

// QueueStats returns enrichment queue statistics.
// GET /admin/enrichment/stats
func (h *AdminHandler) QueueStats(w http.ResponseWriter, r *http.Request) {
	if !h.requireAdmin(w, r) {
		return
	}

	stats, err := h.enrichment.GetStats(r.Context())
	if err != nil {
		h.log.ErrorContext(r.Context(), "get queue stats", slog.String("error", err.Error()))
		writeError(w, http.StatusInternalServerError, "internal server error")
		return
	}

	writeJSON(w, http.StatusOK, stats)
}

// QueueList returns enrichment queue items filtered by status.
// GET /admin/enrichment/queue?status=pending&limit=50&offset=0
func (h *AdminHandler) QueueList(w http.ResponseWriter, r *http.Request) {
	if !h.requireAdmin(w, r) {
		return
	}

	status := r.URL.Query().Get("status")
	limit := 50
	offset := 0

	if v := r.URL.Query().Get("limit"); v != "" {
		json.Unmarshal([]byte(v), &limit) //nolint:errcheck
	}
	if v := r.URL.Query().Get("offset"); v != "" {
		json.Unmarshal([]byte(v), &offset) //nolint:errcheck
	}

	items, err := h.enrichment.List(r.Context(), status, limit, offset)
	if err != nil {
		h.log.ErrorContext(r.Context(), "list queue", slog.String("error", err.Error()))
		writeError(w, http.StatusInternalServerError, "internal server error")
		return
	}

	writeJSON(w, http.StatusOK, items)
}

func (h *AdminHandler) requireAdmin(w http.ResponseWriter, r *http.Request) bool {
	if !ctxutil.IsAdminCtx(r.Context()) {
		writeError(w, http.StatusForbidden, "admin access required")
		return false
	}
	return true
}
