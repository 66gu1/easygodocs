package entity

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/66gu1/easygodocs/internal/infrastructure/apperr"
	"github.com/google/uuid"
)

func ErrEntityNotFound() error {
	return apperr.New("Entity not found", CodeNotFound, apperr.ClassNotFound, apperr.LogLevelWarn)
}

func ErrParentCycle() error {
	return apperr.New("Parent cycle detected", CodeParentCycle, apperr.ClassBadRequest, apperr.LogLevelWarn).
		WithViolation(apperr.Violation{
			Field: FieldParentID, Rule: apperr.RuleCycle,
		})
}

func ErrMaxHierarchyDepthExceeded(maxDepth int) error {
	return apperr.New("Maximum hierarchy depth exceeded", CodeMaxDepthExceeded, apperr.ClassBadRequest, apperr.LogLevelWarn).
		WithViolation(apperr.Violation{
			Field: FieldParentID, Rule: apperr.RuleMaxHierarchy,
			Params: map[string]any{"max_depth": maxDepth},
		})
}

const (
	CodeValidationFailed apperr.Code = "entity/validation_failed"
	CodeNotFound         apperr.Code = "entity/not_found"
	CodeParentCycle      apperr.Code = "entity/parent_cycle"
	CodeMaxDepthExceeded apperr.Code = "entity/max_depth_exceeded"
)

const (
	FieldName     apperr.Field = "name"
	FieldType     apperr.Field = "type"
	FieldParentID apperr.Field = "parent_id"
	FieldEntityID apperr.Field = "entity_id"
	FieldUserID   apperr.Field = "user_id"
)

func ErrNameRequired() error {
	return apperr.New("name is required", CodeValidationFailed, apperr.ClassBadRequest, apperr.LogLevelWarn).
		WithViolation(apperr.Violation{Field: FieldName, Rule: apperr.RuleRequired})
}

func ErrNameTooLong(max int) error {
	return apperr.New("name is too long", CodeValidationFailed, apperr.ClassBadRequest, apperr.LogLevelWarn).
		WithViolation(apperr.Violation{Field: FieldName, Rule: apperr.RuleTooLong, Params: map[string]any{"max": max}})
}

func ErrParentRequired() error {
	return apperr.New("article must have a parent entity", CodeValidationFailed, apperr.ClassBadRequest, apperr.LogLevelWarn).
		WithViolation(apperr.Violation{Field: FieldParentID, Rule: apperr.RuleRequired})
}

type Repository interface {
	GetPermittedHierarchy(ctx context.Context, permissions []uuid.UUID, onlyForRead bool) ([]ListItem, error)
	Get(ctx context.Context, id uuid.UUID) (Entity, error)
	GetVersion(ctx context.Context, id uuid.UUID, version int) (Entity, error)
	GetVersionsList(ctx context.Context, id uuid.UUID) ([]Entity, error)
	Create(ctx context.Context, req CreateEntityReq, id uuid.UUID, createdAt time.Time) error
	CreateDraft(ctx context.Context, req CreateEntityReq, id uuid.UUID) error
	Update(ctx context.Context, req UpdateEntityReq, updatedAt time.Time) error
	UpdateDraft(ctx context.Context, req UpdateEntityReq) error
	Delete(ctx context.Context, id uuid.UUID, deletedAt time.Time) error
	GetAll(ctx context.Context) ([]ListItem, error)
	GetListItem(ctx context.Context, id uuid.UUID) (ListItem, error)
	CheckParentDepthLimit(ctx context.Context, parentID uuid.UUID) error
	ValidateChangedParent(ctx context.Context, id, parentID uuid.UUID) error
	GetMaxNameLength() int
}

type IDGenerator interface {
	New() (uuid.UUID, error)
}

type TimeGenerator interface {
	Now() time.Time
}

type Generators struct {
	ID   IDGenerator
	Time TimeGenerator
}

type core struct {
	repo Repository
	gen  Generators
}

func NewCore(repo Repository, generators Generators) *core {
	return &core{
		repo: repo,
		gen:  generators,
	}
}

