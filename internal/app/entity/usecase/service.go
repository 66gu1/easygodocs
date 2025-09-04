package usecase

import (
	"context"
	"fmt"
	"slices"

	"github.com/66gu1/easygodocs/internal/app/auth"
	"github.com/66gu1/easygodocs/internal/app/entity"
	"github.com/66gu1/easygodocs/internal/infrastructure/apperr"
	"github.com/66gu1/easygodocs/internal/infrastructure/contextx"
	"github.com/66gu1/easygodocs/internal/infrastructure/logger"
	"github.com/google/uuid"
)

type CreateEntityCmd struct {
	Type     entity.Type `json:"type"`
	Name     string      `json:"name"`
	Content  string      `json:"content"`
	ParentID *uuid.UUID  `json:"parent_id,omitempty"`
	IsDraft  bool        `json:"is_draft"`
}

type UpdateEntityCmd struct {
	ID       uuid.UUID  `json:"id"`
	Name     string     `json:"name"`
	Content  string     `json:"content"`
	ParentID *uuid.UUID `json:"parent_id,omitempty"`
	IsDraft  bool       `json:"is_draft,omitempty"`
}

type service struct {
	core     Core
	authCore AuthCore
}

type Core interface {
	GetTree(ctx context.Context, permissions []uuid.UUID, isAdmin bool) (entity.Tree, error)
	GetPermittedHierarchy(ctx context.Context, directPermissions []uuid.UUID, onlyForRead bool) ([]uuid.UUID, error)
	Get(ctx context.Context, id uuid.UUID) (entity.Entity, error)
	GetVersion(ctx context.Context, id uuid.UUID, version int) (entity.Entity, error)
	GetVersionsList(ctx context.Context, id uuid.UUID) ([]entity.Entity, error)
	Create(ctx context.Context, req entity.CreateEntityReq) (uuid.UUID, error)
	GetListItem(ctx context.Context, id uuid.UUID) (entity.ListItem, error)
	Update(ctx context.Context, req entity.UpdateEntityReq) error
	Delete(ctx context.Context, id uuid.UUID) error
}

type AuthCore interface {
	GetCurrentUserDirectPermissions(ctx context.Context, role auth.Role) (ids []uuid.UUID, isAdmin bool, err error)
}

func NewService(repo Core, authCore AuthCore) *service {
	return &service{core: repo, authCore: authCore}
}

func (s *service) GetTree(ctx context.Context) (entity.Tree, error) {
	ids, isAdmin, err := s.authCore.GetCurrentUserDirectPermissions(ctx, auth.RoleRead)
	if err != nil {
		logger.Error(ctx, err).Msg("entity.service.GetTree: getUserPermissions")
		return entity.Tree{}, fmt.Errorf("entity.service.GetTree: %w", err)
	}
	tree, err := s.core.GetTree(ctx, ids, isAdmin)
	if err != nil {
		logger.Error(ctx, err).Msg("entity.service.GetTree: GetTree")
		return entity.Tree{}, fmt.Errorf("entity.service.GetTree: %w", err)
	}

	return tree, nil
}

func (s *service) Get(ctx context.Context, id uuid.UUID) (entity.Entity, error) {
	if err := s.checkEntityPermission(ctx, id, auth.RoleRead); err != nil {
		logger.Error(ctx, err).
			Str(entity.FieldEntityID.String(), id.String()).
			Msg("entity.service.Get: checkEntityPermission")
		return entity.Entity{}, fmt.Errorf("entity.service.Get: %w", err)
	}

	ent, err := s.core.Get(ctx, id)
	if err != nil {
		logger.Error(ctx, err).
			Str(entity.FieldEntityID.String(), id.String()).
			Msg("entity.service.Get: Get")
		return entity.Entity{}, fmt.Errorf("entity.service.Get: %w", err)
	}

	return ent, nil
}

