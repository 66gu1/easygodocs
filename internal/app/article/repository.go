package article

import (
	"context"
	"errors"
	"fmt"
	"github.com/66gu1/easygodocs/internal/app/article/dto"
	"github.com/66gu1/easygodocs/internal/infrastructure/apperror"
	"github.com/66gu1/easygodocs/internal/infrastructure/appslices"
	"github.com/66gu1/easygodocs/internal/infrastructure/tx"
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

func (r *gormRepo) Get(ctx context.Context, id uuid.UUID) (dto.Article, error) {
	var model articleModel

	err := r.db.WithContext(ctx).Where("id = ?", id).First(&model).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			err = errArticleNotFound
		}
		return dto.Article{}, fmt.Errorf("gormRepo.Get: %w", err)
	}

	return model.toDTO(), nil
}

func (r *gormRepo) GetVersion(ctx context.Context, id uuid.UUID, version int) (dto.Article, error) {
	var model articleVersion

	err := r.db.WithContext(ctx).Where("article_id = ? AND version = ?", id, version).First(&model).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			err = errArticleNotFound
		}
		return dto.Article{}, fmt.Errorf("gormRepo.GetVersion: %w", err)
	}

	return model.toDTO(), nil
}

func (r *gormRepo) GetVersionsList(ctx context.Context, id uuid.UUID) ([]dto.Article, error) {
	var models []articleVersion

	err := r.db.WithContext(ctx).Where("article_id = ?", id).Order("version DESC").Find(&models).Error
	if err != nil {
		return nil, fmt.Errorf("gormRepo.GetVersionsList: %w", err)
	}

	dtos := appslices.Map(models, func(v articleVersion) dto.Article { return v.toDTO() })
	return dtos, nil
}

func (r *gormRepo) CreateDraft(ctx context.Context, tx tx.Transaction, req dto.CreateArticleReq) error {
	if tx == nil {
		return fmt.Errorf("gormRepo.CreateDraft: transaction is nil")
	}

	model := &articleModel{
		ID:        req.ID,
		Name:      req.Name,
		Content:   req.Content,
		CreatedBy: req.UserID,
		UpdatedBy: req.UserID,
	}

	err := tx.GetDB(ctx).Create(model).Error
	if err != nil {
		return fmt.Errorf("gormRepo.Create: %w", err)
	}

	return nil
}

func (r *gormRepo) Create(ctx context.Context, tx tx.Transaction, req dto.CreateArticleReq) error {
	if tx == nil {
		return fmt.Errorf("gormRepo.Create: transaction is nil")
	}
	now := time.Now().UTC()
	version := 1
	model := &articleModel{
		ID:             req.ID,
		Name:           req.Name,
		Content:        req.Content,
		CreatedBy:      req.UserID,
		UpdatedBy:      req.UserID,
		CurrentVersion: &version,
	}
	model.CreatedAt = now
	model.UpdatedAt = now

	versionModel := articleVersion{
		ArticleID: req.ID,
		Name:      req.Name,
		Content:   req.Content,
		CreatedBy: req.UserID,
		CreatedAt: now,
		Version:   version,
	}
	if req.Parent != nil {
		versionModel.ParentID = &req.Parent.ID
		versionModel.ParentType = &req.Parent.Type
	}

	if err := tx.GetDB(ctx).Create(&model).Error; err != nil {
		return fmt.Errorf("gormRepo.article.Create: %w", err)
	}
	if err := tx.GetDB(ctx).Create(&versionModel).Error; err != nil {
		return fmt.Errorf("gormRepo.articleVersion.Create: %w", err)
	}

	return nil
}

func (r *gormRepo) Update(ctx context.Context, req dto.UpdateArticleReq) error {
	var model articleModel
	now := time.Now().UTC()
	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("id = ?", req.ID).First(&model).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				err = errArticleNotFound
			}
			return fmt.Errorf("tx.First: %w", err)
		}

		version := 1
		if model.CurrentVersion != nil {
			version = *model.CurrentVersion + 1
		}

		model.Name = req.Name
		model.Content = req.Content
		model.UpdatedBy = req.UserID
		model.CurrentVersion = &version
		model.UpdatedAt = now

		if err := tx.Save(&model).Error; err != nil {
			return fmt.Errorf("tx.article.Save: %w", err)
		}

		versionModel := articleVersion{
			ArticleID: model.ID,
			Name:      model.Name,
			Content:   model.Content,
			CreatedBy: model.UpdatedBy,
			CreatedAt: now,
			Version:   version,
		}
		if req.Parent != nil {
			versionModel.ParentID = &req.Parent.ID
			versionModel.ParentType = &req.Parent.Type
		}

		if err := tx.Create(&versionModel).Error; err != nil {
			return fmt.Errorf("tx.articleVersion.Create: %w", err)
		}

		return nil
	})

	if err != nil {
		return fmt.Errorf("gormRepo.Update: %w", err)
	}

	return nil
}

func (r *gormRepo) Delete(ctx context.Context, tx tx.Transaction, ids []uuid.UUID) error {
	if tx == nil {
		return fmt.Errorf("gormRepo.Delete: transaction is nil")
	}
	err := tx.GetDB(ctx).Where("id = ANY($1)", ids).Delete(&articleModel{}).Error
	if err != nil {
		return fmt.Errorf("gormRepo.Delete: %w", err)
	}

	return nil
}
