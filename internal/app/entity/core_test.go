package entity_test

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/66gu1/easygodocs/internal/app/entity"
	"github.com/66gu1/easygodocs/internal/app/entity/mocks"
	"github.com/66gu1/easygodocs/internal/infrastructure/apperr"
	"github.com/66gu1/easygodocs/internal/infrastructure/contextx"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

//go:generate minimock -o ./mocks -s _mock.go

func Cfg() entity.Config {
	return entity.Config{MaxHierarchyDepth: 1}
}
func TestNewCore(t *testing.T) {
	t.Parallel()

	var (
		repo      = mocks.NewRepositoryMock(t)
		idGen     = mocks.NewIDGeneratorMock(t)
		timeGen   = mocks.NewTimeGeneratorMock(t)
		validator = mocks.NewValidatorMock(t)
	)

	tests := []struct {
		name      string
		repo      entity.Repository
		gen       entity.Generators
		validator entity.Validator
		wantErr   bool
	}{
		{
			name:      "success",
			repo:      repo,
			gen:       entity.Generators{ID: idGen, Time: timeGen},
			validator: validator,
			wantErr:   false,
		},
		{
			name:      "error/nil_repo",
			repo:      nil,
			gen:       entity.Generators{ID: idGen, Time: timeGen},
			validator: validator,
			wantErr:   true,
		},
		{
			name:      "error/nil_id_gen",
			repo:      repo,
			gen:       entity.Generators{ID: nil, Time: timeGen},
			validator: validator,
			wantErr:   true,
		},
		{
			name:      "error/nil_time_gen",
			repo:      repo,
			gen:       entity.Generators{ID: idGen, Time: nil},
			validator: validator,
			wantErr:   true,
		},
		{
			name:      "error/nil_validator",
			repo:      repo,
			gen:       entity.Generators{ID: idGen, Time: timeGen},
			validator: nil,
			wantErr:   true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			_, err := entity.NewCore(tt.repo, tt.gen, tt.validator, entity.Config{MaxHierarchyDepth: 1})
			if tt.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
		})
	}
}

func TestCore_Get(t *testing.T) {
	t.Parallel()

	var (
		ctx  = context.Background()
		id   = uuid.New()
		want = entity.Entity{
			ID:             id,
			Type:           "type",
			Name:           "name",
			Content:        "content",
			ParentID:       &[]uuid.UUID{uuid.New()}[0],
			CreatedBy:      uuid.New(),
			UpdatedBy:      uuid.New(),
			CurrentVersion: &[]int{1}[0],
			CreatedAt:      time.Now(),
			UpdatedAt:      time.Now(),
		}
		expErr = fmt.Errorf("test error")
	)

	tests := []struct {
		name  string
		id    uuid.UUID
		setup func(repo *mocks.RepositoryMock, idGen *mocks.IDGeneratorMock, timeGen *mocks.TimeGeneratorMock)
		want  entity.Entity
		err   error
	}{
		{
			name: "success",
			id:   id,
			setup: func(repo *mocks.RepositoryMock, idGen *mocks.IDGeneratorMock, timeGen *mocks.TimeGeneratorMock) {
				repo.GetMock.Expect(ctx, id).Return(want, nil)
			},
			want: want,
			err:  nil,
		},
		{
			name: "error/nil_id",
			id:   uuid.Nil,
			setup: func(repo *mocks.RepositoryMock, idGen *mocks.IDGeneratorMock, timeGen *mocks.TimeGeneratorMock) {
			},
			err: apperr.ErrNilUUID(entity.FieldEntityID),
		},
		{
			name: "error/repo_error",
			id:   id,
			setup: func(repo *mocks.RepositoryMock, idGen *mocks.IDGeneratorMock, timeGen *mocks.TimeGeneratorMock) {
				repo.GetMock.Expect(ctx, id).Return(entity.Entity{}, expErr)
			},
			err: expErr,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			repo := mocks.NewRepositoryMock(t)
			idGen := mocks.NewIDGeneratorMock(t)
			timeGen := mocks.NewTimeGeneratorMock(t)
			validator := mocks.NewValidatorMock(t)
			if tt.setup != nil {
				tt.setup(repo, idGen, timeGen)
			}
			c, err := entity.NewCore(repo, entity.Generators{ID: idGen, Time: timeGen}, validator, Cfg())
			require.NoError(t, err)

			got, err := c.Get(ctx, tt.id)
			if tt.err != nil {
				require.ErrorIs(t, err, tt.err)
				return
			}
			require.NoError(t, err)
			require.Equal(t, tt.want, got)
		})
	}
}

func TestCore_GetListItem(t *testing.T) {
	t.Parallel()

	var (
		ctx  = context.Background()
		id   = uuid.New()
		want = entity.ListItem{
			ID:       id,
			Type:     "type",
			Name:     "name",
			ParentID: &[]uuid.UUID{uuid.New()}[0],
		}
		expErr = fmt.Errorf("test error")
	)

	tests := []struct {
		name  string
		id    uuid.UUID
		setup func(repo *mocks.RepositoryMock, idGen *mocks.IDGeneratorMock, timeGen *mocks.TimeGeneratorMock)
		want  entity.ListItem
		err   error
	}{
		{
			name: "success",
			id:   id,
			setup: func(repo *mocks.RepositoryMock, idGen *mocks.IDGeneratorMock, timeGen *mocks.TimeGeneratorMock) {
				repo.GetListItemMock.Expect(ctx, id).Return(want, nil)
			},
			want: want,
			err:  nil,
		},
		{
			name: "error/nil_id",
			id:   uuid.Nil,
			setup: func(repo *mocks.RepositoryMock, idGen *mocks.IDGeneratorMock, timeGen *mocks.TimeGeneratorMock) {
			},
			err: apperr.ErrNilUUID(entity.FieldEntityID),
		},
		{
			name: "error/repo_error",
			id:   id,
			setup: func(repo *mocks.RepositoryMock, idGen *mocks.IDGeneratorMock, timeGen *mocks.TimeGeneratorMock) {
				repo.GetListItemMock.Expect(ctx, id).Return(entity.ListItem{}, expErr)
			},
			err: expErr,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			repo := mocks.NewRepositoryMock(t)
			idGen := mocks.NewIDGeneratorMock(t)
			timeGen := mocks.NewTimeGeneratorMock(t)
			validator := mocks.NewValidatorMock(t)
			if tt.setup != nil {
				tt.setup(repo, idGen, timeGen)
			}
			c, err := entity.NewCore(repo, entity.Generators{ID: idGen, Time: timeGen}, validator, Cfg())
			require.NoError(t, err)

			got, err := c.GetListItem(ctx, tt.id)
			if tt.err != nil {
				require.ErrorIs(t, err, tt.err)
				return
			}
			require.NoError(t, err)
			require.Equal(t, tt.want, got)
		})
	}
}

