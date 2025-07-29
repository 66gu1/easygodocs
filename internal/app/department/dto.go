package department

import (
	"github.com/66gu1/easygodocs/internal/infrastructure/apperror"
	"github.com/google/uuid"
	"time"
)

var (
	nameRequiredErr = &apperror.Error{
		Message:  "department name is required",
		Code:     apperror.BadRequest,
		LogLevel: apperror.LogLevelWarn,
	}
	idRequiredErr = &apperror.Error{
		Message:  "department ID is required",
		Code:     apperror.BadRequest,
		LogLevel: apperror.LogLevelWarn,
	}
)

type Department struct {
	ID        uuid.UUID  `json:"id"`
	Name      string     `json:"name"`
	ParentID  *uuid.UUID `json:"parent_id,omitempty"`
	UpdatedAt time.Time  `json:"updated_at"`
	CreatedAt time.Time  `json:"created_at"`
	DeletedAt *time.Time `json:"deleted_at,omitempty"`
}

type CreateDepartmentReq struct {
	Name     string     `json:"name"`
	ParentID *uuid.UUID `json:"parent_id,omitempty"`
}

func (req CreateDepartmentReq) Validate() error {
	if req.Name == "" {
		return nameRequiredErr
	}

	return nil
}

type UpdateDepartmentReq struct {
	ID       uuid.UUID  `json:"id"`
	Name     string     `json:"name"`
	ParentID *uuid.UUID `json:"parent_id,omitempty"`
}

func (req UpdateDepartmentReq) Validate() error {
	if req.ID == uuid.Nil {
		return idRequiredErr
	}
	if req.Name == "" {
		return nameRequiredErr
	}

	if req.ParentID != nil {
		if *req.ParentID == uuid.Nil {
			return &apperror.Error{
				Message:  "Parent department ID must be omitted or a valid value — it cannot be an empty UUID.",
				Code:     apperror.BadRequest,
				LogLevel: apperror.LogLevelWarn,
			}
		}
		if req.ID == *req.ParentID {
			return &apperror.Error{
				Message:  "department cannot be its own parent",
				Code:     apperror.BadRequest,
				LogLevel: apperror.LogLevelWarn,
			}
		}
	}

	return nil
}
