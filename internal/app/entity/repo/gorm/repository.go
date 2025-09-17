package gorm

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/66gu1/easygodocs/internal/app/entity"
	"github.com/google/uuid"
	"github.com/samber/lo"
	"gorm.io/gorm"
)

// buildVisibilityFilter if userID is nil, show all entities, otherwise show only published entities and drafts created by the user.
func buildVisibilityFilter(userID *uuid.UUID) (string, []any) {
	if userID == nil {
		return "TRUE", nil
	}
	return "(current_version IS NOT NULL OR updated_by = ?)", []any{*userID}
}

type gormRepo struct {
	db *gorm.DB
}

func NewRepository(db *gorm.DB) (*gormRepo, error) {
	if db == nil {
		return nil, errors.New("db is nil")
	}
	return &gormRepo{db: db}, nil
}

func (r *gormRepo) Get(ctx context.Context, id uuid.UUID) (entity.Entity, error) {
	var model entityModel

	err := r.db.WithContext(ctx).Where("id = $1", id).First(&model).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			err = entity.ErrEntityNotFound()
		}
		return entity.Entity{}, fmt.Errorf("gormRepo.Get: %w", err)
	}

	return model.toDTO(), nil
}

func (r *gormRepo) GetListItem(ctx context.Context, id uuid.UUID) (entity.ListItem, error) {
	var model entityListItemModel

	err := r.db.WithContext(ctx).Where("id = $1", id).First(&model).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			err = entity.ErrEntityNotFound()
		}
		return entity.ListItem{}, fmt.Errorf("gormRepo.GetListItem: %w", err)
	}

	return model.toDTO(), nil
}

func (r *gormRepo) GetAll(ctx context.Context) ([]entity.ListItem, error) {
	var models []entityListItemModel

	err := r.db.WithContext(ctx).Find(&models).Error
	if err != nil {
		return nil, fmt.Errorf("gormRepo.GetAll: %w", err)
	}

	return lo.Map(models, func(m entityListItemModel, _ int) entity.ListItem { return m.toDTO() }), nil
}

// GetHierarchy if userID is nil, show all entities, otherwise show only published entities and drafts created by the user.
func (r *gormRepo) GetHierarchy(ctx context.Context, ids []uuid.UUID, maxDepth int, userID *uuid.UUID, hType entity.HierarchyType) ([]entity.ListItem, error) {
	if len(ids) == 0 {
		return []entity.ListItem{}, nil
	}
	var models []entityListItemModel

	recursiveQuery, args := r.getRecursiveQuery(hType, maxDepth, ids, userID)
	childrenResult := " SELECT * FROM children "
	parentsResult := " SELECT * FROM parents "
	switch hType {
	case entity.HierarchyTypeChildrenOnly:
		recursiveQuery += childrenResult
	case entity.HierarchyTypeParentsOnly:
		recursiveQuery += parentsResult
	case entity.HierarchyTypeChildrenAndParents:
		recursiveQuery += childrenResult + " UNION " + parentsResult
	default:
		return nil, fmt.Errorf("gormRepo.GetHierarchy: %w", fmt.Errorf("invalid hierarchy type: %v", hType))
	}

	err := r.db.WithContext(ctx).Raw(recursiveQuery, args...).Scan(&models).Error
	if err != nil {
		return nil, fmt.Errorf("gormRepo.GetHierarchy: %w", err)
	}

	return lo.Map(models, func(m entityListItemModel, _ int) entity.ListItem { return m.toDTO() }), nil
}

func (r *gormRepo) GetVersion(ctx context.Context, id uuid.UUID, version int) (entity.Entity, error) {
	var model versionModel

	err := r.db.WithContext(ctx).Where("entity_id = $1 AND version = $2", id, version).First(&model).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			err = entity.ErrEntityNotFound()
		}
		return entity.Entity{}, fmt.Errorf("gormRepo.GetVersion: %w", err)
	}

	return model.toDTO(), nil
}

func (r *gormRepo) GetVersionsList(ctx context.Context, id uuid.UUID) ([]entity.Entity, error) {
	var models []versionModel

	err := r.db.WithContext(ctx).Where("entity_id = $1", id).Order("version DESC").Find(&models).Error
	if err != nil {
		return nil, fmt.Errorf("gormRepo.GetVersionsList: %w", err)
	}

	return lo.Map(models, func(m versionModel, _ int) entity.Entity { return m.toDTO() }), nil
}

func (r *gormRepo) CreateDraft(ctx context.Context, req entity.CreateEntityReq, id uuid.UUID) error {
	model := &entityModel{
		ID:        id,
		Type:      req.Type,
		Name:      req.Name,
		Content:   req.Content,
		ParentID:  req.ParentID,
		CreatedBy: req.UserID,
		UpdatedBy: req.UserID,
	}

	err := r.db.WithContext(ctx).Create(model).Error
	if err != nil {
		return fmt.Errorf("gormRepo.CreateDraft: %w", err)
	}

	return nil
}

