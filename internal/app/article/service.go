package article

import (
	"context"
	"fmt"
	"github.com/66gu1/easygodocs/internal/infrastructure/apperror"
	"github.com/66gu1/easygodocs/internal/infrastructure/logger"
	"github.com/google/uuid"
)

type ArticleService struct {
	repo              Repository
	departmentService DepartmentService
}

type Repository interface {
	Get(ctx context.Context, id uuid.UUID) (Article, error)
	Create(ctx context.Context, req CreateArticleReq) error
	CreateDraft(ctx context.Context, req CreateArticleReq) error
	Update(ctx context.Context, req UpdateArticleReq) error
	Delete(ctx context.Context, id uuid.UUID) error
	GetAllArticleNodes(ctx context.Context) ([]ArticleNode, error)
	GetPermittedArticleNodes(ctx context.Context, permitted []uuid.UUID) ([]ArticleNode, error)
	ValidateParent(ctx context.Context, id uuid.UUID, parentID uuid.UUID) error
	GetVersion(ctx context.Context, id uuid.UUID, version int) (Article, error)
	GetVersionsList(ctx context.Context, id uuid.UUID) ([]Article, error)
}

type DepartmentService interface {
	Exists(ctx context.Context, id uuid.UUID) (bool, error)
}

func NewService(repo Repository, departmentService DepartmentService) *ArticleService {
	return &ArticleService{repo: repo, departmentService: departmentService}
}

func (s *ArticleService) Get(ctx context.Context, id uuid.UUID) (Article, error) {
	article, err := s.repo.Get(ctx, id)
	if err != nil {
		logger.Error(ctx, err).Str("id", id.String()).Msg("ArticleService.Get")
		return Article{}, fmt.Errorf("ArticleService.Get: %w", err)
	}

	return article, nil
}

func (s *ArticleService) GetVersion(ctx context.Context, id uuid.UUID, version int) (Article, error) {
	article, err := s.repo.GetVersion(ctx, id, version)
	if err != nil {
		logger.Error(ctx, err).Str("id", id.String()).Int("version", version).Msg("ArticleService.GetVersion")
		return Article{}, fmt.Errorf("ArticleService.GetVersion: %w", err)
	}

	return article, nil
}

func (s *ArticleService) GetVersionsList(ctx context.Context, id uuid.UUID) ([]Article, error) {
	articles, err := s.repo.GetVersionsList(ctx, id)
	if err != nil {
		logger.Error(ctx, err).Str("id", id.String()).Msg("ArticleService.GetVersionsList")
		return nil, fmt.Errorf("ArticleService.GetVersionsList: %w", err)
	}

	return articles, nil
}

func (s *ArticleService) Create(ctx context.Context, req CreateArticleReq) (uuid.UUID, error) {
	id, err := uuid.NewV7()
	if err != nil {
		logger.Error(ctx, err).Msg("ArticleService.Create.uuid.NewV7")
		return uuid.Nil, fmt.Errorf("ArticleService.Create.uuidV7: %w", err)
	}
	req.id = id
	// todo add userID to request

	err = s.validateParent(ctx, req.ParentType, req.ParentID, id)
	if err != nil {
		logger.Error(ctx, err).Interface("create_article_request", req).Str("id", req.id.String()).
			Str("user_id", req.userID.String()).Msg("ArticleService.Create.validateParent")
		return uuid.Nil, fmt.Errorf("ArticleService.Create: %w", err)
	}

	if err = s.repo.Create(ctx, req); err != nil {
		logger.Error(ctx, err).Interface("create_article_request", req).Str("id", req.id.String()).
			Str("user_id", req.userID.String()).Msg("ArticleService.Create")
		return uuid.Nil, fmt.Errorf("ArticleService.Create: %w", err)
	}

	return id, nil
}

func (s *ArticleService) CreateDraft(ctx context.Context, req CreateArticleReq) (uuid.UUID, error) {
	id, err := uuid.NewV7()
	if err != nil {
		logger.Error(ctx, err).Msg("ArticleService.CreateDraft.uuid.NewV7")
		return uuid.Nil, fmt.Errorf("ArticleService.CreateDraft.uuidV7: %w", err)
	}
	req.id = id
	//todo add userID to request

	err = s.validateParent(ctx, req.ParentType, req.ParentID, id)
	if err != nil {
		logger.Error(ctx, err).Interface("create_article_request", req).Str("id", req.id.String()).
			Str("user_id", req.userID.String()).Msg("ArticleService.CreateDraft.validateParent")
		return uuid.Nil, fmt.Errorf("ArticleService.CreateDraft: %w", err)
	}

	if err = s.repo.CreateDraft(ctx, req); err != nil {
		logger.Error(ctx, err).Interface("create_article_request", req).Str("id", req.id.String()).
			Str("user_id", req.userID.String()).Msg("ArticleService.CreateDraft")
		return uuid.Nil, fmt.Errorf("ArticleService.CreateDraft: %w", err)
	}

	return id, nil
}

func (s *ArticleService) Update(ctx context.Context, req UpdateArticleReq) error {
	//todo add userID to request
	err := s.validateParent(ctx, req.ParentType, req.ParentID, req.ID)
	if err != nil {
		logger.Error(ctx, err).Interface("update_article_request", req).
			Str("user_id", req.userID.String()).Msg("ArticleService.Update.validateParent")
		return fmt.Errorf("ArticleService.Update: %w", err)
	}

	err = s.repo.Update(ctx, req)
	if err != nil {
		logger.Error(ctx, err).Interface("update_article_request", req).Msg("ArticleService.Update")
		return fmt.Errorf("ArticleService.Update: %w", err)
	}

	return nil
}

func (s *ArticleService) Delete(ctx context.Context, id uuid.UUID) error {
	err := s.repo.Delete(ctx, id)
	if err != nil {
		logger.Error(ctx, err).Str("id", id.String()).Msg("ArticleService.Delete")
		return fmt.Errorf("ArticleService.Delete: %w", err)
	}

	return nil
}

func (s *ArticleService) GetPermittedArticleNodes(ctx context.Context) ([]ArticleNode, error) {
	//todo add permission check

	articles, err := s.repo.GetAllArticleNodes(ctx)
	if err != nil {
		logger.Error(ctx, err).Msg("ArticleService.GetPermitted")
		return nil, fmt.Errorf("ArticleService.GetPermitted: %w", err)
	}

	return articles, nil
}

func (s *ArticleService) validateParent(ctx context.Context, pType parentType, pID uuid.UUID, id uuid.UUID) error {
	switch pType {
	case ParentTypeDepartment:
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
	case ParentTypeArticle:
		err := s.repo.ValidateParent(ctx, id, pID)
		if err != nil {
			return fmt.Errorf("ArticleService.validateParent: %w", err)
		}

	default:
		return fmt.Errorf("ArticleService.validateParent: unknown parent type %s", pType)
	}

	return nil
}
