package auth

import (
	"fmt"

	"github.com/66gu1/easygodocs/internal/infrastructure/apperr"
	"github.com/google/uuid"
)

var ErrInvalidRole = fmt.Errorf("invalid role")

type Role string

const (
	roleAdmin Role = "admin"
	RoleRead  Role = "read"
	RoleWrite Role = "write"
)

func (role Role) String() string {
	return string(role)
}

func (role Role) GetHierarchy() []Role {
	switch role {
	case RoleRead:
		return []Role{roleAdmin, RoleWrite, RoleRead}
	case RoleWrite:
		return []Role{roleAdmin, RoleWrite}
	case roleAdmin:
		return []Role{roleAdmin}
	default:
		return []Role{}
	}
}

func (role Role) Validate() error {
	switch role {
	case roleAdmin, RoleRead, RoleWrite:
		return nil
	default:
		return ErrInvalidRole
	}
}

func (role Role) IsOnlyForRead() bool {
	switch role {
	case RoleRead:
		return true
	default:
		return false
	}
}

func (role Role) RequiresEntity() bool {
	switch role {
	case RoleRead, RoleWrite:
		return true
	default:
		return false
	}
}

func (role Role) ValidateEntity(entity *uuid.UUID) error {
	if role.RequiresEntity() && (entity == nil || *entity == uuid.Nil) {
		return apperr.New("role entity is required", CodeValidationFailed, apperr.ClassBadRequest, apperr.LogLevelWarn).
			WithViolation(apperr.Violation{Field: FieldEntity, Rule: apperr.RuleRequired})
	} else if !role.RequiresEntity() && entity != nil {
		return apperr.New("role entity must be nil", CodeValidationFailed, apperr.ClassBadRequest, apperr.LogLevelWarn).
			WithViolation(apperr.Violation{Field: FieldEntity, Rule: apperr.RuleForbidden})
	}

	return nil
}
