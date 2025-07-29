package article

import (
	"context"
	"fmt"
	"github.com/66gu1/easygodocs/internal/app/article/dto"
	hierarchy "github.com/66gu1/easygodocs/internal/app/hierarchy/dto"
	"github.com/66gu1/easygodocs/internal/infrastructure/apperror"
	"github.com/66gu1/easygodocs/internal/infrastructure/auth"
	"github.com/66gu1/easygodocs/internal/infrastructure/logger"
	"github.com/66gu1/easygodocs/internal/infrastructure/tx"
	"github.com/google/uuid"
)

type ArticleService struct {
	repo             Repository
	hierarchyService HierarchyService
	userService      UserService
	tx   tx.Transaction
}

type Repository interface {
	Get(ctx context.Context, id uuid.UUID) (dto.Article, error)
	Create(ctx context.Context, tx tx.Transaction, req dto.CreateArticleReq) error
	CreateDraft(ctx context.Context, tx tx.Transaction, req dto.CreateArticleReq) error
	Update(ctx context.Context, req dto.UpdateArticleReq) error
	Delete(ctx context.Context, tx tx.Transaction, ids []uuid.UUID) error
	GetVersion(ctx context.Context, id uuid.UUID, version int) (dto.Article, error)
	GetVersionsList(ctx context.Context, id uuid.UUID) ([]dto.Article, error)
}

type HierarchyService interface {
	CheckPermission(ctx context.Context, checkPermissionReq hierarchy.CheckPermissionRequest) error
}

type UserService interface {
	CheckIsAdmin(ctx context.Context) error
}

func NewService(repo Repository, hierarchy HierarchyService) *ArticleService {
	return &ArticleService{repo: repo, hierarchyService: hierarchy}
}

func (s *ArticleService) Get(ctx context.Context, id uuid.UUID) (dto.Article, error) {
	err := s.hierarchyService.CheckPermission(ctx, hierarchy.CheckPermissionRequest{
		Entity: hierarchy.Entity{
			ID:   id,
			Type: hierarchy.EntityTypeArticle,
		},
		Role: auth.RoleRead,
	})
	if err != nil {
		logger.Error(ctx, err).Str("id", id.String()).Msg("ArticleService.Get.CheckPermission")
		return dto.Article{}, fmt.Errorf("ArticleService.Get.CheckPermission: %w", err)
	}

	article, err := s.repo.Get(ctx, id)
	if err != nil {
		logger.Error(ctx, err).Str("id", id.String()).Msg("ArticleService.Get")
		return dto.Article{}, fmt.Errorf("ArticleService.Get: %w", err)
	}

	return article, nil
}

func (s *ArticleService) GetVersion(ctx context.Context, id uuid.UUID, version int) (dto.Article, error) {
	err := s.hierarchyService.CheckPermission(ctx, hierarchy.CheckPermissionRequest{
		Entity: hierarchy.Entity{
			ID:   id,
			Type: hierarchy.EntityTypeArticle,
		},
		Role: auth.RoleRead,
	})
	if err != nil {
		logger.Error(ctx, err).Str("id", id.String()).Int("version", version).Msg("ArticleService.GetVersion.CheckPermission")
		return dto.Article{}, fmt.Errorf("ArticleService.GetVersion.CheckPermission: %w", err)
	}

	article, err := s.repo.GetVersion(ctx, id, version)
	if err != nil {
		logger.Error(ctx, err).Str("id", id.String()).Int("version", version).Msg("ArticleService.GetVersion")
		return dto.Article{}, fmt.Errorf("ArticleService.GetVersion: %w", err)
	}

	return article, nil
}

func (s *ArticleService) GetVersionsList(ctx context.Context, id uuid.UUID) ([]dto.Article, error) {
	err := s.hierarchyService.CheckPermission(ctx, hierarchy.CheckPermissionRequest{
		Entity: hierarchy.Entity{
			ID:   id,
			Type: hierarchy.EntityTypeArticle,
		},
		Role: auth.RoleRead,
	})
	if err != nil {
		logger.Error(ctx, err).Str("id", id.String()).Msg("ArticleService.GetVersionsList.CheckPermission")
		return nil, fmt.Errorf("ArticleService.GetVersionsList.CheckPermission: %w", err)
	}

	articles, err := s.repo.GetVersionsList(ctx, id)
	if err != nil {
		logger.Error(ctx, err).Str("id", id.String()).Msg("ArticleService.GetVersionsList")
		return nil, fmt.Errorf("ArticleService.GetVersionsList: %w", err)
	}

	return articles, nil
}

