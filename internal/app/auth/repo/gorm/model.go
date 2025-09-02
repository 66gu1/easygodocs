package gorm

import (
	"time"

	"github.com/66gu1/easygodocs/internal/app/auth"
	"github.com/google/uuid"
)

type sessionModel struct {
	ID               uuid.UUID
	UserID           uuid.UUID
	RefreshTokenHash string `json:"-"`
	CreatedAt        time.Time
	ExpiresAt        time.Time
	SessionVersion   int
}

func (s *sessionModel) toDTO() auth.Session {
	return auth.Session{
		ID:             s.ID,
		UserID:         s.UserID,
		CreatedAt:      s.CreatedAt,
		ExpiresAt:      s.ExpiresAt,
		SessionVersion: s.SessionVersion,
	}
}

type userRole struct {
	UserID   uuid.UUID
	Role     auth.Role
	EntityID *uuid.UUID
}

func (u *userRole) toDTO() auth.UserRole {
	return auth.UserRole{
		UserID:   u.UserID,
		Role:     u.Role,
		EntityID: u.EntityID,
	}
}

func userRoleFromDTO(dto auth.UserRole) userRole {
	return userRole{
		UserID:   dto.UserID,
		Role:     dto.Role,
		EntityID: dto.EntityID,
	}
}
