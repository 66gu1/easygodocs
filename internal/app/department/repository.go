package department

import (
	"context"
	"fmt"
	"github.com/66gu1/easygodocs/internal/infrastructure/apperror"
	"github.com/google/uuid"
	"gorm.io/gorm"
	"time"
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

func (r *gormRepo) Create(ctx context.Context, req CreateDepartmentReq, id uuid.UUID) error {
	model := &department{
		ID:   id,
		Name: req.Name,
	}
	if req.ParentID != nil {
		model.ParentID = req.ParentID
	}

	err := r.db.WithContext(ctx).Create(model).Error
	if err != nil {
		return fmt.Errorf("department.gormRepo.Create: %w", err)
	}

	return nil
}

func (r *gormRepo) Update(ctx context.Context, req UpdateDepartmentReq) error {
	model := &department{
		ID:   req.ID,
		Name: req.Name,
	}
	if req.ParentID != nil {
		model.ParentID = req.ParentID
	}

	err := r.db.WithContext(ctx).Model(model).Updates(model.getMap()).Error
	if err != nil {
		return fmt.Errorf("department.gormRepo.Update: %w", err)
	}

	return nil
}

func (r *gormRepo) List(ctx context.Context) ([]Department, error) {
	var list []department
	err := r.db.WithContext(ctx).Find(&list).Error
	if err != nil {
		return nil, fmt.Errorf("department.gormRepo.List: %w", err)
	}

	return toDTOs(list), nil
}

func (r *gormRepo) Delete(ctx context.Context, id uuid.UUID) error {
	query := `
WITH RECURSIVE subdepartments AS (
  SELECT id FROM departments WHERE id = $1
  UNION ALL
  SELECT d.id
  FROM departments d
  INNER JOIN subdepartments sd ON d.parent_id = sd.id
)
UPDATE departments SET deleted_at = $2
WHERE id IN (SELECT id FROM subdepartments);
`

	err := r.db.WithContext(ctx).Exec(query, id, time.Now().UTC()).Error
	if err != nil {
		return fmt.Errorf("department.gormRepo.Delete: %w", err)
	}

	return nil
}

func (r *gormRepo) ValidateParent(ctx context.Context, id uuid.UUID, parentID uuid.UUID) error {
	type parentValidationStatus string
	const (
		statusNotFound parentValidationStatus = "not_found"
		statusCycle    parentValidationStatus = "cycle"
		statusOK       parentValidationStatus = "ok"
	)

	if id == uuid.Nil {
		return fmt.Errorf("id cannot be empty")
	}

	var status parentValidationStatus
	query := `
    WITH RECURSIVE dept_tree AS (
    SELECT * FROM departments WHERE id = $1 AND deleted_at IS NULL
    UNION ALL
    SELECT d.* FROM departments d
    INNER JOIN dept_tree dt ON dt.parent_id = d.id
    WHERE d.deleted_at IS NULL
)
SELECT
    CASE
        WHEN NOT EXISTS (SELECT 1 FROM dept_tree) THEN 'not_found'
        WHEN EXISTS (SELECT 1 FROM dept_tree WHERE id = $2) THEN 'cycle'
        ELSE 'ok'
    END AS status;
`

	err := r.db.WithContext(ctx).Raw(query, parentID, id).Scan(&status).Error
	if err != nil {
		return fmt.Errorf("department.gormRepo.ValidateParent:: %w", err)
	}

	switch status {
	case statusNotFound:
		return parentNodFoundErr
	case statusCycle:
		return parentCycleErr
	case statusOK:
		return nil
	default:
		return fmt.Errorf("department.gormRepo.ValidateParent: unexpected status %s", status)
	}
}

//// Purge actually removes it from the database
//func (r *gormRepo) purge(ctx context.Context, id uuid.UUID) error {
//	err := r.db.WithContext(ctx).Unscoped().Delete(&department{}, id).Error
//	if err != nil {
//		return fmt.Errorf("department.gormRepo.purge: %w", err)
//	}
//
//	return nil
//}