func (s *service) GetVersion(ctx context.Context, id uuid.UUID, version int) (entity.Entity, error) {
	if err := s.checkEntityPermission(ctx, id, auth.RoleRead); err != nil {
		logger.Error(ctx, err).
			Str(entity.FieldEntityID.String(), id.String()).
			Int(entity.FieldVersion.String(), version).
			Msg("entity.service.GetVersion: checkEntityPermission")
		return entity.Entity{}, fmt.Errorf("entity.service.GetVersion: %w", err)
	}

	ent, err := s.core.GetVersion(ctx, id, version)
	if err != nil {
		logger.Error(ctx, err).
			Str(entity.FieldEntityID.String(), id.String()).
			Int(entity.FieldVersion.String(), version).
			Msg("entity.service.GetVersion: GetVersion")
		return entity.Entity{}, fmt.Errorf("entity.service.GetVersion: %w", err)
	}

	return ent, nil
}

func (s *service) GetVersionsList(ctx context.Context, id uuid.UUID) ([]entity.Entity, error) {
	if err := s.checkEntityPermission(ctx, id, auth.RoleRead); err != nil {
		logger.Error(ctx, err).
			Str(entity.FieldEntityID.String(), id.String()).
			Msg("entity.service.GetVersionsList: checkEntityPermission")
		return nil, fmt.Errorf("entity.service.GetVersionsList: %w", err)
	}

	entities, err := s.core.GetVersionsList(ctx, id)
	if err != nil {
		logger.Error(ctx, err).
			Str(entity.FieldEntityID.String(), id.String()).
			Msg("entity.service.GetVersionsList: GetVersionsList")
		return nil, fmt.Errorf("entity.service.GetVersionsList: %w", err)
	}

	return entities, nil
}

func (s *service) Create(ctx context.Context, cmd CreateEntityCmd) (uuid.UUID, error) {
	permissions, err := s.getEffectivePermissions(ctx, auth.RoleWrite)
	if err != nil {
		logger.Error(ctx, err).
			Interface(apperr.FieldRequest.String(), cmd).
			Msg("entity.service.Create: getEffectivePermissions")
		return uuid.Nil, fmt.Errorf("entity.service.Create: %w", err)
	}
	if err = permissions.checkParentIDs([]*uuid.UUID{cmd.ParentID}); err != nil {
		logger.Error(ctx, err).
			Interface(apperr.FieldRequest.String(), cmd).
			Msg("entity.service.Create: checkParentIDs")
		return uuid.Nil, fmt.Errorf("entity.service.Create: %w", err)
	}

	userID, err := contextx.GetUserID(ctx)
	if err != nil {
		logger.Error(ctx, err).
			Interface(apperr.FieldRequest.String(), cmd).
			Msg("entity.service.Create: GetUserID")
		return uuid.Nil, fmt.Errorf("entity.service.Create: %w", err)
	}
	req := entity.CreateEntityReq{
		Type:     cmd.Type,
		Name:     cmd.Name,
		Content:  cmd.Content,
		ParentID: cmd.ParentID,
		IsDraft:  cmd.IsDraft,
		UserID:   userID,
	}
	id, err := s.core.Create(ctx, req)
	if err != nil {
		logger.Error(ctx, err).
			Interface(apperr.FieldRequest.String(), req).
			Msg("entity.service.Create: Create")
		return uuid.Nil, fmt.Errorf("entity.service.Create: %w", err)
	}

	return id, nil
}

