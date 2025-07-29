package user

import (
	"context"
	"encoding/json"
	"github.com/66gu1/easygodocs/internal/app/user/dto"
	"github.com/66gu1/easygodocs/internal/infrastructure/apperror"
	"github.com/66gu1/easygodocs/internal/infrastructure/httputil"
	"github.com/66gu1/easygodocs/internal/infrastructure/logger"
	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"net/http"
)

// Handler knows how to decode HTTP → service calls and encode responses.
type Handler struct {
	svc Service
	cfg ValidationConfig
}

type ValidationConfig struct {
	MinPasswordLength int
	MaxPasswordLength int
	MaxEmailLength    int
	MaxNameLength     int
}

//go:generate minimock -i github.com/66gu1/easygodocs/internal/app/department.Service -o ./mock -s _mock.go
type Service interface {
	CreateUser(ctx context.Context, req dto.CreateUserReq) error
	GetUser(ctx context.Context, id uuid.UUID) (dto.User, error)
	GetAllUsers(ctx context.Context) ([]dto.User, error)
	UpdateUser(ctx context.Context, req dto.UpdateUserReq) error
	DeleteUser(ctx context.Context, id uuid.UUID) error
	GetSessionsByUserID(ctx context.Context, userID uuid.UUID) ([]dto.Session, error)
	DeleteSession(ctx context.Context, id uuid.UUID) error
	DeleteSessionsByUserID(ctx context.Context, userID uuid.UUID) error
	AddUserRole(ctx context.Context, role dto.UserRole) error
	DeleteUserRole(ctx context.Context, role dto.UserRole) error
	GetUserRoles(ctx context.Context, userID uuid.UUID) ([]dto.UserRole, error)
	RefreshTokens(ctx context.Context, refreshToken string) (string, string, error)
	Login(ctx context.Context, req dto.LoginReq) (string, string, error)
}

func NewHandler(svc Service, cfg ValidationConfig) *Handler {
	return &Handler{svc: svc, cfg: cfg}
}

func (h *Handler) CreateUser(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	var req dto.CreateUserReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		logger.Error(ctx, err).Msg("user.Handler.CreateUser: request json decode failed")
		httputil.ReturnError(ctx, w, apperror.ErrBadRequest)
		return
	}
	if err := req.Validate(h.cfg.MaxNameLength, h.cfg.MaxEmailLength, h.cfg.MaxPasswordLength, h.cfg.MinPasswordLength); err != nil {
		logger.Error(ctx, err).Msg("user.Handler.CreateUser: validation failed")
		httputil.ReturnError(ctx, w, err)
		return
	}

	if err := h.svc.CreateUser(ctx, req); err != nil {
		httputil.ReturnError(ctx, w, err)
		return
	}

	w.WriteHeader(http.StatusCreated)
}

func (h *Handler) GetUser(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	idStr := chi.URLParam(r, "id")
	id, err := uuid.Parse(idStr)
	if err != nil || id == uuid.Nil {
		logger.Warn(ctx, err).Str("id", idStr).Msg("user.Handler.GetUser: invalid user ID format")
		httputil.ReturnError(ctx, w, apperror.ErrBadRequest)
		return
	}

	user, err := h.svc.GetUser(ctx, id)
	if err != nil {
		httputil.ReturnError(ctx, w, err)
		return
	}

	httputil.WriteJSON(ctx, w, http.StatusOK, user)
}

func (h *Handler) GetAllUsers(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	users, err := h.svc.GetAllUsers(ctx)
	if err != nil {
		httputil.ReturnError(ctx, w, err)
		return
	}

	httputil.WriteJSON(ctx, w, http.StatusOK, users)
}

