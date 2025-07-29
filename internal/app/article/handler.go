package article

import (
	"context"
	"encoding/json"
	"github.com/66gu1/easygodocs/internal/app/article/dto"
	"github.com/66gu1/easygodocs/internal/infrastructure/apperror"
	"github.com/66gu1/easygodocs/internal/infrastructure/httputil"
	"github.com/66gu1/easygodocs/internal/infrastructure/logger"
	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"net/http"
	"strconv"
)

var (
	ErrInvalidID = &apperror.Error{
		Message:  "invalid article ID",
		Code:     apperror.BadRequest,
		LogLevel: apperror.LogLevelWarn,
	}
)

type Handler struct {
	svc Service
}

//go:generate minimock -i github.com/66gu1/easygodocs/internal/app/article.Service -o ./mock -s _mock.go
type Service interface {
	Get(ctx context.Context, id uuid.UUID) (dto.Article, error)
	GetVersion(ctx context.Context, id uuid.UUID, version int) (dto.Article, error)
	GetVersionsList(ctx context.Context, id uuid.UUID) ([]dto.Article, error)
	Create(ctx context.Context, req dto.CreateArticleReq) (uuid.UUID, error)
	CreateDraft(ctx context.Context, req dto.CreateArticleReq) (uuid.UUID, error)
	Update(ctx context.Context, req dto.UpdateArticleReq) error
	Delete(ctx context.Context, id uuid.UUID) error
}

func NewHandler(svc Service) *Handler {
	return &Handler{svc: svc}
}

func (h *Handler) Get(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	idStr := chi.URLParam(r, "id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		logger.Warn(ctx, err).Str("id", idStr).Msg("article.Handler.Get: invalid article ID format")
		httputil.ReturnError(ctx, w, apperror.ErrBadRequest)
		return
	}

	article, err := h.svc.Get(ctx, id)
	if err != nil {
		httputil.ReturnError(ctx, w, err)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	err = json.NewEncoder(w).Encode(article)
	if err != nil {
		logger.Error(ctx, err).Msg("article.Handler.Get: response JSON encode failed")
		httputil.ReturnError(ctx, w, err)
		return
	}
}

func (h *Handler) GetVersion(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	idStr := chi.URLParam(r, "id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		logger.Warn(ctx, err).Str("id", idStr).Msg("article.Handler.GetVersion: invalid article ID format")
		httputil.ReturnError(ctx, w, apperror.ErrBadRequest)
		return
	}

	versionStr := chi.URLParam(r, "version")
	version, err := strconv.Atoi(versionStr)
	if err != nil {
		logger.Warn(ctx, err).Str("version", versionStr).Msg("article.Handler.GetVersion: invalid version format")
		httputil.ReturnError(ctx, w, apperror.ErrBadRequest)
		return
	}

	article, err := h.svc.GetVersion(ctx, id, version)
	if err != nil {
		httputil.ReturnError(ctx, w, err)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	err = json.NewEncoder(w).Encode(article)
	if err != nil {
		logger.Error(ctx, err).Msg("article.Handler.GetVersion: response JSON encode failed")
		httputil.ReturnError(ctx, w, err)
		return
	}
}

func (h *Handler) GetVersionsList(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	idStr := chi.URLParam(r, "id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		logger.Warn(ctx, err).Str("id", idStr).Msg("article.Handler.GetVersionsList: invalid article ID format")
		httputil.ReturnError(ctx, w, apperror.ErrBadRequest)
		return
	}

	versions, err := h.svc.GetVersionsList(ctx, id)
	if err != nil {
		httputil.ReturnError(ctx, w, err)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	err = json.NewEncoder(w).Encode(versions)
	if err != nil {
		logger.Error(ctx, err).Msg("article.Handler.GetVersionsList: response JSON encode failed")
		httputil.ReturnError(ctx, w, err)
		return
	}
}

func (h *Handler) Create(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	var req dto.CreateArticleReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		logger.Error(ctx, err).Msg("article.Handler.Create: request JSON decode failed")
		httputil.ReturnError(ctx, w, apperror.ErrBadRequest)
		return
	}

	if err := req.Validate(); err != nil {
		logger.Error(ctx, err).Msg("article.Handler.Create: request validation failed")
		httputil.ReturnError(ctx, w, err)
		return
	}

	id, err := h.svc.Create(ctx, req)
	if err != nil {
		httputil.ReturnError(ctx, w, err)
		return
	}

	httputil.WriteJSON(ctx, w, http.StatusCreated, map[string]string{"id": id.String()})
}

func (h *Handler) CreateDraft(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	var req dto.CreateArticleReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		logger.Error(ctx, err).Msg("article.Handler.CreateDraft: request JSON decode failed")
		httputil.ReturnError(ctx, w, apperror.ErrBadRequest)
		return
	}

	if err := req.Validate(); err != nil {
		logger.Error(ctx, err).Msg("article.Handler.CreateDraft: request validation failed")
		httputil.ReturnError(ctx, w, err)
		return
	}

	id, err := h.svc.CreateDraft(ctx, req)
	if err != nil {
		httputil.ReturnError(ctx, w, err)
		return
	}

	httputil.WriteJSON(ctx, w, http.StatusCreated, map[string]string{"id": id.String()})
}

func (h *Handler) Update(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	var req dto.UpdateArticleReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		logger.Error(ctx, err).Msg("article.Handler.Update: request JSON decode failed")
		httputil.ReturnError(ctx, w, apperror.ErrBadRequest)
		return
	}

	if err := req.Validate(); err != nil {
		logger.Error(ctx, err).Msg("article.Handler.Update: request validation failed")
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
		logger.Warn(ctx, err).Str("id", idStr).Msg("article.Handler.Delete: invalid article ID format")
		httputil.ReturnError(ctx, w, apperror.ErrBadRequest)
		return
	}

	if err := h.svc.Delete(ctx, id); err != nil {
		httputil.ReturnError(ctx, w, err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