func TestCore_GetTree(t *testing.T) {
	t.Parallel()

	var (
		ctx         = context.Background()
		perm1       = uuid.New()
		perm2       = uuid.New()
		permissions = []uuid.UUID{perm1, perm2}
		want        = entity.Tree{
			{
				ListItem: entity.ListItem{
					ID:       perm1,
					Type:     "type1",
					Name:     "name1",
					ParentID: nil,
				},
				Children: []*entity.Node{
					{
						ListItem: entity.ListItem{
							ID:       uuid.New(),
							Type:     "type1.1",
							Name:     "name1.1",
							ParentID: &[]uuid.UUID{perm1}[0],
						},
					},
				},
			},
			{
				ListItem: entity.ListItem{
					ID:       uuid.New(),
					Type:     "type2",
					Name:     "name2",
					ParentID: nil,
				},
			},
		}
		expErr = fmt.Errorf("test error")
		userID = uuid.New()
		hType  = entity.HierarchyTypeChildrenAndParents
		cfg    = entity.Config{MaxHierarchyDepth: 1}
	)

	ctx = contextx.SetUserID(ctx, userID)
	tests := []struct {
		name    string
		perms   []uuid.UUID
		isAdmin bool
		setup   func(repo *mocks.RepositoryMock, idGen *mocks.IDGeneratorMock, timeGen *mocks.TimeGeneratorMock)
		want    entity.Tree
		err     error
	}{
		{
			name:    "success/is_admin",
			perms:   nil,
			isAdmin: true,
			setup: func(repo *mocks.RepositoryMock, idGen *mocks.IDGeneratorMock, timeGen *mocks.TimeGeneratorMock) {
				repo.GetAllMock.Expect(ctx).Return([]entity.ListItem{
					want[0].ListItem,
					want[0].Children[0].ListItem,
					want[1].ListItem,
				}, nil)
			},
			want: want,
		},
		{
			name:    "success/with_permissions",
			perms:   permissions,
			isAdmin: false,
			setup: func(repo *mocks.RepositoryMock, idGen *mocks.IDGeneratorMock, timeGen *mocks.TimeGeneratorMock) {
				repo.GetHierarchyMock.Expect(ctx, permissions, cfg.MaxHierarchyDepth, &userID, hType).Return([]entity.ListItem{
					want[0].ListItem,
					want[0].Children[0].ListItem,
					want[1].ListItem,
				}, nil)
			},
			want: want,
		},
		{
			name:    "success/no_permissions",
			perms:   nil,
			isAdmin: false,
			want:    entity.Tree{},
		},
		{
			name:    "repo_error/not_admin",
			perms:   permissions,
			isAdmin: false,
			setup: func(repo *mocks.RepositoryMock, idGen *mocks.IDGeneratorMock, timeGen *mocks.TimeGeneratorMock) {
				repo.GetHierarchyMock.Expect(ctx, permissions, cfg.MaxHierarchyDepth, &userID, hType).Return(nil, expErr)
			},
			err: expErr,
		},
		{
			name:    "repo_error/is_admin",
			perms:   nil,
			isAdmin: true,
			setup: func(repo *mocks.RepositoryMock, idGen *mocks.IDGeneratorMock, timeGen *mocks.TimeGeneratorMock) {
				repo.GetAllMock.Expect(ctx).Return(nil, expErr)
			},
			err: expErr,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			repo := mocks.NewRepositoryMock(t)
			idGen := mocks.NewIDGeneratorMock(t)
			timeGen := mocks.NewTimeGeneratorMock(t)
			validator := mocks.NewValidatorMock(t)
			if tt.setup != nil {
				tt.setup(repo, idGen, timeGen)
			}
			c, err := entity.NewCore(repo, entity.Generators{ID: idGen, Time: timeGen}, validator, Cfg())
			require.NoError(t, err)

			got, err := c.GetTree(ctx, tt.perms, tt.isAdmin)
			if tt.err != nil {
				require.ErrorIs(t, err, tt.err)
				return
			}
			require.NoError(t, err)
			require.Equal(t, tt.want, got)
		})
	}
}

func TestCore_GetPermittedIDs(t *testing.T) {
	t.Parallel()

	var (
		ctx         = context.Background()
		permissions = []uuid.UUID{uuid.New(), uuid.New()}
		want        = append(permissions, uuid.New())
		items       = []entity.ListItem{{ID: want[0]}, {ID: want[1]}, {ID: want[2]}}
		userID      = uuid.New()
		hType       = entity.HierarchyTypeChildrenOnly
		cfg         = entity.Config{MaxHierarchyDepth: 7}

		expErr = fmt.Errorf("test error")
	)
	ctx = contextx.SetUserID(ctx, userID)
	tests := []struct {
		name  string
		perms []uuid.UUID
		setup func(repo *mocks.RepositoryMock, idGen *mocks.IDGeneratorMock, timeGen *mocks.TimeGeneratorMock)
		want  []uuid.UUID
		err   error
	}{
		{
			name:  "success",
			perms: permissions,
			setup: func(repo *mocks.RepositoryMock, idGen *mocks.IDGeneratorMock, timeGen *mocks.TimeGeneratorMock) {
				repo.GetHierarchyMock.Expect(ctx, permissions, cfg.MaxHierarchyDepth, &userID, hType).Return(items, nil)
			},
			want: want,
			err:  nil,
		},
		{
			name:  "success/no_permissions",
			perms: nil,
			setup: func(repo *mocks.RepositoryMock, idGen *mocks.IDGeneratorMock, timeGen *mocks.TimeGeneratorMock) {
			},
			want: nil,
			err:  nil,
		},
		{
			name:  "repo_error",
			perms: permissions,
			setup: func(repo *mocks.RepositoryMock, idGen *mocks.IDGeneratorMock, timeGen *mocks.TimeGeneratorMock) {
				repo.GetHierarchyMock.Expect(ctx, permissions, cfg.MaxHierarchyDepth, &userID, hType).Return(nil, expErr)
			},
			err: expErr,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			repo := mocks.NewRepositoryMock(t)
			idGen := mocks.NewIDGeneratorMock(t)
			timeGen := mocks.NewTimeGeneratorMock(t)
			validator := mocks.NewValidatorMock(t)
			if tt.setup != nil {
				tt.setup(repo, idGen, timeGen)
			}
			c, err := entity.NewCore(repo, entity.Generators{ID: idGen, Time: timeGen}, validator, cfg)
			require.NoError(t, err)

			got, err := c.GetPermittedIDs(ctx, tt.perms, hType)
			if tt.err != nil {
				require.ErrorIs(t, err, tt.err)
				return
			}
			require.NoError(t, err)
			require.Equal(t, tt.want, got)
		})
	}
}

