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

type CreateEntityResp struct {
	ID uuid.UUID `json:"id"`
}

type UpdateEntityInput struct {
	Name     string     `json:"name"`
	Content  string     `json:"content"`
	ParentID *uuid.UUID `json:"parent_id,omitempty"`
	IsDraft  bool       `json:"is_draft,omitempty"`
}

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
	if svc == nil {
		panic("entity HTTP handler: nil service")
	}
	return &Handler{svc: svc}
}

// GetTree godoc
// @Summary      Get full entity tree
// @Description  Returns the hierarchical tree of all permitted entities.
// @Tags         entities
// @Security     BearerAuth
// @Produce      json
// @Success      200 {object} entity.Tree
// @Failure      default {object} apperr.appError "Error"
// @Router       /entities [get]
func (h *Handler) GetTree(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	tree, err := h.svc.GetTree(ctx)
	if err != nil {
		httpx.ReturnError(ctx, w, err)
		return
	}

	httpx.WriteJSON(ctx, w, http.StatusOK, tree)
}

// Get godoc
// @Summary      Get entity by ID
// @Description  Returns a single entity by its ID. Requires read permission.
// @Tags         entities
// @Security     BearerAuth
// @Produce      json
// @Param        entity_id path string true "Entity ID"
// @Success      200 {object} entity.Entity
// @Failure      default {object} apperr.appError "Error"
// @Router       /entities/{entity_id} [get]
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

// GetVersion godoc
// @Summary      Get specific entity version
// @Description  Returns a specific version of an entity. Requires read permission.
// @Tags         entities
// @Security     BearerAuth
// @Produce      json
// @Param        entity_id path string true "Entity ID"
// @Param        version   path int    true "Version number"
// @Success      200 {object} entity.Entity
// @Failure      default {object} apperr.appError "Error"
// @Router       /entities/{entity_id}/versions/{version} [get]
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

// GetVersionsList godoc
// @Summary      List entity versions
// @Description  Returns list of all versions for an entity. Requires read permission.
// @Tags         entities
// @Security     BearerAuth
// @Produce      json
// @Param        entity_id path string true "Entity ID"
// @Success      200 {array} entity.Entity
// @Failure      default {object} apperr.appError "Error"
// @Router       /entities/{entity_id}/versions [get]
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

// Create godoc
// @Summary      Create entity
// @Description  Creates a new entity. Requires write permission for the parent entity. if root entity, requires admin role.
// @Tags         entities
// @Security     BearerAuth
// @Accept       json
// @Produce      json
// @Param        request body usecase.CreateEntityCmd true "Create entity payload"
// @Success      201 {object} CreateEntityResp
// @Failure      default {object} apperr.appError "Error"
// @Router       /entities [post]
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

	httpx.WriteJSON(ctx, w, http.StatusCreated, CreateEntityResp{ID: id})
}

// Update godoc
// @Summary      Update entity
// @Description  Updates an existing entity. Requires write permission. If changes parent, requires write permission for the new and old parents as well.
// @Tags         entities
// @Security     BearerAuth
// @Accept       json
// @Param        entity_id path string true "Entity ID"
// @Param        request body UpdateEntityInput true "Update entity payload"
// @Success      204 "No Content"
// @Failure      default {object} apperr.appError "Error"
// @Router       /entities/{entity_id} [put]
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

// Delete godoc
// @Summary      Delete entity
// @Description  Deletes an entity by ID. Requires write permission for the entity.
// @Tags         entities
// @Security     BearerAuth
// @Param        entity_id path string true "Entity ID"
// @Success      204 "No Content"
// @Failure      default {object} apperr.appError "Error"
// @Router       /entities/{entity_id} [delete]
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
