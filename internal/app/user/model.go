package user

import (
	hierarchy "github.com/66gu1/easygodocs/internal/app/hierarchy/dto"
	"github.com/66gu1/easygodocs/internal/app/user/dto"
	"github.com/66gu1/easygodocs/internal/infrastructure/auth"
	"github.com/66gu1/easygodocs/internal/infrastructure/db"
	"github.com/google/uuid"
	"time"
)

type user struct {
	db.Base
	ID           uuid.UUID `json:"id"`
	Email        string    `json:"email"`
	PasswordHash string    `json:"-"`
	Name         string    `json:"name"`
}

func (u *user) toDTO() dto.User {
	var deletedAt *time.Time
	if u.DeletedAt.Valid {
		deletedAt = &u.DeletedAt.Time
	}

	return dto.User{
		ID:           u.ID,
		Email:        u.Email,
		Name:         u.Name,
		PasswordHash: u.PasswordHash,
		CreatedAt:    u.CreatedAt,
		UpdatedAt:    u.UpdatedAt,
		DeletedAt:    deletedAt,
	}
}

func toDTOs[T any, DTO any](items []T, mapper func(T) DTO) []DTO {
	dtos := make([]DTO, len(items))
	for i, item := range items {
		dtos[i] = mapper(item)
	}
	return dtos
}

type session struct {
	ID           uuid.UUID `json:"id"`
	UserID       uuid.UUID `json:"user_id"`
	UserAgent    string    `json:"user_agent"`
	RefreshToken string    `json:"-"`
	CreatedAt    time.Time `json:"created_at"`
	ExpiresAt    time.Time `json:"expires_at"`
}

func (s *session) toDTO() dto.Session {
	return dto.Session{
		ID:           s.ID,
		UserID:       s.UserID,
		UserAgent:    s.UserAgent,
		CreatedAt:    s.CreatedAt,
		ExpiresAt:    s.ExpiresAt,
		RefreshToken: s.RefreshToken,
	}
}

type userRole struct {
	UserID     uuid.UUID             `json:"user_id"`
	Role       auth.Role             `json:"role"`
	EntityID   *uuid.UUID            `json:"entity_id,omitempty"`
	EntityType *hierarchy.EntityType `json:"entity_type"`
}

func (u *userRole) toDTO() dto.UserRole {
	dto := dto.UserRole{
		UserID: u.UserID,
		Role:   u.Role,
	}
	if u.EntityID != nil && u.EntityType != nil {
		dto.Entity.ID = *u.EntityID
		dto.Entity.Type = *u.EntityType
	}

	return dto
}

func userRoleFromDTO(dto dto.UserRole) userRole {
	model := userRole{
		UserID: dto.UserID,
		Role:   dto.Role,
	}
	if dto.Entity != nil {
		model.EntityID = &dto.Entity.ID
		model.EntityType = &dto.Entity.Type
	}

	return model
}

type accessScope struct {
	EntityID   uuid.UUID            `json:"entity_id,omitempty"`
	EntityType hierarchy.EntityType `json:"entity_type"`
}

type updateTokenReq struct {
	ID           uuid.UUID `json:"id"`
	RefreshToken string    `json:"refresh_token"`
	ExpiresAt    time.Time `json:"expires_at"`
}