func TestCore_GetVersion(t *testing.T) {
	t.Parallel()

	var (
		ctx     = context.Background()
		id      = uuid.New()
		version = 1
		want    = entity.Entity{
			ID:             id,
			Type:           "type",
			Name:           "name",
			Content:        "content",
			ParentID:       &[]uuid.UUID{uuid.New()}[0],
			CreatedBy:      uuid.New(),
			UpdatedBy:      uuid.New(),
			CurrentVersion: &[]int{1}[0],
			CreatedAt:      time.Now(),
			UpdatedAt:      time.Now(),
		}
		expErr = fmt.Errorf("test error")
	)

	tests := []struct {
		name  string
		id    uuid.UUID
		v     int
		setup func(repo *mocks.RepositoryMock, idGen *mocks.IDGeneratorMock, timeGen *mocks.TimeGeneratorMock)
		want  entity.Entity
		err   error
	}{
		{
			name: "success",
			id:   id,
			v:    version,
			setup: func(repo *mocks.RepositoryMock, idGen *mocks.IDGeneratorMock, timeGen *mocks.TimeGeneratorMock) {
				repo.GetVersionMock.Expect(ctx, id, version).Return(want, nil)
			},
			want: want,
			err:  nil,
		},
		{
			name: "error/nil_id",
			id:   uuid.Nil,
			v:    version,
			err:  apperr.ErrNilUUID(entity.FieldEntityID),
		},
		{
			name: "error/invalid_version",
			id:   id,
			v:    0,
			err:  entity.ErrInvalidVersion(),
		},
		{
			name: "error/repo_error",
			id:   id,
			v:    version,
			setup: func(repo *mocks.RepositoryMock, idGen *mocks.IDGeneratorMock, timeGen *mocks.TimeGeneratorMock) {
				repo.GetVersionMock.Expect(ctx, id, version).Return(entity.Entity{}, expErr)
			},
			err: expErr,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			repo := mocks.NewRepositoryMock(t)
			idGen := mocks.NewIDGeneratorMock(t)
			timeGen := mocks.NewTimeGeneratorMock(t)
			validator := mocks.NewValidatorMock(t)
			if tt.setup != nil {
				tt.setup(repo, idGen, timeGen)
			}
			c, err := entity.NewCore(repo, entity.Generators{ID: idGen, Time: timeGen}, validator, Cfg())
			require.NoError(t, err)

			got, err := c.GetVersion(ctx, tt.id, tt.v)
			if tt.err != nil {
				require.ErrorIs(t, err, tt.err)
				return
			}
			require.NoError(t, err)
			require.Equal(t, tt.want, got)
		})
	}
}

func TestCore_GetVersionsList(t *testing.T) {
	t.Parallel()

	var (
		ctx  = context.Background()
		id   = uuid.New()
		want = []entity.Entity{
			{
				ID:             id,
				Type:           "type",
				Name:           "name",
				Content:        "content",
				ParentID:       &[]uuid.UUID{uuid.New()}[0],
				CreatedBy:      uuid.New(),
				UpdatedBy:      uuid.New(),
				CurrentVersion: &[]int{1}[0],
				CreatedAt:      time.Now(),
				UpdatedAt:      time.Now(),
			},
			{
				ID:             id,
				Type:           "type",
				Name:           "name v2",
				Content:        "content v2",
				ParentID:       &[]uuid.UUID{uuid.New()}[0],
				CreatedBy:      uuid.New(),
				UpdatedBy:      uuid.New(),
				CurrentVersion: &[]int{2}[0],
				CreatedAt:      time.Now(),
				UpdatedAt:      time.Now(),
			},
		}
		expErr = fmt.Errorf("test error")
	)

	tests := []struct {
		name  string
		id    uuid.UUID
		setup func(repo *mocks.RepositoryMock, idGen *mocks.IDGeneratorMock, timeGen *mocks.TimeGeneratorMock)
		want  []entity.Entity
		err   error
	}{
		{
			name: "success",
			id:   id,
			setup: func(repo *mocks.RepositoryMock, idGen *mocks.IDGeneratorMock, timeGen *mocks.TimeGeneratorMock) {
				repo.GetVersionsListMock.Expect(ctx, id).Return(want, nil)
			},
			want: want,
			err:  nil,
		},
		{
			name: "error/nil_id",
			id:   uuid.Nil,
			setup: func(repo *mocks.RepositoryMock, idGen *mocks.IDGeneratorMock, timeGen *mocks.TimeGeneratorMock) {
			},
			err: apperr.ErrNilUUID(entity.FieldEntityID),
		},
		{
			name: "error/repo_error",
			id:   id,
			setup: func(repo *mocks.RepositoryMock, idGen *mocks.IDGeneratorMock, timeGen *mocks.TimeGeneratorMock) {
				repo.GetVersionsListMock.Expect(ctx, id).Return(nil, expErr)
			},
			err: expErr,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			repo := mocks.NewRepositoryMock(t)
			idGen := mocks.NewIDGeneratorMock(t)
			timeGen := mocks.NewTimeGeneratorMock(t)
			validator := mocks.NewValidatorMock(t)
			if tt.setup != nil {
				tt.setup(repo, idGen, timeGen)
			}
			c, err := entity.NewCore(repo, entity.Generators{ID: idGen, Time: timeGen}, validator, Cfg())
			require.NoError(t, err)

			got, err := c.GetVersionsList(ctx, tt.id)
			if tt.err != nil {
				require.ErrorIs(t, err, tt.err)
				return
			}
			require.NoError(t, err)
			require.Equal(t, tt.want, got)
		})
	}
}

