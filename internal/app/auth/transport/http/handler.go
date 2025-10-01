package http

import (
	"context"
	"net/http"

	"github.com/66gu1/easygodocs/internal/app/auth"
	"github.com/66gu1/easygodocs/internal/app/auth/usecase"
	user_http "github.com/66gu1/easygodocs/internal/app/user/transport/http"
	"github.com/66gu1/easygodocs/internal/infrastructure/apperr"
	"github.com/66gu1/easygodocs/internal/infrastructure/httpx"
	"github.com/66gu1/easygodocs/internal/infrastructure/logger"
	"github.com/66gu1/easygodocs/internal/infrastructure/secure"
	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
)

const (
	URLParamSessionID = "session_id"
)

type AuthService interface {
	GetSessionsByUserID(ctx context.Context, userID uuid.UUID) ([]auth.Session, error)
	DeleteSession(ctx context.Context, userID, id uuid.UUID) error
	DeleteSessionsByUserID(ctx context.Context, userID uuid.UUID) error
	AddUserRole(ctx context.Context, role auth.UserRole) error
	DeleteUserRole(ctx context.Context, role auth.UserRole) error
	ListUserRoles(ctx context.Context, userID uuid.UUID) ([]auth.UserRole, error)
	RefreshTokens(ctx context.Context, refreshToken auth.RefreshToken) (auth.Tokens, error)
	Login(ctx context.Context, req usecase.LoginCmd) (auth.Tokens, error)
}

type LoginInput struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

// Handler knows how to decode HTTP → service calls and encode responses.
type Handler struct {
	svc AuthService
}

func NewHandler(svc AuthService) *Handler {
	if svc == nil {
		panic("nil AuthService")
	}
	return &Handler{svc: svc}
}

// GetSessionsByUserID godoc
// @Summary      List sessions by user ID
// @Description  Returns all active sessions for the specified user. Requires admin privileges or self-access.
// @Tags         sessions
// @Security     BearerAuth
// @Produce      json
// @Param        user_id query string true "User ID"
// @Success      200 {array} auth.Session
// @Failure      default {object} apperr.appError "Error"
// @Router       /sessions [get]
func (h *Handler) GetSessionsByUserID(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	idStr := r.URL.Query().Get(user_http.URLParamUserID)
	id, err := uuid.Parse(idStr)
	if err != nil {
		logger.Warn(ctx, err).
			Str(auth.FieldUserID.String(), idStr).
			Msg("auth.Handler.GetSessionsByUserID: invalid user ID format")
		httpx.ReturnError(ctx, w, apperr.ErrBadRequest())
		return
	}

	sessions, err := h.svc.GetSessionsByUserID(ctx, id)
	if err != nil {
		httpx.ReturnError(ctx, w, err)
		return
	}

	httpx.WriteJSON(ctx, w, http.StatusOK, sessions)
}

