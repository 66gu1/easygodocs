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

// noTx runs the function without a transaction, so mocks receive the original ctx.
type noTx struct{}

func (noTx) WithinTx(ctx context.Context, fn func(ctx context.Context) error) error {
	return fn(ctx)
}

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
		tx        entity.TxManager
		gen       entity.Generators
		validator entity.Validator
		wantErr   bool
	}{
		{
			name:      "success",
			repo:      repo,
			tx:        noTx{},
			gen:       entity.Generators{ID: idGen, Time: timeGen},
			validator: validator,
			wantErr:   false,
		},
		{
			name:      "error/nil_repo",
			repo:      nil,
			tx:        noTx{},
			gen:       entity.Generators{ID: idGen, Time: timeGen},
			validator: validator,
			wantErr:   true,
		},
		{
			name:      "error/nil_tx",
			repo:      repo,
			tx:        nil,
			gen:       entity.Generators{ID: idGen, Time: timeGen},
			validator: validator,
			wantErr:   true,
		},
		{
			name:      "error/nil_id_gen",
			repo:      repo,
			tx:        noTx{},
			gen:       entity.Generators{ID: nil, Time: timeGen},
			validator: validator,
			wantErr:   true,
		},
		{
			name:      "error/nil_time_gen",
			repo:      repo,
			tx:        noTx{},
			gen:       entity.Generators{ID: idGen, Time: nil},
			validator: validator,
			wantErr:   true,
		},
		{
			name:      "error/nil_validator",
			repo:      repo,
			tx:        noTx{},
			gen:       entity.Generators{ID: idGen, Time: timeGen},
			validator: nil,
			wantErr:   true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			_, err := entity.NewCore(tt.repo, tt.tx, tt.gen, tt.validator, entity.Config{MaxHierarchyDepth: 1})
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
			c, err := entity.NewCore(repo, noTx{}, entity.Generators{ID: idGen, Time: timeGen}, validator, Cfg())
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
			c, err := entity.NewCore(repo, noTx{}, entity.Generators{ID: idGen, Time: timeGen}, validator, Cfg())
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
			c, err := entity.NewCore(repo, noTx{}, entity.Generators{ID: idGen, Time: timeGen}, validator, Cfg())
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

func TestCore_IsPermitted(t *testing.T) {
	t.Parallel()

	var (
		// root -> parent -> id
		root     = uuid.New()
		parent   = uuid.New()
		id       = uuid.New()
		other    = uuid.New()
		path     = []entity.ListItem{{ID: id}, {ID: parent}, {ID: root}}
		userID   = uuid.New()
		ctx      = contextx.SetUserID(context.Background(), userID)
		cfg      = entity.Config{MaxHierarchyDepth: 7}
		expErr   = fmt.Errorf("test error")
		parents  = entity.HierarchyTypeParentsOnly
		pathCall = func(repo *mocks.RepositoryMock, items []entity.ListItem, err error) {
			repo.GetHierarchyMock.When(ctx, []uuid.UUID{id}, cfg.MaxHierarchyDepth, &userID, parents).Then(items, err)
		}
	)

	tests := []struct {
		name          string
		direct        []uuid.UUID
		withAncestors bool
		setup         func(repo *mocks.RepositoryMock)
		want          bool
		err           error
	}{
		{
			name:   "permission on the entity",
			direct: []uuid.UUID{id},
			setup:  func(repo *mocks.RepositoryMock) { pathCall(repo, path, nil) },
			want:   true,
		},
		{
			name:   "permission inherited from an ancestor",
			direct: []uuid.UUID{other, root},
			setup:  func(repo *mocks.RepositoryMock) { pathCall(repo, path, nil) },
			want:   true,
		},
		{
			name:   "no permission on the path",
			direct: []uuid.UUID{other},
			setup:  func(repo *mocks.RepositoryMock) { pathCall(repo, path, nil) },
			want:   false,
		},
		{
			name:   "entity is not visible",
			direct: []uuid.UUID{id},
			setup:  func(repo *mocks.RepositoryMock) { pathCall(repo, nil, nil) },
			want:   false,
		},
		{
			name:          "with ancestors: entity is an ancestor of a permitted one",
			direct:        []uuid.UUID{other},
			withAncestors: true,
			setup: func(repo *mocks.RepositoryMock) {
				pathCall(repo, path, nil)
				repo.GetHierarchyMock.When(ctx, []uuid.UUID{other}, cfg.MaxHierarchyDepth, &userID, parents).
					Then([]entity.ListItem{{ID: other}, {ID: id}, {ID: parent}}, nil)
			},
			want: true,
		},
		{
			name:          "with ancestors: entity is not an ancestor of a permitted one",
			direct:        []uuid.UUID{other},
			withAncestors: true,
			setup: func(repo *mocks.RepositoryMock) {
				pathCall(repo, path, nil)
				repo.GetHierarchyMock.When(ctx, []uuid.UUID{other}, cfg.MaxHierarchyDepth, &userID, parents).
					Then([]entity.ListItem{{ID: other}}, nil)
			},
			want: false,
		},
		{
			name:   "no direct permissions",
			direct: nil,
			want:   false,
		},
		{
			name:   "repo error/path",
			direct: []uuid.UUID{id},
			setup:  func(repo *mocks.RepositoryMock) { pathCall(repo, nil, expErr) },
			err:    expErr,
		},
		{
			name:          "repo error/ancestors",
			direct:        []uuid.UUID{other},
			withAncestors: true,
			setup: func(repo *mocks.RepositoryMock) {
				pathCall(repo, path, nil)
				repo.GetHierarchyMock.When(ctx, []uuid.UUID{other}, cfg.MaxHierarchyDepth, &userID, parents).Then(nil, expErr)
			},
			err: expErr,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			repo := mocks.NewRepositoryMock(t)
			if tt.setup != nil {
				tt.setup(repo)
			}
			c, err := entity.NewCore(repo, noTx{}, entity.Generators{ID: mocks.NewIDGeneratorMock(t), Time: mocks.NewTimeGeneratorMock(t)}, mocks.NewValidatorMock(t), cfg)
			require.NoError(t, err)

			got, err := c.IsPermitted(ctx, id, tt.direct, tt.withAncestors)
			if tt.err != nil {
				require.ErrorIs(t, err, tt.err)
				return
			}
			require.NoError(t, err)
			require.Equal(t, tt.want, got)
		})
	}

	t.Run("no user in context", func(t *testing.T) {
		t.Parallel()
		c, err := entity.NewCore(mocks.NewRepositoryMock(t), noTx{}, entity.Generators{ID: mocks.NewIDGeneratorMock(t), Time: mocks.NewTimeGeneratorMock(t)}, mocks.NewValidatorMock(t), cfg)
		require.NoError(t, err)

		_, err = c.IsPermitted(context.Background(), id, []uuid.UUID{id}, false)
		require.ErrorIs(t, err, apperr.ErrUnauthorized())
	})
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
			c, err := entity.NewCore(repo, noTx{}, entity.Generators{ID: idGen, Time: timeGen}, validator, Cfg())
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
			c, err := entity.NewCore(repo, noTx{}, entity.Generators{ID: idGen, Time: timeGen}, validator, Cfg())
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

		version = 1
		parent  = entity.ListItem{
			ID:             parentID,
			Type:           entity.TypeDepartment,
			Name:           "parent",
			ParentID:       nil,
			CurrentVersion: &version,
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
				repo.LockHierarchyMock.Expect(ctx).Return(nil)
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
				repo.LockHierarchyMock.Expect(ctx).Return(nil)
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
				repo.LockHierarchyMock.Expect(ctx).Return(nil)
				repo.GetHierarchyMock.Expect(ctx, []uuid.UUID{parentID}, cfg.MaxHierarchyDepth+1, nil, entity.HierarchyTypeParentsOnly).Return([]entity.ListItem{{}, {}, {}, {}}, nil)
			},
			err: entity.ErrMaxHierarchyDepthExceeded(cfg.MaxHierarchyDepth),
		},
		{
			name: "error/repo/lock_hierarchy",
			req:  requestWithParent,
			setup: func(repo *mocks.RepositoryMock, idGen *mocks.IDGeneratorMock, timeGen *mocks.TimeGeneratorMock, validator *mocks.ValidatorMock) {
				validator.NormalizeNameMock.Expect(requestWithParent.Name).Return(requestWithParent.Name)
				validator.ValidateNameMock.Expect(requestWithParent.Name).Return(nil)
				repo.LockHierarchyMock.Expect(ctx).Return(expErr)
			},
			err: expErr,
		},
		{
			name: "error/validation/parent_not_found",
			req:  requestWithParent,
			setup: func(repo *mocks.RepositoryMock, idGen *mocks.IDGeneratorMock, timeGen *mocks.TimeGeneratorMock, validator *mocks.ValidatorMock) {
				validator.NormalizeNameMock.Expect(requestWithParent.Name).Return(requestWithParent.Name)
				validator.ValidateNameMock.Expect(requestWithParent.Name).Return(nil)
				repo.LockHierarchyMock.Expect(ctx).Return(nil)
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
				repo.LockHierarchyMock.Expect(ctx).Return(nil)
				repo.GetHierarchyMock.Expect(ctx, []uuid.UUID{parentID}, cfg.MaxHierarchyDepth+1, nil, entity.HierarchyTypeParentsOnly).Return([]entity.ListItem{
					{
						ID:             parentID,
						Type:           entity.TypeArticle,
						Name:           "parent",
						ParentID:       nil,
						CurrentVersion: &version,
					},
				}, nil)
			},
			err: entity.ErrIncompatibleParentType(),
		},
		{
			name: "error/validation/parent_not_published",
			req:  requestWithParent,
			setup: func(repo *mocks.RepositoryMock, idGen *mocks.IDGeneratorMock, timeGen *mocks.TimeGeneratorMock, validator *mocks.ValidatorMock) {
				validator.NormalizeNameMock.Expect(requestWithParent.Name).Return(requestWithParent.Name)
				validator.ValidateNameMock.Expect(requestWithParent.Name).Return(nil)
				repo.LockHierarchyMock.Expect(ctx).Return(nil)
				repo.GetHierarchyMock.Expect(ctx, []uuid.UUID{parentID}, cfg.MaxHierarchyDepth+1, nil, entity.HierarchyTypeParentsOnly).Return([]entity.ListItem{
					{ID: parentID, Type: entity.TypeDepartment, Name: "parent"},
				}, nil)
			},
			err: entity.ErrParentNotPublished(),
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
				repo.LockHierarchyMock.Expect(ctx).Return(nil)
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
			c, err := entity.NewCore(repo, noTx{}, entity.Generators{ID: idGen, Time: timeGen}, validator, cfg)
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
			UserID:  userID,
		}
		notNormalizedReq = entity.UpdateEntityReq{
			ID:      req.ID,
			Name:    " Name ",
			Content: req.Content,
			UserID:  req.UserID,
		}
		reqParentRemoved = entity.UpdateEntityReq{
			ID:            req.ID,
			Name:          req.Name,
			Content:       req.Content,
			ParentID:      nil,
			ParentChanged: true,
			UserID:        req.UserID,
			EntityType:    entity.TypeArticle,
		}
		parentID         = uuid.MustParse("c4abc05f-91f6-43ca-97b2-1cf4f7de0978")
		reqParentChanged = entity.UpdateEntityReq{
			ID:            req.ID,
			Name:          req.Name,
			Content:       req.Content,
			ParentID:      &parentID,
			UserID:        req.UserID,
			ParentChanged: true,
			EntityType:    entity.TypeDepartment,
		}
		version    = 1
		parentItem = entity.ListItem{
			ID:             parentID,
			Type:           entity.TypeDepartment,
			Name:           "parent",
			ParentID:       nil,
			CurrentVersion: &version,
		}
		parentList = []entity.ListItem{parentItem, {}, {}}
		cfg        = entity.Config{MaxHierarchyDepth: 5}
		expErr     = fmt.Errorf("test error")
		lockRow    = func(repo *mocks.RepositoryMock, structural bool) {
			if structural {
				repo.LockHierarchyMock.Expect(ctx).Return(nil)
			}
			repo.GetListItemForUpdateMock.Expect(ctx, id).Return(entity.ListItem{ID: id}, nil)
		}
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
				lockRow(repo, false)
				timeGen.NowMock.Expect().Return(now)
				repo.UpdateMock.Expect(ctx, req, now).Return(nil)
			},
		},
		{
			name: "success/parent_changed",
			req:  reqParentChanged,
			setup: func(repo *mocks.RepositoryMock, idGen *mocks.IDGeneratorMock, timeGen *mocks.TimeGeneratorMock, validator *mocks.ValidatorMock) {
				validator.NormalizeNameMock.Expect(reqParentChanged.Name).Return(reqParentChanged.Name)
				validator.ValidateNameMock.Expect(reqParentChanged.Name).Return(nil)
				lockRow(repo, true)
				repo.GetHierarchyMock.When(ctx, []uuid.UUID{parentID}, cfg.MaxHierarchyDepth+1, nil, entity.HierarchyTypeParentsOnly).Then(parentList, nil)
				repo.GetHierarchyMock.When(ctx, []uuid.UUID{id}, cfg.MaxHierarchyDepth+1, nil, entity.HierarchyTypeChildrenOnly).Then(nil, nil)
				timeGen.NowMock.Expect().Return(now)
				repo.UpdateMock.Expect(ctx, reqParentChanged, now).Return(nil)
			},
		},
		{
			name: "error/validation/nil_id",
			req: entity.UpdateEntityReq{
				ID:      uuid.Nil,
				Name:    req.Name,
				Content: req.Content,
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
				UserID:        req.UserID,
				EntityType:    entity.TypeDepartment,
			},
			setup: func(repo *mocks.RepositoryMock, idGen *mocks.IDGeneratorMock, timeGen *mocks.TimeGeneratorMock, validator *mocks.ValidatorMock) {
				validator.NormalizeNameMock.Expect(req.Name).Return(reqParentRemoved.Name)
				validator.ValidateNameMock.Expect(req.Name).Return(nil)
				lockRow(repo, true)
			},
			err: entity.ErrParentCycle(),
		},
		{
			name: "error/parent_changed/parent cycle",
			req:  reqParentChanged,
			setup: func(repo *mocks.RepositoryMock, idGen *mocks.IDGeneratorMock, timeGen *mocks.TimeGeneratorMock, validator *mocks.ValidatorMock) {
				validator.NormalizeNameMock.Expect(reqParentChanged.Name).Return(reqParentChanged.Name)
				validator.ValidateNameMock.Expect(reqParentChanged.Name).Return(nil)
				lockRow(repo, true)
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
				lockRow(repo, true)
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
				lockRow(repo, true)
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
				lockRow(repo, true)
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
				lockRow(repo, true)
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
				lockRow(repo, true)
				repo.GetHierarchyMock.Expect(ctx, []uuid.UUID{parentID}, cfg.MaxHierarchyDepth+1, nil, entity.HierarchyTypeParentsOnly).Return([]entity.ListItem{
					{
						ID:             parentID,
						Type:           entity.TypeArticle,
						Name:           "parent",
						ParentID:       nil,
						CurrentVersion: &version,
					},
				}, nil)
			},
			err: entity.ErrIncompatibleParentType(),
		},
		{
			name: "error/validation/parent_not_published",
			req:  reqParentChanged,
			setup: func(repo *mocks.RepositoryMock, idGen *mocks.IDGeneratorMock, timeGen *mocks.TimeGeneratorMock, validator *mocks.ValidatorMock) {
				validator.NormalizeNameMock.Expect(req.Name).Return(req.Name)
				validator.ValidateNameMock.Expect(req.Name).Return(nil)
				lockRow(repo, true)
				repo.GetHierarchyMock.Expect(ctx, []uuid.UUID{parentID}, cfg.MaxHierarchyDepth+1, nil, entity.HierarchyTypeParentsOnly).Return([]entity.ListItem{
					{ID: parentID, Type: entity.TypeDepartment, Name: "parent"},
				}, nil)
			},
			err: entity.ErrParentNotPublished(),
		},
		{
			name: "error/validation/parent_required",
			req:  reqParentRemoved,
			setup: func(repo *mocks.RepositoryMock, idGen *mocks.IDGeneratorMock, timeGen *mocks.TimeGeneratorMock, validator *mocks.ValidatorMock) {
				validator.NormalizeNameMock.Expect(req.Name).Return(req.Name)
				validator.ValidateNameMock.Expect(req.Name).Return(nil)
				lockRow(repo, true)
			},
			err: entity.ErrParentRequired(),
		},
		{
			name: "error/repo/lock_hierarchy",
			req:  reqParentChanged,
			setup: func(repo *mocks.RepositoryMock, idGen *mocks.IDGeneratorMock, timeGen *mocks.TimeGeneratorMock, validator *mocks.ValidatorMock) {
				validator.NormalizeNameMock.Expect(req.Name).Return(req.Name)
				validator.ValidateNameMock.Expect(req.Name).Return(nil)
				repo.LockHierarchyMock.Expect(ctx).Return(expErr)
			},
			err: expErr,
		},
		{
			name: "error/repo/lock_row",
			req:  req,
			setup: func(repo *mocks.RepositoryMock, idGen *mocks.IDGeneratorMock, timeGen *mocks.TimeGeneratorMock, validator *mocks.ValidatorMock) {
				validator.NormalizeNameMock.Expect(req.Name).Return(req.Name)
				validator.ValidateNameMock.Expect(req.Name).Return(nil)
				repo.GetListItemForUpdateMock.Expect(ctx, id).Return(entity.ListItem{}, entity.ErrEntityNotFound())
			},
			err: entity.ErrEntityNotFound(),
		},
		{
			name: "error/moved_concurrently",
			req:  req,
			setup: func(repo *mocks.RepositoryMock, idGen *mocks.IDGeneratorMock, timeGen *mocks.TimeGeneratorMock, validator *mocks.ValidatorMock) {
				validator.NormalizeNameMock.Expect(req.Name).Return(req.Name)
				validator.ValidateNameMock.Expect(req.Name).Return(nil)
				repo.GetListItemForUpdateMock.Expect(ctx, id).Return(entity.ListItem{ID: id, ParentID: &parentID}, nil)
			},
			err: entity.ErrConcurrentModification(),
		},
		{
			name: "error/repo/update",
			req:  req,
			setup: func(repo *mocks.RepositoryMock, idGen *mocks.IDGeneratorMock, timeGen *mocks.TimeGeneratorMock, validator *mocks.ValidatorMock) {
				validator.NormalizeNameMock.Expect(req.Name).Return(normalizedName)
				validator.ValidateNameMock.Expect(normalizedName).Return(nil)
				lockRow(repo, false)
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
			c, err := entity.NewCore(repo, noTx{}, entity.Generators{ID: idGen, Time: timeGen}, validator, cfg)
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

func TestCore_SaveDraft(t *testing.T) {
	t.Parallel()

	var (
		ctx            = context.Background()
		entityID       = uuid.New()
		userID         = uuid.New()
		now            = time.Now()
		version        = 3
		normalizedName = "n_name"
		req            = entity.SaveDraftReq{EntityID: entityID, UserID: userID, Name: normalizedName, Content: "content"}
		rawReq         = entity.SaveDraftReq{EntityID: entityID, UserID: userID, Name: " Name ", Content: req.Content}
		expErr         = fmt.Errorf("test error")
	)

	tests := []struct {
		name  string
		req   entity.SaveDraftReq
		setup func(repo *mocks.RepositoryMock, timeGen *mocks.TimeGeneratorMock, validator *mocks.ValidatorMock)
		err   error
	}{
		{
			name: "success/normalize",
			req:  rawReq,
			setup: func(repo *mocks.RepositoryMock, timeGen *mocks.TimeGeneratorMock, validator *mocks.ValidatorMock) {
				validator.NormalizeNameMock.Expect(rawReq.Name).Return(normalizedName)
				validator.ValidateNameMock.Expect(normalizedName).Return(nil)
				repo.GetListItemMock.Expect(ctx, entityID).Return(entity.ListItem{ID: entityID, CurrentVersion: &version}, nil)
				timeGen.NowMock.Expect().Return(now)
				repo.SaveDraftMock.Expect(ctx, req, &version, now).Return(nil)
			},
		},
		{
			name: "error/validation/nil_entity_id",
			req:  entity.SaveDraftReq{UserID: userID, Name: req.Name},
			err:  apperr.ErrNilUUID(entity.FieldEntityID),
		},
		{
			name: "error/validation/nil_user_id",
			req:  entity.SaveDraftReq{EntityID: entityID, Name: req.Name},
			err:  apperr.ErrNilUUID(entity.FieldUserID),
		},
		{
			name: "error/validation/invalid_name",
			req:  req,
			setup: func(repo *mocks.RepositoryMock, timeGen *mocks.TimeGeneratorMock, validator *mocks.ValidatorMock) {
				validator.NormalizeNameMock.Expect(req.Name).Return(req.Name)
				validator.ValidateNameMock.Expect(req.Name).Return(expErr)
			},
			err: expErr,
		},
		{
			name: "error/repo/get_list_item",
			req:  req,
			setup: func(repo *mocks.RepositoryMock, timeGen *mocks.TimeGeneratorMock, validator *mocks.ValidatorMock) {
				validator.NormalizeNameMock.Expect(req.Name).Return(req.Name)
				validator.ValidateNameMock.Expect(req.Name).Return(nil)
				repo.GetListItemMock.Expect(ctx, entityID).Return(entity.ListItem{}, entity.ErrEntityNotFound())
			},
			err: entity.ErrEntityNotFound(),
		},
		{
			name: "error/repo/save_draft",
			req:  req,
			setup: func(repo *mocks.RepositoryMock, timeGen *mocks.TimeGeneratorMock, validator *mocks.ValidatorMock) {
				validator.NormalizeNameMock.Expect(req.Name).Return(req.Name)
				validator.ValidateNameMock.Expect(req.Name).Return(nil)
				repo.GetListItemMock.Expect(ctx, entityID).Return(entity.ListItem{ID: entityID}, nil)
				timeGen.NowMock.Expect().Return(now)
				repo.SaveDraftMock.Expect(ctx, req, nil, now).Return(expErr)
			},
			err: expErr,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			repo := mocks.NewRepositoryMock(t)
			timeGen := mocks.NewTimeGeneratorMock(t)
			validator := mocks.NewValidatorMock(t)
			if tt.setup != nil {
				tt.setup(repo, timeGen, validator)
			}
			c, err := entity.NewCore(repo, noTx{}, entity.Generators{ID: mocks.NewIDGeneratorMock(t), Time: timeGen}, validator, Cfg())
			require.NoError(t, err)

			err = c.SaveDraft(ctx, tt.req)
			if tt.err != nil {
				require.ErrorIs(t, err, tt.err)
				return
			}
			require.NoError(t, err)
		})
	}
}

func TestCore_GetDraft_DeleteDraft(t *testing.T) {
	t.Parallel()

	var (
		ctx      = context.Background()
		entityID = uuid.New()
		userID   = uuid.New()
		draft    = entity.Draft{EntityID: entityID, UserID: userID, Name: "name", Content: "content"}
		expErr   = fmt.Errorf("test error")
	)

	tests := []struct {
		name     string
		entityID uuid.UUID
		userID   uuid.UUID
		setup    func(repo *mocks.RepositoryMock)
		err      error
	}{
		{
			name:     "success",
			entityID: entityID,
			userID:   userID,
			setup: func(repo *mocks.RepositoryMock) {
				repo.GetDraftMock.Expect(ctx, entityID, userID).Return(draft, nil)
				repo.DeleteDraftMock.Expect(ctx, entityID, userID).Return(nil)
			},
		},
		{
			name:   "error/validation/nil_entity_id",
			userID: userID,
			err:    apperr.ErrNilUUID(entity.FieldEntityID),
		},
		{
			name:     "error/validation/nil_user_id",
			entityID: entityID,
			err:      apperr.ErrNilUUID(entity.FieldUserID),
		},
		{
			name:     "error/repo",
			entityID: entityID,
			userID:   userID,
			setup: func(repo *mocks.RepositoryMock) {
				repo.GetDraftMock.Expect(ctx, entityID, userID).Return(entity.Draft{}, expErr)
				repo.DeleteDraftMock.Expect(ctx, entityID, userID).Return(expErr)
			},
			err: expErr,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			repo := mocks.NewRepositoryMock(t)
			if tt.setup != nil {
				tt.setup(repo)
			}
			c, err := entity.NewCore(repo, noTx{}, entity.Generators{ID: mocks.NewIDGeneratorMock(t), Time: mocks.NewTimeGeneratorMock(t)}, mocks.NewValidatorMock(t), Cfg())
			require.NoError(t, err)

			got, getErr := c.GetDraft(ctx, tt.entityID, tt.userID)
			deleteErr := c.DeleteDraft(ctx, tt.entityID, tt.userID)
			if tt.err != nil {
				require.ErrorIs(t, getErr, tt.err)
				require.ErrorIs(t, deleteErr, tt.err)
				return
			}
			require.NoError(t, getErr)
			require.NoError(t, deleteErr)
			require.Equal(t, draft, got)
		})
	}
}