func TestCore_Create(t *testing.T) {
	t.Parallel()

	var (
		ctx            = context.Background()
		id             = uuid.New()
		userID         = uuid.New()
		now            = time.Now()
		normalizedName = "n_name"
		req            = entity.CreateEntityReq{
			Type:    entity.TypeDepartment,
			Name:    normalizedName,
			Content: "content",
			IsDraft: false,
			UserID:  userID,
		}
		notNormalizedReq = entity.CreateEntityReq{
			Type:    req.Type,
			Name:    " Name ",
			Content: req.Content,
			IsDraft: req.IsDraft,
			UserID:  req.UserID,
		}

		parentID          = uuid.New()
		requestWithParent = entity.CreateEntityReq{
			Type:     req.Type,
			Name:     req.Name,
			Content:  req.Content,
			ParentID: &parentID,
			IsDraft:  true,
			UserID:   req.UserID,
		}

		parent = entity.ListItem{
			ID:       parentID,
			Type:     entity.TypeDepartment,
			Name:     "parent",
			ParentID: nil,
		}
		cfg    = entity.Config{MaxHierarchyDepth: 4}
		list   = []entity.ListItem{parent, {}, {}}
		expErr = fmt.Errorf("test error")
	)

	tests := []struct {
		name   string
		req    entity.CreateEntityReq
		parent entity.ListItem
		setup  func(repo *mocks.RepositoryMock, idGen *mocks.IDGeneratorMock, timeGen *mocks.TimeGeneratorMock, validator *mocks.ValidatorMock)
		err    error
	}{
		{
			name: "success/no_parent/not_draft/normalize",
			req:  notNormalizedReq,
			setup: func(repo *mocks.RepositoryMock, idGen *mocks.IDGeneratorMock, timeGen *mocks.TimeGeneratorMock, validator *mocks.ValidatorMock) {
				validator.NormalizeNameMock.Expect(notNormalizedReq.Name).Return(normalizedName)
				validator.ValidateNameMock.Expect(normalizedName).Return(nil)
				timeGen.NowMock.Expect().Return(now)
				idGen.NewMock.Expect().Return(id, nil)
				repo.CreateMock.Expect(ctx, req, id, now).Return(nil)
			},
		},
		{
			name: "success/with_parent/draft",
			req:  requestWithParent,
			setup: func(repo *mocks.RepositoryMock, idGen *mocks.IDGeneratorMock, timeGen *mocks.TimeGeneratorMock, validator *mocks.ValidatorMock) {
				validator.NormalizeNameMock.Expect(requestWithParent.Name).Return(requestWithParent.Name)
				validator.ValidateNameMock.Expect(requestWithParent.Name).Return(nil)
				repo.GetHierarchyMock.Expect(ctx, []uuid.UUID{parentID}, cfg.MaxHierarchyDepth+1, nil, entity.HierarchyTypeParentsOnly).Return(list, nil)
				timeGen.NowMock.Expect().Return(now)
				idGen.NewMock.Expect().Return(id, nil)
				repo.CreateDraftMock.Expect(ctx, requestWithParent, id).Return(nil)
			},
		},
		{
			name: "error/validation/nil_user_id",
			req: entity.CreateEntityReq{
				Type:    req.Type,
				Name:    req.Name,
				Content: req.Content,
				IsDraft: req.IsDraft,
				UserID:  uuid.Nil,
			},
			err: apperr.ErrNilUUID(entity.FieldUserID),
		},
		{
			name: "error/validation/invalid_type",
			req: entity.CreateEntityReq{
				Type:    "invalid",
				Name:    req.Name,
				Content: req.Content,
				IsDraft: req.IsDraft,
				UserID:  req.UserID,
			},
			err: entity.ErrInvalidType(),
		},
		{
			name: "error/validation/invalid_name",
			req:  req,
			setup: func(repo *mocks.RepositoryMock, idGen *mocks.IDGeneratorMock, timeGen *mocks.TimeGeneratorMock, validator *mocks.ValidatorMock) {
				validator.NormalizeNameMock.Expect(req.Name).Return(normalizedName)
				validator.ValidateNameMock.Expect(normalizedName).Return(expErr)
			},
			err: expErr,
		},
		{
			name: "error/repo/get_parent",
			req:  requestWithParent,
			setup: func(repo *mocks.RepositoryMock, idGen *mocks.IDGeneratorMock, timeGen *mocks.TimeGeneratorMock, validator *mocks.ValidatorMock) {
				validator.NormalizeNameMock.Expect(requestWithParent.Name).Return(requestWithParent.Name)
				validator.ValidateNameMock.Expect(requestWithParent.Name).Return(nil)
				repo.GetHierarchyMock.Expect(ctx, []uuid.UUID{parentID}, cfg.MaxHierarchyDepth+1, nil, entity.HierarchyTypeParentsOnly).Return(nil, expErr)
			},
			err: expErr,
		},
		{
			name: "error/max hierarchy depth exceeded",
			req:  requestWithParent,
			setup: func(repo *mocks.RepositoryMock, idGen *mocks.IDGeneratorMock, timeGen *mocks.TimeGeneratorMock, validator *mocks.ValidatorMock) {
				validator.NormalizeNameMock.Expect(requestWithParent.Name).Return(requestWithParent.Name)
				validator.ValidateNameMock.Expect(requestWithParent.Name).Return(nil)
				repo.GetHierarchyMock.Expect(ctx, []uuid.UUID{parentID}, cfg.MaxHierarchyDepth+1, nil, entity.HierarchyTypeParentsOnly).Return([]entity.ListItem{{}, {}, {}, {}}, nil)
			},
			err: entity.ErrMaxHierarchyDepthExceeded(cfg.MaxHierarchyDepth),
		},
		{
			name: "error/validation/parent_not_found",
			req:  requestWithParent,
			setup: func(repo *mocks.RepositoryMock, idGen *mocks.IDGeneratorMock, timeGen *mocks.TimeGeneratorMock, validator *mocks.ValidatorMock) {
				validator.NormalizeNameMock.Expect(requestWithParent.Name).Return(requestWithParent.Name)
				validator.ValidateNameMock.Expect(requestWithParent.Name).Return(nil)
				repo.GetHierarchyMock.Expect(ctx, []uuid.UUID{parentID}, cfg.MaxHierarchyDepth+1, nil, entity.HierarchyTypeParentsOnly).Return([]entity.ListItem{}, nil)
			},
			err: entity.ErrParentNotFound(),
		},
		{
			name: "error/validation/incompatible_parent_type",
			req:  requestWithParent,
			setup: func(repo *mocks.RepositoryMock, idGen *mocks.IDGeneratorMock, timeGen *mocks.TimeGeneratorMock, validator *mocks.ValidatorMock) {
				validator.NormalizeNameMock.Expect(requestWithParent.Name).Return(requestWithParent.Name)
				validator.ValidateNameMock.Expect(requestWithParent.Name).Return(nil)
				repo.GetHierarchyMock.Expect(ctx, []uuid.UUID{parentID}, cfg.MaxHierarchyDepth+1, nil, entity.HierarchyTypeParentsOnly).Return([]entity.ListItem{
					{
						ID:       parentID,
						Type:     entity.TypeArticle,
						Name:     "parent",
						ParentID: nil,
					},
				}, nil)
			},
			err: entity.ErrIncompatibleParentType(),
		},
		{
			name: "error/validation/parent_required",
			req: entity.CreateEntityReq{
				Type:    entity.TypeArticle,
				Name:    req.Name,
				Content: req.Content,
				IsDraft: req.IsDraft,
				UserID:  req.UserID,
			},
			setup: func(repo *mocks.RepositoryMock, idGen *mocks.IDGeneratorMock, timeGen *mocks.TimeGeneratorMock, validator *mocks.ValidatorMock) {
				validator.NormalizeNameMock.Expect(req.Name).Return(normalizedName)
				validator.ValidateNameMock.Expect(normalizedName).Return(nil)
			},
			err: entity.ErrParentRequired(),
		},
		{
			name: "error/id_gen",
			req:  req,
			setup: func(repo *mocks.RepositoryMock, idGen *mocks.IDGeneratorMock, timeGen *mocks.TimeGeneratorMock, validator *mocks.ValidatorMock) {
				validator.NormalizeNameMock.Expect(req.Name).Return(normalizedName)
				validator.ValidateNameMock.Expect(normalizedName).Return(nil)
				timeGen.NowMock.Expect().Return(now)
				idGen.NewMock.Expect().Return(uuid.UUID{}, expErr)
			},
			err: expErr,
		},
		{
			name: "error/repo/create",
			req:  req,
			setup: func(repo *mocks.RepositoryMock, idGen *mocks.IDGeneratorMock, timeGen *mocks.TimeGeneratorMock, validator *mocks.ValidatorMock) {
				validator.NormalizeNameMock.Expect(req.Name).Return(normalizedName)
				validator.ValidateNameMock.Expect(normalizedName).Return(nil)
				timeGen.NowMock.Expect().Return(now)
				idGen.NewMock.Expect().Return(id, nil)
				repo.CreateMock.Expect(ctx, req, id, now).Return(expErr)
			},
			err: expErr,
		},
		{
			name: "error/repo/create_draft",
			req:  requestWithParent,
			setup: func(repo *mocks.RepositoryMock, idGen *mocks.IDGeneratorMock, timeGen *mocks.TimeGeneratorMock, validator *mocks.ValidatorMock) {
				validator.NormalizeNameMock.Expect(requestWithParent.Name).Return(requestWithParent.Name)
				validator.ValidateNameMock.Expect(requestWithParent.Name).Return(nil)
				repo.GetHierarchyMock.Expect(ctx, []uuid.UUID{parentID}, cfg.MaxHierarchyDepth+1, nil, entity.HierarchyTypeParentsOnly).Return(list, nil)
				timeGen.NowMock.Expect().Return(now)
				idGen.NewMock.Expect().Return(id, nil)
				repo.CreateDraftMock.Expect(ctx, requestWithParent, id).Return(expErr)
			},
			err: expErr,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			repo := mocks.NewRepositoryMock(t)
			idGen := mocks.NewIDGeneratorMock(t)
			timeGen := mocks.NewTimeGeneratorMock(t)
			validator := mocks.NewValidatorMock(t)
			if tt.setup != nil {
				tt.setup(repo, idGen, timeGen, validator)
			}
			c, err := entity.NewCore(repo, entity.Generators{ID: idGen, Time: timeGen}, validator, cfg)
			require.NoError(t, err)

			gotID, err := c.Create(ctx, tt.req)
			if tt.err != nil {
				require.ErrorIs(t, err, tt.err)
				return
			}
			require.NoError(t, err)
			require.Equal(t, id, gotID)
		})
	}
}

