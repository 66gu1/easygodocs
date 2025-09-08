package http

import (
	"context"
	"net/http"

	"github.com/66gu1/easygodocs/internal/app/auth"
	"github.com/66gu1/easygodocs/internal/app/auth/usecase"
	auth_http "github.com/66gu1/easygodocs/internal/app/user/transport/http"
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

type userRoleInput struct {
	Role     auth.Role  `json:"role"`
	EntityID *uuid.UUID `json:"entity_id"`
}

// Handler knows how to decode HTTP → service calls and encode responses.
type Handler struct {
	svc AuthService
}

//go:generate minimock -i github.com/66gu1/easygodocs/internal/app/auth.AuthService -o ./mock -s _mock.go
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

func NewHandler(svc AuthService) *Handler {
	if svc == nil {
		panic("nil AuthService")
	}
	return &Handler{svc: svc}
}

func (h *Handler) GetSessionsByUserID(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	idStr := chi.URLParam(r, auth_http.URLParamUserID)
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

	userIDStr := chi.URLParam(r, auth_http.URLParamUserID)
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

func (h *Handler) DeleteSessionsByUserID(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	idStr := chi.URLParam(r, auth_http.URLParamUserID)
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

func (h *Handler) AddUserRole(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	idStr := chi.URLParam(r, auth_http.URLParamUserID)
	id, err := uuid.Parse(idStr)
	if err != nil {
		logger.Warn(ctx, err).
			Str(auth.FieldUserID.String(), idStr).
			Msg("auth.Handler.AddUserRole: invalid user ID format")
		httpx.ReturnError(ctx, w, apperr.ErrBadRequest())
		return
	}

	var role userRoleInput
	if err = httpx.DecodeJSON(r, &role); err != nil {
		logger.Error(ctx, err).Msg("auth.Handler.AddUserRole: request json decode failed")
		httpx.ReturnError(ctx, w, err)
		return
	}

	if err = h.svc.AddUserRole(ctx, auth.UserRole{
		UserID:   id,
		Role:     role.Role,
		EntityID: role.EntityID,
	}); err != nil {
		httpx.ReturnError(ctx, w, err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) DeleteUserRole(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	idStr := chi.URLParam(r, auth_http.URLParamUserID)
	id, err := uuid.Parse(idStr)
	if err != nil {
		logger.Warn(ctx, err).
			Str(auth.FieldUserID.String(), idStr).
			Msg("auth.Handler.DeleteUserRole: invalid user ID format")
		httpx.ReturnError(ctx, w, apperr.ErrBadRequest())
		return
	}

	var input userRoleInput
	if err = httpx.DecodeJSON(r, &input); err != nil {
		logger.Error(ctx, err).
			Str(auth.FieldUserID.String(), idStr).
			Msg("auth.Handler.DeleteUserRole: request json decode failed")
		httpx.ReturnError(ctx, w, err)
		return
	}

	if err = h.svc.DeleteUserRole(ctx, auth.UserRole{
		UserID:   id,
		Role:     input.Role,
		EntityID: input.EntityID,
	}); err != nil {
		httpx.ReturnError(ctx, w, err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) ListUserRoles(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	idStr := chi.URLParam(r, auth_http.URLParamUserID)
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

func (h *Handler) Login(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	type loginInput struct {
		Email    string `json:"email"`
		Password string `json:"password"`
	}
	var input loginInput
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
