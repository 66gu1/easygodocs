package hierarchy

import (
	"context"
	"fmt"
	"github.com/66gu1/easygodocs/internal/app/hierarchy/dto"
	user "github.com/66gu1/easygodocs/internal/app/user/dto"
	"github.com/66gu1/easygodocs/internal/infrastructure/apperror"
	"github.com/66gu1/easygodocs/internal/infrastructure/auth"
	"github.com/66gu1/easygodocs/internal/infrastructure/logger"
	"github.com/66gu1/easygodocs/internal/infrastructure/tx"
)

type HierarchyService struct {
	repo        Repository
	userService UserService
}

type Repository interface {
	GetAll(ctx context.Context) ([]dto.HierarchyWithName, error)
	GetPermitted(ctx context.Context, permissions user.Permissions) ([]dto.HierarchyWithName, error)
	Create(ctx context.Context, tx tx.Transaction, req dto.Hierarchy) error
	Update(ctx context.Context, tx tx.Transaction, req dto.Hierarchy) error
	Delete(ctx context.Context, tx tx.Transaction, req dto.DeleteRequest) ([]dto.Hierarchy, error)
	ValidateParent(ctx context.Context, hierarchy dto.Hierarchy) error
}

type UserService interface {
	GetPermissionsByUserAndRole(ctx context.Context, role auth.Role) (user.Permissions, error)
}

func NewService(repo Repository, userService UserService) *HierarchyService {
	return &HierarchyService{
		repo:        repo,
		userService: userService,
	}
}

func (s *HierarchyService) GetTree(ctx context.Context) (dto.Tree, error) {
	permissions, err := s.userService.GetPermissionsByUserAndRole(ctx, auth.RoleRead)
	if err != nil {
		logger.Error(ctx, err).Msg("HierarchyService.GetTree")
		return nil, fmt.Errorf("HierarchyService.GetTree: %w", err)
	}

	var hierarchy []dto.HierarchyWithName
	if permissions.All {
		hierarchy, err = s.repo.GetAll(ctx)
		if err != nil {
			logger.Error(ctx, err).Msg("HierarchyService.GetTree.GetAll")
			return nil, fmt.Errorf("HierarchyService.GetTree: %w", err)
		}
	} else {
		hierarchy, err = s.repo.GetPermitted(ctx, permissions)
		if err != nil {
			logger.Error(ctx, err).Msg("HierarchyService.GetTree.GetPermitted")
			return nil, fmt.Errorf("HierarchyService.GetTree: %w", err)
		}
	}

	return dto.BuildTree(ctx, hierarchy), nil
}

func (s *HierarchyService) Create(ctx context.Context, tx tx.Transaction, req dto.Hierarchy) error {
	if err := s.repo.ValidateParent(ctx, req); err != nil {
		logger.Error(ctx, err).Msg("HierarchyService.Create.ValidateParent")
		return fmt.Errorf("HierarchyService.Create: %w", err)
	}

	if err := s.repo.Create(ctx, tx, req); err != nil {
		logger.Error(ctx, err).Msg("HierarchyService.Create")
		return fmt.Errorf("HierarchyService.Create: %w", err)
	}

	return nil
}

func (s *HierarchyService) Update(ctx context.Context, tx tx.Transaction, req dto.Hierarchy) error {
	if err := s.repo.ValidateParent(ctx, req); err != nil {
		logger.Error(ctx, err).Msg("HierarchyService.Update.ValidateParent")
		return fmt.Errorf("HierarchyService.Update: %w", err)
	}
	if err := s.repo.Update(ctx, tx, req); err != nil {
		logger.Error(ctx, err).Msg("HierarchyService.Update")
		return fmt.Errorf("HierarchyService.Update: %w", err)
	}

	return nil
}

func (s *HierarchyService) Delete(ctx context.Context, tx tx.Transaction, req dto.DeleteRequest) ([]dto.Hierarchy, error) {
	entities, err := s.repo.Delete(ctx, tx, req)
	if err != nil {
		logger.Error(ctx, err).Msg("HierarchyService.Delete")
		return nil, fmt.Errorf("HierarchyService.Delete: %w", err)
	}

	return entities, nil
}

func (s *HierarchyService) CheckPermission(ctx context.Context, checkPermissionReq dto.CheckPermissionRequest) error {
	permissions, err := s.userService.GetPermissionsByUserAndRole(ctx, checkPermissionReq.Role)
	if err != nil {
		logger.Error(ctx, err).Msg("HierarchyService.CheckPermission.GetPermissionsByUserAndRole")
		return fmt.Errorf("HierarchyService.CheckPermission: %w", err)
	}
	hierarchy, err := s.repo.GetPermitted(ctx, permissions)
	if err != nil {
		logger.Error(ctx, err).Msg("HierarchyService.CheckPermission.GetPermitted")
		return fmt.Errorf("HierarchyService.CheckPermission: %w", err)
	}
	for _, h := range hierarchy {
		if h.Entity.ID == checkPermissionReq.Entity.ID && h.Entity.Type == checkPermissionReq.Entity.Type {
			return nil
		}
	}

	return &apperror.Error{
		Message:  "Permission denied",
		Code:     apperror.Forbidden,
		LogLevel: apperror.LogLevelWarn,
	}
}