func TestCore_Update(t *testing.T) {
	t.Parallel()

	var (
		ctx            = context.Background()
		id             = uuid.MustParse("e5fb927e-71e2-4e7f-920c-e9a5863c5399")
		userID         = uuid.New()
		now            = time.Now()
		normalizedName = "n_name"
		req            = entity.UpdateEntityReq{
			ID:      id,
			Name:    normalizedName,
			Content: "content",
			IsDraft: false,
			UserID:  userID,
		}
		notNormalizedReq = entity.UpdateEntityReq{
			ID:      req.ID,
			Name:    " Name ",
			Content: req.Content,
			IsDraft: req.IsDraft,
			UserID:  req.UserID,
		}
		reqParentRemoved = entity.UpdateEntityReq{
			ID:            req.ID,
			Name:          req.Name,
			Content:       req.Content,
			ParentID:      nil,
			ParentChanged: true,
			IsDraft:       true,
			UserID:        req.UserID,
			EntityType:    entity.TypeArticle,
		}
		parentID         = uuid.MustParse("c4abc05f-91f6-43ca-97b2-1cf4f7de0978")
		reqParentChanged = entity.UpdateEntityReq{
			ID:            req.ID,
			Name:          req.Name,
			Content:       req.Content,
			ParentID:      &parentID,
			IsDraft:       true,
			UserID:        req.UserID,
			ParentChanged: true,
			EntityType:    entity.TypeDepartment,
		}
		parentItem = entity.ListItem{
			ID:       parentID,
			Type:     entity.TypeDepartment,
			Name:     "parent",
			ParentID: nil,
		}
		parentList = []entity.ListItem{parentItem, {}, {}}
		cfg        = entity.Config{MaxHierarchyDepth: 5}
		expErr     = fmt.Errorf("test error")
	)

	tests := []struct {
		name   string
		req    entity.UpdateEntityReq
		parent entity.ListItem
		setup  func(repo *mocks.RepositoryMock, idGen *mocks.IDGeneratorMock, timeGen *mocks.TimeGeneratorMock, validator *mocks.ValidatorMock)
		err    error
	}{
		{
			name: "success/parent_not_changed/not_draft/normalize",
			req:  notNormalizedReq,
			setup: func(repo *mocks.RepositoryMock, idGen *mocks.IDGeneratorMock, timeGen *mocks.TimeGeneratorMock, validator *mocks.ValidatorMock) {
				validator.NormalizeNameMock.Expect(notNormalizedReq.Name).Return(normalizedName)
				validator.ValidateNameMock.Expect(normalizedName).Return(nil)
				timeGen.NowMock.Expect().Return(now)
				repo.UpdateMock.Expect(ctx, req, now).Return(nil)
			},
		},
		{
			name: "success/parent_changed/draft",
			req:  reqParentChanged,
			setup: func(repo *mocks.RepositoryMock, idGen *mocks.IDGeneratorMock, timeGen *mocks.TimeGeneratorMock, validator *mocks.ValidatorMock) {
				validator.NormalizeNameMock.Expect(reqParentChanged.Name).Return(reqParentChanged.Name)
				validator.ValidateNameMock.Expect(reqParentChanged.Name).Return(nil)
				repo.GetHierarchyMock.When(ctx, []uuid.UUID{parentID}, cfg.MaxHierarchyDepth+1, nil, entity.HierarchyTypeParentsOnly).Then(parentList, nil)
				repo.GetHierarchyMock.When(ctx, []uuid.UUID{id}, cfg.MaxHierarchyDepth+1, nil, entity.HierarchyTypeChildrenOnly).Then(nil, nil)
				repo.UpdateDraftMock.Expect(ctx, reqParentChanged).Return(nil)
			},
		},
		{
			name: "error/validation/nil_id",
			req: entity.UpdateEntityReq{
				ID:      uuid.Nil,
				Name:    req.Name,
				Content: req.Content,
				IsDraft: req.IsDraft,
				UserID:  req.UserID,
			},
			err: apperr.ErrNilUUID(entity.FieldEntityID),
		},
		{
			name: "error/validation/nil_user_id",
			req: entity.UpdateEntityReq{
				ID:      req.ID,
				Name:    req.Name,
				Content: req.Content,
				IsDraft: req.IsDraft,
				UserID:  uuid.Nil,
			},
			err: apperr.ErrNilUUID(entity.FieldUserID),
		},
		{
			name: "error/validation/invalid_name",
			req:  req,
			setup: func(repo *mocks.RepositoryMock, idGen *mocks.IDGeneratorMock, timeGen *mocks.TimeGeneratorMock, validator *mocks.ValidatorMock) {
				validator.NormalizeNameMock.Expect(req.Name).Return(normalizedName)
				validator.ValidateNameMock.Expect(normalizedName).Return(expErr)
			},
			err: expErr,
		},
		{
			name: "error/parent_changed/id == parent_id",
			req: entity.UpdateEntityReq{
				ID:            id,
				Name:          req.Name,
				Content:       req.Content,
				ParentID:      &id,
				ParentChanged: true,
				IsDraft:       req.IsDraft,
				UserID:        req.UserID,
				EntityType:    entity.TypeDepartment,
			},
			setup: func(repo *mocks.RepositoryMock, idGen *mocks.IDGeneratorMock, timeGen *mocks.TimeGeneratorMock, validator *mocks.ValidatorMock) {
				validator.NormalizeNameMock.Expect(req.Name).Return(reqParentRemoved.Name)
				validator.ValidateNameMock.Expect(req.Name).Return(nil)
			},
			err: entity.ErrParentCycle(),
		},
		{
			name: "error/parent_changed/parent cycle",
			req:  reqParentChanged,
			setup: func(repo *mocks.RepositoryMock, idGen *mocks.IDGeneratorMock, timeGen *mocks.TimeGeneratorMock, validator *mocks.ValidatorMock) {
				validator.NormalizeNameMock.Expect(reqParentChanged.Name).Return(reqParentChanged.Name)
				validator.ValidateNameMock.Expect(reqParentChanged.Name).Return(nil)
				repo.GetHierarchyMock.When(ctx, []uuid.UUID{parentID}, cfg.MaxHierarchyDepth+1, nil, entity.HierarchyTypeParentsOnly).Then([]entity.ListItem{
					{
						ID:   req.ID,
						Type: entity.TypeDepartment,
						Name: "child",
					},
				}, nil)
			},
			err: entity.ErrParentCycle(),
		},
		{
			name: "error/parent_changed/parent not found",
			req:  reqParentChanged,
			setup: func(repo *mocks.RepositoryMock, idGen *mocks.IDGeneratorMock, timeGen *mocks.TimeGeneratorMock, validator *mocks.ValidatorMock) {
				validator.NormalizeNameMock.Expect(reqParentChanged.Name).Return(reqParentChanged.Name)
				validator.ValidateNameMock.Expect(reqParentChanged.Name).Return(nil)
				repo.GetHierarchyMock.When(ctx, []uuid.UUID{parentID}, cfg.MaxHierarchyDepth+1, nil, entity.HierarchyTypeParentsOnly).Then([]entity.ListItem{}, nil)
			},
			err: entity.ErrParentNotFound(),
		},
		{
			name: "error/parent_changed/max hierarchy depth exceeded",
			req:  reqParentChanged,
			setup: func(repo *mocks.RepositoryMock, idGen *mocks.IDGeneratorMock, timeGen *mocks.TimeGeneratorMock, validator *mocks.ValidatorMock) {
				validator.NormalizeNameMock.Expect(reqParentChanged.Name).Return(reqParentChanged.Name)
				validator.ValidateNameMock.Expect(reqParentChanged.Name).Return(nil)
				repo.GetHierarchyMock.When(ctx, []uuid.UUID{parentID}, cfg.MaxHierarchyDepth+1, nil, entity.HierarchyTypeParentsOnly).Then(parentList, nil)
				repo.GetHierarchyMock.When(ctx, []uuid.UUID{id}, cfg.MaxHierarchyDepth+1, nil, entity.HierarchyTypeChildrenOnly).Then([]entity.ListItem{{Depth: 3}}, nil)
			},
			err: entity.ErrMaxHierarchyDepthExceeded(cfg.MaxHierarchyDepth),
		},
		{
			name: "error/repo/get_children",
			req:  reqParentChanged,
			setup: func(repo *mocks.RepositoryMock, idGen *mocks.IDGeneratorMock, timeGen *mocks.TimeGeneratorMock, validator *mocks.ValidatorMock) {
				validator.NormalizeNameMock.Expect(reqParentRemoved.Name).Return(reqParentRemoved.Name)
				validator.ValidateNameMock.Expect(reqParentRemoved.Name).Return(nil)
				repo.GetHierarchyMock.Expect(ctx, []uuid.UUID{parentID}, cfg.MaxHierarchyDepth+1, nil, entity.HierarchyTypeParentsOnly).Return(nil, expErr)
			},
			err: expErr,
		},
		{
			name: "error/repo/get_parents",
			req:  reqParentChanged,
			setup: func(repo *mocks.RepositoryMock, idGen *mocks.IDGeneratorMock, timeGen *mocks.TimeGeneratorMock, validator *mocks.ValidatorMock) {
				validator.NormalizeNameMock.Expect(req.Name).Return(req.Name)
				validator.ValidateNameMock.Expect(req.Name).Return(nil)
				repo.GetHierarchyMock.When(ctx, []uuid.UUID{parentID}, cfg.MaxHierarchyDepth+1, nil, entity.HierarchyTypeParentsOnly).Then(parentList, nil)
				repo.GetHierarchyMock.When(ctx, []uuid.UUID{id}, cfg.MaxHierarchyDepth+1, nil, entity.HierarchyTypeChildrenOnly).Then(nil, expErr)
			},
			err: expErr,
		},
		{
			name: "error/validation/incompatible_parent_type",
			req:  reqParentChanged,
			setup: func(repo *mocks.RepositoryMock, idGen *mocks.IDGeneratorMock, timeGen *mocks.TimeGeneratorMock, validator *mocks.ValidatorMock) {
				validator.NormalizeNameMock.Expect(req.Name).Return(req.Name)
				validator.ValidateNameMock.Expect(req.Name).Return(nil)
				repo.GetHierarchyMock.Expect(ctx, []uuid.UUID{parentID}, cfg.MaxHierarchyDepth+1, nil, entity.HierarchyTypeParentsOnly).Return([]entity.ListItem{
					{
						ID:       parentID,
						Type:     entity.TypeArticle,
						Name:     "parent",
						ParentID: nil,
					},
				}, nil)
			},
			err: entity.ErrIncompatibleParentType(),
		},
		{
			name: "error/validation/parent_required",
			req:  reqParentRemoved,
			setup: func(repo *mocks.RepositoryMock, idGen *mocks.IDGeneratorMock, timeGen *mocks.TimeGeneratorMock, validator *mocks.ValidatorMock) {
				validator.NormalizeNameMock.Expect(req.Name).Return(req.Name)
				validator.ValidateNameMock.Expect(req.Name).Return(nil)
			},
			err: entity.ErrParentRequired(),
		},
		{
			name: "error/update_draft/has_children",
			req: entity.UpdateEntityReq{
				ID:      req.ID,
				UserID:  userID,
				Name:    reqParentChanged.Name,
				IsDraft: reqParentChanged.IsDraft,
			},
			setup: func(repo *mocks.RepositoryMock, idGen *mocks.IDGeneratorMock, timeGen *mocks.TimeGeneratorMock, validator *mocks.ValidatorMock) {
				validator.NormalizeNameMock.Expect(reqParentRemoved.Name).Return(reqParentRemoved.Name)
				validator.ValidateNameMock.Expect(reqParentRemoved.Name).Return(nil)
				repo.GetHierarchyMock.When(ctx, []uuid.UUID{id}, 2, nil, entity.HierarchyTypeChildrenOnly).Then([]entity.ListItem{{}, {}}, nil)
			},
			err: entity.ErrCannotDraftEntityWithChildren(),
		},
		{
			name: "error/update_draft/repo/GetHierarchy",
			req: entity.UpdateEntityReq{
				ID:      req.ID,
				UserID:  userID,
				Name:    reqParentChanged.Name,
				IsDraft: reqParentChanged.IsDraft,
			},
			setup: func(repo *mocks.RepositoryMock, idGen *mocks.IDGeneratorMock, timeGen *mocks.TimeGeneratorMock, validator *mocks.ValidatorMock) {
				validator.NormalizeNameMock.Expect(reqParentRemoved.Name).Return(reqParentRemoved.Name)
				validator.ValidateNameMock.Expect(reqParentRemoved.Name).Return(nil)
				repo.GetHierarchyMock.When(ctx, []uuid.UUID{id}, 2, nil, entity.HierarchyTypeChildrenOnly).Then(nil, expErr)
			},
			err: expErr,
		},
		{
			name: "error/repo/update",
			req:  req,
			setup: func(repo *mocks.RepositoryMock, idGen *mocks.IDGeneratorMock, timeGen *mocks.TimeGeneratorMock, validator *mocks.ValidatorMock) {
				validator.NormalizeNameMock.Expect(req.Name).Return(normalizedName)
				validator.ValidateNameMock.Expect(normalizedName).Return(nil)
				timeGen.NowMock.Expect().Return(now)
				repo.UpdateMock.Expect(ctx, req, now).Return(expErr)
			},
			err: expErr,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			repo := mocks.NewRepositoryMock(t)
			idGen := mocks.NewIDGeneratorMock(t)
			timeGen := mocks.NewTimeGeneratorMock(t)
			validator := mocks.NewValidatorMock(t)
			if tt.setup != nil {
				tt.setup(repo, idGen, timeGen, validator)
			}
			c, err := entity.NewCore(repo, entity.Generators{ID: idGen, Time: timeGen}, validator, cfg)
			require.NoError(t, err)

			err = c.Update(ctx, tt.req)
			if tt.err != nil {
				require.ErrorIs(t, err, tt.err)
				return
			}
			require.NoError(t, err)
		})
	}
}

