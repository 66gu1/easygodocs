package user

import (
	"fmt"
	"github.com/66gu1/easygodocs/internal/infrastructure/apperror"
	"github.com/google/uuid"
	"net/mail"
	"time"
)

var (
	ErrInvalidEmail = &apperror.Error{
		Message:  "invalid email address",
		Code:     apperror.BadRequest,
		LogLevel: apperror.LogLevelWarn,
	}
	ErrNameEmpty = &apperror.Error{
		Message:  "name cannot be empty",
		Code:     apperror.BadRequest,
		LogLevel: apperror.LogLevelWarn,
	}
)

func newErrNameTooLong(maxLength int) *apperror.Error {
	return &apperror.Error{
		Message:  fmt.Sprintf("name cannot be longer than %d characters", maxLength),
		Code:     apperror.BadRequest,
		LogLevel: apperror.LogLevelWarn,
	}
}

func newErrEmailTooLong(maxLength int) *apperror.Error {
	return &apperror.Error{
		Message:  fmt.Sprintf("email cannot be longer than %d characters", maxLength),
		Code:     apperror.BadRequest,
		LogLevel: apperror.LogLevelWarn,
	}
}

func newErrPasswordTooShort(minLength int) *apperror.Error {
	return &apperror.Error{
		Message:  fmt.Sprintf("password must be at least %d characters long", minLength),
		Code:     apperror.BadRequest,
		LogLevel: apperror.LogLevelWarn,
	}
}

func newErrPasswordTooLong(maxLength int) *apperror.Error {
	return &apperror.Error{
		Message:  fmt.Sprintf("password cannot be longer than %d characters", maxLength),
		Code:     apperror.BadRequest,
		LogLevel: apperror.LogLevelWarn,
	}
}

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
	UserID       uuid.UUID  `json:"user_id"`
	Role         role       `json:"role"`
	DepartmentID *uuid.UUID `json:"department_id,omitempty"`
	ArticleID    *uuid.UUID `json:"article_id,omitempty"`
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
	Role         role       `json:"role"`
	DepartmentID *uuid.UUID `json:"department_id,omitempty"`
	ArticleID    *uuid.UUID `json:"article_id,omitempty"`
}

type updateTokenReq struct {
	ID           uuid.UUID `json:"id"`
	RefreshToken string    `json:"refresh_token"`
	ExpiresAt    time.Time `json:"expires_at"`
}
