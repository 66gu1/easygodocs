package entity

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/66gu1/easygodocs/internal/infrastructure/apperr"
	"github.com/google/uuid"
)

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
}

type IDGenerator interface {
	New() (uuid.UUID, error)
}

type TimeGenerator interface {
	Now() time.Time
}

type Validator interface {
	NormalizeName(name string) string
	ValidateName(name string) error
}

type Generators struct {
	ID   IDGenerator
	Time TimeGenerator
}

type core struct {
	repo      Repository
	gen       Generators
	validator Validator
}

func NewCore(repo Repository, generators Generators, validator Validator) (*core, error) {
	if repo == nil || generators.ID == nil || generators.Time == nil || validator == nil {
		return nil, fmt.Errorf("entity.NewCore: %w", fmt.Errorf("nil dependency"))
	}
	return &core{
		repo:      repo,
		gen:       generators,
		validator: validator,
	}, nil
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
		return Entity{}, fmt.Errorf("entity.core.GetVersion: %w", ErrInvalidVersion())
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
	req.Name = c.validator.NormalizeName(req.Name)
	if err := c.validator.ValidateName(req.Name); err != nil {
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
	req.Name = c.validator.NormalizeName(req.Name)
	if err := c.validator.ValidateName(req.Name); err != nil {
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

type ValidationConfig struct {
	MaxNameLength int `mapstructure:"max_name_length" json:"max_name_length"`
}

type validator struct {
	cfg ValidationConfig
}

func NewValidator(cfg ValidationConfig) (*validator, error) {
	if cfg.MaxNameLength <= 0 {
		return nil, fmt.Errorf("entity.NewValidator: %w", fmt.Errorf("max name length must be positive"))
	}
	return &validator{cfg: cfg}, nil
}

func (c *validator) NormalizeName(name string) string {
	return strings.TrimSpace(name)
}

func (c *validator) ValidateName(name string) error {
	if name == "" {
		return fmt.Errorf("validateName: %w", ErrNameRequired())
	}
	if len(name) > c.cfg.MaxNameLength {
		return fmt.Errorf("validateName: %w", ErrNameTooLong(c.cfg.MaxNameLength))
	}

	return nil
}