func TestCore_Delete(t *testing.T) {
	t.Parallel()

	var (
		ctx = context.Background()
		id  = uuid.New()

		ids    = []uuid.UUID{id, uuid.New(), uuid.New()}
		list   = []entity.ListItem{{ID: id}, {ID: ids[1]}, {ID: ids[2]}}
		cfg    = entity.Config{MaxHierarchyDepth: 5}
		expErr = fmt.Errorf("test error")
	)

	tests := []struct {
		name  string
		setup func(repo *mocks.RepositoryMock, timeGen *mocks.TimeGeneratorMock)
		err   error
	}{
		{
			name: "success",
			setup: func(repo *mocks.RepositoryMock, timeGen *mocks.TimeGeneratorMock) {
				repo.GetHierarchyMock.Expect(ctx, []uuid.UUID{id}, cfg.MaxHierarchyDepth+1, nil, entity.HierarchyTypeChildrenOnly).Return(list, nil)
				repo.DeleteMock.Expect(ctx, ids).Return(nil)
			},
		},
		{
			name: "error/repo/GetHierarchyMock",
			setup: func(repo *mocks.RepositoryMock, timeGen *mocks.TimeGeneratorMock) {
				repo.GetHierarchyMock.Expect(ctx, []uuid.UUID{id}, cfg.MaxHierarchyDepth+1, nil, entity.HierarchyTypeChildrenOnly).Return(nil, expErr)
			},
			err: expErr,
		},
		{
			name: "error/repo/Delete",
			setup: func(repo *mocks.RepositoryMock, timeGen *mocks.TimeGeneratorMock) {
				repo.GetHierarchyMock.Expect(ctx, []uuid.UUID{id}, cfg.MaxHierarchyDepth+1, nil, entity.HierarchyTypeChildrenOnly).Return(list, nil)
				repo.DeleteMock.Expect(ctx, ids).Return(expErr)
			},
			err: expErr,
		},
		{
			name: "error/not found",
			setup: func(repo *mocks.RepositoryMock, timeGen *mocks.TimeGeneratorMock) {
				repo.GetHierarchyMock.Expect(ctx, []uuid.UUID{id}, cfg.MaxHierarchyDepth+1, nil, entity.HierarchyTypeChildrenOnly).Return([]entity.ListItem{}, nil)
			},
			err: entity.ErrEntityNotFound(),
		},
		{
			name: "error/max hierarchy depth exceeded",
			setup: func(repo *mocks.RepositoryMock, timeGen *mocks.TimeGeneratorMock) {
				repo.GetHierarchyMock.Expect(ctx, []uuid.UUID{id}, cfg.MaxHierarchyDepth+1, nil, entity.HierarchyTypeChildrenOnly).Return([]entity.ListItem{{Depth: 6}}, nil)
			},
			err: entity.ErrMaxHierarchyDepthExceeded(cfg.MaxHierarchyDepth),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			repo := mocks.NewRepositoryMock(t)
			timeGen := mocks.NewTimeGeneratorMock(t)
			idGen := mocks.NewIDGeneratorMock(t)
			validator := mocks.NewValidatorMock(t)
			if tt.setup != nil {
				tt.setup(repo, timeGen)
			}
			c, err := entity.NewCore(repo, entity.Generators{Time: timeGen, ID: idGen}, validator, cfg)
			require.NoError(t, err)

			err = c.Delete(ctx, id)
			if tt.err != nil {
				require.ErrorIs(t, err, tt.err)
				return
			}
			require.NoError(t, err)
		})
	}
}

