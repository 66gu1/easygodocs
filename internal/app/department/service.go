package department

import (
	"context"
	"fmt"
	"github.com/66gu1/easygodocs/internal/infrastructure/logger"
	"github.com/google/uuid"
	"github.com/rs/zerolog/log"
)

const (
	departmentIDField = "department_id"
)

type DepartmentService struct {
	repo Repository
}

//go:generate minimock -i github.com/66gu1/easygodocs/internal/app/department.Repository -o ./mock -s _mock.go
type Repository interface {
	Create(ctx context.Context, req CreateDepartmentReq, id uuid.UUID) error
	Update(ctx context.Context, req UpdateDepartmentReq) error
	List(ctx context.Context) ([]Department, error)
	Delete(ctx context.Context, id uuid.UUID) error
	ValidateParent(ctx context.Context, id uuid.UUID, parentID uuid.UUID) error
}

func NewService(repo Repository) *DepartmentService {
	return &DepartmentService{repo: repo}
}

func (s *DepartmentService) Create(ctx context.Context, req CreateDepartmentReq) (uuid.UUID, error) {
	id, err := uuid.NewV7()
	if err != nil {
		logger.Error(ctx, err).Msg("department.Service.Create.uuid.NewV7")
		return uuid.Nil, fmt.Errorf("department.Service.Create.uuidV7: %w", err)
	}

	if req.ParentID != nil {
		err = s.repo.ValidateParent(ctx, id, *req.ParentID)
		if err != nil {
			logger.Error(ctx, err).Str(departmentIDField, id.String()).Msg("department.Service.Update.ValidateParent")
			return uuid.Nil, fmt.Errorf("department.Service.Update.ValidateParent: %w", err)
		}
	}

	err = s.repo.Create(ctx, req, id)
	if err != nil {
		logger.Error(ctx, err).Interface("create_department_request", req).Str("id", id.String()).
			Msg("department.Service.Create")
		return uuid.Nil, fmt.Errorf("department.Service.Create: %w", err)
	}

	return id, nil
}

func (s *DepartmentService) Update(ctx context.Context, req UpdateDepartmentReq) error {
	if req.ParentID != nil {
		err := s.repo.ValidateParent(ctx, req.ID, *req.ParentID)
		if err != nil {
			logger.Error(ctx, err).Str(departmentIDField, req.ID.String()).Msg("department.Service.Update.ValidateParent")
			return fmt.Errorf("department.Service.Update.ValidateParent: %w", err)
		}
	}
	err := s.repo.Update(ctx, req)
	if err != nil {
		logger.Error(ctx, err).Interface("update_department_request", req).Msg("department.Service.Update")
		return fmt.Errorf("department.Service.Update: %w", err)
	}

	return nil
}

func (s *DepartmentService) GetDepartmentTree(ctx context.Context) (Tree, error) {
	deps, err := s.repo.List(ctx)
	if err != nil {
		log.Error().Msg("department.Service.List")
		return nil, fmt.Errorf("department.Service.List: %w", err)
	}

	t := Tree{}
	t.build(deps)

	return t, nil
}

func (s *DepartmentService) Delete(ctx context.Context, id uuid.UUID) error {
	err := s.repo.Delete(ctx, id)
	if err != nil {
		logger.Error(ctx, err).Str(departmentIDField, id.String()).Msg("department.Service.Delete")
		return fmt.Errorf("department.Service.Delete: %w", err)
	}

	return nil
}
