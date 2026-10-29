package usecase

import (
	"context"
	"fmt"
	"slices"

	"github.com/66gu1/easygodocs/internal/app/auth"
	"github.com/66gu1/easygodocs/internal/app/entity"
	"github.com/66gu1/easygodocs/internal/infrastructure/apperr"
	"github.com/66gu1/easygodocs/internal/infrastructure/contextx"
	"github.com/google/uuid"
)

type Core interface {
	GetTree(ctx context.Context, permissions []uuid.UUID, isAdmin bool) (entity.Tree, error)
	GetPermittedIDs(ctx context.Context, directPermissions []uuid.UUID, hType entity.HierarchyType) ([]uuid.UUID, error)
	Get(ctx context.Context, id uuid.UUID) (entity.Entity, error)
	GetVersion(ctx context.Context, id uuid.UUID, version int) (entity.Entity, error)
	GetVersionsList(ctx context.Context, id uuid.UUID) ([]entity.Entity, error)
	Create(ctx context.Context, req entity.CreateEntityReq) (uuid.UUID, error)
	GetListItem(ctx context.Context, id uuid.UUID) (entity.ListItem, error)
	Update(ctx context.Context, req entity.UpdateEntityReq) error
	Delete(ctx context.Context, id uuid.UUID) error
	SaveDraft(ctx context.Context, req entity.SaveDraftReq) error
	GetDraft(ctx context.Context, entityID, userID uuid.UUID) (entity.Draft, error)
	DeleteDraft(ctx context.Context, entityID, userID uuid.UUID) error
	PublishDraft(ctx context.Context, entityID, userID uuid.UUID) error
}

type AuthCore interface {
	GetCurrentUserDirectPermissions(ctx context.Context, role auth.Role) (ids []uuid.UUID, isAdmin bool, err error)
}

type PermissionChecker interface {
	CheckEntityPermission(ctx context.Context, id uuid.UUID, role auth.Role) error
	GetEffectivePermissions(ctx context.Context, role auth.Role) (EffectivePermissions, error)
	GetDirectPermissions(ctx context.Context, role auth.Role) ([]uuid.UUID, bool, error)
}

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
}

type SaveDraftCmd struct {
	EntityID uuid.UUID `json:"entity_id"`
	Name     string    `json:"name"`
	Content  string    `json:"content"`
}

type Service struct {
	core Core
	perm PermissionChecker
}

func NewService(repo Core, perm PermissionChecker) *Service {
	if perm == nil || repo == nil {
		panic("entity.NewService: nil core or perm")
	}
	return &Service{core: repo, perm: perm}
}

func (s *Service) GetTree(ctx context.Context) (entity.Tree, error) {
	ids, isAdmin, err := s.perm.GetDirectPermissions(ctx, auth.RoleRead)
	if err != nil {
		return entity.Tree{}, fmt.Errorf("entity.service.GetTree: %w", err)
	}
	tree, err := s.core.GetTree(ctx, ids, isAdmin)
	if err != nil {
		return entity.Tree{}, fmt.Errorf("entity.service.GetTree: %w", err)
	}

	return tree, nil
}

func (s *Service) Get(ctx context.Context, id uuid.UUID) (entity.Entity, error) {
	if err := s.perm.CheckEntityPermission(ctx, id, auth.RoleRead); err != nil {
		return entity.Entity{}, fmt.Errorf("entity.service.Get: %w", err)
	}

	ent, err := s.core.Get(ctx, id)
	if err != nil {
		return entity.Entity{}, fmt.Errorf("entity.service.Get: %w", err)
	}

	return ent, nil
}

func (s *Service) GetVersion(ctx context.Context, id uuid.UUID, version int) (entity.Entity, error) {
	if err := s.perm.CheckEntityPermission(ctx, id, auth.RoleRead); err != nil {
		return entity.Entity{}, fmt.Errorf("entity.service.GetVersion: %w", err)
	}

	ent, err := s.core.GetVersion(ctx, id, version)
	if err != nil {
		return entity.Entity{}, fmt.Errorf("entity.service.GetVersion: %w", err)
	}

	return ent, nil
}

func (s *Service) GetVersionsList(ctx context.Context, id uuid.UUID) ([]entity.Entity, error) {
	if err := s.perm.CheckEntityPermission(ctx, id, auth.RoleRead); err != nil {
		return nil, fmt.Errorf("entity.service.GetVersionsList: %w", err)
	}

	entities, err := s.core.GetVersionsList(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("entity.service.GetVersionsList: %w", err)
	}

	return entities, nil
}

