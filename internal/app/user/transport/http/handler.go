package http

import (
	"context"
	"net/http"

	"github.com/66gu1/easygodocs/internal/app/user"
	"github.com/66gu1/easygodocs/internal/app/user/usecase"
	"github.com/66gu1/easygodocs/internal/infrastructure/apperr"
	"github.com/66gu1/easygodocs/internal/infrastructure/httpx"
	"github.com/66gu1/easygodocs/internal/infrastructure/logger"
	"github.com/66gu1/easygodocs/internal/infrastructure/secure"
	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
)

const (
	URLParamUserID = "user_id"
)

type CreateUserInput struct {
	Email    string `json:"email"`
	Name     string `json:"name"`
	Password string `json:"password"`
}

type UpdateUserInput struct {
	Email string `json:"email"`
	Name  string `json:"name"`
}

type ChangePasswordInput struct {
	OldPassword string `json:"old_password"`
	NewPassword string `json:"new_password"`
}

// Handler knows how to decode HTTP → service calls and encode responses.
type Handler struct {
	svc Service
}

//go:generate minimock -i github.com/66gu1/easygodocs/internal/app/user/usecase.Service -o ./mock -s _mock.go
type Service interface {
	CreateUser(ctx context.Context, req user.CreateUserReq) error
	GetUser(ctx context.Context, id uuid.UUID) (user.User, error)
	GetAllUsers(ctx context.Context) ([]user.User, error)
	UpdateUser(ctx context.Context, req user.UpdateUserReq) error
	DeleteUser(ctx context.Context, id uuid.UUID) error
	ChangePassword(ctx context.Context, req usecase.ChangePasswordCmd) error
}

func NewHandler(svc Service) *Handler {
	if svc == nil {
		panic("user HTTP handler: nil service")
	}
	return &Handler{svc: svc}
}

// CreateUser godoc
// @Summary      Create user
// @Description  Creates a new user.
// @Tags         users
// @Security     BearerAuth
// @Accept       json
// @Param        request body CreateUserInput true "Create user payload"
// @Success      201 "Created"
// @Failure      default {object} apperr.appError "Error"
// @Router       /register [post]
func (h *Handler) CreateUser(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	var in CreateUserInput
	if err := httpx.DecodeJSON(r, &in); err != nil {
		logger.Error(ctx, err).
			Msg("user.Handler.CreateUser: request json decode failed")
		httpx.ReturnError(ctx, w, apperr.ErrBadRequest())
		return
	}

	req := user.CreateUserReq{
		Name:     in.Name,
		Email:    in.Email,
		Password: []byte(in.Password),
	}
	defer secure.ZeroBytes(req.Password)
	in.Password = ""

	if err := h.svc.CreateUser(ctx, req); err != nil {
		httpx.ReturnError(ctx, w, err)
		return
	}

	w.WriteHeader(http.StatusCreated)
}

// GetUser godoc
// @Summary      Get user by ID
// @Description  Returns a single user by ID. Requires admin role or self.
// @Tags         users
// @Security     BearerAuth
// @Produce      json
// @Param        user_id path string true "User ID"
// @Success      200 {object} user.User
// @Failure      default {object} apperr.appError "Error"
// @Router       /users/{user_id} [get]
func (h *Handler) GetUser(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	idStr := chi.URLParam(r, URLParamUserID)
	id, err := uuid.Parse(idStr)
	if err != nil {
		logger.Warn(ctx, err).Str(user.FieldUserID.String(), idStr).
			Msg("user.Handler.GetUser: invalid user ID format")
		httpx.ReturnError(ctx, w, apperr.ErrBadRequest())
		return
	}

	usr, err := h.svc.GetUser(ctx, id)
	if err != nil {
		httpx.ReturnError(ctx, w, err)
		return
	}

	httpx.WriteJSON(ctx, w, http.StatusOK, usr)
}

// GetAllUsers godoc
// @Summary      List users
// @Description  Returns all users. Requires admin role.
// @Tags         users
// @Security     BearerAuth
// @Produce      json
// @Success      200 {array} user.User
// @Failure      default {object} apperr.appError "Error"
// @Router       /users [get]
func (h *Handler) GetAllUsers(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	users, err := h.svc.GetAllUsers(ctx)
	if err != nil {
		httpx.ReturnError(ctx, w, err)
		return
	}

	httpx.WriteJSON(ctx, w, http.StatusOK, users)
}