func (h *Handler) UpdateUser(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	var req dto.UpdateUserReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		logger.Error(ctx, err).Msg("user.Handler.UpdateUser: request json decode failed")
		httputil.ReturnError(ctx, w, apperror.ErrBadRequest)
		return
	}
	if err := req.Validate(h.cfg.MaxNameLength, h.cfg.MaxEmailLength); err != nil {
		logger.Error(ctx, err).Msg("user.Handler.UpdateUser: validation failed")
		httputil.ReturnError(ctx, w, err)
		return
	}

	if err := h.svc.UpdateUser(ctx, req); err != nil {
		httputil.ReturnError(ctx, w, err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) DeleteUser(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	idStr := chi.URLParam(r, "id")
	id, err := uuid.Parse(idStr)
	if err != nil || id == uuid.Nil {
		logger.Warn(ctx, err).Str("id", idStr).Msg("user.Handler.DeleteUser: invalid user ID format")
		httputil.ReturnError(ctx, w, apperror.ErrBadRequest)
		return
	}

	if err := h.svc.DeleteUser(ctx, id); err != nil {
		httputil.ReturnError(ctx, w, err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) GetSessionsByUserID(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	idStr := chi.URLParam(r, "id")
	id, err := uuid.Parse(idStr)
	if err != nil || id == uuid.Nil {
		logger.Warn(ctx, err).Str("id", idStr).Msg("user.Handler.GetSessionsByUserID: invalid user ID format")
		httputil.ReturnError(ctx, w, apperror.ErrBadRequest)
		return
	}

	sessions, err := h.svc.GetSessionsByUserID(ctx, id)
	if err != nil {
		httputil.ReturnError(ctx, w, err)
		return
	}

	httputil.WriteJSON(ctx, w, http.StatusOK, sessions)
}

func (h *Handler) DeleteSession(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	idStr := chi.URLParam(r, "id")
	id, err := uuid.Parse(idStr)
	if err != nil || id == uuid.Nil {
		logger.Warn(ctx, err).Str("id", idStr).Msg("user.Handler.DeleteSession: invalid session ID format")
		httputil.ReturnError(ctx, w, apperror.ErrBadRequest)
		return
	}

	if err := h.svc.DeleteSession(ctx, id); err != nil {
		httputil.ReturnError(ctx, w, err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) DeleteSessionsByUserID(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	idStr := chi.URLParam(r, "id")
	id, err := uuid.Parse(idStr)
	if err != nil || id == uuid.Nil {
		logger.Warn(ctx, err).Str("id", idStr).Msg("user.Handler.DeleteSessionsByUserID: invalid user ID format")
		httputil.ReturnError(ctx, w, apperror.ErrBadRequest)
		return
	}

	if err := h.svc.DeleteSessionsByUserID(ctx, id); err != nil {
		httputil.ReturnError(ctx, w, err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) AddUserRole(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	var role dto.UserRole
	if err := json.NewDecoder(r.Body).Decode(&role); err != nil {
		logger.Error(ctx, err).Msg("user.Handler.AddUserRole: request json decode failed")
		httputil.ReturnError(ctx, w, apperror.ErrBadRequest)
		return
	}

	if err := h.svc.AddUserRole(ctx, role); err != nil {
		httputil.ReturnError(ctx, w, err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) DeleteUserRole(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	var role dto.UserRole
	if err := json.NewDecoder(r.Body).Decode(&role); err != nil {
		logger.Error(ctx, err).Msg("user.Handler.DeleteUserRole: request json decode failed")
		httputil.ReturnError(ctx, w, apperror.ErrBadRequest)
		return
	}

	if err := h.svc.DeleteUserRole(ctx, role); err != nil {
		httputil.ReturnError(ctx, w, err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) GetUserRoles(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	idStr := chi.URLParam(r, "id")
	id, err := uuid.Parse(idStr)
	if err != nil || id == uuid.Nil {
		logger.Warn(ctx, err).Str("id", idStr).Msg("user.Handler.GetUserRoles: invalid user ID format")
		httputil.ReturnError(ctx, w, apperror.ErrBadRequest)
		return
	}

	roles, err := h.svc.GetUserRoles(ctx, id)
	if err != nil {
		httputil.ReturnError(ctx, w, err)
		return
	}

	httputil.WriteJSON(ctx, w, http.StatusOK, roles)
}

func (h *Handler) RefreshTokens(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	type refreshTokenRequest struct {
		RefreshToken string `json:"refresh_token"`
	}
	var req refreshTokenRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		logger.Error(ctx, err).Msg("user.Handler.RefreshTokens: request json decode failed")
		httputil.ReturnError(ctx, w, apperror.ErrBadRequest)
		return
	}

	if req.RefreshToken == "" {
		logger.Error(ctx, nil).Msg("user.Handler.RefreshTokens: empty refresh token")
		httputil.ReturnError(ctx, w, apperror.ErrBadRequest)
		return
	}

	newAccessToken, newRefreshToken, err := h.svc.RefreshTokens(ctx, req.RefreshToken)
	if err != nil {
		httputil.ReturnError(ctx, w, err)
		return
	}

	type refreshTokenResponse struct {
		AccessToken  string `json:"access_token"`
		RefreshToken string `json:"refresh_token"`
	}
	response := refreshTokenResponse{
		AccessToken:  newAccessToken,
		RefreshToken: newRefreshToken,
	}
	httputil.WriteJSON(ctx, w, http.StatusOK, response)
}

func (h *Handler) Login(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	var req dto.LoginReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		logger.Error(ctx, err).Msg("user.Handler.Login: request json decode failed")
		httputil.ReturnError(ctx, w, apperror.ErrBadRequest)
		return
	}
	if err := req.Validate(); err != nil {
		logger.Error(ctx, err).Msg("user.Handler.Login: validation failed")
		httputil.ReturnError(ctx, w, err)
		return
	}
	req.UserAgent = r.UserAgent()

	newAccessToken, newRefreshToken, err := h.svc.Login(ctx, req)
	if err != nil {
		httputil.ReturnError(ctx, w, err)
		return
	}

	type loginResponse struct {
		AccessToken  string `json:"access_token"`
		RefreshToken string `json:"refresh_token"`
	}
	response := loginResponse{
		AccessToken:  newAccessToken,
		RefreshToken: newRefreshToken,
	}
	httputil.WriteJSON(ctx, w, http.StatusOK, response)

}
