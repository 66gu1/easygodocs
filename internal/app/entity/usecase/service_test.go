package usecase_test

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/66gu1/easygodocs/internal/app/auth"
	"github.com/66gu1/easygodocs/internal/app/entity"
	"github.com/66gu1/easygodocs/internal/app/entity/usecase"
	"github.com/66gu1/easygodocs/internal/app/entity/usecase/mocks"
	"github.com/66gu1/easygodocs/internal/infrastructure/apperr"
	"github.com/66gu1/easygodocs/internal/infrastructure/contextx"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

//go:generate minimock -o ./mocks -s _mock.go

type serviceMocks struct {
	core *mocks.CoreMock
	perm *mocks.PermissionCheckerMock
}

func newServiceMocks(t *testing.T) serviceMocks {
	t.Helper()
	return serviceMocks{
		core: mocks.NewCoreMock(t),
		perm: mocks.NewPermissionCheckerMock(t),
	}
}

func TestService_GetTree(t *testing.T) {
	t.Parallel()

	var (
		ctx     = t.Context()
		isAdmin = true
		ids     = []uuid.UUID{uuid.New(), uuid.New(), uuid.New(), uuid.New()}
		expErr  = fmt.Errorf("exp")
	)

	tests := []struct {
		name  string
		setup func(mock serviceMocks)
		err   error
	}{
		{
			name: "ok",
			setup: func(mock serviceMocks) {
				mock.perm.GetDirectPermissionsMock.Expect(ctx, auth.RoleRead).Return(ids, isAdmin, nil)
				mock.core.GetTreeMock.Return(nil, nil)
			},
		},
		{
			name: "core.GetTree error",
			setup: func(mock serviceMocks) {
				mock.perm.GetDirectPermissionsMock.Expect(ctx, auth.RoleRead).Return(ids, isAdmin, nil)
				mock.core.GetTreeMock.Return(nil, expErr)
			},
			err: expErr,
		},
		{
			name: "perm.GetDirectPermissions error",
			setup: func(mock serviceMocks) {
				mock.perm.GetDirectPermissionsMock.Expect(ctx, auth.RoleRead).Return(nil, false, expErr)
			},
			err: expErr,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			m := newServiceMocks(t)
			if tt.setup != nil {
				tt.setup(m)
			}

			s := usecase.NewService(m.core, m.perm)
			_, err := s.GetTree(ctx)
			if tt.err != nil {
				require.ErrorIs(t, err, tt.err)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestService_Get(t *testing.T) {
	t.Parallel()

	var (
		ctx      = t.Context()
		id       = uuid.New()
		parentID = uuid.New()
		want     = entity.Entity{
			ID:             id,
			Type:           "type",
			Name:           "name",
			Content:        "content",
			ParentID:       &parentID,
			CreatedBy:      uuid.New(),
			UpdatedBy:      uuid.New(),
			CurrentVersion: &[]int{1}[0],
			CreatedAt:      time.Now(),
			UpdatedAt:      time.Now(),
		}
		expErr = fmt.Errorf("exp")
	)

	tests := []struct {
		name  string
		setup func(mock serviceMocks)
		err   error
	}{
		{
			name: "ok",
			setup: func(mock serviceMocks) {
				mock.perm.CheckEntityPermissionMock.Expect(ctx, id, auth.RoleRead).Return(nil)
				mock.core.GetMock.Expect(ctx, id).Return(want, nil)
			},
		},
		{
			name: "core.Get error",
			setup: func(mock serviceMocks) {
				mock.perm.CheckEntityPermissionMock.Expect(ctx, id, auth.RoleRead).Return(nil)
				mock.core.GetMock.Expect(ctx, id).Return(entity.Entity{}, expErr)
			},
			err: expErr,
		},
		{
			name: "perm.CheckEntityPermissionMock error",
			setup: func(mock serviceMocks) {
				mock.perm.CheckEntityPermissionMock.Expect(ctx, id, auth.RoleRead).Return(expErr)
			},
			err: expErr,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			m := newServiceMocks(t)
			if tt.setup != nil {
				tt.setup(m)
			}

			s := usecase.NewService(m.core, m.perm)
			got, err := s.Get(ctx, id)
			if tt.err != nil {
				require.ErrorIs(t, err, tt.err)
			} else {
				require.NoError(t, err)
				require.Equal(t, want, got)
			}
		})
	}
}

func TestService_GetVersion(t *testing.T) {
	t.Parallel()
	var (
		ctx     = t.Context()
		id      = uuid.New()
		version = 1
		want    = entity.Entity{
			ID:             id,
			Type:           "type",
			Name:           "name",
			Content:        "content",
			CreatedBy:      uuid.New(),
			UpdatedBy:      uuid.New(),
			CurrentVersion: &[]int{1}[0],
			CreatedAt:      time.Now(),
			UpdatedAt:      time.Now(),
		}
		expErr = fmt.Errorf("exp")
	)
	tests := []struct {
		name  string
		setup func(mock serviceMocks)
		err   error
	}{
		{
			name: "ok",
			setup: func(mock serviceMocks) {
				mock.perm.CheckEntityPermissionMock.Expect(ctx, id, auth.RoleRead).Return(nil)
				mock.core.GetVersionMock.Expect(ctx, id, version).Return(want, nil)
			},
		},
		{
			name: "core.GetVersion error",
			setup: func(mock serviceMocks) {
				mock.perm.CheckEntityPermissionMock.Expect(ctx, id, auth.RoleRead).Return(nil)
				mock.core.GetVersionMock.Expect(ctx, id, version).Return(entity.Entity{}, expErr)
			},
			err: expErr,
		},
		{
			name: "perm.CheckEntityPermissionMock error",
			setup: func(mock serviceMocks) {
				mock.perm.CheckEntityPermissionMock.Expect(ctx, id, auth.RoleRead).Return(expErr)
			},
			err: expErr,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			m := newServiceMocks(t)
			if tt.setup != nil {
				tt.setup(m)
			}

			s := usecase.NewService(m.core, m.perm)
			got, err := s.GetVersion(ctx, id, version)
			if tt.err != nil {
				require.ErrorIs(t, err, tt.err)
			} else {
				require.NoError(t, err)
				require.Equal(t, want, got)
			}
		})
	}
}

func TestService_GetVersionsList(t *testing.T) {
	t.Parallel()
	var (
		ctx  = t.Context()
		id   = uuid.New()
		want = []entity.Entity{
			{
				ID:             id,
				Type:           "type",
				Name:           "name",
				Content:        "content",
				CreatedBy:      uuid.New(),
				UpdatedBy:      uuid.New(),
				CurrentVersion: &[]int{1}[0],
				CreatedAt:      time.Now(),
				UpdatedAt:      time.Now(),
			},
			{
				ID:             id,
				Type:           "type",
				Name:           "name",
				Content:        "content",
				CreatedBy:      uuid.New(),
				UpdatedBy:      uuid.New(),
				CurrentVersion: &[]int{2}[0],
				CreatedAt:      time.Now(),
				UpdatedAt:      time.Now(),
			},
		}
		expErr = fmt.Errorf("exp")
	)
	tests := []struct {
		name  string
		setup func(mock serviceMocks)
		err   error
	}{
		{
			name: "ok",
			setup: func(mock serviceMocks) {
				mock.perm.CheckEntityPermissionMock.Expect(ctx, id, auth.RoleRead).Return(nil)
				mock.core.GetVersionsListMock.Expect(ctx, id).Return(want, nil)
			},
		},
		{
			name: "core.GetVersionsList error",
			setup: func(mock serviceMocks) {
				mock.perm.CheckEntityPermissionMock.Expect(ctx, id, auth.RoleRead).Return(nil)
				mock.core.GetVersionsListMock.Expect(ctx, id).Return(nil, expErr)
			},
			err: expErr,
		},
		{
			name: "perm.CheckEntityPermissionMock error",
			setup: func(mock serviceMocks) {
				mock.perm.CheckEntityPermissionMock.Expect(ctx, id, auth.RoleRead).Return(expErr)
			},
			err: expErr,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			m := newServiceMocks(t)
			if tt.setup != nil {
				tt.setup(m)
			}

			s := usecase.NewService(m.core, m.perm)
			got, err := s.GetVersionsList(ctx, id)
			if tt.err != nil {
				require.ErrorIs(t, err, tt.err)
			} else {
				require.NoError(t, err)
				require.Equal(t, want, got)
			}
		})
	}
}

func TestService_Create(t *testing.T) {
	t.Parallel()
	var (
		ctx      = t.Context()
		parentID = uuid.New()
		cmd      = usecase.CreateEntityCmd{
			Type:     "type",
			Name:     "name",
			Content:  "content",
			ParentID: &parentID,
			IsDraft:  true,
		}
		userID = uuid.New()
		req    = entity.CreateEntityReq{
			Type:     cmd.Type,
			Name:     cmd.Name,
			Content:  cmd.Content,
			ParentID: cmd.ParentID,
			IsDraft:  cmd.IsDraft,
			UserID:   userID,
		}
		expErr = fmt.Errorf("exp")
	)
	ctx = contextx.SetUserID(ctx, userID)
	tests := []struct {
		name  string
		ctx   context.Context
		setup func(mock serviceMocks)
		err   error
	}{
		{
			name: "ok",
			ctx:  ctx,
			setup: func(mock serviceMocks) {
				mock.perm.CheckParentPermissionMock.Expect(ctx, auth.RoleWrite, &parentID).Return(nil)
				mock.core.CreateMock.Expect(ctx, req).Return(uuid.New(), nil)
			},
		},
		{
			name: "core.Create error",
			ctx:  ctx,
			setup: func(mock serviceMocks) {
				mock.perm.CheckParentPermissionMock.Expect(ctx, auth.RoleWrite, &parentID).Return(nil)
				mock.core.CreateMock.Expect(ctx, req).Return(uuid.Nil, expErr)
			},
			err: expErr,
		},
		{
			name: "no user id in context",
			ctx:  t.Context(),
			setup: func(mock serviceMocks) {
				mock.perm.CheckParentPermissionMock.Expect(t.Context(), auth.RoleWrite, &parentID).Return(nil)
			},
			err: apperr.ErrUnauthorized(),
		},
		{
			name: "no write permission",
			ctx:  ctx,
			setup: func(mock serviceMocks) {
				mock.perm.CheckParentPermissionMock.Expect(ctx, auth.RoleWrite, &parentID).Return(apperr.ErrForbidden())
			},
			err: apperr.ErrForbidden(),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			m := newServiceMocks(t)
			if tt.setup != nil {
				tt.setup(m)
			}

			s := usecase.NewService(m.core, m.perm)
			_, err := s.Create(tt.ctx, cmd)
			if tt.err != nil {
				require.ErrorIs(t, err, tt.err)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestService_Update(t *testing.T) {
	t.Parallel()
	var (
		ctx         = t.Context()
		id          = uuid.New()
		parentID    = uuid.New()
		oldParentID = uuid.New()

		cmd = usecase.UpdateEntityCmd{
			ID:       id,
			Name:     "name",
			Content:  "content",
			ParentID: &parentID,
		}
		userID   = uuid.New()
		listItem = entity.ListItem{
			ID:       id,
			Type:     "type",
			Name:     "name",
			ParentID: &oldParentID,
		}
		req = entity.UpdateEntityReq{
			ID:            cmd.ID,
			Name:          cmd.Name,
			Content:       cmd.Content,
			ParentID:      &parentID,
			UserID:        userID,
			ParentChanged: true,
			EntityType:    listItem.Type,

			ExpectedParentID: listItem.ParentID,
		}
		expErr = fmt.Errorf("exp")
	)
	ctx = contextx.SetUserID(ctx, userID)
	tests := []struct {
		name  string
		ctx   context.Context
		setup func(mock serviceMocks)
		err   error
	}{
		{
			name: "ok",
			ctx:  ctx,
			setup: func(mock serviceMocks) {
				mock.perm.CheckEntityPermissionMock.Expect(ctx, id, auth.RoleWrite).Return(nil)
				mock.core.GetListItemMock.Expect(ctx, req.ID).Return(listItem, nil)
				mock.perm.CheckParentPermissionMock.Expect(ctx, auth.RoleWrite, &parentID, &oldParentID).Return(nil)
				mock.core.UpdateMock.Expect(ctx, req).Return(nil)
			},
		},
		{
			name: "core.Update error",
			ctx:  ctx,
			setup: func(mock serviceMocks) {
				mock.perm.CheckEntityPermissionMock.Expect(ctx, id, auth.RoleWrite).Return(nil)
				mock.core.GetListItemMock.Expect(ctx, req.ID).Return(listItem, nil)
				mock.perm.CheckParentPermissionMock.Expect(ctx, auth.RoleWrite, &parentID, &oldParentID).Return(nil)
				mock.core.UpdateMock.Expect(ctx, req).Return(expErr)
			},
			err: expErr,
		},
		{
			name: "no user id in context",
			ctx:  t.Context(),
			setup: func(mock serviceMocks) {
				mock.perm.CheckEntityPermissionMock.Expect(t.Context(), id, auth.RoleWrite).Return(nil)
				mock.core.GetListItemMock.Expect(t.Context(), req.ID).Return(listItem, nil)
				mock.perm.CheckParentPermissionMock.Expect(t.Context(), auth.RoleWrite, &parentID, &oldParentID).Return(nil)
			},
			err: apperr.ErrUnauthorized(),
		},
		{
			name: "no parent permission",
			ctx:  ctx,
			setup: func(mock serviceMocks) {
				mock.perm.CheckEntityPermissionMock.Expect(ctx, id, auth.RoleWrite).Return(nil)
				mock.core.GetListItemMock.Expect(ctx, req.ID).Return(listItem, nil)
				mock.perm.CheckParentPermissionMock.Expect(ctx, auth.RoleWrite, &parentID, &oldParentID).Return(apperr.ErrForbidden())
			},
			err: apperr.ErrForbidden(),
		},
		{
			name: "no entity permission",
			ctx:  ctx,
			setup: func(mock serviceMocks) {
				mock.perm.CheckEntityPermissionMock.Expect(ctx, id, auth.RoleWrite).Return(apperr.ErrForbidden())
			},
			err: apperr.ErrForbidden(),
		},
		{
			name: "core.GetListItem error",
			ctx:  ctx,
			setup: func(mock serviceMocks) {
				mock.perm.CheckEntityPermissionMock.Expect(ctx, id, auth.RoleWrite).Return(nil)
				mock.core.GetListItemMock.Expect(ctx, req.ID).Return(entity.ListItem{}, expErr)
			},
			err: expErr,
		},
		{
			name: "ok/parent not changed",
			ctx:  ctx,
			setup: func(mock serviceMocks) {
				mock.perm.CheckEntityPermissionMock.Expect(ctx, id, auth.RoleWrite).Return(nil)
				mock.core.GetListItemMock.Expect(ctx, req.ID).Return(entity.ListItem{ID: id, Type: listItem.Type, ParentID: &parentID}, nil)
				mock.core.UpdateMock.Expect(ctx, entity.UpdateEntityReq{
					ID: cmd.ID, Name: cmd.Name, Content: cmd.Content, ParentID: &parentID, UserID: userID,
					EntityType: listItem.Type, ExpectedParentID: &parentID,
				}).Return(nil)
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			m := newServiceMocks(t)
			if tt.setup != nil {
				tt.setup(m)
			}

			s := usecase.NewService(m.core, m.perm)
			err := s.Update(tt.ctx, cmd)
			if tt.err != nil {
				require.ErrorIs(t, err, tt.err)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestService_Delete(t *testing.T) {
	t.Parallel()
	var (
		ctx    = t.Context()
		id     = uuid.New()
		errExp = fmt.Errorf("exp")
	)
	tests := []struct {
		name  string
		setup func(mock serviceMocks)
		err   error
	}{
		{
			name: "ok",
			setup: func(mock serviceMocks) {
				mock.perm.CheckEntityPermissionMock.Expect(ctx, id, auth.RoleWrite).Return(nil)
				mock.core.DeleteMock.Expect(ctx, id).Return(nil)
			},
		},
		{
			name: "core.Delete error",
			setup: func(mock serviceMocks) {
				mock.perm.CheckEntityPermissionMock.Expect(ctx, id, auth.RoleWrite).Return(nil)
				mock.core.DeleteMock.Expect(ctx, id).Return(errExp)
			},
			err: errExp,
		},
		{
			name: "perm.CheckEntityPermission error",
			setup: func(mock serviceMocks) {
				mock.perm.CheckEntityPermissionMock.Expect(ctx, id, auth.RoleWrite).Return(errExp)
			},
			err: errExp,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			m := newServiceMocks(t)
			if tt.setup != nil {
				tt.setup(m)
			}

			s := usecase.NewService(m.core, m.perm)
			err := s.Delete(ctx, id)
			if tt.err != nil {
				require.ErrorIs(t, err, tt.err)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestService_Drafts(t *testing.T) {
	t.Parallel()
	var (
		userID   = uuid.New()
		entityID = uuid.New()
		ctx      = contextx.SetUserID(t.Context(), userID)
		cmd      = usecase.SaveDraftCmd{EntityID: entityID, Name: "name", Content: "content"}
		draft    = entity.Draft{EntityID: entityID, UserID: userID, Name: cmd.Name, Content: cmd.Content}
		expErr   = fmt.Errorf("exp")
	)

	// every draft operation checks write permission for the entity and works with the current user's draft
	ops := []struct {
		name   string
		expect func(mock serviceMocks, err error)
		call   func(s *usecase.Service, ctx context.Context) error
	}{
		{
			name: "SaveDraft",
			expect: func(mock serviceMocks, err error) {
				mock.core.SaveDraftMock.Expect(ctx, entity.SaveDraftReq{
					EntityID: entityID, UserID: userID, Name: cmd.Name, Content: cmd.Content,
				}).Return(err)
			},
			call: func(s *usecase.Service, ctx context.Context) error { return s.SaveDraft(ctx, cmd) },
		},
		{
			name: "GetDraft",
			expect: func(mock serviceMocks, err error) {
				mock.core.GetDraftMock.Expect(ctx, entityID, userID).Return(draft, err)
			},
			call: func(s *usecase.Service, ctx context.Context) error {
				got, err := s.GetDraft(ctx, entityID)
				if err != nil {
					return fmt.Errorf("GetDraft: %w", err)
				}
				if got != draft {
					return fmt.Errorf("unexpected draft: %+v", got)
				}
				return nil
			},
		},
		{
			name: "DeleteDraft",
			expect: func(mock serviceMocks, err error) {
				mock.core.DeleteDraftMock.Expect(ctx, entityID, userID).Return(err)
			},
			call: func(s *usecase.Service, ctx context.Context) error { return s.DeleteDraft(ctx, entityID) },
		},
		{
			name: "PublishDraft",
			expect: func(mock serviceMocks, err error) {
				mock.core.PublishDraftMock.Expect(ctx, entityID, userID).Return(err)
			},
			call: func(s *usecase.Service, ctx context.Context) error { return s.PublishDraft(ctx, entityID) },
		},
	}
	for _, op := range ops {
		tests := []struct {
			name  string
			ctx   context.Context
			setup func(mock serviceMocks)
			err   error
		}{
			{
				name: "ok",
				ctx:  ctx,
				setup: func(mock serviceMocks) {
					mock.perm.CheckEntityPermissionMock.Expect(ctx, entityID, auth.RoleWrite).Return(nil)
					op.expect(mock, nil)
				},
			},
			{
				name: "core error",
				ctx:  ctx,
				setup: func(mock serviceMocks) {
					mock.perm.CheckEntityPermissionMock.Expect(ctx, entityID, auth.RoleWrite).Return(nil)
					op.expect(mock, expErr)
				},
				err: expErr,
			},
			{
				name: "no permission",
				ctx:  ctx,
				setup: func(mock serviceMocks) {
					mock.perm.CheckEntityPermissionMock.Expect(ctx, entityID, auth.RoleWrite).Return(apperr.ErrForbidden())
				},
				err: apperr.ErrForbidden(),
			},
			{
				name: "no user id in context",
				ctx:  t.Context(),
				setup: func(mock serviceMocks) {
					mock.perm.CheckEntityPermissionMock.Expect(t.Context(), entityID, auth.RoleWrite).Return(nil)
				},
				err: apperr.ErrUnauthorized(),
			},
		}
		for _, tt := range tests {
			t.Run(op.name+"/"+tt.name, func(t *testing.T) {
				t.Parallel()
				m := newServiceMocks(t)
				tt.setup(m)

				err := op.call(usecase.NewService(m.core, m.perm), tt.ctx)
				if tt.err != nil {
					require.ErrorIs(t, err, tt.err)
				} else {
					require.NoError(t, err)
				}
			})
		}
	}
}

type permMocks struct {
	core *mocks.CoreMock
	auth *mocks.AuthCoreMock
}

func newPermMocks(t *testing.T) permMocks {
	return permMocks{
		core: mocks.NewCoreMock(t),
		auth: mocks.NewAuthCoreMock(t),
	}
}

func TestPermissionChecker_GetDirectPermissions(t *testing.T) {
	t.Parallel()
	var (
		ctx     = t.Context()
		role    = auth.RoleRead
		ids     = []uuid.UUID{uuid.New(), uuid.New(), uuid.New()}
		isAdmin = true
		expErr  = fmt.Errorf("exp")
	)
	tests := []struct {
		name  string
		setup func(mock permMocks)
		err   error
	}{
		{
			name: "ok",
			setup: func(mock permMocks) {
				mock.auth.GetCurrentUserDirectPermissionsMock.Expect(ctx, role).Return(ids, isAdmin, nil)
			},
		},
		{
			name: "auth.GetCurrentUserDirectPermissions error",
			setup: func(mock permMocks) {
				mock.auth.GetCurrentUserDirectPermissionsMock.Expect(ctx, role).Return(nil, false, expErr)
			},
			err: expErr,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			m := newPermMocks(t)
			if tt.setup != nil {
				tt.setup(m)
			}

			p := usecase.NewPermissionChecker(m.core, m.auth)
			gotIDs, gotAdmin, err := p.GetDirectPermissions(ctx, role)
			if tt.err != nil {
				require.ErrorIs(t, err, tt.err)
			} else {
				require.NoError(t, err)
				require.Equal(t, ids, gotIDs)
				require.Equal(t, isAdmin, gotAdmin)
			}
		})
	}
}

func TestPermissionChecker_Check(t *testing.T) {
	t.Parallel()
	var (
		userCtx  = t.Context()
		id       = uuid.New()
		parentID = uuid.New()
		direct   = []uuid.UUID{uuid.New()}
		expErr   = fmt.Errorf("exp")
	)

	tests := []struct {
		name  string
		role  auth.Role
		call  func(p *usecase.DefaultPermissionChecker) error
		setup func(mock permMocks)
		err   error
	}{
		{
			name: "entity/admin",
			call: func(p *usecase.DefaultPermissionChecker) error {
				return p.CheckEntityPermission(userCtx, id, auth.RoleWrite)
			},
			setup: func(mock permMocks) {
				mock.auth.GetCurrentUserDirectPermissionsMock.Expect(userCtx, auth.RoleWrite).Return(nil, true, nil)
			},
		},
		{
			name: "entity/permitted",
			call: func(p *usecase.DefaultPermissionChecker) error {
				return p.CheckEntityPermission(userCtx, id, auth.RoleWrite)
			},
			setup: func(mock permMocks) {
				mock.auth.GetCurrentUserDirectPermissionsMock.Expect(userCtx, auth.RoleWrite).Return(direct, false, nil)
				mock.core.IsPermittedMock.Expect(userCtx, id, direct, false).Return(true, nil)
			},
		},
		{
			name: "entity/read opens ancestors",
			call: func(p *usecase.DefaultPermissionChecker) error {
				return p.CheckEntityPermission(userCtx, id, auth.RoleRead)
			},
			setup: func(mock permMocks) {
				mock.auth.GetCurrentUserDirectPermissionsMock.Expect(userCtx, auth.RoleRead).Return(direct, false, nil)
				mock.core.IsPermittedMock.Expect(userCtx, id, direct, true).Return(true, nil)
			},
		},
		{
			name: "entity/not permitted",
			call: func(p *usecase.DefaultPermissionChecker) error {
				return p.CheckEntityPermission(userCtx, id, auth.RoleWrite)
			},
			setup: func(mock permMocks) {
				mock.auth.GetCurrentUserDirectPermissionsMock.Expect(userCtx, auth.RoleWrite).Return(direct, false, nil)
				mock.core.IsPermittedMock.Expect(userCtx, id, direct, false).Return(false, nil)
			},
			err: apperr.ErrForbidden(),
		},
		{
			name: "entity/core error",
			call: func(p *usecase.DefaultPermissionChecker) error {
				return p.CheckEntityPermission(userCtx, id, auth.RoleWrite)
			},
			setup: func(mock permMocks) {
				mock.auth.GetCurrentUserDirectPermissionsMock.Expect(userCtx, auth.RoleWrite).Return(direct, false, nil)
				mock.core.IsPermittedMock.Expect(userCtx, id, direct, false).Return(false, expErr)
			},
			err: expErr,
		},
		{
			name: "entity/auth error",
			call: func(p *usecase.DefaultPermissionChecker) error {
				return p.CheckEntityPermission(userCtx, id, auth.RoleWrite)
			},
			setup: func(mock permMocks) {
				mock.auth.GetCurrentUserDirectPermissionsMock.Expect(userCtx, auth.RoleWrite).Return(nil, false, expErr)
			},
			err: expErr,
		},
		{
			name: "parent/admin can use root",
			call: func(p *usecase.DefaultPermissionChecker) error {
				return p.CheckParentPermission(userCtx, auth.RoleWrite, nil, &parentID)
			},
			setup: func(mock permMocks) {
				mock.auth.GetCurrentUserDirectPermissionsMock.Expect(userCtx, auth.RoleWrite).Return(nil, true, nil)
			},
		},
		{
			name: "parent/all permitted",
			call: func(p *usecase.DefaultPermissionChecker) error {
				return p.CheckParentPermission(userCtx, auth.RoleWrite, &parentID, &id)
			},
			setup: func(mock permMocks) {
				mock.auth.GetCurrentUserDirectPermissionsMock.Expect(userCtx, auth.RoleWrite).Return(direct, false, nil)
				mock.core.IsPermittedMock.When(userCtx, parentID, direct, false).Then(true, nil)
				mock.core.IsPermittedMock.When(userCtx, id, direct, false).Then(true, nil)
			},
		},
		{
			name: "parent/one not permitted",
			call: func(p *usecase.DefaultPermissionChecker) error {
				return p.CheckParentPermission(userCtx, auth.RoleWrite, &parentID, &id)
			},
			setup: func(mock permMocks) {
				mock.auth.GetCurrentUserDirectPermissionsMock.Expect(userCtx, auth.RoleWrite).Return(direct, false, nil)
				mock.core.IsPermittedMock.When(userCtx, parentID, direct, false).Then(true, nil)
				mock.core.IsPermittedMock.When(userCtx, id, direct, false).Then(false, nil)
			},
			err: apperr.ErrForbidden(),
		},
		{
			name: "parent/root requires admin",
			call: func(p *usecase.DefaultPermissionChecker) error {
				return p.CheckParentPermission(userCtx, auth.RoleWrite, nil)
			},
			setup: func(mock permMocks) {
				mock.auth.GetCurrentUserDirectPermissionsMock.Expect(userCtx, auth.RoleWrite).Return(direct, false, nil)
			},
			err: apperr.ErrForbidden(),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			m := newPermMocks(t)
			tt.setup(m)

			err := tt.call(usecase.NewPermissionChecker(m.core, m.auth))
			if tt.err != nil {
				require.ErrorIs(t, err, tt.err)
			} else {
				require.NoError(t, err)
			}
		})
	}
}
