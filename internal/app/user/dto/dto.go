package dto

import (
	hierarchy "github.com/66gu1/easygodocs/internal/app/hierarchy/dto"
	"github.com/66gu1/easygodocs/internal/infrastructure/auth"
	"github.com/google/uuid"

	"net/mail"
	"time"
)

type User struct {
	ID           uuid.UUID  `json:"id"`
	Email        string     `json:"email"`
	Name         string     `json:"name"`
	PasswordHash string     `json:"-"`
	CreatedAt    time.Time  `json:"created_at"`
	UpdatedAt    time.Time  `json:"updated_at"`
	DeletedAt    *time.Time `json:"deleted_at"`
}

type Session struct {
	ID           uuid.UUID `json:"id"`
	UserID       uuid.UUID `json:"user_id"`
	UserAgent    string    `json:"user_agent"`
	RefreshToken string    `json:"-"`
	CreatedAt    time.Time `json:"created_at"`
	ExpiresAt    time.Time `json:"expires_at"`
}

type UserRole struct {
	UserID uuid.UUID         `json:"user_id"`
	Role   auth.Role         `json:"role"`
	Entity *hierarchy.Entity `json:"entity"`
}

type CreateUserReq struct {
	Email    string `json:"email"`
	Name     string `json:"name"`
	Password string `json:"-"`

	// filled by the service layer
	ID           uuid.UUID `json:"id"`
	PasswordHash string    `json:"-"`
}

func (req *CreateUserReq) Validate(maxNameLength, maxEmailLength, maxPasswordLength, minPasswordLength int) error {
	_, err := mail.ParseAddress(req.Email)
	if err != nil {
		return ErrInvalidEmail
	}
	if req.Name == "" {
		return ErrNameEmpty
	}
	if len(req.Name) > maxNameLength {
		return newErrNameTooLong(maxNameLength)
	}
	if len(req.Email) > maxEmailLength {
		return newErrEmailTooLong(maxEmailLength)
	}
	if len(req.Password) < minPasswordLength {
		return newErrPasswordTooShort(minPasswordLength)
	}
	if len(req.Password) > maxPasswordLength {
		return newErrPasswordTooLong(maxPasswordLength)
	}

	return nil
}

type UpdateUserReq struct {
	ID    uuid.UUID `json:"id"`
	Email string    `json:"email"`
	Name  string    `json:"name"`
}

func (req *UpdateUserReq) Validate(maxNameLength, maxEmailLength int) error {
	_, err := mail.ParseAddress(req.Email)
	if err != nil {
		return ErrInvalidEmail
	}
	if req.Name == "" {
		return ErrNameEmpty
	}
	if len(req.Name) > maxNameLength {
		return newErrNameTooLong(maxNameLength)
	}
	if len(req.Email) > maxEmailLength {
		return newErrEmailTooLong(maxEmailLength)
	}

	return nil
}

type ChangePasswordReq struct {
	ID          uuid.UUID `json:"id"`
	OldPassword string    `json:"old_password"`
	NewPassword string    `json:"new_password"`

	// filled by the service layer
	NewPasswordHash string `json:"new_password_hash"`
}

func (req *ChangePasswordReq) Validate(maxPasswordLength, minPasswordLength int) error {
	if len(req.NewPassword) < minPasswordLength {
		return newErrPasswordTooShort(minPasswordLength)
	}
	if len(req.NewPassword) > maxPasswordLength {
		return newErrPasswordTooLong(maxPasswordLength)
	}

	return nil
}

type CheckUserRoleReq struct {
	UserID       uuid.UUID  `json:"user_id"`
	Role         auth.Role  `json:"role"`
	DepartmentID *uuid.UUID `json:"department_id,omitempty"`
	ArticleID    *uuid.UUID `json:"article_id,omitempty"`
}

type LoginReq struct {
	Email     string `json:"email"`
	Password  string `json:"password"`
	UserAgent string `json:"-"`
}

func (req *LoginReq) Validate() error {
	if _, err := mail.ParseAddress(req.Email); err != nil {
		return ErrInvalidEmail
	}
	if len(req.Password) == 0 {
		return ErrPasswordEmpty
	}

	return nil
}

type Permissions struct {
	ArticleIDs    []uuid.UUID `json:"article_ids"`
	DepartmentIDs []uuid.UUID `json:"department_ids"`
	All           bool        `json:"all"`
}
