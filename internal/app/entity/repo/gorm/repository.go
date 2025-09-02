package gorm

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/66gu1/easygodocs/internal/app/entity"
	"github.com/66gu1/easygodocs/internal/infrastructure/apperr"
	"github.com/google/uuid"
	"github.com/samber/lo"
	"gorm.io/gorm"
)

func ErrEntityNotFound() error {
	return apperr.New("Entity not found", entity.CodeNotFound, apperr.ClassNotFound, apperr.LogLevelWarn)
}

func ErrParentCycle() error {
	return apperr.New("Parent cycle detected", entity.CodeParentCycle, apperr.ClassBadRequest, apperr.LogLevelWarn).
		WithViolation(apperr.Violation{
			Field: entity.FieldParentID, Rule: apperr.RuleCycle,
		})
}

func ErrMaxHierarchyDepthExceeded(maxDepth int) error {
	return apperr.New("Maximum hierarchy depth exceeded", entity.CodeMaxDepthExceeded, apperr.ClassBadRequest, apperr.LogLevelWarn).
		WithViolation(apperr.Violation{
			Field: entity.FieldParentID, Rule: apperr.RuleMaxHierarchy,
			Params: map[string]any{"max_depth": maxDepth},
		})
}

type gormRepo struct {
	db  *gorm.DB
	cfg Config
}

type Config struct {
	MaxHierarchyDepth int `mapstructure:"max_hierarchy_depth" json:"max_hierarchy_depth"`

	MaxNameLength int `mapstructure:"max_name_length" json:"max_name_length"`
}

func NewRepository(db *gorm.DB, cfg Config) *gormRepo {
	if cfg.MaxHierarchyDepth <= 0 {
		panic("Config.MaxHierarchyDepth must be > 0")
	}
	if cfg.MaxNameLength <= 0 {
		panic("Config.MaxNameLength must be > 0")
	}
	return &gormRepo{db: db, cfg: cfg}
}