func (s *ArticleService) Create(ctx context.Context, req dto.CreateArticleReq) (uuid.UUID, error) {
	if req.Parent == nil {
		err := s.userService.CheckIsAdmin(ctx)
		if err != nil {
			logger.Error(ctx, err).Msg("ArticleService.Create.CheckIsAdmin")
			return uuid.Nil, fmt.Errorf("ArticleService.Create: %w", err)
		}
	} else {
		err := s.hierarchyService.CheckPermission(ctx, hierarchy.CheckPermissionRequest{
			Entity: hierarchy.Entity{
				ID:   req.Parent.ID,
				Type: req.Parent.Type,
			},
			Role: auth.RoleWrite,
		})
		if err != nil {
			logger.Error(ctx, err).Interface("create_article_request", req).Str("id", req.ID.String()).
				Str("user_id", req.UserID.String()).Msg("ArticleService.Create.CheckPermission")
			return uuid.Nil, fmt.Errorf("ArticleService.Create.CheckPermission: %w", err)
		}
	}


	}
	id, err := uuid.NewV7()
	if err != nil {
		logger.Error(ctx, err).Msg("ArticleService.Create.uuid.NewV7")
		return uuid.Nil, fmt.Errorf("ArticleService.Create.uuidV7: %w", err)
	}
	req.ID = id
	// todo add userID to request

	err = s.validateParent(ctx, req.ParentType, req.ParentID, id)
	if err != nil {
		logger.Error(ctx, err).Interface("create_article_request", req).Str("id", req.ID.String()).
			Str("user_id", req.UserID.String()).Msg("ArticleService.Create.validateParent")
		return uuid.Nil, fmt.Errorf("ArticleService.Create: %w", err)
	}
	err = s.tx.Transaction(func(tx repo.Tx) error {
		if err = s.repo.Create(ctx, req); err != nil {
			logger.Error(ctx, err).Interface("create_article_request", req).Str("id", req.ID.String()).
				Str("user_id", req.UserID.String()).Msg("ArticleService.Create")
			return fmt.Errorf("ArticleService.Create: %w", err)
		}

		return nil
	})

	return id, nil
}

func (s *ArticleService) CreateDraft(ctx context.Context, req dto.CreateArticleReq) (uuid.UUID, error) {
	id, err := uuid.NewV7()
	if err != nil {
		logger.Error(ctx, err).Msg("ArticleService.CreateDraft.uuid.NewV7")
		return uuid.Nil, fmt.Errorf("ArticleService.CreateDraft.uuidV7: %w", err)
	}
	req.ID = id
	//todo add userID to request

	err = s.validateParent(ctx, req.ParentType, req.ParentID, id)
	if err != nil {
		logger.Error(ctx, err).Interface("create_article_request", req).Str("id", req.ID.String()).
			Str("user_id", req.UserID.String()).Msg("ArticleService.CreateDraft.validateParent")
		return uuid.Nil, fmt.Errorf("ArticleService.CreateDraft: %w", err)
	}

	if err = s.repo.CreateDraft(ctx, req); err != nil {
		logger.Error(ctx, err).Interface("create_article_request", req).Str("id", req.ID.String()).
			Str("user_id", req.UserID.String()).Msg("ArticleService.CreateDraft")
		return uuid.Nil, fmt.Errorf("ArticleService.CreateDraft: %w", err)
	}

	return id, nil
}

func (s *ArticleService) Update(ctx context.Context, req dto.UpdateArticleReq) error {
	//todo add userID to request
	err := s.validateParent(ctx, req.ParentType, req.ParentID, req.ID)
	if err != nil {
		logger.Error(ctx, err).Interface("update_article_request", req).
			Str("user_id", req.UserID.String()).Msg("ArticleService.Update.validateParent")
		return fmt.Errorf("ArticleService.Update: %w", err)
	}

	err = s.repo.Update(ctx, req)
	if err != nil {
		logger.Error(ctx, err).Interface("update_article_request", req).Msg("ArticleService.Update")
		return fmt.Errorf("ArticleService.Update: %w", err)
	}

	return nil
}

func (s *ArticleService) Delete(ctx context.Context, ids []uuid.UUID) error {
	err := s.repo.Delete(ctx, ids)
	if err != nil {
		logger.Error(ctx, err).Interface("ids", ids).Msg("ArticleService.Delete")
		return fmt.Errorf("ArticleService.Delete: %w", err)
	}

	return nil
}

func (s *ArticleService) GetPermittedArticleNodes(ctx context.Context) ([]dto.ArticleNode, error) {
	//todo add permission check

	articles, err := s.repo.GetAllArticleNodes(ctx)
	if err != nil {
		logger.Error(ctx, err).Msg("ArticleService.GetPermitted")
		return nil, fmt.Errorf("ArticleService.GetPermitted: %w", err)
	}

	return articles, nil
}

func (s *ArticleService) validateParent(ctx context.Context, pType dto.parentType, pID uuid.UUID, id uuid.UUID) error {
	switch pType {
	case dto.ParentTypeDepartment:
		exists, err := s.departmentService.Exists(ctx, pID)
		if err != nil {
			return fmt.Errorf("ArticleService.validateParent: %w", err)
		}

		if !exists {
			return &apperror.Error{
				Message:  "Parent department does not exist",
				Code:     apperror.NotFound,
				LogLevel: apperror.LogLevelWarn,
			}
		}
	case dto.ParentTypeArticle:
		err := s.repo.ValidateParent(ctx, id, pID)
		if err != nil {
			return fmt.Errorf("ArticleService.validateParent: %w", err)
		}

	default:
		return fmt.Errorf("ArticleService.validateParent: unknown parent type %s", pType)
	}

	return nil
}
