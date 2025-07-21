package article

import (
	"github.com/66gu1/easygodocs/internal/infrastructure/apperror"
	"github.com/google/uuid"
	"time"
)

type parentType string

const (
	ParentTypeArticle    parentType = "article"
	ParentTypeDepartment parentType = "department"
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

func (pt parentType) Validate() error {
	switch pt {
	case ParentTypeArticle, ParentTypeDepartment:
		return nil
	default:
		return &apperror.Error{
			Message:  "invalid parent type",
			Code:     apperror.BadRequest,
			LogLevel: apperror.LogLevelWarn,
		}
	}
}

type Article struct {
	ID         uuid.UUID  `json:"id"`
	Name       string     `json:"name"`
	Content    string     `json:"content"`
	ParentType parentType `json:"parent_type"`
	ParentID   uuid.UUID  `json:"parent_id"`
	CreatedBy  uuid.UUID  `json:"created_by"`
	UpdatedBy  uuid.UUID  `json:"updated_by"`
	CreatedAt  time.Time  `json:"created_at"`
	UpdatedAt  time.Time  `json:"updated_at"`
	Version    *int       `json:"version"`
}

type ArticleNode struct {
	ID         uuid.UUID  `json:"id"`
	Name       string     `json:"name"`
	ParentType parentType `json:"parent_type"`
	ParentID   uuid.UUID  `json:"parent_id"`
}

type CreateArticleReq struct {
	Name       string     `json:"name"`
	Content    string     `json:"content"`
	ParentType parentType `json:"parent_type"`
	ParentID   uuid.UUID  `json:"parent_id"`
	id         uuid.UUID
	userID     uuid.UUID
}

func (req CreateArticleReq) Validate() error {
	if req.Name == "" {
		return ErrNameIsEmpty
	}
	if req.Content == "" {
		return ErrContentIsEmpty
	}
	if req.ParentID == uuid.Nil {
		return ErrParentIDIsEmpty
	}
	err := req.ParentType.Validate()
	if err != nil {
		return err
	}
	return nil
}

type UpdateArticleReq struct {
	ID         uuid.UUID  `json:"id"`
	Name       string     `json:"name"`
	Content    string     `json:"content"`
	ParentType parentType `json:"parent_type"`
	ParentID   uuid.UUID  `json:"parent_id"`
	userID     uuid.UUID
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
	if req.ParentID == uuid.Nil {
		return ErrParentIDIsEmpty
	}
	err := req.ParentType.Validate()
	if err != nil {
		return err
	}

	return nil
}
