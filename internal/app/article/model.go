package article

import (
	"fmt"
	"github.com/66gu1/easygodocs/internal/infrastructure/db"
	"github.com/google/uuid"
	"time"
)

type articleModel struct {
	db.Base
	ID             uuid.UUID            `json:"id"`
	Name           string               `json:"name"`
	Content        string               `json:"content"`
	ParentType     db.ArticleParentType `json:"parent_type"`
	ParentID       uuid.UUID            `json:"parent_id"`
	CreatedBy      uuid.UUID            `json:"created_by"`
	UpdatedBy      uuid.UUID            `json:"updated_by"`
	CurrentVersion *int                 `json:"current_version"`
}

func (m *articleModel) fromDTO(dto Article) error {
	pt, err := parentTypeToModel(dto.ParentType)
	if err != nil {
		return fmt.Errorf("articleModel.fromDTO: %w", err)
	}

	m.ID = dto.ID
	m.Name = dto.Name
	m.Content = dto.Content
	m.ParentType = pt
	m.ParentID = dto.ParentID
	m.CreatedBy = dto.CreatedBy
	m.UpdatedBy = dto.UpdatedBy
	m.CurrentVersion = dto.Version

	return nil
}

func (m *articleModel) toDTO() (Article, error) {
	pt, err := parentTypeFromModel(m.ParentType)
	if err != nil {
		return Article{}, fmt.Errorf("articleModel.toDTO: %w", err)
	}
	return Article{
		ID:         m.ID,
		Name:       m.Name,
		Content:    m.Content,
		ParentID:   m.ParentID,
		ParentType: pt,
		CreatedBy:  m.CreatedBy,
		UpdatedBy:  m.UpdatedBy,
		Version:    m.CurrentVersion,
		UpdatedAt:  m.UpdatedAt,
		CreatedAt:  m.CreatedAt,
	}, nil
}

type articleNode struct {
	db.Base
	ID         uuid.UUID            `json:"id"`
	Name       string               `json:"name"`
	ParentType db.ArticleParentType `json:"parent_type"`
	ParentID   uuid.UUID            `json:"parent_id"`
}

func (m *articleNode) toDTO() (ArticleNode, error) {
	pt, err := parentTypeFromModel(m.ParentType)
	if err != nil {
		return ArticleNode{}, fmt.Errorf("articleNode.toDTO: %w", err)
	}
	return ArticleNode{
		ID:         m.ID,
		Name:       m.Name,
		ParentID:   m.ParentID,
		ParentType: pt,
	}, nil
}
func toArticleNodes(models []articleNode) ([]ArticleNode, error) {
	nodes := make([]ArticleNode, len(models))
	var err error
	for i, model := range models {
		nodes[i], err = model.toDTO()
		if err != nil {
			return nil, fmt.Errorf("toArticleNodes: %w", err)
		}
	}
	return nodes, nil
}

type articleVersion struct {
	ArticleID  uuid.UUID            `json:"article_id"`
	Name       string               `json:"name"`
	Content    string               `json:"content"`
	ParentType db.ArticleParentType `json:"parent_type"`
	ParentID   uuid.UUID            `json:"parent_id"`
	CreatedBy  uuid.UUID            `json:"created_by"`
	CreatedAt  time.Time            `json:"created_at"`
	Version    int                  `json:"version"`
}

func (m *articleVersion) toDTO() (Article, error) {
	pt, err := parentTypeFromModel(m.ParentType)
	if err != nil {
		return Article{}, fmt.Errorf("articleVersion.toDTO: %w", err)
	}

	return Article{
		ID:         m.ArticleID,
		Name:       m.Name,
		Content:    m.Content,
		ParentID:   m.ParentID,
		ParentType: pt,
		CreatedBy:  m.CreatedBy,
		UpdatedBy:  m.CreatedBy,
		Version:    &m.Version,
		CreatedAt:  m.CreatedAt,
		UpdatedAt:  m.CreatedAt,
	}, nil
}

func parentTypeFromModel(t db.ArticleParentType) (parentType, error) {
	switch t {
	case db.ArticleParentTypeArticle:
		return ParentTypeArticle, nil
	case db.ArticleParentTypeDepartment:
		return ParentTypeDepartment, nil
	default:
		return "", fmt.Errorf("parentTypeFromModel: unknown parent type")
	}
}

func parentTypeToModel(t parentType) (db.ArticleParentType, error) {
	switch t {
	case ParentTypeArticle:
		return db.ArticleParentTypeArticle, nil
	case ParentTypeDepartment:
		return db.ArticleParentTypeDepartment, nil
	default:
		return "", fmt.Errorf("parentTypeToModel: unknown parent type")
	}
}
