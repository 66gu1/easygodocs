package department

import (
	"context"
	"errors"
	"fmt"
	"github.com/66gu1/easygodocs/internal/infrastructure/apperror"
	"github.com/google/uuid"
	"github.com/pressly/goose/v3"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"testing"
)

func TestRepository_Create(t *testing.T) {
	t.Parallel()

	id1 := uuid.New()
	id2 := uuid.New()

	tests := []struct {
		name  string
		input CreateDepartmentReq
		want  Department
		id    uuid.UUID
	}{
		{
			name:  "with parent id",
			input: CreateDepartmentReq{Name: "HR", ParentID: &id2},
			want:  Department{ID: id1, Name: "HR", ParentID: &id2},
			id:    id1,
		},
		{
			name:  "without parent id",
			input: CreateDepartmentReq{Name: "Engineering"},
			want:  Department{ID: id1, Name: "Engineering"},
			id:    id1,
		},
	}

	ctx := context.Background()

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			db := setupTestDB(t)
			repo := NewRepository(db)
			err := repo.Create(ctx, tc.input, tc.id)
			require.NoError(t, err)

			list, err := repo.List(ctx)
			require.NoError(t, err)

			require.Equal(t, 1, len(list))
			assertDepartment(t, tc.want, list[0], false)
		})
	}
}

func TestRepository_Update(t *testing.T) {
	t.Parallel()

	id1 := uuid.New()
	id2 := uuid.New()

	tests := []struct {
		name        string
		createInput CreateDepartmentReq
		updateInput UpdateDepartmentReq
		want        Department
		id          uuid.UUID
	}{
		{
			name:        "with parent ID",
			createInput: CreateDepartmentReq{Name: "HR"},
			updateInput: UpdateDepartmentReq{ID: id1, Name: "Engineering", ParentID: &id2},
			want:        Department{ID: id1, Name: "Engineering", ParentID: &id2},
			id:          id1,
		},
		{
			name:        "without parent ID",
			createInput: CreateDepartmentReq{Name: "HR", ParentID: &id2},
			updateInput: UpdateDepartmentReq{ID: id1, Name: "Engineering"},
			want:        Department{ID: id1, Name: "Engineering"},
			id:          id1,
		},
	}

	ctx := context.Background()

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			db := setupTestDB(t)
			repo := NewRepository(db)
			err := repo.Create(ctx, tc.createInput, tc.id)
			require.NoError(t, err)
			err = repo.Update(ctx, tc.updateInput)
			require.NoError(t, err)

			list, err := repo.List(ctx)
			require.NoError(t, err)

			require.Equal(t, 1, len(list))
			assertDepartment(t, tc.want, list[0], true)
		})
	}
}

func TestRepository_Delete(t *testing.T) {
	t.Parallel()

	id1 := uuid.New()
	id2 := uuid.New()
	id3 := uuid.New()
	id4 := uuid.New()
	tests := []struct {
		name        string
		createInput []CreateDepartmentReq
		deleteID    uuid.UUID
		want        []Department
		id          []uuid.UUID
	}{
		{
			name: "delete existing department",
			createInput: []CreateDepartmentReq{
				{Name: "HR"},
				{Name: "Engineering", ParentID: &id1},
				{Name: "other"},
				{Name: "new", ParentID: &id3},
			},
			id:       []uuid.UUID{id1, id2, id3, id4},
			deleteID: id3,
			want: []Department{
				{ID: id1, Name: "HR"},
				{ID: id2, Name: "Engineering", ParentID: &id1},
			},
		},
	}

	ctx := context.Background()

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			db := setupTestDB(t)
			repo := NewRepository(db)
			for i, input := range tc.createInput {
				err := repo.Create(ctx, input, tc.id[i])
				require.NoError(t, err)
			}
			err := repo.Delete(ctx, tc.deleteID)
			require.NoError(t, err)

			list, err := repo.List(ctx)
			require.NoError(t, err)

			assertDepartments(t, tc.want, list, false)
		})
	}
}

