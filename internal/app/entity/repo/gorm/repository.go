package gorm

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/66gu1/easygodocs/internal/app/entity"
	"github.com/66gu1/easygodocs/internal/infrastructure/db"
	"github.com/google/uuid"
	"github.com/samber/lo"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// hierarchyLockKey identifies the advisory lock that serializes changes of the entity tree structure.
const hierarchyLockKey = "entities_hierarchy"

// buildVisibilityFilter if userID is nil, show all entities, otherwise show only published entities
// and the never published ones created by the user. The filter is applied to the "e" alias.
func buildVisibilityFilter(userID *uuid.UUID) (string, []any) {
	if userID == nil {
		return "TRUE", nil
	}
	return "(e.current_version IS NOT NULL OR e.created_by = ?)", []any{*userID}
}

type Repository struct {
	db *gorm.DB
}

func NewRepository(db *gorm.DB) (*Repository, error) {
	if db == nil {
		return nil, errors.New("db is nil")
	}
	return &Repository{db: db}, nil
}

// conn returns the transaction from ctx if there is one.
func (r *Repository) conn(ctx context.Context) *gorm.DB {
	return db.Conn(ctx, r.db)
}

// LockHierarchy takes a transaction-level advisory lock that serializes structural changes
// (creating children, moving, deleting), so concurrent requests can't create cycles
// or exceed the depth limit. It must be called inside a transaction.
func (r *Repository) LockHierarchy(ctx context.Context) error {
	if err := r.conn(ctx).Exec("SELECT pg_advisory_xact_lock(hashtext(?))", hierarchyLockKey).Error; err != nil {
		return fmt.Errorf("entity.repo.LockHierarchy: %w", err)
	}

	return nil
}

// GetListItemForUpdate locks the entity row until the end of the transaction.
func (r *Repository) GetListItemForUpdate(ctx context.Context, id uuid.UUID) (entity.ListItem, error) {
	var model entityListItemModel

	err := r.conn(ctx).Clauses(clause.Locking{Strength: clause.LockingStrengthUpdate}).
		Where("id = ?", id).First(&model).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			err = entity.ErrEntityNotFound()
		}
		return entity.ListItem{}, fmt.Errorf("entity.repo.GetListItemForUpdate: %w", err)
	}

	return model.toDTO(), nil
}

func (r *Repository) Get(ctx context.Context, id uuid.UUID) (entity.Entity, error) {
	var model entityModel

	err := r.conn(ctx).Where("id = $1", id).First(&model).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			err = entity.ErrEntityNotFound()
		}
		return entity.Entity{}, fmt.Errorf("entity.repo.Get: %w", err)
	}

	return model.toDTO(), nil
}

func (r *Repository) GetListItem(ctx context.Context, id uuid.UUID) (entity.ListItem, error) {
	var model entityListItemModel

	err := r.conn(ctx).Where("id = $1", id).First(&model).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			err = entity.ErrEntityNotFound()
		}
		return entity.ListItem{}, fmt.Errorf("entity.repo.GetListItem: %w", err)
	}

	return model.toDTO(), nil
}

func (r *Repository) GetAll(ctx context.Context) ([]entity.ListItem, error) {
	var models []entityListItemModel

	err := r.conn(ctx).Find(&models).Error
	if err != nil {
		return nil, fmt.Errorf("entity.repo.GetAll: %w", err)
	}

	return lo.Map(models, func(m entityListItemModel, _ int) entity.ListItem { return m.toDTO() }), nil
}

// GetHierarchy if userID is nil, show all entities, otherwise show only published entities
// and the never published ones created by the user.
func (r *Repository) GetHierarchy(ctx context.Context, ids []uuid.UUID, maxDepth int, userID *uuid.UUID, hType entity.HierarchyType) ([]entity.ListItem, error) {
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
		return nil, fmt.Errorf("entity.repo.GetHierarchy: %w", fmt.Errorf("invalid hierarchy type: %v", hType))
	}

	err := r.conn(ctx).Raw(recursiveQuery, args...).Scan(&models).Error
	if err != nil {
		return nil, fmt.Errorf("entity.repo.GetHierarchy: %w", err)
	}

	return lo.Map(models, func(m entityListItemModel, _ int) entity.ListItem { return m.toDTO() }), nil
}

func (r *Repository) GetVersion(ctx context.Context, id uuid.UUID, version int) (entity.Entity, error) {
	var model versionModel

	err := r.conn(ctx).Where("entity_id = $1 AND version = $2", id, version).First(&model).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			err = entity.ErrEntityNotFound()
		}
		return entity.Entity{}, fmt.Errorf("entity.repo.GetVersion: %w", err)
	}

	return model.toDTO(), nil
}

func (r *Repository) GetVersionsList(ctx context.Context, id uuid.UUID) ([]entity.Entity, error) {
	var models []versionModel

	err := r.conn(ctx).Where("entity_id = $1", id).Order("version DESC").Find(&models).Error
	if err != nil {
		return nil, fmt.Errorf("entity.repo.GetVersionsList: %w", err)
	}

	return lo.Map(models, func(m versionModel, _ int) entity.Entity { return m.toDTO() }), nil
}

