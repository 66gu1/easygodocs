package department

import (
	"context"
	"encoding/json"
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
}

//go:generate minimock -i github.com/66gu1/easygodocs/internal/app/department.Service -o ./mock -s _mock.go
type Service interface {
	Create(ctx context.Context, req CreateDepartmentReq) (uuid.UUID, error)
	Update(ctx context.Context, req UpdateDepartmentReq) error
	Delete(ctx context.Context, id uuid.UUID) error
	GetDepartmentTree(ctx context.Context) (Tree, error)
}

func NewHandler(svc Service) *Handler {
	return &Handler{svc: svc}
}

func (h *Handler) GetDepartmentTree(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	deps, err := h.svc.GetDepartmentTree(ctx)
	if err != nil {
		httputil.ReturnError(ctx, w, err)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	err = json.NewEncoder(w).Encode(deps)
	if err != nil {
		logger.Error(ctx, err).Msg("department.Handler.GetDepartmentTree: response JSON encode failed")
		httputil.ReturnError(ctx, w, err)
		return
	}
}

func (h *Handler) Create(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	var req CreateDepartmentReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		logger.Error(ctx, err).Msg("department.Handler.Create: request json decode failed")
		httputil.ReturnError(ctx, w, err)
		return
	}
	if err := req.Validate(); err != nil {
		logger.Error(ctx, err).Msg("department.Handler.Create")
		httputil.ReturnError(ctx, w, err)
		return
	}

	id, err := h.svc.Create(ctx, req)
	if err != nil {
		httputil.ReturnError(ctx, w, err)
		return
	}

	w.WriteHeader(http.StatusCreated)
	err = json.NewEncoder(w).Encode(map[string]string{"id": id.String()})
	if err != nil {
		logger.Error(ctx, err).Msg("department.Handler.Create: response JSON encode failed")
		httputil.ReturnError(ctx, w, err)
	}
}

func (h *Handler) Update(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	var req UpdateDepartmentReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		logger.Error(ctx, err).Msg("department.Handler.Update: request json decode failed")
		httputil.ReturnError(ctx, w, err)
		return
	}

	if err := req.Validate(); err != nil {
		logger.Error(ctx, err).Msg("department.Handler.Update")
		httputil.ReturnError(ctx, w, err)
		return
	}

	if err := h.svc.Update(ctx, req); err != nil {
		httputil.ReturnError(ctx, w, err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) Delete(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	idStr := chi.URLParam(r, "id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		err = &apperror.Error{
			Message:  "invalid department ID",
			Code:     apperror.BadRequest,
			LogLevel: apperror.LogLevelWarn,
		}
		logger.Error(ctx, err).Msg("department.Handler.Delete")
		httputil.ReturnError(ctx, w, err)
		return
	}

	if id == uuid.Nil {
		err = idRequiredErr
		logger.Error(ctx, err).Msg("department.Handler.Delete")
		httputil.ReturnError(ctx, w, err)
		return
	}

	if err = h.svc.Delete(ctx, id); err != nil {
		httputil.ReturnError(ctx, w, err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