func TestRepository_ValidateParent(t *testing.T) {
	t.Parallel()

	id1 := uuid.New()
	id2 := uuid.New()

	tests := []struct {
		name        string
		createInput []CreateDepartmentReq
		parentID    uuid.UUID
		checkingID  uuid.UUID
		wantErr     bool
		err         *apperror.Error
		id          []uuid.UUID
	}{
		{
			name: "valid parent ID",
			createInput: []CreateDepartmentReq{
				{Name: "HR"},
				{Name: "Engineering"},
			},
			parentID:   id1,
			checkingID: id2,
			id:         []uuid.UUID{id1, id2},
		},
		{
			name:        "not found parent ID",
			createInput: []CreateDepartmentReq{{Name: "Engineering"}},
			parentID:    uuid.New(), // not created yet
			checkingID:  id1,
			wantErr:     true,
			err:         parentNodFoundErr,
			id:          []uuid.UUID{id1},
		},
		{
			name:        "empty ID",
			createInput: []CreateDepartmentReq{{Name: "Engineering"}},
			parentID:    uuid.New(), // not created yet
			checkingID:  uuid.Nil,
			wantErr:     true,
			id:          []uuid.UUID{id1},
		},
		{
			name: "cycle parent ID",
			createInput: []CreateDepartmentReq{
				{Name: "Engineering"},
				{Name: "HR", ParentID: &id1},
			},
			id:         []uuid.UUID{id1, id2},
			parentID:   id2,
			checkingID: id1,
			wantErr:    true,
			err:        parentCycleErr,
		},
	}

	ctx := context.Background()

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			db := setupTestDB(t)
			repo := NewRepository(db)

			for i, input := range tc.createInput {
				err := repo.Create(ctx, input, tc.id[i])
				require.NoError(t, err)
			}

			err := repo.ValidateParent(ctx, tc.checkingID, tc.parentID)
			if tc.wantErr {
				require.Error(t, err)
				if tc.err != nil {
					require.True(t, errors.Is(err, tc.err))
				}
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func setupTestDB(t *testing.T) *gorm.DB {
	t.Helper()

	// Use isolated in-memory database
	dsn := fmt.Sprintf("file:%s?mode=memory&cache=private", uuid.New().String())
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{})
	require.NoError(t, err)

	err = goose.SetDialect("sqlite3")
	require.NoError(t, err)

	sqlDB, err := db.DB()
	require.NoError(t, err)

	err = goose.Up(sqlDB, "../../../migrations")
	require.NoError(t, err)

	return db
}

func assertDepartments(t *testing.T, want, got []Department, updated bool) {
	t.Helper()

	require.Equal(t, len(want), len(got), "department count mismatch")

	gotMap := make(map[uuid.UUID]Department)
	for _, dep := range got {
		gotMap[dep.ID] = dep
	}

	for _, wantDep := range want {
		gotDep, ok := gotMap[wantDep.ID]
		require.True(t, ok, "expected department not found: %s", wantDep.ID)
		assertDepartment(t, wantDep, gotDep, updated)
	}
}

func assertDepartment(t *testing.T, want Department, actual Department, updated bool) {
	t.Helper()

	assert.Equal(t, want.Name, actual.Name)
	assert.Equal(t, want.ID, actual.ID)
	assert.Equal(t, want.ParentID, actual.ParentID)
	assert.NotEmpty(t, actual.CreatedAt)
	assert.NotEmpty(t, actual.UpdatedAt)
	assert.Empty(t, actual.DeletedAt)
	if updated {
		assert.True(t, actual.UpdatedAt.After(actual.CreatedAt), "UpdatedAt should be after CreatedAt")
	} else {
		assert.Equal(t, actual.CreatedAt, actual.UpdatedAt, "CreatedAt should equal UpdatedAt for non-updated departments")
	}
}