func (c *core) Get(ctx context.Context, id uuid.UUID) (Entity, error) {
	if id == uuid.Nil {
		return Entity{}, fmt.Errorf("entity.core.Get: %w", apperr.ErrNilUUID(FieldEntityID))
	}
	entity, err := c.repo.Get(ctx, id)
	if err != nil {
		return Entity{}, fmt.Errorf("entity.core.Get: %w", err)
	}

	return entity, nil
}

func (c *core) GetListItem(ctx context.Context, id uuid.UUID) (ListItem, error) {
	if id == uuid.Nil {
		return ListItem{}, fmt.Errorf("entity.core.GetListItem: %w", apperr.ErrNilUUID(FieldEntityID))
	}
	item, err := c.repo.GetListItem(ctx, id)
	if err != nil {
		return ListItem{}, fmt.Errorf("entity.core.GetListItem: %w", err)
	}

	return item, nil
}

func (c *core) GetTree(ctx context.Context, permissions []uuid.UUID, isAdmin bool) (Tree, error) {
	var (
		err       error
		permitted []ListItem
	)
	if isAdmin {
		permitted, err = c.repo.GetAll(ctx)
	} else {
		if len(permissions) == 0 {
			return Tree{}, nil
		}
		permitted, err = c.repo.GetPermittedHierarchy(ctx, permissions, true)
	}
	if err != nil {
		return nil, fmt.Errorf("entity.Service.GetTree: %w", err)
	}

	return BuildTree(ctx, permitted), nil
}

func (c *core) GetPermittedHierarchy(ctx context.Context, directPermissions []uuid.UUID, onlyForRead bool) ([]uuid.UUID, error) {
	if len(directPermissions) == 0 {
		return nil, nil
	}
	permitted, err := c.repo.GetPermittedHierarchy(ctx, directPermissions, onlyForRead)
	if err != nil {
		return nil, fmt.Errorf("entity.core.GetPermittedHierarchy: %w", err)
	}

	ids := make([]uuid.UUID, len(permitted))
	for i, item := range permitted {
		ids[i] = item.ID
	}

	return ids, nil
}

func (c *core) GetVersion(ctx context.Context, id uuid.UUID, version int) (Entity, error) {
	if id == uuid.Nil {
		return Entity{}, fmt.Errorf("entity.core.GetVersion: %w", apperr.ErrNilUUID(FieldEntityID))
	}
	if version <= 0 {
		return Entity{}, fmt.Errorf("entity.core.GetVersion: %w",
			apperr.New("version must be positive", CodeValidationFailed, apperr.ClassBadRequest, apperr.LogLevelWarn).
				WithViolation(apperr.Violation{Field: FieldVersion, Rule: apperr.RuleInvalidFormat}))
	}

	entity, err := c.repo.GetVersion(ctx, id, version)
	if err != nil {
		return Entity{}, fmt.Errorf("entity.core.GetVersion: %w", err)
	}

	return entity, nil
}

func (c *core) GetVersionsList(ctx context.Context, id uuid.UUID) ([]Entity, error) {
	if id == uuid.Nil {
		return nil, fmt.Errorf("entity.core.GetVersionsList: %w", apperr.ErrNilUUID(FieldEntityID))
	}
	entities, err := c.repo.GetVersionsList(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("entity.core.GetVersionsList: %w", err)
	}

	return entities, nil
}