func (s *Service) Create(ctx context.Context, cmd CreateEntityCmd) (uuid.UUID, error) {
	permissions, err := s.perm.GetEffectivePermissions(ctx, auth.RoleWrite)
	if err != nil {
		return uuid.Nil, fmt.Errorf("entity.service.Create: %w", err)
	}
	if err = permissions.CheckParentIDs([]*uuid.UUID{cmd.ParentID}); err != nil {
		return uuid.Nil, fmt.Errorf("entity.service.Create: %w", err)
	}

	userID, err := contextx.GetUserID(ctx)
	if err != nil {
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
		return uuid.Nil, fmt.Errorf("entity.service.Create: %w", err)
	}

	return id, nil
}

func (s *Service) Update(ctx context.Context, cmd UpdateEntityCmd) error {
	permissions, err := s.perm.GetEffectivePermissions(ctx, auth.RoleWrite)
	if err != nil {
		return fmt.Errorf("entity.service.Update: %w", err)
	}
	if err = permissions.CheckID(cmd.ID); err != nil {
		return fmt.Errorf("entity.service.Update: %w", err)
	}

	oldEntity, err := s.core.GetListItem(ctx, cmd.ID)
	if err != nil {
		return fmt.Errorf("entity.service.Update: %w", err)
	}
	parentChanged := !equalUUIDPtr(oldEntity.ParentID, cmd.ParentID)
	if parentChanged {
		if err = permissions.CheckParentIDs([]*uuid.UUID{cmd.ParentID, oldEntity.ParentID}); err != nil {
			return fmt.Errorf("entity.service.Update: %w", err)
		}
	}

	userID, err := contextx.GetUserID(ctx)
	if err != nil {
		return fmt.Errorf("entity.service.Update: %w", err)
	}

	req := entity.UpdateEntityReq{
		ID:            cmd.ID,
		Name:          cmd.Name,
		Content:       cmd.Content,
		ParentID:      cmd.ParentID,
		UserID:        userID,
		ParentChanged: parentChanged,
		EntityType:    oldEntity.Type,

		ExpectedParentID: oldEntity.ParentID,
	}

	if err = s.core.Update(ctx, req); err != nil {
		return fmt.Errorf("entity.service.Update: %w", err)
	}

	return nil
}

func (s *Service) Delete(ctx context.Context, id uuid.UUID) error {
	err := s.perm.CheckEntityPermission(ctx, id, auth.RoleWrite)
	if err != nil {
		return fmt.Errorf("entity.service.Delete: %w", err)
	}
	err = s.core.Delete(ctx, id)
	if err != nil {
		return fmt.Errorf("entity.service.Delete: %w", err)
	}

	return nil
}

// Drafts are personal: every operation works with the current user's draft
// and requires write permission for the entity.

func (s *Service) SaveDraft(ctx context.Context, cmd SaveDraftCmd) error {
	userID, err := s.checkDraftAccess(ctx, cmd.EntityID)
	if err != nil {
		return fmt.Errorf("entity.service.SaveDraft: %w", err)
	}
	req := entity.SaveDraftReq{
		EntityID: cmd.EntityID,
		UserID:   userID,
		Name:     cmd.Name,
		Content:  cmd.Content,
	}
	if err = s.core.SaveDraft(ctx, req); err != nil {
		return fmt.Errorf("entity.service.SaveDraft: %w", err)
	}

	return nil
}

func (s *Service) GetDraft(ctx context.Context, entityID uuid.UUID) (entity.Draft, error) {
	userID, err := s.checkDraftAccess(ctx, entityID)
	if err != nil {
		return entity.Draft{}, fmt.Errorf("entity.service.GetDraft: %w", err)
	}
	draft, err := s.core.GetDraft(ctx, entityID, userID)
	if err != nil {
		return entity.Draft{}, fmt.Errorf("entity.service.GetDraft: %w", err)
	}

	return draft, nil
}

func (s *Service) DeleteDraft(ctx context.Context, entityID uuid.UUID) error {
	userID, err := s.checkDraftAccess(ctx, entityID)
	if err != nil {
		return fmt.Errorf("entity.service.DeleteDraft: %w", err)
	}
	if err = s.core.DeleteDraft(ctx, entityID, userID); err != nil {
		return fmt.Errorf("entity.service.DeleteDraft: %w", err)
	}

	return nil
}