func (r *gormRepo) Get(ctx context.Context, id uuid.UUID) (entity.Entity, error) {
	var model entityModel

	err := r.db.WithContext(ctx).Where("id = $1", id).First(&model).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			err = ErrEntityNotFound()
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
			err = ErrEntityNotFound()
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

func (r *gormRepo) GetPermittedHierarchy(ctx context.Context, permissions []uuid.UUID) ([]entity.ListItem, error) {
	var models []entityListItemModel
	if len(permissions) == 0 {
		return []entity.ListItem{}, nil
	}

	recursiveQuery, err := r.getRecursiveQuery(childrenAndParents, r.cfg.MaxHierarchyDepth)
	if err != nil {
		return nil, fmt.Errorf("gormRepo.GetPermittedHierarchy: %w", err)
	}

	err = r.db.WithContext(ctx).Raw(recursiveQuery+`
	/* noinspection SqlNoDataSourceInspection */
    SELECT *
    FROM children
    UNION
    SELECT *
    FROM parents;
`,
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
			err = ErrEntityNotFound()
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
INSERT INTO entities (
    id, type, name, content,
    parent_id, created_by, updated_by,
    current_version, created_at, updated_at
  ) VALUES (
    $1, $2, $3, $4,
    $5, $6, $6,
    1,   $7, $7
  );

INSERT INTO entity_versions (
  entity_id, name, content, parent_id,
  created_by, created_at, version 
) VALUES (
  $1, $3, $4, $5,
  $6, $7, 1
);
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
	result := r.db.WithContext(ctx).Where("id = $1", req.ID).Updates(&updates)
	if result.Error != nil {
		return fmt.Errorf("gormRepo.UpdateDraft: %w", result.Error)
	}
	if result.RowsAffected == 0 {
		return fmt.Errorf("gormRepo.UpdateDraft: %w", ErrEntityNotFound())
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
		return fmt.Errorf("entity.update: %w", ErrEntityNotFound())
	}

	return nil
}

func (r *gormRepo) Delete(ctx context.Context, id uuid.UUID, deletedAt time.Time) error {
	recursiveQuery, err := r.getRecursiveQuery(onlyChildren, r.cfg.MaxHierarchyDepth)
	if err != nil {
		return fmt.Errorf("gormRepo.Delete: %w", err)
	}

	err = r.db.WithContext(ctx).Raw(recursiveQuery+
		`
	/* noinspection SqlNoDataSourceInspection */
	UPDATE entities SET deleted_at = $2
	WHERE id IN (
    	SELECT id FROM children
	);
`, []uuid.UUID{id}, deletedAt).Error
	if err != nil {
		return fmt.Errorf("gormRepo.Delete: %w", err)
	}

	return nil
}

func (r *gormRepo) ValidateChangedParent(ctx context.Context, id, parentID uuid.UUID) error {
	type validationResult struct {
		Depth   int
		IsCycle bool
	}
	var result validationResult

	recursiveQuery, err := r.getRecursiveQuery(onlyChildren, r.cfg.MaxHierarchyDepth)
	if err != nil {
		return fmt.Errorf("gormRepo.ValidateChangedParent: %w", err)
	}

	err = r.db.WithContext(ctx).Raw(recursiveQuery+`
	/* noinspection SqlNoDataSourceInspection */
SELECT
    COALESCE((SELECT MAX(depth) FROM children), 0) AS depth,
    EXISTS (SELECT 1 FROM children WHERE id = $2) AS is_cycle;
	`, []uuid.UUID{id}, parentID).Scan(&result).Error
	if err != nil {
		return fmt.Errorf("gormRepo.ValidateChangedParent: %w", err)
	}
	if result.IsCycle {
		return fmt.Errorf("gormRepo.ValidateChangedParent: %w", ErrParentCycle())
	}

	recursiveQuery, err = r.getRecursiveQuery(onlyParents, r.cfg.MaxHierarchyDepth)
	if err != nil {
		return fmt.Errorf("gormRepo.ValidateChangedParent: %w", err)
	}

	var parentDepth int
	err = r.db.WithContext(ctx).Raw(recursiveQuery+`
	/* noinspection SqlNoDataSourceInspection */
SELECT
    COALESCE((SELECT MAX(depth) FROM parents), 0) AS depth,
	`, []uuid.UUID{parentID}).Scan(&parentDepth).Error
	if err != nil {
		return fmt.Errorf("gormRepo.ValidateChangedParent: %w", err)
	}
	if result.Depth+parentDepth > r.cfg.MaxHierarchyDepth {
		return fmt.Errorf("gormRepo.ValidateChangedParent: %w", ErrMaxHierarchyDepthExceeded(r.cfg.MaxHierarchyDepth))
	}

	return nil
}

func (r *gormRepo) CheckParentDepthLimit(ctx context.Context, parentID uuid.UUID) error {
	recursiveQuery, err := r.getRecursiveQuery(onlyParents, r.cfg.MaxHierarchyDepth)
	if err != nil {
		return fmt.Errorf("gormRepo.CheckParentDepthLimit: %w", err)
	}
	var maxDepthExceeded bool

	err = r.db.WithContext(ctx).Raw(recursiveQuery+`
	/* noinspection SqlNoDataSourceInspection */
	SELECT COALESCE((MAX(depth) + 1 > $2), FALSE) AS max_depth_exceeded FROM parents;
	`, []uuid.UUID{parentID}, r.cfg.MaxHierarchyDepth).Scan(&maxDepthExceeded).Error
	if err != nil {
		return fmt.Errorf("gormRepo.CheckParentDepthLimit: %w", err)
	}

	if maxDepthExceeded {
		err = ErrMaxHierarchyDepthExceeded(r.cfg.MaxHierarchyDepth)
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

func (r *gormRepo) getRecursiveQuery(rqType recursiveQueryType, maxDepth int) (string, error) {
	maxDepth++ // Traverse up to maxDepth + 1 to detect if actual depth exceeds the allowed maxDepth.
	base := `
WITH RECURSIVE
    base AS (
        SELECT id, type, parent_id, name, 0 as depth
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
		return base + childrenQuery, nil
	case onlyParents:
		return base + parentsQuery, nil
	case childrenAndParents:
		return base + childrenQuery + parentsQuery, nil
	default:
		return "", fmt.Errorf("invalid recursive query type: %s", rqType)

	}
}

func (r *gormRepo) GetMaxNameLength() int {
	return r.cfg.MaxNameLength
}
