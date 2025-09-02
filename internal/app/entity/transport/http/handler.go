package http

import (
	"context"
	"net/http"
	"strconv"

	"github.com/66gu1/easygodocs/internal/app/entity"
	"github.com/66gu1/easygodocs/internal/app/entity/usecase"
	"github.com/66gu1/easygodocs/internal/infrastructure/apperr"
	"github.com/66gu1/easygodocs/internal/infrastructure/httpx"
	"github.com/66gu1/easygodocs/internal/infrastructure/logger"
	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
)

const (
	URLParamEntityID = "entity_id"
	URLParamVersion  = "version"
)

// Handler knows how to decode HTTP → service calls and encode responses.
type Handler struct {
	svc Service
}

//go:generate minimock -i github.com/66gu1/easygodocs/internal/app/entity.EntityService -o ./mock -s _mock.go
type Service interface {
	GetTree(ctx context.Context) (entity.Tree, error)
	Get(ctx context.Context, id uuid.UUID) (entity.Entity, error)
	GetVersion(ctx context.Context, id uuid.UUID, version int) (entity.Entity, error)
	GetVersionsList(ctx context.Context, id uuid.UUID) ([]entity.Entity, error)
	Create(ctx context.Context, req usecase.CreateEntityCmd) (uuid.UUID, error)
	Update(ctx context.Context, req usecase.UpdateEntityCmd) error
	Delete(ctx context.Context, id uuid.UUID) error
}

func NewHandler(svc Service) *Handler {
	return &Handler{svc: svc}
}

func (h *Handler) GetTree(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	tree, err := h.svc.GetTree(ctx)
	if err != nil {
		httpx.ReturnError(ctx, w, err)
		return
	}

	httpx.WriteJSON(ctx, w, http.StatusOK, tree)
}

func (h *Handler) Get(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	idStr := chi.URLParam(r, URLParamEntityID)
	id, err := uuid.Parse(idStr)
	if err != nil {
		logger.Warn(ctx, err).
			Str(entity.FieldEntityID.String(), idStr).
			Msg("entity.Handler.Get: invalid entity ID format")
		httpx.ReturnError(ctx, w, apperr.ErrBadRequest())
		return
	}

	ent, err := h.svc.Get(ctx, id)
	if err != nil {
		httpx.ReturnError(ctx, w, err)
		return
	}

	httpx.WriteJSON(ctx, w, http.StatusOK, ent)
}

func (h *Handler) GetVersion(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	idStr := chi.URLParam(r, URLParamEntityID)
	id, err := uuid.Parse(idStr)
	if err != nil {
		logger.Warn(ctx, err).
			Str(entity.FieldEntityID.String(), idStr).
			Msg("entity.Handler.GetVersion: invalid entity ID format")
		httpx.ReturnError(ctx, w, apperr.ErrBadRequest())
		return
	}

	versionStr := chi.URLParam(r, URLParamVersion)
	version, err := strconv.Atoi(versionStr)
	if err != nil {
		logger.Warn(ctx, err).
			Str(entity.FieldEntityID.String(), idStr).
			Str(entity.FieldVersion.String(), versionStr).
			Msg("entity.Handler.GetVersion: invalid version format")
		httpx.ReturnError(ctx, w, apperr.ErrBadRequest())
		return
	}

	ent, err := h.svc.GetVersion(ctx, id, version)
	if err != nil {
		httpx.ReturnError(ctx, w, err)
		return
	}

	httpx.WriteJSON(ctx, w, http.StatusOK, ent)
}

func (h *Handler) GetVersionsList(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	idStr := chi.URLParam(r, URLParamEntityID)
	id, err := uuid.Parse(idStr)
	if err != nil {
		logger.Warn(ctx, err).
			Str(entity.FieldEntityID.String(), idStr).
			Msg("entity.Handler.GetVersionsList: invalid entity ID format")
		httpx.ReturnError(ctx, w, apperr.ErrBadRequest())
		return
	}

	versions, err := h.svc.GetVersionsList(ctx, id)
	if err != nil {
		httpx.ReturnError(ctx, w, err)
		return
	}

	httpx.WriteJSON(ctx, w, http.StatusOK, versions)
}

func (h *Handler) Create(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	var cmd usecase.CreateEntityCmd
	if err := httpx.DecodeJSON(r, &cmd); err != nil {
		logger.Error(ctx, err).
			Msg("entity.Handler.Create.DecodeJSON")
		httpx.ReturnError(ctx, w, apperr.ErrBadRequest())
		return
	}

	id, err := h.svc.Create(ctx, cmd)
	if err != nil {
		httpx.ReturnError(ctx, w, err)
		return
	}

	w.Header().Set("Location", "/entities/"+id.String())

	httpx.WriteJSON(ctx, w, http.StatusCreated, map[string]uuid.UUID{URLParamEntityID: id})
}

func (h *Handler) Update(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	idStr := chi.URLParam(r, URLParamEntityID)
	id, err := uuid.Parse(idStr)
	if err != nil {
		logger.Warn(ctx, err).
			Str(entity.FieldEntityID.String(), idStr).
			Msg("entity.Handler.Update: invalid entity ID format")
		httpx.ReturnError(ctx, w, apperr.ErrBadRequest())
		return
	}

	type UpdateEntityInput struct {
		Name     string     `json:"name"`
		Content  string     `json:"content"`
		ParentID *uuid.UUID `json:"parent_id,omitempty"`
		IsDraft  bool       `json:"is_draft,omitempty"`
	}
	var input UpdateEntityInput
	if err = httpx.DecodeJSON(r, &input); err != nil {
		logger.Error(ctx, err).
			Msg("entity.Handler.Update: failed to decode JSON")
		httpx.ReturnError(ctx, w, apperr.ErrBadRequest())
		return
	}

	if err = h.svc.Update(ctx, usecase.UpdateEntityCmd{
		ID:       id,
		Name:     input.Name,
		Content:  input.Content,
		ParentID: input.ParentID,
		IsDraft:  input.IsDraft,
	}); err != nil {
		httpx.ReturnError(ctx, w, err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) Delete(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	idStr := chi.URLParam(r, URLParamEntityID)
	id, err := uuid.Parse(idStr)
	if err != nil {
		logger.Warn(ctx, err).
			Str(entity.FieldEntityID.String(), idStr).
			Msg("entity.Handler.Delete: invalid entity ID format")
		httpx.ReturnError(ctx, w, apperr.ErrBadRequest())
		return
	}

	if err = h.svc.Delete(ctx, id); err != nil {
		httpx.ReturnError(ctx, w, err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