func TestNewValidator(t *testing.T) {
	t.Parallel()

	cfg := entity.ValidationConfig{
		MaxNameLength: 50,
	}
	_, err := entity.NewValidator(cfg)
	require.NoError(t, err)

	cfg.MaxNameLength = 0
	_, err = entity.NewValidator(cfg)
	require.Error(t, err)
}

func TestValidator_NormalizeName(t *testing.T) {
	t.Parallel()
	validator, err := entity.NewValidator(entity.ValidationConfig{MaxNameLength: 50})
	require.NoError(t, err)

	require.Equal(t, "name", validator.NormalizeName(" name "))
}

func TestValidator_ValidateName(t *testing.T) {
	t.Parallel()
	validator, err := entity.NewValidator(entity.ValidationConfig{MaxNameLength: 10})
	require.NoError(t, err)

	tests := []struct {
		name string
		err  error
	}{
		{
			name: "valid",
			err:  nil,
		},
		{
			name: "",
			err:  entity.ErrNameRequired(),
		},
		{
			name: "a_very_long_name_exceeding_the_maximum_length_set_in_validation_config",
			err:  entity.ErrNameTooLong(10),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			err := validator.ValidateName(tt.name)
			if tt.err != nil {
				require.ErrorIs(t, err, tt.err)
				return
			}
			require.NoError(t, err)
		})
	}
}
