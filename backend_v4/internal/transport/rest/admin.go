package rest

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"

	"github.com/google/uuid"
	"github.com/heartmarshall/myenglish-backend/internal/domain"
	"github.com/heartmarshall/myenglish-backend/pkg/ctxutil"
)

type adminEnrichmentService interface {
	GetStats(ctx context.Context) (domain.EnrichmentQueueStats, error)
	List(ctx context.Context, status string, limit, offset int) ([]domain.EnrichmentQueueItem, error)
	Enqueue(ctx context.Context, refEntryID uuid.UUID) error
	RetryAllFailed(ctx context.Context) (int, error)
	ResetProcessing(ctx context.Context) (int, error)
}

type adminUserService interface {
	SetUserRole(ctx context.Context, targetUserID uuid.UUID, role domain.UserRole) (*domain.User, error)
	ListUsers(ctx context.Context, limit, offset int) ([]domain.User, int, error)
}

// AdminHandler serves admin REST endpoints.
type AdminHandler struct {
	enrichment adminEnrichmentService
	users      adminUserService
	log        *slog.Logger
}

// NewAdminHandler creates an AdminHandler.
func NewAdminHandler(enrichment adminEnrichmentService, users adminUserService, logger *slog.Logger) *AdminHandler {
	return &AdminHandler{
		enrichment: enrichment,
		users:      users,
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

// RetryFailed resets all failed enrichment items to pending.
// POST /admin/enrichment/retry
func (h *AdminHandler) RetryFailed(w http.ResponseWriter, r *http.Request) {
	if !h.requireAdmin(w, r) {
		return
	}

	n, err := h.enrichment.RetryAllFailed(r.Context())
	if err != nil {
		h.log.ErrorContext(r.Context(), "retry failed", slog.String("error", err.Error()))
		writeError(w, http.StatusInternalServerError, "internal server error")
		return
	}

	writeJSON(w, http.StatusOK, map[string]int{"retried": n})
}

// ResetProcessing resets stuck processing items to pending.
// POST /admin/enrichment/reset-processing
func (h *AdminHandler) ResetProcessing(w http.ResponseWriter, r *http.Request) {
	if !h.requireAdmin(w, r) {
		return
	}

	n, err := h.enrichment.ResetProcessing(r.Context())
	if err != nil {
		h.log.ErrorContext(r.Context(), "reset processing", slog.String("error", err.Error()))
		writeError(w, http.StatusInternalServerError, "internal server error")
		return
	}

	writeJSON(w, http.StatusOK, map[string]int{"reset": n})
}

// EnqueueWord manually enqueues a ref entry for enrichment.
// POST /admin/enrichment/enqueue  body: {"refEntryId": "uuid"}
func (h *AdminHandler) EnqueueWord(w http.ResponseWriter, r *http.Request) {
	if !h.requireAdmin(w, r) {
		return
	}

	var body struct {
		RefEntryID string `json:"refEntryId"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	id, err := uuid.Parse(body.RefEntryID)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid refEntryId: must be a valid UUID")
		return
	}

	if err := h.enrichment.Enqueue(r.Context(), id); err != nil {
		h.log.ErrorContext(r.Context(), "enqueue word", slog.String("error", err.Error()))
		writeError(w, http.StatusInternalServerError, "internal server error")
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"status": "enqueued", "refEntryId": id.String()})
}

// ListUsers returns paginated list of users.
// GET /admin/users?limit=50&offset=0
func (h *AdminHandler) ListUsers(w http.ResponseWriter, r *http.Request) {
	if !h.requireAdmin(w, r) {
		return
	}

	limit := 50
	offset := 0

	if v := r.URL.Query().Get("limit"); v != "" {
		json.Unmarshal([]byte(v), &limit) //nolint:errcheck
	}
	if v := r.URL.Query().Get("offset"); v != "" {
		json.Unmarshal([]byte(v), &offset) //nolint:errcheck
	}

	users, total, err := h.users.ListUsers(r.Context(), limit, offset)
	if err != nil {
		h.log.ErrorContext(r.Context(), "list users", slog.String("error", err.Error()))
		writeError(w, http.StatusInternalServerError, "internal server error")
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{"users": users, "total": total})
}

// SetUserRole changes a user's role.
// PUT /admin/users/{id}/role  body: {"role": "admin"}
func (h *AdminHandler) SetUserRole(w http.ResponseWriter, r *http.Request) {
	if !h.requireAdmin(w, r) {
		return
	}

	idStr := r.PathValue("id")
	userID, err := uuid.Parse(idStr)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid user id")
		return
	}

	var body struct {
		Role string `json:"role"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	user, err := h.users.SetUserRole(r.Context(), userID, domain.UserRole(body.Role))
	if err != nil {
		switch {
		case errors.Is(err, domain.ErrNotFound):
			writeError(w, http.StatusNotFound, "user not found")
		case errors.Is(err, domain.ErrValidation):
			writeError(w, http.StatusBadRequest, err.Error())
		case errors.Is(err, domain.ErrForbidden):
			writeError(w, http.StatusForbidden, err.Error())
		default:
			h.log.ErrorContext(r.Context(), "set user role", slog.String("error", err.Error()))
			writeError(w, http.StatusInternalServerError, "internal server error")
		}
		return
	}

	writeJSON(w, http.StatusOK, user)
}

func (h *AdminHandler) requireAdmin(w http.ResponseWriter, r *http.Request) bool {
	if !ctxutil.IsAdminCtx(r.Context()) {
		writeError(w, http.StatusForbidden, "admin access required")
		return false
	}
	return true
}
