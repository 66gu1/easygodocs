package article

import (
	"context"
	"errors"
	"fmt"
	"github.com/66gu1/easygodocs/internal/infrastructure/apperror"
	"github.com/66gu1/easygodocs/internal/infrastructure/db"
	"github.com/google/uuid"
	"gorm.io/gorm"
	"time"
)

var (
	errArticleNotFound = &apperror.Error{
		Message:  "article not found",
		Code:     apperror.NotFound,
		LogLevel: apperror.LogLevelWarn,
	}
)

type gormRepo struct {
	db *gorm.DB
}

func NewRepository(db *gorm.DB) *gormRepo {
	return &gormRepo{db: db}
}

func (r *gormRepo) Get(ctx context.Context, id uuid.UUID) (Article, error) {
	var model articleModel

	err := r.db.WithContext(ctx).Where("id = ?", id).First(&model).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			err = errArticleNotFound
		}
		return Article{}, fmt.Errorf("gormRepo.Get: %w", err)
	}

	dto, err := model.toDTO()
	if err != nil {
		return Article{}, fmt.Errorf("gormRepo.Get: %w", err)
	}

	return dto, nil
}

func (r *gormRepo) GetVersion(ctx context.Context, id uuid.UUID, version int) (Article, error) {
	var model articleVersion

	err := r.db.WithContext(ctx).Where("article_id = ? AND version = ?", id, version).First(&model).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			err = errArticleNotFound
		}
		return Article{}, fmt.Errorf("gormRepo.GetVersion: %w", err)
	}

	dto, err := model.toDTO()
	if err != nil {
		return Article{}, fmt.Errorf("gormRepo.GetVersion: %w", err)
	}

	return dto, nil
}

func (r *gormRepo) GetVersionsList(ctx context.Context, id uuid.UUID) ([]Article, error) {
	var models []articleVersion

	err := r.db.WithContext(ctx).Where("article_id = ?", id).Order("version DESC").Find(&models).Error
	if err != nil {
		return nil, fmt.Errorf("gormRepo.GetVersionsList: %w", err)
	}

	dtos := make([]Article, len(models))
	for i, model := range models {
		dto, err := model.toDTO()
		if err != nil {
			return nil, fmt.Errorf("gormRepo.GetVersionsList: %w", err)
		}
		dtos[i] = dto
	}

	return dtos, nil
}

func (r *gormRepo) CreateDraft(ctx context.Context, req CreateArticleReq) error {
	pt, err := parentTypeToModel(req.ParentType)
	if err != nil {
		return fmt.Errorf("gormRepo.CreateDraft: %w", err)
	}
	model := &articleModel{
		ID:         req.id,
		Name:       req.Name,
		Content:    req.Content,
		ParentType: pt,
		ParentID:   req.ParentID,
		CreatedBy:  req.userID,
		UpdatedBy:  req.userID,
	}

	err = r.db.WithContext(ctx).Create(model).Error
	if err != nil {
		return fmt.Errorf("gormRepo.Create: %w", err)
	}

	return nil
}

func (r *gormRepo) Create(ctx context.Context, req CreateArticleReq) error {
	pt, err := parentTypeToModel(req.ParentType)
	if err != nil {
		return fmt.Errorf("gormRepo.Create: %w", err)
	}
	now := time.Now().UTC()
	version := 1
	model := &articleModel{
		ID:             req.id,
		Name:           req.Name,
		Content:        req.Content,
		ParentType:     pt,
		ParentID:       req.ParentID,
		CreatedBy:      req.userID,
		UpdatedBy:      req.userID,
		CurrentVersion: &version,
	}
	model.CreatedAt = now
	model.UpdatedAt = now

	versionModel := articleVersion{
		ArticleID:  req.id,
		Name:       req.Name,
		Content:    req.Content,
		ParentType: pt,
		ParentID:   req.ParentID,
		CreatedBy:  req.userID,
		CreatedAt:  now,
		Version:    version,
	}

	err = r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(&model).Error; err != nil {
			return fmt.Errorf("transaction.article.Create: %w", err)
		}
		if err := tx.Create(&versionModel).Error; err != nil {
			return fmt.Errorf("transaction.articleVersion.Create: %w", err)
		}
		return nil
	})

	if err != nil {
		return fmt.Errorf("gormRepo.Create: %w", err)
	}

	return nil
}

func (r *gormRepo) Update(ctx context.Context, req UpdateArticleReq) error {
	pt, err := parentTypeToModel(req.ParentType)
	if err != nil {
		return fmt.Errorf("gormRepo.Update: %w", err)
	}

	var model articleModel
	now := time.Now().UTC()
	err = r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("id = ?", req.ID).First(&model).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				err = errArticleNotFound
			}
			return fmt.Errorf("transaction.First: %w", err)
		}

		version := 1
		if model.CurrentVersion != nil {
			version = *model.CurrentVersion + 1
		}

		model.Name = req.Name
		model.Content = req.Content
		model.ParentType = pt
		model.ParentID = req.ParentID
		model.UpdatedBy = req.userID
		model.CurrentVersion = &version
		model.UpdatedAt = now

		if err := tx.Save(&model).Error; err != nil {
			return fmt.Errorf("transaction.article.Save: %w", err)
		}

		versionModel := articleVersion{
			ArticleID:  model.ID,
			Name:       model.Name,
			Content:    model.Content,
			ParentType: model.ParentType,
			ParentID:   model.ParentID,
			CreatedBy:  model.UpdatedBy,
			CreatedAt:  now,
			Version:    version,
		}

		if err := tx.Create(&versionModel).Error; err != nil {
			return fmt.Errorf("transaction.articleVersion.Create: %w", err)
		}

		return nil
	})

	if err != nil {
		return fmt.Errorf("gormRepo.Update: %w", err)
	}

	return nil
}

func (r *gormRepo) Delete(ctx context.Context, id uuid.UUID) error {
	err := r.db.WithContext(ctx).Exec(db.GetRecursiveDeleteQuery(db.ArticleTableName), id, time.Now().UTC()).Error
	if err != nil {
		return fmt.Errorf("gormRepo.Delete: %w", err)
	}

	return nil
}

func (r *gormRepo) GetAllArticleNodes(ctx context.Context) ([]ArticleNode, error) {
	var models []articleNode
	err := r.db.WithContext(ctx).Find(&models).Error
	if err != nil {
		return nil, fmt.Errorf("gormRepo.GetAllArticleNodes: %w", err)
	}

	dtos, err := toArticleNodes(models)
	if err != nil {
		return nil, fmt.Errorf("gormRepo.GetAllArticleNodes: %w", err)
	}

	return dtos, nil
}

func (r *gormRepo) GetPermittedArticleNodes(ctx context.Context, permitted []uuid.UUID) ([]ArticleNode, error) {
	if len(permitted) == 0 {
		return nil, nil
	}

	var models []articleNode

	err := r.db.WithContext(ctx).Raw(db.GetRecursiveFetcherQuery(db.ArticleTableName), permitted).Scan(&models).Error
	if err != nil {
		return nil, fmt.Errorf("gormRepo.GetPermittedArticleNodes.Exec: %w", err)
	}

	dtos, err := toArticleNodes(models)
	if err != nil {
		return nil, fmt.Errorf("gormRepo.GetPermittedArticleNodes: %w", err)
	}

	return dtos, nil
}

func (r *gormRepo) ValidateParent(ctx context.Context, id uuid.UUID, parentID uuid.UUID) error {
	if id == uuid.Nil {
		return fmt.Errorf("id cannot be empty")
	}

	query, err := db.GetRecursiveValidateParentQuery(db.ArticleTableName)
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
