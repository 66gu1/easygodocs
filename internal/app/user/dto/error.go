package dto

import (
	"fmt"
	"github.com/66gu1/easygodocs/internal/infrastructure/apperror"
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
	ErrPasswordEmpty = &apperror.Error{
		Message:  "password cannot be empty",
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
