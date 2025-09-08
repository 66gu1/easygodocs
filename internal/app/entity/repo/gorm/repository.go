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

type gormRepo struct {
	db  *gorm.DB
	cfg Config
}

type Config struct {
	MaxHierarchyDepth int `mapstructure:"max_hierarchy_depth" json:"max_hierarchy_depth"`
}

func NewRepository(db *gorm.DB, cfg Config) (*gormRepo, error) {
	if cfg.MaxHierarchyDepth <= 0 {
		return nil, errors.New("max_hierarchy_depth must be positive")
	}
	if db == nil {
		return nil, errors.New("db is nil")
	}
	return &gormRepo{db: db, cfg: cfg}, nil
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

func (r *gormRepo) GetPermittedHierarchy(ctx context.Context, permissions []uuid.UUID, onlyForRead bool) ([]entity.ListItem, error) {
	var models []entityListItemModel
	if len(permissions) == 0 {
		return []entity.ListItem{}, nil
	}
	query := `/* noinspection SqlNoDataSourceInspection */
    SELECT *
    FROM children `
	rqType := onlyChildren

	if onlyForRead {
		rqType = childrenAndParents
		query += `
    	UNION
    	SELECT *
    	FROM parents`
	}

	recursiveQuery := r.getRecursiveQuery(rqType, r.cfg.MaxHierarchyDepth)

	err := r.db.WithContext(ctx).Raw(recursiveQuery+query,
		permissions).Scan(&models).Error
	if err != nil {
		return nil, fmt.Errorf("gormRepo.GetPermittedHierarchy: %w", err)
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

func (r *gormRepo) Delete(ctx context.Context, id uuid.UUID, deletedAt time.Time) error {
	recursiveQuery := r.getRecursiveQuery(onlyChildren, r.cfg.MaxHierarchyDepth)

	resp := r.db.WithContext(ctx).Exec(recursiveQuery+
		`
	/* noinspection SqlNoDataSourceInspection */
	UPDATE entities SET deleted_at = $2
	WHERE id IN (
    	SELECT id FROM children
	);
`, []uuid.UUID{id}, deletedAt)
	if resp.Error != nil {
		return fmt.Errorf("gormRepo.Delete: %w", resp.Error)
	}
	if resp.RowsAffected == 0 {
		return fmt.Errorf("gormRepo.Delete: %w", entity.ErrEntityNotFound())
	}

	return nil
}

func (r *gormRepo) ValidateChangedParent(ctx context.Context, id, parentID uuid.UUID) error {
	type validationResult struct {
		Depth   int
		IsCycle bool
	}
	var result validationResult

	recursiveQuery := r.getRecursiveQuery(onlyChildren, r.cfg.MaxHierarchyDepth)

	err := r.db.WithContext(ctx).Raw(recursiveQuery+`
	/* noinspection SqlNoDataSourceInspection */
SELECT
    COALESCE((SELECT MAX(depth) FROM children), 0) AS depth,
    EXISTS (SELECT 1 FROM children WHERE id = $2) AS is_cycle;
	`, []uuid.UUID{id}, parentID).Scan(&result).Error
	if err != nil {
		return fmt.Errorf("gormRepo.ValidateChangedParent: %w", err)
	}
	if result.IsCycle {
		return fmt.Errorf("gormRepo.ValidateChangedParent: %w", entity.ErrParentCycle())
	}

	recursiveQuery = r.getRecursiveQuery(onlyParents, r.cfg.MaxHierarchyDepth)

	var parentDepth int
	err = r.db.WithContext(ctx).Raw(recursiveQuery+`
	/* noinspection SqlNoDataSourceInspection */
SELECT
    COALESCE((SELECT MAX(depth) FROM parents), 0) AS depth
	`, []uuid.UUID{parentID}).Scan(&parentDepth).Error
	if err != nil {
		return fmt.Errorf("gormRepo.ValidateChangedParent: %w", err)
	}
	if result.Depth+parentDepth > r.cfg.MaxHierarchyDepth {
		return fmt.Errorf("gormRepo.ValidateChangedParent: %w", entity.ErrMaxHierarchyDepthExceeded(r.cfg.MaxHierarchyDepth))
	}

	return nil
}

func (r *gormRepo) CheckParentDepthLimit(ctx context.Context, parentID uuid.UUID) error {
	recursiveQuery := r.getRecursiveQuery(onlyParents, r.cfg.MaxHierarchyDepth)
	var maxDepthExceeded bool

	err := r.db.WithContext(ctx).Raw(recursiveQuery+`
	/* noinspection SqlNoDataSourceInspection */
	SELECT COALESCE((MAX(depth) + 1 > $2), FALSE) AS max_depth_exceeded FROM parents;
	`, []uuid.UUID{parentID}, r.cfg.MaxHierarchyDepth).Scan(&maxDepthExceeded).Error
	if err != nil {
		return fmt.Errorf("gormRepo.CheckParentDepthLimit: %w", err)
	}

	if maxDepthExceeded {
		err = entity.ErrMaxHierarchyDepthExceeded(r.cfg.MaxHierarchyDepth)
		return fmt.Errorf("gormRepo.CheckParentDepthLimit: %w", err)
	}

	return nil
}

type recursiveQueryType string

const (
	onlyChildren       recursiveQueryType = "only_children"
	onlyParents        recursiveQueryType = "only_parents"
	childrenAndParents recursiveQueryType = "children_and_parents"
)

func (r *gormRepo) getRecursiveQuery(rqType recursiveQueryType, maxDepth int) string {
	maxDepth++ // Traverse up to maxDepth + 1 to detect if actual depth exceeds the allowed maxDepth.
	base := `
WITH RECURSIVE
    base AS (
        SELECT id, type, parent_id, name, 1 as depth
        FROM entities 
        WHERE id = ANY($1) AND deleted_at ISNULL
    )
`
	childrenQuery := fmt.Sprintf(`,
    children AS (
        SELECT *
        FROM base

        UNION ALL

        SELECT e.id, e.type, e.parent_id, e.name, c.depth + 1 as depth
        FROM children c
        JOIN entities e ON c.id = e.parent_id AND e.deleted_at ISNULL
		WHERE c.depth <= %d
    )
`, maxDepth)
	parentsQuery := fmt.Sprintf(`,
    parents AS (
        SELECT *
        FROM base

        UNION ALL

        SELECT e.id, e.type, e.parent_id, e.name, p.depth + 1 as depth
        FROM parents p
        JOIN entities e ON p.parent_id = e.id AND e.deleted_at ISNULL
		WHERE p.depth <= %d
    )
`, maxDepth)
	switch rqType {
	case onlyChildren:
		return base + childrenQuery
	case onlyParents:
		return base + parentsQuery
	case childrenAndParents:
		return base + childrenQuery + parentsQuery
	default:
		return ""
	}
}
