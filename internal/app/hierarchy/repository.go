package hierarchy

import (
	"context"
	"fmt"
	"github.com/66gu1/easygodocs/internal/app/hierarchy/dto"
	user "github.com/66gu1/easygodocs/internal/app/user/dto"
	"github.com/66gu1/easygodocs/internal/infrastructure/apperror"
	"github.com/66gu1/easygodocs/internal/infrastructure/appslices"
	"github.com/66gu1/easygodocs/internal/infrastructure/tx"
	"gorm.io/gorm"
)

var (
	parentNodFoundErr = &apperror.Error{
		Message:  "parent department not found",
		Code:     apperror.BadRequest,
		LogLevel: apperror.LogLevelWarn,
	}
	parentCycleErr = &apperror.Error{
		Message:  "cannot assign department as its own descendant (cycle detected)",
		Code:     apperror.BadRequest,
		LogLevel: apperror.LogLevelWarn,
	}
)

type gormRepo struct {
	db *gorm.DB
}

func NewRepository(db *gorm.DB) *gormRepo {
	return &gormRepo{db: db}
}

func (r *gormRepo) GetAll(ctx context.Context) ([]dto.HierarchyWithName, error) {
	var models []modelWithName

	err := r.db.WithContext(ctx).Raw(`
	SELECT h.entity_id, h.entity_type, h.parent_id, h.parent_type,
	       CASE WHEN h.entity_type = $1 THEN ar.name ELSE d.name END AS entity_name
             FROM entity_hierarchy h
	LEFT JOIN articles ar ON h.entity_type = $1 AND h.entity_id = ar.id
         LEFT JOIN departments d ON h.entity_type = $2 AND h.entity_id = d.id
         WHERE h.deleted_at ISNULL`).Scan(&models).Error
	if err != nil {
		return nil, err
	}

	return appslices.Map(models, func(n modelWithName) dto.HierarchyWithName { return n.toDTO() }), nil
}

func (r *gormRepo) GetPermitted(ctx context.Context, permissions user.Permissions) ([]dto.HierarchyWithName, error) {
	var models []modelWithName

	err := r.db.WithContext(ctx).Raw(`
WITH RECURSIVE
    base AS (SELECT h.entity_id, h.entity_type, h.parent_id, h.parent_type
             FROM entity_hierarchy h
             WHERE ((h.entity_type = $1 AND h.entity_id = ANY($3))
                OR (h.entity_type = $2 AND h.entity_id = ANY($4))) AND h.deleted_at ISNULL),
    children AS (SELECT h.entity_id, h.entity_type, h.parent_id, h.parent_type
                 FROM base b
                          JOIN entity_hierarchy h ON b.entity_id = h.parent_id AND b.entity_type = h.parent_type
                 UNION ALL
                 SELECT h.entity_id, h.entity_type, h.parent_id, h.parent_type
                 FROM children c
                          JOIN entity_hierarchy h ON c.entity_id = h.parent_id AND c.entity_type = h.parent_type),
    parents AS (SELECT h.entity_id, h.entity_type, h.parent_id, h.parent_type
                FROM base b
                         JOIN entity_hierarchy h ON b.parent_id = h.entity_id AND b.parent_type = h.entity_type
                UNION ALL
                SELECT h.entity_id, h.entity_type, h.parent_id, h.parent_type
                FROM parents p
                         JOIN entity_hierarchy h ON p.parent_id = h.entity_id AND p.parent_type = h.entity_type)
SELECT DISTINCT a.entity_id,
                a.entity_type,
                a.parent_id,
                a.parent_type,
                CASE WHEN a.entity_type = $1 THEN ar.name ELSE d.name END AS entity_name
FROM (SELECT *
      FROM base
      UNION ALL
      SELECT *
      FROM children
      UNION ALL
      SELECT *
      FROM parents) as a
         LEFT JOIN articles ar ON a.entity_type = $1 AND a.entity_id = ar.id
         LEFT JOIN departments d ON a.entity_type = $2 AND a.entity_id = d.id;`,
		dto.EntityTypeArticle, dto.EntityTypeDepartment, permissions.ArticleIDs, permissions.DepartmentIDs).Scan(&models).Error
	if err != nil {
		return nil, fmt.Errorf("gormRepo.GetPermitted.Scan: %w", err)
	}

	return appslices.Map(models, func(n modelWithName) dto.HierarchyWithName { return n.toDTO() }), nil
}

