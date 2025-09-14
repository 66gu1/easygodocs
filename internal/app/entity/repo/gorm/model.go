package gorm

import (
	"time"

	"github.com/66gu1/easygodocs/internal/app/entity"
	"github.com/66gu1/easygodocs/internal/infrastructure/db"
	"github.com/google/uuid"
)

type entityModel struct {
	db.Base
	ID             uuid.UUID
	Type           entity.Type
	Name           string
	Content        string
	ParentID       *uuid.UUID
	CreatedBy      uuid.UUID
	UpdatedBy      uuid.UUID
	CurrentVersion *int
}

func (m *entityModel) TableName() string {
	return "entities"
}

func (m *entityModel) toDTO() entity.Entity {
	return entity.Entity{
		ID:             m.ID,
		Type:           m.Type,
		Name:           m.Name,
		Content:        m.Content,
		ParentID:       m.ParentID,
		CreatedBy:      m.CreatedBy,
		UpdatedBy:      m.UpdatedBy,
		CurrentVersion: m.CurrentVersion,
		CreatedAt:      m.CreatedAt,
		UpdatedAt:      m.UpdatedAt,
	}
}

type versionModel struct {
	EntityID  uuid.UUID
	Name      string
	Content   string
	ParentID  *uuid.UUID
	CreatedBy uuid.UUID
	CreatedAt time.Time
	Version   int
}

func (m *versionModel) TableName() string {
	return "entity_versions"
}

func (m *versionModel) toDTO() entity.Entity {
	return entity.Entity{
		ID:             m.EntityID,
		Name:           m.Name,
		Content:        m.Content,
		ParentID:       m.ParentID,
		CreatedBy:      m.CreatedBy,
		UpdatedBy:      m.CreatedBy,
		CurrentVersion: &m.Version,
		CreatedAt:      m.CreatedAt,
		UpdatedAt:      m.CreatedAt,
	}
}

type entityListItemModel struct {
	db.Base
	ID       uuid.UUID
	Type     entity.Type
	Name     string
	ParentID *uuid.UUID
}

func (m *entityListItemModel) TableName() string {
	return "entities"
}

func (m entityListItemModel) toDTO() entity.ListItem {
	return entity.ListItem{
		ID:       m.ID,
		Type:     m.Type,
		Name:     m.Name,
		ParentID: m.ParentID,
	}
}
