package user

import (
	"fmt"

	"github.com/66gu1/easygodocs/internal/infrastructure/apperr"
)

const (
	CodeValidationFailed apperr.Code = "user/validation_failed"
	CodeNotFound         apperr.Code = "user/not_found"
	CodeEmailDuplicate   apperr.Code = "user/email_duplicate"
	CodeSamePassword     apperr.Code = "user/same_password"
	CodePasswordMismatch apperr.Code = "user/password_mismatch"
)

const (
	FieldEmail    apperr.Field = "email"
	FieldName     apperr.Field = "name"
	FieldPassword apperr.Field = "password"
	FieldUserID   apperr.Field = "user_id"
	FieldUser     apperr.Field = "user"
)

// Validation errors

func ErrInvalidEmail() error {
	return apperr.New("Invalid email", CodeValidationFailed, apperr.ClassBadRequest, apperr.LogLevelWarn).
		WithViolation(apperr.Violation{
			Field: FieldEmail, Rule: apperr.RuleInvalidFormat,
		})
}

func ErrNameEmpty() error {
	return apperr.New("Name cannot be empty", CodeValidationFailed, apperr.ClassBadRequest, apperr.LogLevelWarn).
		WithViolation(apperr.Violation{
			Field: FieldName, Rule: apperr.RuleRequired,
		})
}

func ErrNameTooLong(max int) error {
	return apperr.New("Name is too long", CodeValidationFailed, apperr.ClassBadRequest, apperr.LogLevelWarn).
		WithViolation(apperr.Violation{
			Field: FieldName, Rule: apperr.RuleTooLong, Params: map[string]any{"max": max},
		})
}

func ErrEmailTooLong(max int) error {
	return apperr.New("Email is too long", CodeValidationFailed, apperr.ClassBadRequest, apperr.LogLevelWarn).
		WithViolation(apperr.Violation{
			Field: FieldEmail, Rule: apperr.RuleTooLong, Params: map[string]any{"max": max},
		})
}

func ErrPasswordTooShort(min int) error {
	return apperr.New("password is too short", CodeValidationFailed, apperr.ClassBadRequest, apperr.LogLevelWarn).
		WithViolation(apperr.Violation{
			Field: FieldPassword, Rule: apperr.RuleTooShort, Params: map[string]any{"min": min},
		}).WithUserMessage(fmt.Sprintf("Password must be at least %d characters", min))
}

func ErrPasswordTooLong(max int) error {
	return apperr.New("password is too long", CodeValidationFailed, apperr.ClassBadRequest, apperr.LogLevelWarn).
		WithViolation(apperr.Violation{
			Field: FieldPassword, Rule: apperr.RuleTooLong, Params: map[string]any{"max": max},
		}).WithUserMessage(fmt.Sprintf("Password must be at most %d characters", max))
}

// Business logic errors

func ErrUserNotFound() error {
	return apperr.New("User not found", CodeNotFound, apperr.ClassNotFound, apperr.LogLevelWarn)
}

func ErrUserWithEmailAlreadyExists() error {
	return apperr.New("User with this email already exists", CodeEmailDuplicate, apperr.ClassConflict, apperr.LogLevelWarn).
		WithViolation(apperr.Violation{
			Field: FieldEmail, Rule: apperr.RuleDuplicate,
		})
}
