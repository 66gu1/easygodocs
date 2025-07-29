package test

import (
	"context"
	"fmt"
	"github.com/66gu1/easygodocs/internal/app/department"
	"github.com/66gu1/easygodocs/internal/app/department/mock"
	"github.com/google/go-cmp/cmp"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"testing"

	"github.com/gojuno/minimock/v3"
	"github.com/google/uuid"
)

func TestDepartmentService_Create(t *testing.T) {
	t.Parallel()
	mc := minimock.NewController(t)
	parentID := uuid.New()
	wantedCtx := context.Background()
	reqWithParent := department.CreateDepartmentReq{
		Name:     "HR",
		ParentID: &parentID,
	}
	reqWithoutParent := department.CreateDepartmentReq{
		Name: "Engineering",
	}
	wantedErr := fmt.Errorf("error")

	tests := []struct {
		name    string
		setup   func(repo *mock.RepositoryMock)
		wantErr bool
		req     department.CreateDepartmentReq
	}{
		{
			name: "with parent id",
			req:  reqWithParent,
			setup: func(repo *mock.RepositoryMock) {
				repo.ValidateParentMock.Times(1).Set(func(ctx context.Context, id uuid.UUID, parentID uuid.UUID) error {
					require.NotEqual(t, uuid.Nil, id)
					require.Equal(t, *reqWithParent.ParentID, parentID)
					require.Equal(t, wantedCtx, ctx)
					return nil
				})
				repo.CreateMock.Times(1).Set(func(ctx context.Context, req department.CreateDepartmentReq, id uuid.UUID) error {
					require.Equal(t, reqWithParent, req)
					require.NotEqual(t, uuid.Nil, id)
					return nil
				})
			},
			wantErr: false,
		},
		{
			name: "no parent id",
			req:  reqWithoutParent,
			setup: func(repo *mock.RepositoryMock) {
				repo.CreateMock.Times(1).Set(func(ctx context.Context, req department.CreateDepartmentReq, id uuid.UUID) error {
					require.Equal(t, reqWithoutParent, req)
					require.NotEqual(t, uuid.Nil, id)
					return nil
				})
			},
			wantErr: false,
		},
		{
			name: "validate parent error",
			req:  reqWithParent,
			setup: func(repo *mock.RepositoryMock) {
				repo.ValidateParentMock.Times(1).Set(func(ctx context.Context, id uuid.UUID, parentID uuid.UUID) error {
					return wantedErr
				})
			},
			wantErr: true,
		},
		{
			name: "create error",
			req:  reqWithoutParent,
			setup: func(repo *mock.RepositoryMock) {
				repo.CreateMock.Times(1).Set(func(ctx context.Context, req department.CreateDepartmentReq, id uuid.UUID) error {
					return wantedErr
				})
			},
			wantErr: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			repoMock := mock.NewRepositoryMock(mc)
			tc.setup(repoMock)
			svc := department.NewService(repoMock)

			id, err := svc.Create(wantedCtx, tc.req)
			if tc.wantErr {
				require.Error(t, err)
				assert.Equal(t, uuid.Nil, id)
			} else {
				require.NoError(t, err)
				assert.NotEqual(t, uuid.Nil, id)
			}
		})
	}
}