func (r *Repository) CreateDraft(ctx context.Context, req entity.CreateEntityReq, id uuid.UUID) error {
	model := &entityModel{
		ID:        id,
		Type:      req.Type,
		Name:      req.Name,
		Content:   req.Content,
		ParentID:  req.ParentID,
		CreatedBy: req.UserID,
		UpdatedBy: req.UserID,
	}

	err := r.conn(ctx).Create(model).Error
	if err != nil {
		return fmt.Errorf("entity.repo.CreateDraft: %w", err)
	}

	return nil
}

func (r *Repository) Create(ctx context.Context, req entity.CreateEntityReq, id uuid.UUID, createdAt time.Time) error {
	const sqlCTE = `
WITH ins AS (
  INSERT INTO entities (id, type, name, content, parent_id, created_by, updated_by, current_version, created_at, updated_at)
  VALUES ($1,$2,$3,$4,$5,$6,$6,1,$7,$7)
)
INSERT INTO entity_versions (entity_id, name, content, parent_id, created_by, created_at, version)
VALUES ($1, $3, $4, $5, $6, $7, 1)
`

	res := r.conn(ctx).
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
		return fmt.Errorf("entity.repo.Create: %w", res.Error)
	}

	return nil
}

func (r *Repository) Update(ctx context.Context, req entity.UpdateEntityReq, updatedAt time.Time) error {
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

	res := r.conn(ctx).
		Exec(sqlCTE,
			req.Name,
			req.Content,
			req.ParentID,
			req.UserID,
			updatedAt,
			req.ID,
		)
	if res.Error != nil {
		return fmt.Errorf("entity.repo.Update: %w", res.Error)
	}
	if res.RowsAffected == 0 {
		return fmt.Errorf("entity.repo.Update: %w", entity.ErrEntityNotFound())
	}

	return nil
}

// Delete soft-deletes the entities and deletes their drafts.
func (r *Repository) Delete(ctx context.Context, ids []uuid.UUID) error {
	resp := r.conn(ctx).Model(&entityModel{}).Where("id IN ?", ids).Delete(&entityModel{})
	if resp.Error != nil {
		return fmt.Errorf("entity.repo.Delete: %w", resp.Error)
	}
	if resp.RowsAffected == 0 {
		return fmt.Errorf("entity.repo.Delete: %w", entity.ErrEntityNotFound())
	}
	// Soft delete doesn't trigger ON DELETE CASCADE.
	if err := r.conn(ctx).Where("entity_id IN ?", ids).Delete(&draftModel{}).Error; err != nil {
		return fmt.Errorf("entity.repo.Delete: delete drafts: %w", err)
	}

	return nil
}

// SaveDraft creates the user's draft or overwrites its content.
// baseVersion is stored only when the draft is created.
func (r *Repository) SaveDraft(ctx context.Context, req entity.SaveDraftReq, baseVersion *int, updatedAt time.Time) error {
	const query = `
INSERT INTO entity_drafts (entity_id, user_id, name, content, base_version, updated_at)
VALUES ($1, $2, $3, $4, $5, $6)
ON CONFLICT (entity_id, user_id) DO UPDATE
SET name       = EXCLUDED.name,
    content    = EXCLUDED.content,
    updated_at = EXCLUDED.updated_at
`
	err := r.conn(ctx).Exec(query, req.EntityID, req.UserID, req.Name, req.Content, baseVersion, updatedAt).Error
	if err != nil {
		return fmt.Errorf("entity.repo.SaveDraft: %w", err)
	}

	return nil
}

func (r *Repository) GetDraft(ctx context.Context, entityID, userID uuid.UUID) (entity.Draft, error) {
	var model draftModel

	err := r.conn(ctx).Where("entity_id = ? AND user_id = ?", entityID, userID).First(&model).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			err = entity.ErrDraftNotFound()
		}
		return entity.Draft{}, fmt.Errorf("entity.repo.GetDraft: %w", err)
	}

	return model.toDTO(), nil
}

func (r *Repository) DeleteDraft(ctx context.Context, entityID, userID uuid.UUID) error {
	resp := r.conn(ctx).Where("entity_id = ? AND user_id = ?", entityID, userID).Delete(&draftModel{})
	if resp.Error != nil {
		return fmt.Errorf("entity.repo.DeleteDraft: %w", resp.Error)
	}
	if resp.RowsAffected == 0 {
		return fmt.Errorf("entity.repo.DeleteDraft: %w", entity.ErrDraftNotFound())
	}

	return nil
}

func (r *Repository) getRecursiveQuery(hType entity.HierarchyType, maxDepth int, ids []uuid.UUID, userID *uuid.UUID) (string, []any) {
	vFilter, vArgs := buildVisibilityFilter(userID)
	args := make([]any, 0, 3+len(vArgs)*3)

	args = append(args, ids)
	args = append(args, vArgs...)
	base := fmt.Sprintf(`
WITH RECURSIVE
    base AS (
        SELECT e.id, e.type, e.parent_id, e.name, e.current_version, 1 as depth
        FROM entities e
        WHERE e.id IN (?) AND e.deleted_at ISNULL AND %s
    )
`, vFilter)

	childrenQuery := fmt.Sprintf(`,
    children AS (
        SELECT *
        FROM base

        UNION ALL

        SELECT e.id, e.type, e.parent_id, e.name, e.current_version, c.depth + 1 as depth
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

        SELECT e.id, e.type, e.parent_id, e.name, e.current_version, p.depth + 1 as depth
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
