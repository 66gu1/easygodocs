package article

import (
	"github.com/66gu1/easygodocs/internal/app/article/dto"
	hierarchy "github.com/66gu1/easygodocs/internal/app/hierarchy/dto"
	"github.com/66gu1/easygodocs/internal/infrastructure/db"
	"github.com/google/uuid"
	"time"
)

type articleModel struct {
	db.Base
	ID             uuid.UUID `json:"id"`
	Name           string    `json:"name"`
	Content        string    `json:"content"`
	CreatedBy      uuid.UUID `json:"created_by"`
	UpdatedBy      uuid.UUID `json:"updated_by"`
	CurrentVersion *int      `json:"current_version"`
}

func fromDTO(dto dto.Article) articleModel {
	return articleModel{
		ID:             dto.ID,
		Name:           dto.Name,
		Content:        dto.Content,
		CreatedBy:      dto.CreatedBy,
		UpdatedBy:      dto.UpdatedBy,
		CurrentVersion: dto.Version,
	}
}

func (m *articleModel) toDTO() dto.Article {

	return dto.Article{
		ID:        m.ID,
		Name:      m.Name,
		Content:   m.Content,
		CreatedBy: m.CreatedBy,
		UpdatedBy: m.UpdatedBy,
		Version:   m.CurrentVersion,
		UpdatedAt: m.UpdatedAt,
		CreatedAt: m.CreatedAt,
	}
}

type articleVersion struct {
	ArticleID  uuid.UUID             `json:"article_id"`
	Name       string                `json:"name"`
	Content    string                `json:"content"`
	ParentType *hierarchy.EntityType `json:"parent_type"`
	ParentID   *uuid.UUID            `json:"parent_id"`
	CreatedBy  uuid.UUID             `json:"created_by"`
	CreatedAt  time.Time             `json:"created_at"`
	Version    int                   `json:"version"`
}

func (m *articleVersion) toDTO() dto.Article {
	return dto.Article{
		ID:        m.ArticleID,
		Name:      m.Name,
		Content:   m.Content,
		CreatedBy: m.CreatedBy,
		UpdatedBy: m.CreatedBy,
		Version:   &m.Version,
		CreatedAt: m.CreatedAt,
		UpdatedAt: m.CreatedAt,
	}
}