// UpdateUser godoc
// @Summary      Update user
// @Description  Updates user's basic fields. Requires admin role or self.
// @Tags         users
// @Security     BearerAuth
// @Accept       json
// @Param        user_id path string true "User ID"
// @Param        request body UpdateUserInput true "Update user payload"
// @Success      204 "No Content"
// @Failure      default {object} apperr.appError "Error"
// @Router       /users/{user_id} [put]
func (h *Handler) UpdateUser(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	idStr := chi.URLParam(r, URLParamUserID)
	id, err := uuid.Parse(idStr)
	if err != nil {
		logger.Warn(ctx, err).
			Str(user.FieldUserID.String(), idStr).
			Msg("user.Handler.UpdateUser: invalid user ID format")
		httpx.ReturnError(ctx, w, apperr.ErrBadRequest())
		return
	}

	var in UpdateUserInput
	if err = httpx.DecodeJSON(r, &in); err != nil {
		logger.Error(ctx, err).
			Str(user.FieldUserID.String(), idStr).
			Msg("user.Handler.UpdateUser: request json decode failed")
		httpx.ReturnError(ctx, w, apperr.ErrBadRequest())
		return
	}

	req := user.UpdateUserReq{
		UserID: id,
		Email:  in.Email,
		Name:   in.Name,
	}

	if err = h.svc.UpdateUser(ctx, req); err != nil {
		httpx.ReturnError(ctx, w, err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// DeleteUser godoc
// @Summary      Delete user
// @Description  Deletes a user by ID. Requires admin role.
// @Tags         users
// @Security     BearerAuth
// @Param        user_id path string true "User ID"
// @Success      204 "No Content"
// @Failure      default {object} apperr.appError "Error"
// @Router       /users/{user_id} [delete]
func (h *Handler) DeleteUser(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	idStr := chi.URLParam(r, URLParamUserID)
	id, err := uuid.Parse(idStr)
	if err != nil {
		logger.Warn(ctx, err).
			Str(user.FieldUserID.String(), idStr).
			Msg("user.Handler.DeleteUser: invalid user ID format")
		httpx.ReturnError(ctx, w, apperr.ErrBadRequest())
		return
	}

	if err = h.svc.DeleteUser(ctx, id); err != nil {
		httpx.ReturnError(ctx, w, err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// ChangePassword godoc
// @Summary      Change user password
// @Description  Changes password for the specified user (old -> new). Requires admin role or self. If admin changes password, old password is not checked.
// @Tags         users
// @Security     BearerAuth
// @Accept       json
// @Param        user_id path string true "User ID"
// @Param        request body ChangePasswordInput true "Change password payload"
// @Success      204 "No Content"
// @Failure      default {object} apperr.appError "Error"
// @Router       /users/{user_id}/password [post]
func (h *Handler) ChangePassword(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	idStr := chi.URLParam(r, URLParamUserID)
	id, err := uuid.Parse(idStr)
	if err != nil {
		logger.Warn(ctx, err).
			Str(user.FieldUserID.String(), idStr).
			Msg("user.Handler.ChangePassword: invalid user ID format")
		httpx.ReturnError(ctx, w, apperr.ErrBadRequest())
		return
	}

	var in ChangePasswordInput
	if err = httpx.DecodeJSON(r, &in); err != nil {
		logger.Error(ctx, err).
			Str(user.FieldUserID.String(), idStr).
			Msg("user.Handler.ChangePassword: request json decode failed")
		httpx.ReturnError(ctx, w, apperr.ErrBadRequest())
		return
	}

	cmd := usecase.ChangePasswordCmd{
		ID:          id,
		NewPassword: []byte(in.NewPassword),
		OldPassword: []byte(in.OldPassword),
	}
	defer secure.ZeroBytes(cmd.NewPassword)
	defer secure.ZeroBytes(cmd.OldPassword)
	in.OldPassword = ""
	in.NewPassword = ""

	if err = h.svc.ChangePassword(ctx, cmd); err != nil {
		httpx.ReturnError(ctx, w, err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