func (r *gormRepo) Create(ctx context.Context, tx tx.Transaction, req dto.Hierarchy) error {
	db := tx.GetDB(ctx)
	if db == nil {
		return fmt.Errorf("gormRepo.Create: transaction db is nil")
	}
	m := fromDTO(req)
	resp := db.Create(&m)
	if resp.Error != nil {
		return fmt.Errorf("gormRepo.Create: %w", resp.Error)
	}

	return nil
}

func (r *gormRepo) Update(ctx context.Context, tx tx.Transaction, req dto.Hierarchy) error {
	db := tx.GetDB(ctx)
	if db == nil {
		return fmt.Errorf("gormRepo.Update: transaction db is nil")
	}

	m := fromDTO(req)
	err := db.Model(&m).Where("entity_id = $1 AND entity_type = $2", req.Entity.ID, req.Entity.Type).Updates(m).Error
	if err != nil {
		return fmt.Errorf("gormRepo.Update: %w", err)
	}

	return nil
}

func (r *gormRepo) Delete(ctx context.Context, tx tx.Transaction, req dto.DeleteRequest) ([]dto.Hierarchy, error) {
	db := tx.GetDB(ctx)
	if db == nil {
		return nil, fmt.Errorf("gormRepo.Delete: transaction db is nil")
	}

	var deletedEntities []dto.Hierarchy
	err := db.Raw(
		`
	WITH RECURSIVE sub AS (
		SELECT entity_type, entity_id, parent_id, parent_type
		FROM entity_hierarchy WHERE entity_type = $1 AND entity_id = $2 
	UNION ALL
	SELECT h.entity_type, h.entity_id, h.parent_id, h.parent_type
	FROM entity_hierarchy h
	INNER JOIN sub s ON h.parent_type = s.entity_type AND h.parent_id = s.entity_id
	)
UPDATE entity_hierarchy SET deleted_at = $3
WHERE (entity_type, entity_id) IN (
    SELECT entity_type, entity_id FROM sub
)
RETURNING entity_type, entity_id, parent_id, parent_type;
`, req.Entity.Type, req.Entity.ID, req.DeletedAt).Scan(&deletedEntities).Error
	if err != nil {
		return nil, fmt.Errorf("gormRepo.Delete: %w", err)
	}

	return deletedEntities, nil
}

func (r *gormRepo) ValidateParent(ctx context.Context, hierarchy dto.Hierarchy) error {
	var status string
	err := r.db.WithContext(ctx).Raw(`
		WITH RECURSIVE sub AS (
			SELECT entity_type, entity_id, parent_id, parent_type
			FROM entity_hierarchy
			WHERE entity_type = ? AND entity_id = ? AND deleted_at ISNULL

			UNION ALL

			SELECT h.entity_type, h.entity_id, h.parent_id, h.parent_type
			FROM entity_hierarchy h
			INNER JOIN sub s ON h.parent_type = s.entity_type AND h.parent_id = s.entity_id
		)
		SELECT CASE
			WHEN NOT EXISTS (SELECT 1 FROM sub) THEN 'not_found'
			WHEN EXISTS (SELECT 1 FROM sub WHERE entity_type = ? AND entity_id = ?) THEN 'cycle'
			ELSE 'ok'
		END AS status
	`, hierarchy.Entity.Type, hierarchy.Entity.ID, hierarchy.Parent.Type, hierarchy.Parent.ID).Scan(&status).Error

	if err != nil {
		return fmt.Errorf("gormRepo.ValidateParent: %w", err)
	}

	switch validationStatus(status) {
	case statusNotFound:
		return parentNodFoundErr
	case statusCycle:
		return parentCycleErr
	case statusOK:
		return nil
	default:
		return fmt.Errorf("gormRepo.ValidateParent: unexpected status %q", status)
	}
}