func TestCore_PublishDraft(t *testing.T) {
	t.Parallel()

	var (
		ctx      = context.Background()
		entityID = uuid.New()
		userID   = uuid.New()
		parentID = uuid.New()
		now      = time.Now()
		v1, v2   = 1, 2
		current  = entity.ListItem{ID: entityID, ParentID: &parentID, CurrentVersion: &v1}
		draft    = entity.Draft{EntityID: entityID, UserID: userID, Name: "name", Content: "content", BaseVersion: &v1}
		// the draft is published under the current parent: moving is not part of a draft
		updateReq = entity.UpdateEntityReq{
			ID:       entityID,
			Name:     draft.Name,
			Content:  draft.Content,
			ParentID: &parentID,
			UserID:   userID,
		}
		expErr = fmt.Errorf("test error")
	)

	tests := []struct {
		name     string
		entityID uuid.UUID
		userID   uuid.UUID
		setup    func(repo *mocks.RepositoryMock, timeGen *mocks.TimeGeneratorMock)
		err      error
	}{
		{
			name:     "success",
			entityID: entityID,
			userID:   userID,
			setup: func(repo *mocks.RepositoryMock, timeGen *mocks.TimeGeneratorMock) {
				repo.GetListItemForUpdateMock.Expect(ctx, entityID).Return(current, nil)
				repo.GetDraftMock.Expect(ctx, entityID, userID).Return(draft, nil)
				timeGen.NowMock.Expect().Return(now)
				repo.UpdateMock.Expect(ctx, updateReq, now).Return(nil)
				repo.DeleteDraftMock.Expect(ctx, entityID, userID).Return(nil)
			},
		},
		{
			name:     "success/never_published",
			entityID: entityID,
			userID:   userID,
			setup: func(repo *mocks.RepositoryMock, timeGen *mocks.TimeGeneratorMock) {
				repo.GetListItemForUpdateMock.Expect(ctx, entityID).Return(entity.ListItem{ID: entityID, ParentID: &parentID}, nil)
				repo.GetDraftMock.Expect(ctx, entityID, userID).Return(entity.Draft{Name: draft.Name, Content: draft.Content}, nil)
				timeGen.NowMock.Expect().Return(now)
				repo.UpdateMock.Expect(ctx, updateReq, now).Return(nil)
				repo.DeleteDraftMock.Expect(ctx, entityID, userID).Return(nil)
			},
		},
		{
			name:     "error/draft_outdated",
			entityID: entityID,
			userID:   userID,
			setup: func(repo *mocks.RepositoryMock, timeGen *mocks.TimeGeneratorMock) {
				repo.GetListItemForUpdateMock.Expect(ctx, entityID).Return(entity.ListItem{ID: entityID, CurrentVersion: &v2}, nil)
				repo.GetDraftMock.Expect(ctx, entityID, userID).Return(draft, nil)
			},
			err: entity.ErrDraftOutdated(),
		},
		{
			name:   "error/validation/nil_entity_id",
			userID: userID,
			err:    apperr.ErrNilUUID(entity.FieldEntityID),
		},
		{
			name:     "error/validation/nil_user_id",
			entityID: entityID,
			err:      apperr.ErrNilUUID(entity.FieldUserID),
		},
		{
			name:     "error/repo/lock_row",
			entityID: entityID,
			userID:   userID,
			setup: func(repo *mocks.RepositoryMock, timeGen *mocks.TimeGeneratorMock) {
				repo.GetListItemForUpdateMock.Expect(ctx, entityID).Return(entity.ListItem{}, entity.ErrEntityNotFound())
			},
			err: entity.ErrEntityNotFound(),
		},
		{
			name:     "error/repo/get_draft",
			entityID: entityID,
			userID:   userID,
			setup: func(repo *mocks.RepositoryMock, timeGen *mocks.TimeGeneratorMock) {
				repo.GetListItemForUpdateMock.Expect(ctx, entityID).Return(current, nil)
				repo.GetDraftMock.Expect(ctx, entityID, userID).Return(entity.Draft{}, entity.ErrDraftNotFound())
			},
			err: entity.ErrDraftNotFound(),
		},
		{
			name:     "error/repo/update",
			entityID: entityID,
			userID:   userID,
			setup: func(repo *mocks.RepositoryMock, timeGen *mocks.TimeGeneratorMock) {
				repo.GetListItemForUpdateMock.Expect(ctx, entityID).Return(current, nil)
				repo.GetDraftMock.Expect(ctx, entityID, userID).Return(draft, nil)
				timeGen.NowMock.Expect().Return(now)
				repo.UpdateMock.Expect(ctx, updateReq, now).Return(expErr)
			},
			err: expErr,
		},
		{
			name:     "error/repo/delete_draft",
			entityID: entityID,
			userID:   userID,
			setup: func(repo *mocks.RepositoryMock, timeGen *mocks.TimeGeneratorMock) {
				repo.GetListItemForUpdateMock.Expect(ctx, entityID).Return(current, nil)
				repo.GetDraftMock.Expect(ctx, entityID, userID).Return(draft, nil)
				timeGen.NowMock.Expect().Return(now)
				repo.UpdateMock.Expect(ctx, updateReq, now).Return(nil)
				repo.DeleteDraftMock.Expect(ctx, entityID, userID).Return(expErr)
			},
			err: expErr,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			repo := mocks.NewRepositoryMock(t)
			timeGen := mocks.NewTimeGeneratorMock(t)
			if tt.setup != nil {
				tt.setup(repo, timeGen)
			}
			c, err := entity.NewCore(repo, noTx{}, entity.Generators{ID: mocks.NewIDGeneratorMock(t), Time: timeGen}, mocks.NewValidatorMock(t), Cfg())
			require.NoError(t, err)

			err = c.PublishDraft(ctx, tt.entityID, tt.userID)
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
				repo.LockHierarchyMock.Expect(ctx).Return(nil)
				repo.GetHierarchyMock.Expect(ctx, []uuid.UUID{id}, cfg.MaxHierarchyDepth+1, nil, entity.HierarchyTypeChildrenOnly).Return(list, nil)
				repo.DeleteMock.Expect(ctx, ids).Return(nil)
			},
		},
		{
			name: "error/repo/LockHierarchy",
			setup: func(repo *mocks.RepositoryMock, timeGen *mocks.TimeGeneratorMock) {
				repo.LockHierarchyMock.Expect(ctx).Return(expErr)
			},
			err: expErr,
		},
		{
			name: "error/repo/GetHierarchyMock",
			setup: func(repo *mocks.RepositoryMock, timeGen *mocks.TimeGeneratorMock) {
				repo.LockHierarchyMock.Expect(ctx).Return(nil)
				repo.GetHierarchyMock.Expect(ctx, []uuid.UUID{id}, cfg.MaxHierarchyDepth+1, nil, entity.HierarchyTypeChildrenOnly).Return(nil, expErr)
			},
			err: expErr,
		},
		{
			name: "error/repo/Delete",
			setup: func(repo *mocks.RepositoryMock, timeGen *mocks.TimeGeneratorMock) {
				repo.LockHierarchyMock.Expect(ctx).Return(nil)
				repo.GetHierarchyMock.Expect(ctx, []uuid.UUID{id}, cfg.MaxHierarchyDepth+1, nil, entity.HierarchyTypeChildrenOnly).Return(list, nil)
				repo.DeleteMock.Expect(ctx, ids).Return(expErr)
			},
			err: expErr,
		},
		{
			name: "error/not found",
			setup: func(repo *mocks.RepositoryMock, timeGen *mocks.TimeGeneratorMock) {
				repo.LockHierarchyMock.Expect(ctx).Return(nil)
				repo.GetHierarchyMock.Expect(ctx, []uuid.UUID{id}, cfg.MaxHierarchyDepth+1, nil, entity.HierarchyTypeChildrenOnly).Return([]entity.ListItem{}, nil)
			},
			err: entity.ErrEntityNotFound(),
		},
		{
			name: "error/max hierarchy depth exceeded",
			setup: func(repo *mocks.RepositoryMock, timeGen *mocks.TimeGeneratorMock) {
				repo.LockHierarchyMock.Expect(ctx).Return(nil)
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
			c, err := entity.NewCore(repo, noTx{}, entity.Generators{Time: timeGen, ID: idGen}, validator, cfg)
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
