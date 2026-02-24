package rest

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"strings"

	"github.com/google/uuid"
	"github.com/heartmarshall/myenglish-backend/internal/domain"
	"github.com/heartmarshall/myenglish-backend/internal/service/auth"
	"github.com/heartmarshall/myenglish-backend/pkg/ctxutil"
)

// authService defines the minimal interface needed by AuthHandler.
type authService interface {
	Login(ctx context.Context, input auth.LoginInput) (*auth.AuthResult, error)
	LoginWithPassword(ctx context.Context, input auth.LoginPasswordInput) (*auth.AuthResult, error)
	Register(ctx context.Context, input auth.RegisterInput) (*auth.AuthResult, error)
	Refresh(ctx context.Context, input auth.RefreshInput) (*auth.AuthResult, error)
	Logout(ctx context.Context) error
	ValidateToken(ctx context.Context, token string) (uuid.UUID, error)
}

// AuthHandler serves auth REST endpoints.
type AuthHandler struct {
	svc authService
	log *slog.Logger
}

// NewAuthHandler creates an AuthHandler.
func NewAuthHandler(svc authService, logger *slog.Logger) *AuthHandler {
	return &AuthHandler{svc: svc, log: logger.With("handler", "auth")}
}

type loginRequest struct {
	Provider string `json:"provider"`
	Code     string `json:"code"`
}

type loginPasswordRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

type registerRequest struct {
	Email    string `json:"email"`
	Username string `json:"username"`
	Password string `json:"password"`
}

type refreshRequest struct {
	RefreshToken string `json:"refreshToken"`
}

type authResponse struct {
	AccessToken  string       `json:"accessToken"`
	RefreshToken string       `json:"refreshToken"`
	User         userResponse `json:"user"`
}

type userResponse struct {
	ID        string  `json:"id"`
	Email     string  `json:"email"`
	Username  string  `json:"username"`
	Name      string  `json:"name"`
	AvatarURL *string `json:"avatarUrl,omitempty"`
	Role      string  `json:"role"`
}

// Login handles POST /auth/login (OAuth).
func (h *AuthHandler) Login(w http.ResponseWriter, r *http.Request) {
	var req loginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	result, err := h.svc.Login(r.Context(), auth.LoginInput{
		Provider: req.Provider,
		Code:     req.Code,
	})
	if err != nil {
		h.handleError(w, r, err)
		return
	}

	writeJSON(w, http.StatusOK, toAuthResponse(result))
}

// LoginWithPassword handles POST /auth/login/password.
func (h *AuthHandler) LoginWithPassword(w http.ResponseWriter, r *http.Request) {
	var req loginPasswordRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	result, err := h.svc.LoginWithPassword(r.Context(), auth.LoginPasswordInput{
		Email:    req.Email,
		Password: req.Password,
	})
	if err != nil {
		h.handleError(w, r, err)
		return
	}

	writeJSON(w, http.StatusOK, toAuthResponse(result))
}

// Register handles POST /auth/register.
func (h *AuthHandler) Register(w http.ResponseWriter, r *http.Request) {
	var req registerRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	result, err := h.svc.Register(r.Context(), auth.RegisterInput{
		Email:    req.Email,
		Username: req.Username,
		Password: req.Password,
	})
	if err != nil {
		h.handleError(w, r, err)
		return
	}

	writeJSON(w, http.StatusCreated, toAuthResponse(result))
}

// Refresh handles POST /auth/refresh.
func (h *AuthHandler) Refresh(w http.ResponseWriter, r *http.Request) {
	var req refreshRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	result, err := h.svc.Refresh(r.Context(), auth.RefreshInput{
		RefreshToken: req.RefreshToken,
	})
	if err != nil {
		h.handleError(w, r, err)
		return
	}

	writeJSON(w, http.StatusOK, toAuthResponse(result))
}

// Logout handles POST /auth/logout.
func (h *AuthHandler) Logout(w http.ResponseWriter, r *http.Request) {
	token := extractBearer(r)
	if token == "" {
		writeError(w, http.StatusUnauthorized, "missing bearer token")
		return
	}

	userID, err := h.svc.ValidateToken(r.Context(), token)
	if err != nil {
		writeError(w, http.StatusUnauthorized, "invalid token")
		return
	}

	ctx := ctxutil.WithUserID(r.Context(), userID)
	if err := h.svc.Logout(ctx); err != nil {
		h.handleError(w, r, err)
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func (h *AuthHandler) handleError(w http.ResponseWriter, r *http.Request, err error) {
	switch {
	case errors.Is(err, domain.ErrValidation):
		writeError(w, http.StatusBadRequest, err.Error())
	case errors.Is(err, domain.ErrUnauthorized):
		writeError(w, http.StatusUnauthorized, "unauthorized")
	case errors.Is(err, domain.ErrAlreadyExists):
		writeError(w, http.StatusConflict, "already exists")
	default:
		h.log.ErrorContext(r.Context(), "internal error", slog.String("error", err.Error()))
		writeError(w, http.StatusInternalServerError, "internal server error")
	}
}

func writeError(w http.ResponseWriter, status int, message string) {
	writeJSON(w, status, map[string]string{"error": message})
}

func extractBearer(r *http.Request) string {
	h := r.Header.Get("Authorization")
	if h == "" {
		return ""
	}
	parts := strings.SplitN(h, " ", 2)
	if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") {
		return ""
	}
	return parts[1]
}

func toAuthResponse(result *auth.AuthResult) authResponse {
	return authResponse{
		AccessToken:  result.AccessToken,
		RefreshToken: result.RefreshToken,
		User: userResponse{
			ID:        result.User.ID.String(),
			Email:     result.User.Email,
			Username:  result.User.Username,
			Name:      result.User.Name,
			AvatarURL: result.User.AvatarURL,
			Role:      result.User.Role.String(),
		},
	}
}
