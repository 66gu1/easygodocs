package dto

import (
	hierarchy "github.com/66gu1/easygodocs/internal/app/hierarchy/dto"
	"github.com/66gu1/easygodocs/internal/infrastructure/apperror"
	"github.com/google/uuid"
	"time"
)

var (
	ErrNameIsEmpty = &apperror.Error{
		Message:  "article name is required",
		Code:     apperror.BadRequest,
		LogLevel: apperror.LogLevelWarn,
	}
	ErrContentIsEmpty = &apperror.Error{
		Message:  "article content is required",
		Code:     apperror.BadRequest,
		LogLevel: apperror.LogLevelWarn,
	}
	ErrParentIDIsEmpty = &apperror.Error{
		Message:  "parent ID cannot be an empty UUID.",
		Code:     apperror.BadRequest,
		LogLevel: apperror.LogLevelWarn,
	}
)

type Article struct {
	ID        uuid.UUID `json:"id"`
	Name      string    `json:"name"`
	Content   string    `json:"content"`
	CreatedBy uuid.UUID `json:"created_by"`
	UpdatedBy uuid.UUID `json:"updated_by"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
	Version   *int      `json:"version"`
}

type CreateArticleReq struct {
	Name    string            `json:"name"`
	Content string            `json:"content"`
	Parent  *hierarchy.Entity `json:"parent"`

	// filled by the service layer
	ID     uuid.UUID `json:"id"`
	UserID uuid.UUID `json:"user_id"`
}

func (req CreateArticleReq) Validate() error {
	if req.Name == "" {
		return ErrNameIsEmpty
	}
	if req.Content == "" {
		return ErrContentIsEmpty
	}

	return nil
}

type UpdateArticleReq struct {
	ID      uuid.UUID         `json:"id"`
	Name    string            `json:"name"`
	Content string            `json:"content"`
	Parent  *hierarchy.Entity `json:"parent"`

	// filled by the service layer
	UserID uuid.UUID `json:"user_id"`
}

func (req UpdateArticleReq) Validate() error {
	if req.ID == uuid.Nil {
		return &apperror.Error{
			Message:  "article ID cannot be an empty UUID",
			Code:     apperror.BadRequest,
			LogLevel: apperror.LogLevelWarn,
		}
	}
	if req.Name == "" {
		return ErrNameIsEmpty
	}
	if req.Content == "" {
		return ErrContentIsEmpty
	}

	return nil
}