func (s *service) Update(ctx context.Context, cmd UpdateEntityCmd) error {
	permissions, err := s.getEffectivePermissions(ctx, auth.RoleWrite)
	if err != nil {
		logger.Error(ctx, err).
			Interface(apperr.FieldRequest.String(), cmd).
			Msg("entity.service.Update: getEffectivePermissions")
		return fmt.Errorf("entity.service.Update: %w", err)
	}
	if err = permissions.checkID(cmd.ID); err != nil {
		logger.Error(ctx, err).
			Interface(apperr.FieldRequest.String(), cmd).
			Msg("entity.service.Update: checkID")
		return fmt.Errorf("entity.service.Update: %w", err)
	}

	oldEntity, err := s.core.GetListItem(ctx, cmd.ID)
	if err != nil {
		logger.Error(ctx, err).
			Interface(apperr.FieldRequest.String(), cmd).
			Msg("entity.service.Update: GetListItem")
		return fmt.Errorf("entity.service.Update: %w", err)
	}
	parentChanged := !equalUUIDPtr(oldEntity.ParentID, cmd.ParentID)
	if parentChanged {
		if err = permissions.checkParentIDs([]*uuid.UUID{cmd.ParentID, oldEntity.ParentID}); err != nil {
			logger.Error(ctx, err).
				Interface(apperr.FieldRequest.String(), cmd).
				Msg("entity.service.Update: checkParentIDs")
			return fmt.Errorf("entity.service.Update: %w", err)
		}
	}

	userID, err := contextx.GetUserID(ctx)
	if err != nil {
		logger.Error(ctx, err).
			Interface(apperr.FieldRequest.String(), cmd).
			Msg("entity.service.Update: GetUserID")
		return fmt.Errorf("entity.service.Update: %w", err)
	}

	req := entity.UpdateEntityReq{
		ID:            cmd.ID,
		Name:          cmd.Name,
		Content:       cmd.Content,
		ParentID:      cmd.ParentID,
		IsDraft:       cmd.IsDraft,
		UserID:        userID,
		ParentChanged: parentChanged,
	}

	if err = s.core.Update(ctx, req); err != nil {
		logger.Error(ctx, err).
			Interface(apperr.FieldRequest.String(), req).
			Msg("entity.service.Update: Update")
		return fmt.Errorf("entity.service.Update: %w", err)
	}

	return nil
}

func (s *service) Delete(ctx context.Context, id uuid.UUID) error {
	err := s.checkEntityPermission(ctx, id, auth.RoleWrite)
	if err != nil {
		logger.Error(ctx, err).
			Str(entity.FieldEntityID.String(), id.String()).
			Msg("entity.service.Delete: checkEntityPermission")
		return fmt.Errorf("entity.service.Delete: %w", err)
	}
	err = s.core.Delete(ctx, id)
	if err != nil {
		logger.Error(ctx, err).
			Str(entity.FieldEntityID.String(), id.String()).
			Msg("entity.service.Delete: Delete")
		return fmt.Errorf("entity.service.Delete: %w", err)
	}

	return nil
}

func (s *service) checkEntityPermission(ctx context.Context, id uuid.UUID, role auth.Role) error {
	permissions, err := s.getEffectivePermissions(ctx, role)
	if err != nil {
		return fmt.Errorf("checkEntityPermission: %w", err)
	}

	err = permissions.checkID(id)
	if err != nil {
		return fmt.Errorf("checkEntityPermission: %w", err)
	}

	return nil
}

// getEffectivePermissions returns all permissions including inherited ones
// doesn't return ids if isAdmin is true
func (s *service) getEffectivePermissions(ctx context.Context, role auth.Role) (effectivePermissions, error) {
	ids, isAdmin, err := s.authCore.GetCurrentUserDirectPermissions(ctx, role)
	if err != nil {
		return effectivePermissions{}, fmt.Errorf("getEffectivePermissions: %w", err)
	}
	if isAdmin {
		return effectivePermissions{isAdmin: true}, nil
	}

	effectiveIDs, err := s.core.GetPermittedHierarchy(ctx, ids, role.IsOnlyForRead())
	if err != nil {
		return effectivePermissions{}, fmt.Errorf("getEffectivePermissions: %w", err)
	}

	return effectivePermissions{ids: effectiveIDs}, nil
}

type effectivePermissions struct {
	isAdmin bool
	ids     []uuid.UUID
}

func (ep *effectivePermissions) checkID(id uuid.UUID) error {
	if ep.isAdmin {
		return nil
	}
	if slices.Contains(ep.ids, id) {
		return nil
	}

	return fmt.Errorf("effectivePermissions.checkID: %w", apperr.ErrForbidden())
}

func (ep *effectivePermissions) checkParentIDs(parentIDs []*uuid.UUID) error {
	for _, id := range parentIDs {
		if id == nil {
			if !ep.isAdmin {
				return fmt.Errorf("effectivePermissions.checkParentIDs: %w", apperr.ErrForbidden())
			}
			return nil
		}
		if err := ep.checkID(*id); err != nil {
			return fmt.Errorf("effectivePermissions.checkParentIDs: %w", err)
		}
	}
	return nil
}

func equalUUIDPtr(a, b *uuid.UUID) bool {
	switch {
	case a == nil && b == nil:
		return true
	case a == nil || b == nil:
		return false
	default:
		return *a == *b
	}
}