func (r *gormRepo) Create(ctx context.Context, req entity.CreateEntityReq, id uuid.UUID, createdAt time.Time) error {
	const sqlCTE = `
WITH ins AS (
  INSERT INTO entities (id, type, name, content, parent_id, created_by, updated_by, current_version, created_at, updated_at)
  VALUES ($1,$2,$3,$4,$5,$6,$6,1,$7,$7)
)
INSERT INTO entity_versions (entity_id, name, content, parent_id, created_by, created_at, version)
VALUES ($1, $3, $4, $5, $6, $7, 1)
`

	res := r.db.WithContext(ctx).
		Exec(sqlCTE,
			id,
			req.Type,
			req.Name,
			req.Content,
			req.ParentID,
			req.UserID,
			createdAt,
		)

	if res.Error != nil {
		return fmt.Errorf("entity.create: %w", res.Error)
	}

	return nil
}

func (r *gormRepo) UpdateDraft(ctx context.Context, req entity.UpdateEntityReq) error {
	updates := map[string]interface{}{
		"name":            req.Name,
		"content":         req.Content,
		"parent_id":       req.ParentID,
		"updated_by":      req.UserID,
		"current_version": gorm.Expr("NULL"),
	}
	result := r.db.WithContext(ctx).Model(&entityModel{}).Where("id = ?", req.ID).Updates(&updates)
	if result.Error != nil {
		return fmt.Errorf("gormRepo.UpdateDraft: %w", result.Error)
	}
	if result.RowsAffected == 0 {
		return fmt.Errorf("gormRepo.UpdateDraft: %w", entity.ErrEntityNotFound())
	}
	return nil
}

func (r *gormRepo) Update(ctx context.Context, req entity.UpdateEntityReq, updatedAt time.Time) error {
	const sqlCTE = `
WITH bumped AS (
  UPDATE entities
  SET
    name            = $1,
    content         = $2,
    parent_id       = $3,
    updated_by      = $4,
    updated_at      = $5,
    current_version = COALESCE((
      SELECT MAX(version)
      FROM entity_versions
      WHERE entity_id = $6
    ), 0) + 1
  WHERE id = $6
  RETURNING id, current_version
)
INSERT INTO entity_versions (
  entity_id, name, content, parent_id,
  created_by, created_at, version
)
SELECT
  id, $1, $2, $3,
  $4,     $5,       current_version
FROM bumped;
`

	res := r.db.
		WithContext(ctx).
		Exec(sqlCTE,
			req.Name,
			req.Content,
			req.ParentID,
			req.UserID,
			updatedAt,
			req.ID,
		)
	if res.Error != nil {
		return fmt.Errorf("entity.update: %w", res.Error)
	}
	if res.RowsAffected == 0 {
		return fmt.Errorf("entity.update: %w", entity.ErrEntityNotFound())
	}

	return nil
}

func (r *gormRepo) Delete(ctx context.Context, ids []uuid.UUID) error {
	resp := r.db.WithContext(ctx).Model(&entityModel{}).Where("id IN ?", ids).Delete(&entityModel{})
	if resp.Error != nil {
		return fmt.Errorf("gormRepo.Delete: %w", resp.Error)
	}
	if resp.RowsAffected == 0 {
		return fmt.Errorf("gormRepo.Delete: %w", entity.ErrEntityNotFound())
	}

	return nil
}

func (r *gormRepo) getRecursiveQuery(hType entity.HierarchyType, maxDepth int, ids []uuid.UUID, userID *uuid.UUID) (string, []any) {
	vFilter, vArgs := buildVisibilityFilter(userID)
	args := make([]any, 0, 3+len(vArgs)*3)

	args = append(args, ids)
	args = append(args, vArgs...)
	base := fmt.Sprintf(`
WITH RECURSIVE
    base AS (
        SELECT id, type, parent_id, name, 1 as depth
        FROM entities 
        WHERE id IN (?) AND deleted_at ISNULL AND %s
    )
`, vFilter)

	childrenQuery := fmt.Sprintf(`,
    children AS (
        SELECT *
        FROM base

        UNION ALL

        SELECT e.id, e.type, e.parent_id, e.name, c.depth + 1 as depth
        FROM children c
        JOIN entities e ON c.id = e.parent_id AND e.deleted_at ISNULL  AND %s
		WHERE c.depth < ?
    )
`, vFilter)

	parentsQuery := fmt.Sprintf(`,
    parents AS (
        SELECT *
        FROM base

        UNION ALL

        SELECT e.id, e.type, e.parent_id, e.name, p.depth + 1 as depth
        FROM parents p
        JOIN entities e ON p.parent_id = e.id AND e.deleted_at ISNULL AND %s
		WHERE p.depth < ?
    )
`, vFilter)

	args = append(args, vArgs...)
	args = append(args, maxDepth)
	switch hType {
	case entity.HierarchyTypeChildrenOnly:
		return base + childrenQuery, args
	case entity.HierarchyTypeParentsOnly:
		return base + parentsQuery, args
	case entity.HierarchyTypeChildrenAndParents:
		args = append(args, vArgs...)
		args = append(args, maxDepth)
		return base + childrenQuery + parentsQuery, args
	default:
		return "", nil
	}
}
