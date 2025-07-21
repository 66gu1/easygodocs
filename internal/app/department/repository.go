package department

import (
	"context"
	"fmt"
	"github.com/66gu1/easygodocs/internal/infrastructure/db"
	"github.com/google/uuid"
	"gorm.io/gorm"
	"time"
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
		return fmt.Errorf("gormRepo.Create: %w", err)
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
		return fmt.Errorf("gormRepo.Update: %w", err)
	}

	return nil
}

func (r *gormRepo) GetAll(ctx context.Context) ([]Department, error) {
	var models []department
	err := r.db.WithContext(ctx).Find(&models).Error
	if err != nil {
		return nil, fmt.Errorf("gormRepo.GetAll: %w", err)
	}

	return toDTOs(models), nil
}
func (r *gormRepo) GetPermitted(ctx context.Context, permitted []uuid.UUID) ([]Department, error) {
	if len(permitted) == 0 {
		return nil, nil
	}

	var models []department

	err := r.db.WithContext(ctx).Raw(db.GetRecursiveFetcherQuery(db.DepartmentTableName), permitted).Scan(&models).Error
	if err != nil {
		return nil, fmt.Errorf("gormRepo.GetList.Exec: %w", err)
	}

	return toDTOs(models), nil
}

func (r *gormRepo) GetList(ctx context.Context, ids []uuid.UUID) ([]Department, error) {
	var models []department
	err := r.db.WithContext(ctx).Find(&models, "id = ANY(?)", ids).Error
	if err != nil {
		return nil, fmt.Errorf("gormRepo.Get: %w", err)
	}

	return toDTOs(models), nil
}

func (r *gormRepo) Delete(ctx context.Context, id uuid.UUID) error {
	err := r.db.WithContext(ctx).Exec(db.GetRecursiveDeleteQuery(db.DepartmentTableName), id, time.Now().UTC()).Error
	if err != nil {
		return fmt.Errorf("gormRepo.Delete: %w", err)
	}

	return nil
}

func (r *gormRepo) ValidateParent(ctx context.Context, id uuid.UUID, parentID uuid.UUID) error {
	if id == uuid.Nil {
		return fmt.Errorf("id cannot be empty")
	}

	query, err := db.GetRecursiveValidateParentQuery(db.DepartmentTableName)
	if err != nil {
		return fmt.Errorf("gormRepo.ValidateParent: %w", err)
	}

	var status string
	err = r.db.WithContext(ctx).Raw(query, parentID, id).
		Scan(&status).Error
	if err != nil {
		return fmt.Errorf("gormRepo.ValidateParent:: %w", err)
	}

	err = db.GetValidateParentErrorByStatus(status)
	if err != nil {
		return fmt.Errorf("gormRepo.ValidateParent: %w", err)
	}

	return nil
}

//// Purge actually removes it from the database
//func (r *gormRepo) purge(ctx context.Context, id uuid.UUID) error {
//	err := r.db.WithContext(ctx).Unscoped().Delete(&department{}, id).Error
//	if err != nil {
//		return fmt.Errorf("gormRepo.purge: %w", err)
//	}
//
//	return nil
//}
