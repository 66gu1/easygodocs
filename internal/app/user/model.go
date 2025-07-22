package user

import (
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

func (u *user) toDTO() User {
	var deletedAt *time.Time
	if u.DeletedAt.Valid {
		deletedAt = &u.DeletedAt.Time
	}

	return User{
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

func (s *session) toDTO() Session {
	return Session{
		ID:           s.ID,
		UserID:       s.UserID,
		UserAgent:    s.UserAgent,
		CreatedAt:    s.CreatedAt,
		ExpiresAt:    s.ExpiresAt,
		RefreshToken: s.RefreshToken,
	}
}

type userRole struct {
	UserID       uuid.UUID  `json:"user_id"`
	Role         role       `json:"role"`
	DepartmentID *uuid.UUID `json:"department_id,omitempty"`
	ArticleID    *uuid.UUID `json:"article_id,omitempty"`
}

func (u *userRole) toDTO() UserRole {
	return UserRole{
		UserID:       u.UserID,
		Role:         u.Role,
		DepartmentID: u.DepartmentID,
		ArticleID:    u.ArticleID,
	}
}

func (u *userRole) fromDTO(dto UserRole) {
	u.UserID = dto.UserID
	u.Role = dto.Role
	u.DepartmentID = dto.DepartmentID
	u.ArticleID = dto.ArticleID
}

type accessScope struct {
	DepartmentID *uuid.UUID
	ArticleID    *uuid.UUID
}