func TestDepartmentService_GetDepartmentTree(t *testing.T) {
	t.Parallel()
	mc := minimock.NewController(t)

	idA := uuid.New()
	idB := uuid.New()
	idC := uuid.New()
	idD := uuid.New()
	idE := uuid.New()
	deps := []department.Department{
		{ID: idA, Name: "A", ParentID: nil},
		{ID: idB, Name: "B", ParentID: &idA},
		{ID: idC, Name: "C", ParentID: &idB},
		{ID: idD, Name: "D", ParentID: &idA},
		{ID: idE, Name: "E", ParentID: nil},
	}
	wantedTree := department.Tree{
		{
			ID:       idA,
			Name:     "A",
			ParentID: nil,
			Children: []*department.Node{
				{
					ID:       idB,
					Name:     "B",
					ParentID: &idA,
					Children: []*department.Node{
						{
							ID:       idC,
							Name:     "C",
							ParentID: &idB,
							Children: nil,
						},
					},
				},
				{
					ID:       idD,
					Name:     "D",
					ParentID: &idA,
					Children: nil,
				},
			},
		},
		{
			ID:       idE,
			Name:     "E",
			ParentID: nil,
			Children: nil,
		},
	}
	ctx := context.Background()
	expErr := fmt.Errorf("error")

	tests := []struct {
		name     string
		setup    func(repo *mock.RepositoryMock)
		wantTree department.Tree
		wantErr  bool
	}{
		{
			name: "simple tree",
			setup: func(repo *mock.RepositoryMock) {
				repo.ListMock.Times(1).Expect(ctx).Return(deps, nil)
			},
			wantTree: wantedTree,
		},
		{
			name: "empty tree",
			setup: func(repo *mock.RepositoryMock) {
				repo.ListMock.Times(1).Expect(ctx).Return(nil, nil)
			},
			wantTree: nil,
		},
		{
			name: "tx error",
			setup: func(repo *mock.RepositoryMock) {
				repo.ListMock.Times(1).Expect(ctx).Return(nil, expErr)
			},
			wantErr: true,
		},
	}

	for _, tc := range tests {
		repoMock := mock.NewRepositoryMock(mc)
		tc.setup(repoMock)
		svc := department.NewService(repoMock)

		tree, err := svc.GetDepartmentTree(ctx)
		if tc.wantErr {
			require.Error(t, err)
		} else {
			require.NoError(t, err)
		}

		if diff := cmp.Diff(tc.wantTree, tree); diff != "" {
			t.Errorf("tree mismatch (-want +got):\n%s", diff)
		}
	}
}

func TestDepartmentService_Update(t *testing.T) {
	t.Parallel()
	mc := minimock.NewController(t)

	parentID := uuid.New()
	reqWithParent := department.UpdateDepartmentReq{
		ID:       uuid.New(),
		Name:     "HR",
		ParentID: &parentID,
	}
	reqWithoutParent := department.UpdateDepartmentReq{
		ID:   uuid.New(),
		Name: "Engineering",
	}
	err := fmt.Errorf("error")
	ctx := context.Background()

	tests := []struct {
		name                 string
		setup                func(repo *mock.RepositoryMock)
		wantErr              bool
		req                  department.UpdateDepartmentReq
		expectUpdateCalled   int
		expectValidateCalled int
	}{
		{
			name: "update with parent id",
			req:  reqWithParent,
			setup: func(repo *mock.RepositoryMock) {
				repo.ValidateParentMock.Times(1).Expect(ctx, reqWithParent.ID, parentID).Return(nil)
				repo.UpdateMock.Times(1).Expect(ctx, reqWithParent).Return(nil)
			},
			wantErr: false,
		},
		{
			name: "update without parent id",
			req:  reqWithoutParent,
			setup: func(repo *mock.RepositoryMock) {
				repo.UpdateMock.Expect(ctx, reqWithoutParent).Return(nil)
			},
			wantErr: false,
		},
		{
			name: "validate parent error",
			req:  reqWithParent,
			setup: func(repo *mock.RepositoryMock) {
				repo.ValidateParentMock.Expect(ctx, reqWithParent.ID, parentID).Return(err)
			},
			wantErr: true,
		},
		{
			name: "update error",
			req:  reqWithoutParent,
			setup: func(repo *mock.RepositoryMock) {
				repo.UpdateMock.Expect(ctx, reqWithoutParent).Return(err)
			},
			wantErr: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			repoMock := mock.NewRepositoryMock(mc)
			tc.setup(repoMock)
			svc := department.NewService(repoMock)

			err := svc.Update(ctx, tc.req)
			if tc.wantErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestDepartmentService_Delete(t *testing.T) {
	t.Parallel()
	mc := minimock.NewController(t)

	id := uuid.New()
	ctx := context.Background()
	expErr := fmt.Errorf("error")

	tests := []struct {
		name    string
		setup   func(repo *mock.RepositoryMock)
		wantErr bool
	}{
		{
			name: "success",
			setup: func(repo *mock.RepositoryMock) {
				repo.DeleteMock.Times(1).Expect(ctx, id).Return(nil)
			},
			wantErr: false,
		},
		{
			name: "tx error",
			setup: func(repo *mock.RepositoryMock) {
				repo.DeleteMock.Times(1).Expect(ctx, id).Return(expErr)
			},
			wantErr: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			repoMock := mock.NewRepositoryMock(mc)
			tc.setup(repoMock)
			svc := department.NewService(repoMock)

			err := svc.Delete(ctx, id)
			if tc.wantErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}
		})
	}
}
