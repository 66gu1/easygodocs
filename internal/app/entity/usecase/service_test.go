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
		permissions = usecase.EffectivePermissions{
			IsAdmin: false,
			IDs:     []uuid.UUID{parentID},
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
				mock.perm.GetEffectivePermissionsMock.Expect(ctx, auth.RoleWrite).Return(permissions, nil)
				mock.core.CreateMock.Expect(ctx, req).Return(uuid.New(), nil)
			},
		},
		{
			name: "core.Create error",
			ctx:  ctx,
			setup: func(mock serviceMocks) {
				mock.perm.GetEffectivePermissionsMock.Expect(ctx, auth.RoleWrite).Return(permissions, nil)
				mock.core.CreateMock.Expect(ctx, req).Return(uuid.Nil, expErr)
			},
			err: expErr,
		},
		{
			name: "no user id in context",
			ctx:  t.Context(),
			setup: func(mock serviceMocks) {
				mock.perm.GetEffectivePermissionsMock.Expect(t.Context(), auth.RoleWrite).Return(permissions, nil)
			},
			err: apperr.ErrUnauthorized(),
		},
		{
			name: "no write permission",
			ctx:  ctx,
			setup: func(mock serviceMocks) {
				mock.perm.GetEffectivePermissionsMock.Expect(ctx, auth.RoleWrite).Return(usecase.EffectivePermissions{
					IsAdmin: false,
					IDs:     []uuid.UUID{},
				}, nil)
			},
			err: apperr.ErrForbidden(),
		},
		{
			name: "perm.GetEffectivePermissions error",
			ctx:  ctx,
			setup: func(mock serviceMocks) {
				mock.perm.GetEffectivePermissionsMock.Expect(ctx, auth.RoleWrite).Return(usecase.EffectivePermissions{}, expErr)
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
			IsDraft:  true,
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
			IsDraft:       cmd.IsDraft,
			UserID:        userID,
			ParentChanged: true,
			EntityType:    listItem.Type,
		}
		permissions = usecase.EffectivePermissions{
			IsAdmin: false,
			IDs:     []uuid.UUID{parentID, id, oldParentID},
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
				mock.perm.GetEffectivePermissionsMock.Expect(ctx, auth.RoleWrite).Return(permissions, nil)
				mock.core.GetListItemMock.Expect(ctx, req.ID).Return(listItem, nil)
				mock.core.UpdateMock.Expect(ctx, req).Return(nil)
			},
		},
		{
			name: "core.Update error",
			ctx:  ctx,
			setup: func(mock serviceMocks) {
				mock.perm.GetEffectivePermissionsMock.Expect(ctx, auth.RoleWrite).Return(permissions, nil)
				mock.core.GetListItemMock.Expect(ctx, req.ID).Return(listItem, nil)
				mock.core.UpdateMock.Expect(ctx, req).Return(expErr)
			},
			err: expErr,
		},
		{
			name: "no user id in context",
			ctx:  t.Context(),
			setup: func(mock serviceMocks) {
				mock.perm.GetEffectivePermissionsMock.Expect(t.Context(), auth.RoleWrite).Return(permissions, nil)
				mock.core.GetListItemMock.Expect(t.Context(), req.ID).Return(listItem, nil)
			},
			err: apperr.ErrUnauthorized(),
		},
		{
			name: "no parent permission",
			ctx:  ctx,
			setup: func(mock serviceMocks) {
				mock.perm.GetEffectivePermissionsMock.Expect(ctx, auth.RoleWrite).Return(usecase.EffectivePermissions{
					IsAdmin: false,
					IDs:     []uuid.UUID{id, oldParentID},
				}, nil)
				mock.core.GetListItemMock.Expect(ctx, req.ID).Return(listItem, nil)
			},
			err: apperr.ErrForbidden(),
		},
		{
			name: "no entity permission",
			ctx:  ctx,
			setup: func(mock serviceMocks) {
				mock.perm.GetEffectivePermissionsMock.Expect(ctx, auth.RoleWrite).Return(usecase.EffectivePermissions{
					IsAdmin: false,
					IDs:     []uuid.UUID{parentID, oldParentID},
				}, nil)
			},
			err: apperr.ErrForbidden(),
		},
		{
			name: "core.GetListItem error",
			ctx:  ctx,
			setup: func(mock serviceMocks) {
				mock.perm.GetEffectivePermissionsMock.Expect(ctx, auth.RoleWrite).Return(permissions, nil)
				mock.core.GetListItemMock.Expect(ctx, req.ID).Return(entity.ListItem{}, expErr)
			},
			err: expErr,
		},
		{
			name: "perm.GetEffectivePermissions error",
			ctx:  ctx,
			setup: func(mock serviceMocks) {
				mock.perm.GetEffectivePermissionsMock.Expect(ctx, auth.RoleWrite).Return(usecase.EffectivePermissions{}, expErr)
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

func TestPermissionChecker_GetEffectivePermissions(t *testing.T) {
	t.Parallel()
	var (
		ctx              = t.Context()
		role             = auth.RoleWrite
		ids              = []uuid.UUID{uuid.New(), uuid.New(), uuid.New()}
		adminPermissions = usecase.EffectivePermissions{IsAdmin: true}
		expErr           = fmt.Errorf("exp")
	)
	tests := []struct {
		name  string
		setup func(mock permMocks)
		want  usecase.EffectivePermissions
		err   error
	}{
		{
			name: "ok/admin",
			setup: func(mock permMocks) {
				mock.auth.GetCurrentUserDirectPermissionsMock.Expect(ctx, role).Return(ids, true, nil)
			},
			want: adminPermissions,
		},
		{
			name: "ok/not admin",
			setup: func(mock permMocks) {
				mock.auth.GetCurrentUserDirectPermissionsMock.Expect(ctx, role).Return(ids, false, nil)
				mock.core.GetPermittedIDsMock.Expect(ctx, ids, entity.HierarchyTypeChildrenOnly).Return(ids, nil)
			},
			want: usecase.EffectivePermissions{IsAdmin: false, IDs: ids},
		},
		{
			name: "auth.GetCurrentUserDirectPermissions error",
			setup: func(mock permMocks) {
				mock.auth.GetCurrentUserDirectPermissionsMock.Expect(ctx, role).Return(nil, false, expErr)
			},
			err: expErr,
		},
		{
			name: "core.GetPermittedHierarchy error",
			setup: func(mock permMocks) {
				mock.auth.GetCurrentUserDirectPermissionsMock.Expect(ctx, role).Return(ids, false, nil)
				mock.core.GetPermittedIDsMock.Expect(ctx, ids, entity.HierarchyTypeChildrenOnly).Return(nil, expErr)
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
			got, err := p.GetEffectivePermissions(ctx, role)
			if tt.err != nil {
				require.ErrorIs(t, err, tt.err)
			} else {
				require.NoError(t, err)
				require.Equal(t, tt.want, got)
			}
		})
	}
}

func TestPermissionChecker_CheckEntityPermission(t *testing.T) {
	t.Parallel()
	var (
		ctx    = t.Context()
		id     = uuid.New()
		role   = auth.RoleWrite
		expErr = fmt.Errorf("exp")
	)
	tests := []struct {
		name  string
		setup func(mock permMocks)
		err   error
	}{
		{
			name: "ok",
			setup: func(mock permMocks) {
				mock.auth.GetCurrentUserDirectPermissionsMock.Expect(ctx, role).Return(nil, true, nil)
			},
		},
		{
			name: "no permission",
			setup: func(mock permMocks) {
				mock.auth.GetCurrentUserDirectPermissionsMock.Expect(ctx, role).Return(nil, false, nil)
				mock.core.GetPermittedIDsMock.Expect(ctx, nil, entity.HierarchyTypeChildrenOnly).Return(nil, nil)
			},
			err: apperr.ErrForbidden(),
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
			err := p.CheckEntityPermission(ctx, id, role)
			if tt.err != nil {
				require.ErrorIs(t, err, tt.err)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestEffectivePermissions_CheckID(t *testing.T) {
	t.Parallel()
	id := uuid.New()
	tests := []struct {
		name string
		perm usecase.EffectivePermissions
		err  error
	}{
		{
			name: "is admin",
			perm: usecase.EffectivePermissions{IsAdmin: true},
		},
		{
			name: "has id",
			perm: usecase.EffectivePermissions{IDs: []uuid.UUID{id, uuid.New()}},
		},
		{
			name: "no id",
			perm: usecase.EffectivePermissions{IDs: []uuid.UUID{uuid.New(), uuid.New()}},
			err:  apperr.ErrForbidden(),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			err := tt.perm.CheckID(id)
			if tt.err != nil {
				require.ErrorIs(t, err, tt.err)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestEffectivePermissions_CheckParentIDs(t *testing.T) {
	t.Parallel()
	var (
		id1        = uuid.New()
		id2        = uuid.New()
		reqWithNil = []*uuid.UUID{nil, nil}
		reqWithIDs = []*uuid.UUID{&id1, &id2}
	)
	tests := []struct {
		name string
		req  []*uuid.UUID
		perm usecase.EffectivePermissions
		err  error
	}{
		{
			name: "is admin with parents",
			req:  reqWithIDs,
			perm: usecase.EffectivePermissions{IsAdmin: true},
		},
		{
			name: "is admin with nil parents",
			req:  reqWithNil,
			perm: usecase.EffectivePermissions{IsAdmin: true},
		},
		{
			name: "has all ids",
			req:  reqWithIDs,
			perm: usecase.EffectivePermissions{IDs: []uuid.UUID{id1, id2, uuid.New()}},
		},
		{
			name: "no ids",
			req:  reqWithIDs,
			perm: usecase.EffectivePermissions{IDs: []uuid.UUID{uuid.New(), uuid.New()}},
			err:  apperr.ErrForbidden(),
		},
		{
			name: "no parents",
			req:  reqWithNil,
			perm: usecase.EffectivePermissions{IDs: []uuid.UUID{uuid.New(), uuid.New()}},
			err:  apperr.ErrForbidden(),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			err := tt.perm.CheckParentIDs(tt.req)
			if tt.err != nil {
				require.ErrorIs(t, err, tt.err)
			} else {
				require.NoError(t, err)
			}
		})
	}
}