func (s *Service) PublishDraft(ctx context.Context, entityID uuid.UUID) error {
	userID, err := s.checkDraftAccess(ctx, entityID)
	if err != nil {
		return fmt.Errorf("entity.service.PublishDraft: %w", err)
	}
	if err = s.core.PublishDraft(ctx, entityID, userID); err != nil {
		return fmt.Errorf("entity.service.PublishDraft: %w", err)
	}

	return nil
}

// checkDraftAccess checks write permission for the entity and returns the current user ID.
func (s *Service) checkDraftAccess(ctx context.Context, entityID uuid.UUID) (uuid.UUID, error) {
	if err := s.perm.CheckEntityPermission(ctx, entityID, auth.RoleWrite); err != nil {
		return uuid.Nil, fmt.Errorf("checkDraftAccess: %w", err)
	}
	userID, err := contextx.GetUserID(ctx)
	if err != nil {
		return uuid.Nil, fmt.Errorf("checkDraftAccess: %w", err)
	}

	return userID, nil
}

type DefaultPermissionChecker struct {
	core     Core
	authCore AuthCore
}

func NewPermissionChecker(core Core, authCore AuthCore) *DefaultPermissionChecker {
	if core == nil || authCore == nil {
		panic("NewPermissionChecker: nil core or authCore")
	}
	return &DefaultPermissionChecker{core: core, authCore: authCore}
}

func (p *DefaultPermissionChecker) CheckEntityPermission(ctx context.Context, id uuid.UUID, role auth.Role) error {
	permissions, err := p.GetEffectivePermissions(ctx, role)
	if err != nil {
		return fmt.Errorf("permissionChecker.CheckEntityPermission: %w", err)
	}

	err = permissions.CheckID(id)
	if err != nil {
		return fmt.Errorf("permissionChecker.CheckEntityPermission: %w", err)
	}

	return nil
}

// GetEffectivePermissions returns all permissions including inherited ones
// doesn't return ids if isAdmin is true
func (p *DefaultPermissionChecker) GetEffectivePermissions(ctx context.Context, role auth.Role) (EffectivePermissions, error) {
	ids, isAdmin, err := p.authCore.GetCurrentUserDirectPermissions(ctx, role)
	if err != nil {
		return EffectivePermissions{}, fmt.Errorf("permissionChecker.GetEffectivePermissions: %w", err)
	}
	if isAdmin {
		return EffectivePermissions{IsAdmin: true}, nil
	}

	hType := entity.HierarchyTypeChildrenOnly
	if role.IsOnlyForRead() {
		hType = entity.HierarchyTypeChildrenAndParents
	}
	effectiveIDs, err := p.core.GetPermittedIDs(ctx, ids, hType)
	if err != nil {
		return EffectivePermissions{}, fmt.Errorf("permissionChecker.GetEffectivePermissions: %w", err)
	}

	return EffectivePermissions{IDs: effectiveIDs}, nil
}

func (p *DefaultPermissionChecker) GetDirectPermissions(ctx context.Context, role auth.Role) ([]uuid.UUID, bool, error) {
	ids, isAdmin, err := p.authCore.GetCurrentUserDirectPermissions(ctx, role)
	if err != nil {
		return nil, false, fmt.Errorf("permissionChecker.GetDirectPermissions: %w", err)
	}
	return ids, isAdmin, nil
}

type EffectivePermissions struct {
	IsAdmin bool
	IDs     []uuid.UUID
}

func (ep *EffectivePermissions) CheckID(id uuid.UUID) error {
	if ep.IsAdmin {
		return nil
	}
	if slices.Contains(ep.IDs, id) {
		return nil
	}

	return fmt.Errorf("EffectivePermissions.checkID: %w", apperr.ErrForbidden())
}

func (ep *EffectivePermissions) CheckParentIDs(parentIDs []*uuid.UUID) error {
	for _, id := range parentIDs {
		if id == nil {
			if !ep.IsAdmin {
				return fmt.Errorf("EffectivePermissions.checkParentIDs: %w", apperr.ErrForbidden())
			}
			return nil
		}
		if err := ep.CheckID(*id); err != nil {
			return fmt.Errorf("EffectivePermissions.checkParentIDs: %w", err)
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
