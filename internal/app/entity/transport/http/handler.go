package http

import (
	"context"
	"fmt"
	"net/http"
	"strconv"

	"github.com/66gu1/easygodocs/internal/app/entity"
	"github.com/66gu1/easygodocs/internal/app/entity/usecase"
	"github.com/66gu1/easygodocs/internal/infrastructure/apperr"
	"github.com/66gu1/easygodocs/internal/infrastructure/httpx"
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
}

type SaveDraftInput struct {
	Name    string `json:"name"`
	Content string `json:"content"`
}

// Handler knows how to decode HTTP → service calls and encode responses.
type Handler struct {
	svc Service
}

type Service interface {
	GetTree(ctx context.Context) (entity.Tree, error)
	Get(ctx context.Context, id uuid.UUID) (entity.Entity, error)
	GetVersion(ctx context.Context, id uuid.UUID, version int) (entity.Entity, error)
	GetVersionsList(ctx context.Context, id uuid.UUID) ([]entity.Entity, error)
	Create(ctx context.Context, req usecase.CreateEntityCmd) (uuid.UUID, error)
	Update(ctx context.Context, req usecase.UpdateEntityCmd) error
	Delete(ctx context.Context, id uuid.UUID) error
	SaveDraft(ctx context.Context, cmd usecase.SaveDraftCmd) error
	GetDraft(ctx context.Context, entityID uuid.UUID) (entity.Draft, error)
	DeleteDraft(ctx context.Context, entityID uuid.UUID) error
	PublishDraft(ctx context.Context, entityID uuid.UUID) error
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
// @Failure      default {object} apperr.AppError "Error"
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
// @Failure      default {object} apperr.AppError "Error"
// @Router       /entities/{entity_id} [get]
func (h *Handler) Get(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	idStr := chi.URLParam(r, URLParamEntityID)
	id, err := uuid.Parse(idStr)
	if err != nil {
		httpx.ReturnError(ctx, w, fmt.Errorf("entity.Handler.Get: invalid entity ID format: %w", apperr.ErrBadRequest().WithDetail(err.Error())))
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
// @Failure      default {object} apperr.AppError "Error"
// @Router       /entities/{entity_id}/versions/{version} [get]
func (h *Handler) GetVersion(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	idStr := chi.URLParam(r, URLParamEntityID)
	id, err := uuid.Parse(idStr)
	if err != nil {
		httpx.ReturnError(ctx, w, fmt.Errorf("entity.Handler.GetVersion: invalid entity ID format: %w", apperr.ErrBadRequest().WithDetail(err.Error())))
		return
	}

	versionStr := chi.URLParam(r, URLParamVersion)
	version, err := strconv.Atoi(versionStr)
	if err != nil {
		httpx.ReturnError(ctx, w, fmt.Errorf("entity.Handler.GetVersion: invalid version format: %w", apperr.ErrBadRequest().WithDetail(err.Error())))
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
// @Failure      default {object} apperr.AppError "Error"
// @Router       /entities/{entity_id}/versions [get]
func (h *Handler) GetVersionsList(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	idStr := chi.URLParam(r, URLParamEntityID)
	id, err := uuid.Parse(idStr)
	if err != nil {
		httpx.ReturnError(ctx, w, fmt.Errorf("entity.Handler.GetVersionsList: invalid entity ID format: %w", apperr.ErrBadRequest().WithDetail(err.Error())))
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
// @Description  With is_draft the entity is not published: it is visible only to its creator and can't have children.
// @Tags         entities
// @Security     BearerAuth
// @Accept       json
// @Produce      json
// @Param        request body usecase.CreateEntityCmd true "Create entity payload"
// @Success      201 {object} CreateEntityResp
// @Failure      default {object} apperr.AppError "Error"
// @Router       /entities [post]
func (h *Handler) Create(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	var cmd usecase.CreateEntityCmd
	if err := httpx.DecodeJSON(r, &cmd); err != nil {
		httpx.ReturnError(ctx, w, fmt.Errorf("entity.Handler.Create: invalid request body: %w", apperr.ErrBadRequest().WithDetail(err.Error())))
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
// @Description  Updates an existing entity and publishes the changes as a new version. Requires write permission. If changes parent, requires write permission for the new and old parents as well.
// @Tags         entities
// @Security     BearerAuth
// @Accept       json
// @Param        entity_id path string true "Entity ID"
// @Param        request body UpdateEntityInput true "Update entity payload"
// @Success      204 "No Content"
// @Failure      default {object} apperr.AppError "Error"
// @Router       /entities/{entity_id} [put]
func (h *Handler) Update(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	idStr := chi.URLParam(r, URLParamEntityID)
	id, err := uuid.Parse(idStr)
	if err != nil {
		httpx.ReturnError(ctx, w, fmt.Errorf("entity.Handler.Update: invalid entity ID format: %w", apperr.ErrBadRequest().WithDetail(err.Error())))
		return
	}

	var input UpdateEntityInput
	if err = httpx.DecodeJSON(r, &input); err != nil {
		httpx.ReturnError(ctx, w, fmt.Errorf("entity.Handler.Update: invalid request body: %w", apperr.ErrBadRequest().WithDetail(err.Error())))
		return
	}

	if err = h.svc.Update(ctx, usecase.UpdateEntityCmd{
		ID:       id,
		Name:     input.Name,
		Content:  input.Content,
		ParentID: input.ParentID,
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
// @Failure      default {object} apperr.AppError "Error"
// @Router       /entities/{entity_id} [delete]
func (h *Handler) Delete(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	idStr := chi.URLParam(r, URLParamEntityID)
	id, err := uuid.Parse(idStr)
	if err != nil {
		httpx.ReturnError(ctx, w, fmt.Errorf("entity.Handler.Delete: invalid entity ID format: %w", apperr.ErrBadRequest().WithDetail(err.Error())))
		return
	}

	if err = h.svc.Delete(ctx, id); err != nil {
		httpx.ReturnError(ctx, w, err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// GetDraft godoc
// @Summary      Get own draft
// @Description  Returns the current user's draft of the entity. Requires write permission.
// @Tags         drafts
// @Security     BearerAuth
// @Produce      json
// @Param        entity_id path string true "Entity ID"
// @Success      200 {object} entity.Draft
// @Failure      default {object} apperr.AppError "Error"
// @Router       /entities/{entity_id}/draft [get]
func (h *Handler) GetDraft(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	id, err := uuid.Parse(chi.URLParam(r, URLParamEntityID))
	if err != nil {
		httpx.ReturnError(ctx, w, fmt.Errorf("entity.Handler.GetDraft: invalid entity ID format: %w", apperr.ErrBadRequest().WithDetail(err.Error())))
		return
	}

	draft, err := h.svc.GetDraft(ctx, id)
	if err != nil {
		httpx.ReturnError(ctx, w, err)
		return
	}

	httpx.WriteJSON(ctx, w, http.StatusOK, draft)
}

// SaveDraft godoc
// @Summary      Save own draft
// @Description  Creates or overwrites the current user's draft of the entity. The published version doesn't change. Requires write permission.
// @Tags         drafts
// @Security     BearerAuth
// @Accept       json
// @Param        entity_id path string true "Entity ID"
// @Param        request body SaveDraftInput true "Draft payload"
// @Success      204 "No Content"
// @Failure      default {object} apperr.AppError "Error"
// @Router       /entities/{entity_id}/draft [put]
func (h *Handler) SaveDraft(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	id, err := uuid.Parse(chi.URLParam(r, URLParamEntityID))
	if err != nil {
		httpx.ReturnError(ctx, w, fmt.Errorf("entity.Handler.SaveDraft: invalid entity ID format: %w", apperr.ErrBadRequest().WithDetail(err.Error())))
		return
	}

	var input SaveDraftInput
	if err = httpx.DecodeJSON(r, &input); err != nil {
		httpx.ReturnError(ctx, w, fmt.Errorf("entity.Handler.SaveDraft: invalid request body: %w", apperr.ErrBadRequest().WithDetail(err.Error())))
		return
	}

	if err = h.svc.SaveDraft(ctx, usecase.SaveDraftCmd{
		EntityID: id,
		Name:     input.Name,
		Content:  input.Content,
	}); err != nil {
		httpx.ReturnError(ctx, w, err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// DeleteDraft godoc
// @Summary      Discard own draft
// @Description  Deletes the current user's draft of the entity. Requires write permission.
// @Tags         drafts
// @Security     BearerAuth
// @Param        entity_id path string true "Entity ID"
// @Success      204 "No Content"
// @Failure      default {object} apperr.AppError "Error"
// @Router       /entities/{entity_id}/draft [delete]
func (h *Handler) DeleteDraft(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	id, err := uuid.Parse(chi.URLParam(r, URLParamEntityID))
	if err != nil {
		httpx.ReturnError(ctx, w, fmt.Errorf("entity.Handler.DeleteDraft: invalid entity ID format: %w", apperr.ErrBadRequest().WithDetail(err.Error())))
		return
	}

	if err = h.svc.DeleteDraft(ctx, id); err != nil {
		httpx.ReturnError(ctx, w, err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// PublishDraft godoc
// @Summary      Publish own draft
// @Description  Publishes the current user's draft as a new version of the entity and deletes the draft.
// @Description  Fails with 409 if the entity has been published since the draft was started. Requires write permission.
// @Tags         drafts
// @Security     BearerAuth
// @Param        entity_id path string true "Entity ID"
// @Success      204 "No Content"
// @Failure      default {object} apperr.AppError "Error"
// @Router       /entities/{entity_id}/draft/publish [post]
func (h *Handler) PublishDraft(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	id, err := uuid.Parse(chi.URLParam(r, URLParamEntityID))
	if err != nil {
		httpx.ReturnError(ctx, w, fmt.Errorf("entity.Handler.PublishDraft: invalid entity ID format: %w", apperr.ErrBadRequest().WithDetail(err.Error())))
		return
	}

	if err = h.svc.PublishDraft(ctx, id); err != nil {
		httpx.ReturnError(ctx, w, err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