// DeleteSession godoc
// @Summary      Delete session by ID
// @Description  Deletes a specific session for a given user. Requires admin privileges or self-access.
// @Tags         sessions
// @Security     BearerAuth
// @Param        session_id path string true "Session ID"
// @Param        user_id    query string true "User ID"
// @Success      204 "No Content"
// @Failure      default {object} apperr.appError "Error"
// @Router       /sessions/{session_id} [delete]
func (h *Handler) DeleteSession(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	idStr := chi.URLParam(r, URLParamSessionID)
	id, err := uuid.Parse(idStr)
	if err != nil {
		logger.Warn(ctx, err).
			Str(auth.FieldSessionID.String(), idStr).
			Msg("auth.Handler.DeleteSession: invalid session ID format")
		httpx.ReturnError(ctx, w, apperr.ErrBadRequest())
		return
	}

	userIDStr := r.URL.Query().Get(user_http.URLParamUserID)
	userID, err := uuid.Parse(userIDStr)
	if err != nil {
		logger.Warn(ctx, err).
			Str(auth.FieldSessionID.String(), idStr).
			Str(auth.FieldUserID.String(), userIDStr).
			Msg("auth.Handler.DeleteSession: invalid user ID format")
		httpx.ReturnError(ctx, w, apperr.ErrBadRequest())
		return
	}

	if err = h.svc.DeleteSession(ctx, userID, id); err != nil {
		httpx.ReturnError(ctx, w, err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// DeleteSessionsByUserID godoc
// @Summary      Delete all sessions for user
// @Description  Deletes all active sessions for the specified user. Requires admin privileges or self-access.
// @Tags         sessions
// @Security     BearerAuth
// @Param        user_id query string true "User ID"
// @Success      204 "No Content"
// @Failure      default {object} apperr.appError "Error"
// @Router       /sessions [delete]
func (h *Handler) DeleteSessionsByUserID(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	idStr := r.URL.Query().Get(user_http.URLParamUserID)
	id, err := uuid.Parse(idStr)
	if err != nil {
		logger.Warn(ctx, err).
			Str(auth.FieldUserID.String(), idStr).
			Msg("auth.Handler.DeleteSessionsByUserID: invalid user ID format")
		httpx.ReturnError(ctx, w, apperr.ErrBadRequest())
		return
	}

	if err = h.svc.DeleteSessionsByUserID(ctx, id); err != nil {
		httpx.ReturnError(ctx, w, err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// AddUserRole godoc
// @Summary      Assign role to user
// @Description  Adds a role for a user in relation to an entity. Requires admin privileges.
// @Tags         roles
// @Security     BearerAuth
// @Accept       json
// @Param        request body auth.UserRole true "User role payload"
// @Success      204 "No Content"
// @Failure      default {object} apperr.appError "Error"
// @Router       /roles [post]
func (h *Handler) AddUserRole(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	var input auth.UserRole
	if err := httpx.DecodeJSON(r, &input); err != nil {
		logger.Error(ctx, err).Msg("auth.Handler.AddUserRole: request json decode failed")
		httpx.ReturnError(ctx, w, err)
		return
	}

	if err := h.svc.AddUserRole(ctx, input); err != nil {
		httpx.ReturnError(ctx, w, err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// DeleteUserRole godoc
// @Summary      Remove role from user
// @Description  Deletes a user role assignment for an entity. Requires admin privileges.
// @Tags         roles
// @Security     BearerAuth
// @Accept       json
// @Param        request body auth.UserRole true "User role payload"
// @Success      204 "No Content"
// @Failure      default {object} apperr.appError "Error"
// @Router       /roles [delete]
func (h *Handler) DeleteUserRole(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	var input auth.UserRole
	if err := httpx.DecodeJSON(r, &input); err != nil {
		logger.Error(ctx, err).
			Msg("auth.Handler.DeleteUserRole: request json decode failed")
		httpx.ReturnError(ctx, w, err)
		return
	}

	if err := h.svc.DeleteUserRole(ctx, input); err != nil {
		httpx.ReturnError(ctx, w, err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// ListUserRoles godoc
// @Summary      List roles assigned to a user
// @Description  Returns a list of roles for the specified user ID. Requires admin privileges or self-access.
// @Tags         roles
// @Security     BearerAuth
// @Produce      json
// @Param        user_id query string true "User ID"
// @Success      200 {array} auth.UserRole
// @Failure      default {object} apperr.appError "Error"
// @Router       /roles [get]
func (h *Handler) ListUserRoles(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	idStr := r.URL.Query().Get(user_http.URLParamUserID)
	id, err := uuid.Parse(idStr)
	if err != nil {
		logger.Warn(ctx, err).
			Str(auth.FieldUserID.String(), idStr).
			Msg("auth.Handler.ListUserRoles: invalid user ID format")
		httpx.ReturnError(ctx, w, apperr.ErrBadRequest())
		return
	}

	roles, err := h.svc.ListUserRoles(ctx, id)
	if err != nil {
		httpx.ReturnError(ctx, w, err)
		return
	}

	httpx.WriteJSON(ctx, w, http.StatusOK, roles)
}

// RefreshTokens godoc
// @Summary      Refresh access token
// @Description  Refreshes the access and refresh tokens using a valid refresh token
// @Tags         auth
// @Accept       json
// @Produce      json
// @Param        request body auth.RefreshToken true "Refresh token payload"
// @Success      200 {object} auth.Tokens
// @Failure      default {object} apperr.appError "Error"
// @Router       /refresh [post]
func (h *Handler) RefreshTokens(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	var input auth.RefreshToken
	if err := httpx.DecodeJSON(r, &input); err != nil {
		logger.Error(ctx, err).Msg("auth.Handler.RefreshTokens: request json decode failed")
		httpx.ReturnError(ctx, w, err)
		return
	}

	resp, err := h.svc.RefreshTokens(ctx, input)
	if err != nil {
		httpx.ReturnError(ctx, w, err)
		return
	}

	httpx.WriteJSON(ctx, w, http.StatusOK, resp)
}

// Login godoc
// @Summary      Login
// @Description  Authenticate user and get tokens
// @Tags         auth
// @Accept       json
// @Produce      json
// @Param        request body LoginInput true "credentials"
// @Success      200 {object} auth.Tokens
// @Failure      400 {object} apperr.appError
// @Router       /login [post]
func (h *Handler) Login(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	var input LoginInput
	if err := httpx.DecodeJSON(r, &input); err != nil {
		logger.Error(ctx, err).Msg("auth.Handler.Login: request json decode failed")
		httpx.ReturnError(ctx, w, err)
		return
	}
	cmd := usecase.LoginCmd{
		Email:    input.Email,
		Password: []byte(input.Password),
	}
	defer secure.ZeroBytes(cmd.Password)
	input.Password = ""

	resp, err := h.svc.Login(ctx, cmd)
	if err != nil {
		httpx.ReturnError(ctx, w, err)
		return
	}

	httpx.WriteJSON(ctx, w, http.StatusOK, resp)
}
