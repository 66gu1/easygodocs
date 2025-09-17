package entity

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/66gu1/easygodocs/internal/infrastructure/apperr"
	"github.com/66gu1/easygodocs/internal/infrastructure/contextx"
	"github.com/google/uuid"
)

type Repository interface {
	GetHierarchy(ctx context.Context, ids []uuid.UUID, maxDepth int, userID *uuid.UUID, hType HierarchyType) ([]ListItem, error)
	Get(ctx context.Context, id uuid.UUID) (Entity, error)
	GetVersion(ctx context.Context, id uuid.UUID, version int) (Entity, error)
	GetVersionsList(ctx context.Context, id uuid.UUID) ([]Entity, error)
	Create(ctx context.Context, req CreateEntityReq, id uuid.UUID, createdAt time.Time) error
	CreateDraft(ctx context.Context, req CreateEntityReq, id uuid.UUID) error
	Update(ctx context.Context, req UpdateEntityReq, updatedAt time.Time) error
	UpdateDraft(ctx context.Context, req UpdateEntityReq) error
	Delete(ctx context.Context, ids []uuid.UUID) error
	GetAll(ctx context.Context) ([]ListItem, error)
	GetListItem(ctx context.Context, id uuid.UUID) (ListItem, error)
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

type HierarchyType int

const (
	HierarchyTypeChildrenAndParents HierarchyType = 1
	HierarchyTypeChildrenOnly       HierarchyType = 2
	HierarchyTypeParentsOnly        HierarchyType = 3
)

type Generators struct {
	ID   IDGenerator
	Time TimeGenerator
}

type Config struct {
	MaxHierarchyDepth int `mapstructure:"max_hierarchy_depth" json:"max_hierarchy_depth"`
}
type core struct {
	repo      Repository
	gen       Generators
	validator Validator
	cfg       Config
}

func NewCore(repo Repository, generators Generators, validator Validator, cfg Config) (*core, error) {
	if repo == nil || generators.ID == nil || generators.Time == nil || validator == nil {
		return nil, fmt.Errorf("entity.NewCore: %w", fmt.Errorf("nil dependency"))
	}
	if cfg.MaxHierarchyDepth <= 0 {
		return nil, fmt.Errorf("entity.NewCore: %w", fmt.Errorf("Config.MaxHierarchyDepth must be positive"))
	}
	return &core{
		repo:      repo,
		gen:       generators,
		validator: validator,
		cfg:       cfg,
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
		var userID uuid.UUID
		userID, err = contextx.GetUserID(ctx)
		if err != nil {
			return nil, fmt.Errorf("entity.Service.GetTree: %w", err)
		}
		permitted, err = c.repo.GetHierarchy(ctx, permissions, c.cfg.MaxHierarchyDepth, &userID, HierarchyTypeChildrenAndParents)
	}
	if err != nil {
		return nil, fmt.Errorf("entity.Service.GetTree: %w", err)
	}

	return BuildTree(ctx, permitted), nil
}

func (c *core) GetPermittedIDs(ctx context.Context, directPermissions []uuid.UUID, hType HierarchyType) ([]uuid.UUID, error) {
	if len(directPermissions) == 0 {
		return nil, nil
	}
	userID, err := contextx.GetUserID(ctx)
	if err != nil {
		return nil, fmt.Errorf("entity.Service.GetTree: %w", err)
	}
	permitted, err := c.repo.GetHierarchy(ctx, directPermissions, c.cfg.MaxHierarchyDepth, &userID, hType)
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

	if req.ParentID != nil {
		list, err := c.repo.GetHierarchy(ctx, []uuid.UUID{*req.ParentID}, c.cfg.MaxHierarchyDepth+1, nil, HierarchyTypeParentsOnly)
		if err != nil {
			return uuid.Nil, fmt.Errorf("entity.core.Create: %w", err)
		}
		if len(list)+1 > c.cfg.MaxHierarchyDepth {
			return uuid.Nil, fmt.Errorf("entity.core.Create: %w", ErrMaxHierarchyDepthExceeded(c.cfg.MaxHierarchyDepth))
		}
		var (
			parent ListItem
			found  bool
		)
		for _, item := range list {
			if item.ID == *req.ParentID {
				found = true
				parent = item
				break
			}
		}
		if !found {
			return uuid.Nil, fmt.Errorf("entity.core.Create: %w", ErrParentNotFound())
		}
		if err = req.Type.ValidateParentTypeCompatibility(parent.Type); err != nil {
			return uuid.Nil, fmt.Errorf("entity.core.Create: %w", err)
		}
	} else if req.Type == TypeArticle {
		return uuid.Nil, fmt.Errorf("entity.core.Create: %w", ErrParentRequired())
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
	var (
		hasChildren         bool
		hasChildrenComputed bool
	)
	if req.ParentChanged {
		if req.ParentID != nil {
			if *req.ParentID == req.ID {
				return fmt.Errorf("entity.core.Update: %w", ErrParentCycle())
			}
			list, err := c.repo.GetHierarchy(ctx, []uuid.UUID{*req.ParentID}, c.cfg.MaxHierarchyDepth+1, nil, HierarchyTypeParentsOnly)
			if err != nil {
				return fmt.Errorf("entity.core.Update: %w", err)
			}
			var (
				parent ListItem
				found  bool
			)
			for _, item := range list {
				if !found && item.ID == *req.ParentID {
					found = true
					parent = item
				}
				if item.ID == req.ID {
					return fmt.Errorf("entity.core.Update: %w", ErrParentCycle())
				}
			}
			if !found {
				return fmt.Errorf("entity.core.Update: %w", ErrParentNotFound())
			}
			if err = req.EntityType.ValidateParentTypeCompatibility(parent.Type); err != nil {
				return fmt.Errorf("entity.core.Update: %w", err)
			}
			parentDepth := len(list)

			list, err = c.repo.GetHierarchy(ctx, []uuid.UUID{req.ID}, c.cfg.MaxHierarchyDepth+1, nil, HierarchyTypeChildrenOnly)
			if err != nil {
				return fmt.Errorf("entity.core.Update: %w", err)
			}
			var maxChildDepth int
			for _, item := range list {
				if item.Depth > maxChildDepth {
					maxChildDepth = item.Depth
				}
			}

			if parentDepth+maxChildDepth > c.cfg.MaxHierarchyDepth {
				return fmt.Errorf("entity.core.Update: %w", ErrMaxHierarchyDepthExceeded(c.cfg.MaxHierarchyDepth))
			}
			hasChildren = len(list) > 1
			hasChildrenComputed = true
		} else if req.EntityType == TypeArticle {
			return fmt.Errorf("entity.core.Update: %w", ErrParentRequired())
		}
	}

	var err error
	if req.IsDraft {
		if !hasChildrenComputed {
			list, err := c.repo.GetHierarchy(ctx, []uuid.UUID{req.ID}, 2, nil, HierarchyTypeChildrenOnly)
			if err != nil {
				return fmt.Errorf("entity.core.Update: %w", err)
			}
			hasChildren = len(list) > 1
		}
		if hasChildren {
			return fmt.Errorf("entity.core.Update: %w", ErrCannotDraftEntityWithChildren())
		}
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
	list, err := c.repo.GetHierarchy(ctx, []uuid.UUID{id}, c.cfg.MaxHierarchyDepth+1, nil, HierarchyTypeChildrenOnly)
	if err != nil {
		return fmt.Errorf("entity.core.Delete: %w", err)
	}
	if len(list) == 0 {
		return fmt.Errorf("entity.core.Delete: %w", ErrEntityNotFound())
	}
	ids := make([]uuid.UUID, 0, len(list))
	maxDepth := 0
	for _, item := range list {
		if item.Depth > maxDepth {
			maxDepth = item.Depth
		}
		ids = append(ids, item.ID)
	}
	if maxDepth > c.cfg.MaxHierarchyDepth {
		return fmt.Errorf("entity.core.Delete: %w", ErrMaxHierarchyDepthExceeded(c.cfg.MaxHierarchyDepth))
	}

	if err = c.repo.Delete(ctx, ids); err != nil {
		return fmt.Errorf("entity.core.Delete: %w", err)
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
