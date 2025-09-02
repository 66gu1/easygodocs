package gorm

import (
	"time"

	"github.com/66gu1/easygodocs/internal/app/user"
	"github.com/66gu1/easygodocs/internal/infrastructure/db"
	"github.com/google/uuid"
)

type userModel struct {
	db.Base
	ID             uuid.UUID
	Email          string
	PasswordHash   string `json:"-"`
	Name           string
	SessionVersion int
}

func (u *userModel) toDTO() user.User {
	var deletedAt *time.Time
	if u.DeletedAt.Valid {
		deletedAt = &u.DeletedAt.Time
	}

	return user.User{
		ID:             u.ID,
		Email:          u.Email,
		Name:           u.Name,
		CreatedAt:      u.CreatedAt,
		UpdatedAt:      u.UpdatedAt,
		DeletedAt:      deletedAt,
		SessionVersion: u.SessionVersion,
	}
}
