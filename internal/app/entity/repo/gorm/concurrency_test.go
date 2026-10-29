//go:build testutil

package gorm

import (
	"context"
	"fmt"
	"sync"
	"testing"

	"github.com/66gu1/easygodocs/internal/app/entity"
	"github.com/66gu1/easygodocs/internal/infrastructure/db"
	"github.com/66gu1/easygodocs/internal/infrastructure/system"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

// These tests run the real core against Postgres to make sure
// concurrent requests can't break versioning or the hierarchy.

const maxDepth = 15

type entityCore interface {
	Create(ctx context.Context, req entity.CreateEntityReq) (uuid.UUID, error)
	Update(ctx context.Context, req entity.UpdateEntityReq) error
	Delete(ctx context.Context, id uuid.UUID) error
	SaveDraft(ctx context.Context, req entity.SaveDraftReq) error
	PublishDraft(ctx context.Context, entityID, userID uuid.UUID) error
}

func newEntityCore(t *testing.T, repo *Repository, gdb *gorm.DB) entityCore {
	t.Helper()
	txManager, err := db.NewTxManager(gdb)
	require.NoError(t, err)
	validator, err := entity.NewValidator(entity.ValidationConfig{MaxNameLength: 100})
	require.NoError(t, err)
	core, err := entity.NewCore(
		repo,
		txManager,
		entity.Generators{ID: &system.UUIDv7Generator{}, Time: &system.TimeGenerator{}},
		validator,
		entity.Config{MaxHierarchyDepth: maxDepth},
	)
	require.NoError(t, err)
	return core
}

func createDepartment(t *testing.T, core entityCore, userID uuid.UUID, parentID *uuid.UUID) uuid.UUID {
	t.Helper()
	id, err := core.Create(t.Context(), entity.CreateEntityReq{
		Type:     entity.TypeDepartment,
		Name:     "dep",
		ParentID: parentID,
		UserID:   userID,
	})
	require.NoError(t, err)
	return id
}

// runConcurrently starts all functions at the same time and waits for them.
func runConcurrently(fns ...func()) {
	var start, done sync.WaitGroup
	start.Add(1)
	for _, fn := range fns {
		done.Add(1)
		go func() {
			defer done.Done()
			start.Wait()
			fn()
		}()
	}
	start.Done()
	done.Wait()
}

func TestConcurrency_UpdateCreatesSequentialVersions(t *testing.T) {
	t.Parallel()
	repo, gdb, _ := newEntityRepo(t)
	core := newEntityCore(t, repo, gdb)
	userID := createUserForEntity(t, gdb)
	id := createDepartment(t, core, userID, nil)

	const updates = 20
	errs := make([]error, updates)
	fns := make([]func(), updates)
	for i := range updates {
		fns[i] = func() {
			errs[i] = core.Update(context.Background(), entity.UpdateEntityReq{
				ID:         id,
				Name:       "dep",
				Content:    fmt.Sprintf("content %d", i),
				UserID:     userID,
				EntityType: entity.TypeDepartment,
			})
		}
	}
	runConcurrently(fns...)

	for i, err := range errs {
		require.NoError(t, err, "update %d", i)
	}
	versions, err := repo.GetVersionsList(t.Context(), id)
	require.NoError(t, err)
	require.Len(t, versions, updates+1)

	got, err := repo.Get(t.Context(), id)
	require.NoError(t, err)
	require.Equal(t, updates+1, *got.CurrentVersion)
}

func TestConcurrency_CrossMoveCannotCreateCycle(t *testing.T) {
	t.Parallel()
	repo, gdb, _ := newEntityRepo(t)
	core := newEntityCore(t, repo, gdb)
	userID := createUserForEntity(t, gdb)
	root := createDepartment(t, core, userID, nil)

	move := func(id, newParent uuid.UUID) error {
		return core.Update(context.Background(), entity.UpdateEntityReq{
			ID:               id,
			Name:             "dep",
			ParentID:         &newParent,
			UserID:           userID,
			ParentChanged:    true,
			EntityType:       entity.TypeDepartment,
			ExpectedParentID: &root,
		})
	}

	for i := range 20 {
		a := createDepartment(t, core, userID, &root)
		b := createDepartment(t, core, userID, &root)

		var errA, errB error
		runConcurrently(
			func() { errA = move(a, b) }, // a -> under b
			func() { errB = move(b, a) }, // b -> under a
		)

		// exactly one move wins, the other one sees the result and is rejected
		// as a cycle (not with a deadlock or another internal error)
		if errA == nil {
			require.ErrorIs(t, errB, entity.ErrParentCycle(), "iteration %d", i)
		} else {
			require.ErrorIs(t, errA, entity.ErrParentCycle(), "iteration %d", i)
			require.NoError(t, errB, "iteration %d", i)
		}
		requireNoCycle(t, repo, a)
	}
}

func TestConcurrency_CreateChildWhileDeletingParentLeavesNoOrphans(t *testing.T) {
	t.Parallel()
	repo, gdb, _ := newEntityRepo(t)
	core := newEntityCore(t, repo, gdb)
	userID := createUserForEntity(t, gdb)

	for i := range 20 {
		parent := createDepartment(t, core, userID, nil)

		var (
			childID     uuid.UUID
			errCreate   error
			errDelete   error
			createChild = func() {
				childID, errCreate = core.Create(context.Background(), entity.CreateEntityReq{
					Type:     entity.TypeArticle,
					Name:     "child",
					ParentID: &parent,
					UserID:   userID,
				})
			}
		)
		runConcurrently(
			createChild,
			func() { errDelete = core.Delete(context.Background(), parent) },
		)
		require.NoError(t, errDelete, "iteration %d", i)

		if errCreate != nil {
			// the delete won: the parent was already gone
			require.ErrorIs(t, errCreate, entity.ErrParentNotFound(), "iteration %d", i)
			continue
		}
		// the create won: the child must have been deleted together with the parent
		_, err := repo.GetListItem(t.Context(), childID)
		require.ErrorIs(t, err, entity.ErrEntityNotFound(), "iteration %d: orphaned child", i)
	}
}

func TestConcurrency_PublishDraftsStartedFromSameVersion(t *testing.T) {
	t.Parallel()
	repo, gdb, _ := newEntityRepo(t)
	core := newEntityCore(t, repo, gdb)
	userA := createUserForEntity(t, gdb)
	userB := createUserForEntity(t, gdb)

	for i := range 20 {
		id := createDepartment(t, core, userA, nil)
		for _, userID := range []uuid.UUID{userA, userB} {
			require.NoError(t, core.SaveDraft(t.Context(), entity.SaveDraftReq{
				EntityID: id, UserID: userID, Name: "dep", Content: userID.String(),
			}))
		}

		var errA, errB error
		runConcurrently(
			func() { errA = core.PublishDraft(context.Background(), id, userA) },
			func() { errB = core.PublishDraft(context.Background(), id, userB) },
		)

		// exactly one draft is published, the other one must not silently overwrite it
		winner := userA
		if errA == nil {
			require.ErrorIs(t, errB, entity.ErrDraftOutdated(), "iteration %d", i)
		} else {
			require.ErrorIs(t, errA, entity.ErrDraftOutdated(), "iteration %d", i)
			require.NoError(t, errB, "iteration %d", i)
			winner = userB
		}
		got, err := repo.Get(t.Context(), id)
		require.NoError(t, err)
		require.Equal(t, 2, *got.CurrentVersion, "iteration %d", i)
		require.Equal(t, winner.String(), got.Content, "iteration %d", i)
	}
}

// requireNoCycle walks up the parent chain and fails if it returns to the start.
func requireNoCycle(t *testing.T, repo *Repository, start uuid.UUID) {
	t.Helper()
	current := start
	for range maxDepth + 1 {
		item, err := repo.GetListItem(t.Context(), current)
		require.NoError(t, err)
		if item.ParentID == nil {
			return
		}
		require.NotEqual(t, start, *item.ParentID, "cycle detected")
		current = *item.ParentID
	}
	t.Fatalf("parent chain of %s is longer than max depth", start)
}
