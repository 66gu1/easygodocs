package hierarchy

import (
	"github.com/66gu1/easygodocs/internal/app/hierarchy/dto"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

type model struct {
	EntityID   uuid.UUID
	EntityType dto.EntityType
	ParentID   *uuid.UUID
	ParentType *dto.EntityType
	DeletedAt  gorm.DeletedAt
}

func (m *model) toDTO() dto.Hierarchy {
	h := dto.Hierarchy{
		Entity: dto.Entity{
			Type: m.EntityType,
			ID:   m.EntityID,
		},
	}
	if m.ParentID != nil && m.ParentType != nil {
		h.Parent = &dto.Entity{
			Type: *m.ParentType,
			ID:   *m.ParentID,
		}
	}

	return h
}

func fromDTO(dto dto.Hierarchy) model {
	m := model{
		EntityID:   dto.Entity.ID,
		EntityType: dto.Entity.Type,
	}
	if dto.Parent != nil {
		m.ParentID = &dto.Parent.ID
		m.ParentType = &dto.Parent.Type
	}

	return m
}

type validationStatus string

const (
	statusOK       validationStatus = "ok"
	statusNotFound validationStatus = "not_found"
	statusCycle    validationStatus = "cycle"
)

type modelWithName struct {
	EntityType dto.EntityType  `json:"entity_type"`
	EntityID   uuid.UUID       `json:"entity_id"`
	ParentType *dto.EntityType `json:"parent_type"`
	ParentID   *uuid.UUID      `json:"parent_id"`
	EntityName string          `json:"entity_name"`
}

func (n *modelWithName) toDTO() dto.HierarchyWithName {
	h := dto.HierarchyWithName{
		Entity: dto.Entity{
			Type: n.EntityType,
			ID:   n.EntityID,
		},
		EntityName: n.EntityName,
	}

	if n.ParentID != nil && n.ParentType != nil {
		h.Parent = &dto.Entity{
			Type: *n.ParentType,
			ID:   *n.ParentID,
		}
	}

	return h
}
