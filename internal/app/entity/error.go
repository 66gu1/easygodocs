package entity

import "github.com/66gu1/easygodocs/internal/infrastructure/apperr"

func ErrEntityNotFound() error {
	return apperr.New("Entity not found", CodeNotFound, apperr.ClassNotFound, apperr.LogLevelWarn)
}

func ErrParentCycle() error {
	return apperr.New("Parent cycle detected", CodeParentCycle, apperr.ClassBadRequest, apperr.LogLevelWarn).
		WithViolation(apperr.Violation{
			Field: FieldParentID, Rule: apperr.RuleCycle,
		})
}

func ErrMaxHierarchyDepthExceeded(maxDepth int) error {
	return apperr.New("Maximum hierarchy depth exceeded", CodeMaxDepthExceeded, apperr.ClassBadRequest, apperr.LogLevelWarn).
		WithViolation(apperr.Violation{
			Field: FieldParentID, Rule: apperr.RuleMaxHierarchy,
			Params: map[string]any{"max_depth": maxDepth},
		})
}

const (
	CodeValidationFailed apperr.Code = "entity/validation_failed"
	CodeNotFound         apperr.Code = "entity/not_found"
	CodeParentCycle      apperr.Code = "entity/parent_cycle"
	CodeMaxDepthExceeded apperr.Code = "entity/max_depth_exceeded"
)

const (
	FieldName     apperr.Field = "name"
	FieldType     apperr.Field = "type"
	FieldParentID apperr.Field = "parent_id"
	FieldEntityID apperr.Field = "entity_id"
	FieldUserID   apperr.Field = "user_id"
)

func ErrNameRequired() error {
	return apperr.New("name is required", CodeValidationFailed, apperr.ClassBadRequest, apperr.LogLevelWarn).
		WithViolation(apperr.Violation{Field: FieldName, Rule: apperr.RuleRequired})
}

func ErrNameTooLong(max int) error {
	return apperr.New("name is too long", CodeValidationFailed, apperr.ClassBadRequest, apperr.LogLevelWarn).
		WithViolation(apperr.Violation{Field: FieldName, Rule: apperr.RuleTooLong, Params: map[string]any{"max": max}})
}

func ErrParentRequired() error {
	return apperr.New("article must have a parent entity", CodeValidationFailed, apperr.ClassBadRequest, apperr.LogLevelWarn).
		WithViolation(apperr.Violation{Field: FieldParentID, Rule: apperr.RuleRequired})
}

func ErrInvalidVersion() error {
	return apperr.New("version must be positive", CodeValidationFailed, apperr.ClassBadRequest, apperr.LogLevelWarn).
		WithViolation(apperr.Violation{Field: FieldVersion, Rule: apperr.RuleInvalidFormat})
}

func ErrInvalidType() error {
	return apperr.New("invalid entity type", CodeValidationFailed, apperr.ClassBadRequest, apperr.LogLevelWarn).
		WithViolation(apperr.Violation{
			Field: FieldType, Rule: apperr.RuleInvalidFormat,
		})
}

func ErrIncompatibleParentType() error {
	return apperr.New("invalid parent type", CodeValidationFailed, apperr.ClassBadRequest, apperr.LogLevelWarn).
		WithViolation(apperr.Violation{
			Field: FieldParentID, Rule: apperr.RuleInvalidFormat,
		})
}
