package department

import (
	"github.com/66gu1/easygodocs/internal/infrastructure/db"
	"github.com/google/uuid"
)

type department struct {
	db.Base
	ID       uuid.UUID
	Name     string
	ParentID *uuid.UUID
}

func (m *department) fromDTO(dto Department) {
	m.ID = dto.ID
	m.Name = dto.Name
	m.ParentID = dto.ParentID
}

func (m *department) toDTO() Department {
	dto := Department{
		ID:        m.ID,
		Name:      m.Name,
		ParentID:  m.ParentID,
		UpdatedAt: m.UpdatedAt,
		CreatedAt: m.CreatedAt,
	}
	if m.DeletedAt.Valid {
		dto.DeletedAt = &m.DeletedAt.Time
	}

	return dto
}

func (m *department) getMap() map[string]interface{} {
	resp := make(map[string]interface{})
	resp["id"] = m.ID
	resp["name"] = m.Name
	resp["parent_id"] = m.ParentID

	return resp
}

func toDTOs(models []department) []Department {
	dtos := make([]Department, 0, len(models))
	for _, model := range models {
		dtos = append(dtos, model.toDTO())
	}

	return dtos
}