func (c *core) Create(ctx context.Context, req CreateEntityReq) (uuid.UUID, error) {
	if req.UserID == uuid.Nil {
		return uuid.Nil, fmt.Errorf("entity.core.Create: %w", apperr.ErrNilUUID(FieldUserID))
	}
	if err := req.Type.CheckIsValid(); err != nil {
		return uuid.Nil, fmt.Errorf("entity.core.Create: %w", err)
	}
	req.Name = c.NormalizeName(req.Name)
	if err := c.ValidateName(req.Name); err != nil {
		return uuid.Nil, fmt.Errorf("entity.core.Create: %w", err)
	}

	err := c.validateParent(ctx, req.ParentID, req.Type)
	if err != nil {
		return uuid.Nil, fmt.Errorf("entity.core.Create: %w", err)
	}
	if req.ParentID != nil {
		if err = c.repo.CheckParentDepthLimit(ctx, *req.ParentID); err != nil {
			return uuid.Nil, fmt.Errorf("entity.core.Create: %w", err)
		}
	}

	now := c.gen.Time.Now()
	id, err := c.gen.ID.New()
	if err != nil {
		return uuid.Nil, fmt.Errorf("entity.core.Create: %w", err)
	}
	if req.IsDraft {
		err = c.repo.CreateDraft(ctx, req, id)
	} else {
		err = c.repo.Create(ctx, req, id, now)
	}
	if err != nil {
		return uuid.Nil, fmt.Errorf("entity.core.Create: %w", err)
	}

	return id, nil
}

func (c *core) Update(ctx context.Context, req UpdateEntityReq) error {
	if req.ID == uuid.Nil {
		return fmt.Errorf("entity.core.Update: %w", apperr.ErrNilUUID(FieldEntityID))
	}
	if req.UserID == uuid.Nil {
		return fmt.Errorf("entity.core.Update: %w", apperr.ErrNilUUID(FieldUserID))
	}
	req.Name = c.NormalizeName(req.Name)
	if err := c.ValidateName(req.Name); err != nil {
		return fmt.Errorf("entity.core.Update: %w", err)
	}

	if req.ParentChanged {
		entity, err := c.repo.GetListItem(ctx, req.ID)
		if err != nil {
			return fmt.Errorf("entity.core.Update: %w", err)
		}
		if err = c.validateParent(ctx, req.ParentID, entity.Type); err != nil {
			return fmt.Errorf("entity.core.Update: %w", err)
		}
		if req.ParentID != nil {
			if err = c.repo.ValidateChangedParent(ctx, req.ID, *req.ParentID); err != nil {
				return fmt.Errorf("entity.core.Update: %w", err)
			}
		}
	}

	var err error
	if req.IsDraft {
		err = c.repo.UpdateDraft(ctx, req)
	} else {
		now := c.gen.Time.Now()
		err = c.repo.Update(ctx, req, now)
	}
	if err != nil {
		return fmt.Errorf("entity.core.Update: %w", err)
	}
	return nil
}

func (c *core) Delete(ctx context.Context, id uuid.UUID) error {
	now := c.gen.Time.Now()
	if err := c.repo.Delete(ctx, id, now); err != nil {
		return fmt.Errorf("entity.core.Delete: %w", err)
	}

	return nil
}

func (c *core) GetAll(ctx context.Context) ([]ListItem, error) {
	entities, err := c.repo.GetAll(ctx)
	if err != nil {
		return nil, fmt.Errorf("entity.core.GetAll: %w", err)
	}

	return entities, nil
}

func (c *core) validateParent(ctx context.Context, parentID *uuid.UUID, entityType Type) error {
	if parentID != nil {
		if *parentID == uuid.Nil {
			return fmt.Errorf("validateParent: %w", apperr.ErrNilUUID(FieldParentID))
		}

		parent, err := c.repo.GetListItem(ctx, *parentID)
		if err != nil {
			return fmt.Errorf("validateParent: %w", err)
		}
		if err = entityType.ValidateParentTypeCompatibility(parent.Type); err != nil {
			return fmt.Errorf("validateParent: %w", err)
		}
	} else if entityType == TypeArticle {
		return fmt.Errorf("validateParent: %w", ErrParentRequired())
	}

	return nil
}

func (c *core) NormalizeName(name string) string {
	return strings.TrimSpace(name)
}

func (c *core) ValidateName(name string) error {
	maxLen := c.repo.GetMaxNameLength()
	if name == "" {
		return fmt.Errorf("validateName: %w", ErrNameRequired())
	}
	if len(name) > maxLen {
		return fmt.Errorf("validateName: %w", ErrNameTooLong(maxLen))
	}

	return nil
}
