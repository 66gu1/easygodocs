package gorm

import (
	"os"
	"testing"
	"time"

	"github.com/66gu1/easygodocs/internal/app/entity"
	"github.com/66gu1/easygodocs/internal/infrastructure/db"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

var shared *db.TestDB

func TestMain(m *testing.M) {
	var stop func()
	shared, stop = db.StartPostgres()
	code := m.Run()
	stop()
	os.Exit(code)
}

func newEntityRepo(t *testing.T, cfg Config) (*gormRepo, *gorm.DB, func()) {
	gdb, _, cleanup := shared.CreateIsolatedDB(t)
	t.Cleanup(cleanup)
	repo, err := NewRepository(gdb, cfg)
	require.NoError(t, err)
	return repo, gdb, cleanup
}

/* --- helpers --- */

func createUserForEntity(t *testing.T, gdb *gorm.DB) uuid.UUID {
	t.Helper()
	id := uuid.New()
	err := gdb.WithContext(t.Context()).Exec(
		`INSERT INTO users(id,email,name,password_hash,created_at,updated_at,session_version)
		 VALUES ($1,$2,'name','hash',NOW(),NOW(),0)`,
		id, uuid.New().String(),
	).Error
	require.NoError(t, err)
	return id
}

func compareEntityDTO(t *testing.T, e entity.Entity, eType entity.Type, name, content string, id, createdBy, updatedBy uuid.UUID, parentID *uuid.UUID, currentVersion *int) {
	t.Helper()
	require.Equal(t, id, e.ID)
	require.Equal(t, eType, e.Type)
	require.Equal(t, name, e.Name)
	require.Equal(t, content, e.Content)
	require.Equal(t, parentID, e.ParentID)
	require.Equal(t, createdBy, e.CreatedBy)
	require.Equal(t, updatedBy, e.UpdatedBy)
	require.Equal(t, currentVersion, e.CurrentVersion)
	require.NotZero(t, e.CreatedAt)
	require.NotZero(t, e.UpdatedAt)
}

/* --- tests --- */

func TestEntity_Create_Get_Versions_Update(t *testing.T) {
	t.Parallel()
	repo, gdb, cleanup := newEntityRepo(t, Config{MaxHierarchyDepth: 4})

	userID := createUserForEntity(t, gdb)
	userID2 := createUserForEntity(t, gdb)

	now := time.Now().UTC().Truncate(time.Second)
	id := uuid.New()
	req := entity.CreateEntityReq{
		Type:     entity.Type("t"),
		Name:     "root",
		Content:  "v1",
		ParentID: nil,
		UserID:   userID,
	}
	require.NoError(t, repo.Create(t.Context(), req, id, now))

	// Get + GetVersion(1)
	dto, err := repo.Get(t.Context(), id)
	require.NoError(t, err)
	compareEntityDTO(t, dto, req.Type, req.Name, req.Content, id, userID, userID, req.ParentID, &[]int{1}[0])
	dto, err = repo.GetVersion(t.Context(), id, 1)
	require.NoError(t, err)
	compareEntityDTO(t, dto, "", req.Name, req.Content, id, userID, userID, req.ParentID, &[]int{1}[0])

	// Update -> version 2
	reqUp := entity.UpdateEntityReq{
		ID:       id,
		Name:     "root-2",
		Content:  "v2",
		ParentID: nil,
		UserID:   userID2,
	}
	require.NoError(t, repo.Update(t.Context(), reqUp, now.Add(time.Minute)))

	// Get + GetVersion(2)
	dto, err = repo.Get(t.Context(), id)
	require.NoError(t, err)
	compareEntityDTO(t, dto, req.Type, reqUp.Name, reqUp.Content, id, userID, userID2, reqUp.ParentID, &[]int{2}[0])
	dto, err = repo.GetVersion(t.Context(), id, 2)
	require.NoError(t, err)
	compareEntityDTO(t, dto, "", reqUp.Name, reqUp.Content, id, userID2, userID2, reqUp.ParentID, &[]int{2}[0])

	// Versions list: [2,1]
	vs, err := repo.GetVersionsList(t.Context(), id)
	require.NoError(t, err)
	require.Len(t, vs, 2)
	compareEntityDTO(t, vs[0], "", reqUp.Name, reqUp.Content, id, userID2, userID2, reqUp.ParentID, &[]int{2}[0])
	compareEntityDTO(t, vs[1], "", req.Name, req.Content, id, userID, userID, req.ParentID, &[]int{1}[0])

	// not found
	_, err = repo.Get(t.Context(), uuid.New())
	require.ErrorIs(t, err, entity.ErrEntityNotFound())
	_, err = repo.GetVersion(t.Context(), id, 999)
	require.ErrorIs(t, err, entity.ErrEntityNotFound())
	err = repo.Update(t.Context(), entity.UpdateEntityReq{}, time.Now().UTC())
	require.ErrorIs(t, err, entity.ErrEntityNotFound())

	// err
	cleanup()
	_, err = repo.Get(t.Context(), id)
	require.Error(t, err)
	err = repo.Delete(t.Context(), id, time.Now().UTC())
	require.Error(t, err)
	_, err = repo.GetVersion(t.Context(), id, 1)
	require.Error(t, err)
	_, err = repo.GetVersionsList(t.Context(), id)
	require.Error(t, err)
	err = repo.Update(t.Context(), reqUp, time.Now().UTC())
	require.Error(t, err)
	err = repo.Create(t.Context(), req, uuid.New(), time.Now().UTC())
	require.Error(t, err)
}

func TestEntity_CreateDraft_And_UpdateDraft(t *testing.T) {
	t.Parallel()
	repo, gdb, cleanup := newEntityRepo(t, Config{MaxHierarchyDepth: 4})

	userID := createUserForEntity(t, gdb)

	now := time.Now().UTC().Truncate(time.Second)
	id := uuid.New()
	req := entity.CreateEntityReq{
		Type:    "t",
		Name:    "draft",
		Content: "d0",
		UserID:  userID,
	}

	// create draft, version = nil
	require.NoError(t, repo.CreateDraft(t.Context(), req, id))
	dto, err := repo.Get(t.Context(), id)
	require.NoError(t, err)
	compareEntityDTO(t, dto, req.Type, req.Name, req.Content, id, userID, userID, req.ParentID, nil)
	vs, err := repo.GetVersionsList(t.Context(), id)
	require.NoError(t, err)
	require.Len(t, vs, 0)

	// update, version = 1
	reqUpd := entity.UpdateEntityReq{
		ID:      id,
		Name:    "draft-1",
		Content: "d1",
		UserID:  userID,
	}
	require.NoError(t, repo.Update(t.Context(), reqUpd, now))
	dto, err = repo.Get(t.Context(), id)
	require.NoError(t, err)
	require.Equal(t, &[]int{1}[0], dto.CurrentVersion)
	vs, err = repo.GetVersionsList(t.Context(), id)
	require.NoError(t, err)
	require.Len(t, vs, 1)

	// UpdateDraft, version = nil
	reqUpd = entity.UpdateEntityReq{
		ID:       id,
		Name:     "draft-2",
		Content:  "d1",
		ParentID: nil,
		UserID:   userID,
	}
	require.NoError(t, repo.UpdateDraft(t.Context(), reqUpd))
	dto, err = repo.Get(t.Context(), id)
	require.NoError(t, err)
	compareEntityDTO(t, dto, req.Type, reqUpd.Name, reqUpd.Content, id, userID, userID, reqUpd.ParentID, nil)
	vs, err = repo.GetVersionsList(t.Context(), id)
	require.NoError(t, err)
	require.Len(t, vs, 1)

	// not found
	err = repo.UpdateDraft(t.Context(), entity.UpdateEntityReq{})
	require.ErrorIs(t, err, entity.ErrEntityNotFound())

	// err
	cleanup()
	err = repo.UpdateDraft(t.Context(), reqUpd)
	require.Error(t, err)
	err = repo.CreateDraft(t.Context(), req, id)
	require.Error(t, err)
}

func TestEntity_GetListItem_And_GetAll(t *testing.T) {
	t.Parallel()
	repo, gdb, cleanup := newEntityRepo(t, Config{MaxHierarchyDepth: 4})

	userID := createUserForEntity(t, gdb)

	req1 := entity.CreateEntityReq{
		Type: "t", Name: "A", Content: "c1", UserID: userID,
	}
	id1 := uuid.New()
	require.NoError(t, repo.Create(t.Context(), req1, id1, time.Now().UTC()))
	id2 := uuid.New()
	req2 := entity.CreateEntityReq{
		Type: entity.Type("t"), Name: "B", Content: "c2", UserID: userID,
	}
	require.NoError(t, repo.Create(t.Context(), req2, id2, time.Now().UTC()))

	exp1 := entity.ListItem{ID: id1, Type: req1.Type, Name: req1.Name, ParentID: req1.ParentID}
	exp2 := entity.ListItem{ID: id2, Type: req2.Type, Name: req2.Name, ParentID: req2.ParentID}
	li, err := repo.GetListItem(t.Context(), id1)
	require.NoError(t, err)
	require.Equal(t, exp1, li)

	expSlice := []entity.ListItem{exp1, exp2}
	all, err := repo.GetAll(t.Context())
	require.NoError(t, err)
	require.ElementsMatch(t, all, expSlice)

	// not found
	_, err = repo.GetListItem(t.Context(), uuid.New())
	require.ErrorIs(t, err, entity.ErrEntityNotFound())

	// негатив
	cleanup()
	_, err = repo.GetListItem(t.Context(), id1)
	require.Error(t, err)
	_, err = repo.GetAll(t.Context())
	require.Error(t, err)
}

func TestEntity_PermittedHierarchy(t *testing.T) {
	t.Parallel()
	repo, gdb, cleanup := newEntityRepo(t, Config{MaxHierarchyDepth: 6})
	userID := createUserForEntity(t, gdb)

	// root -> c1 -> gc1 ; root -> c2
	root := uuid.New()
	require.NoError(t, repo.Create(t.Context(), entity.CreateEntityReq{
		Type: "t", Name: "root", Content: "", UserID: userID,
	}, root, time.Now().UTC()))
	c1 := uuid.New()
	require.NoError(t, repo.Create(t.Context(), entity.CreateEntityReq{
		Type: "t", Name: "c1", Content: "", ParentID: &root, UserID: userID,
	}, c1, time.Now().UTC()))
	gc1 := uuid.New()
	require.NoError(t, repo.Create(t.Context(), entity.CreateEntityReq{
		Type: "t", Name: "gc1", Content: "", ParentID: &c1, UserID: userID,
	}, gc1, time.Now().UTC()))
	c2 := uuid.New()
	require.NoError(t, repo.Create(t.Context(), entity.CreateEntityReq{
		Type: "t", Name: "c2", Content: "", ParentID: &root, UserID: userID,
	}, c2, time.Now().UTC()))

	// empty permissions
	res, err := repo.GetPermittedHierarchy(t.Context(), []uuid.UUID{}, true)
	require.NoError(t, err)
	require.Equal(t, []entity.ListItem{}, res)
	// only for read
	// permissions = [c1] → {root, c1, gc1}
	res, err = repo.GetPermittedHierarchy(t.Context(), []uuid.UUID{c1}, true)
	require.NoError(t, err)

	ids := map[uuid.UUID]bool{}
	for _, li := range res {
		ids[li.ID] = true
	}
	require.True(t, ids[root] && ids[c1] && ids[gc1])
	require.False(t, ids[c2])

	// not only for read
	// permissions = [c1] → {c1, gc1}
	res, err = repo.GetPermittedHierarchy(t.Context(), []uuid.UUID{c1}, false)
	require.NoError(t, err)
	ids = map[uuid.UUID]bool{}
	for _, li := range res {
		ids[li.ID] = true
	}
	require.False(t, ids[root])
	require.True(t, ids[c1] && ids[gc1])
	require.False(t, ids[c2])

	// негатив
	cleanup()
	_, err = repo.GetPermittedHierarchy(t.Context(), []uuid.UUID{c1}, true)
	require.Error(t, err)
}

func TestEntity_Delete_SoftMarksRecursively(t *testing.T) {
	t.Parallel()
	repo, gdb, _ := newEntityRepo(t, Config{MaxHierarchyDepth: 6})
	userID := createUserForEntity(t, gdb)

	root := uuid.New()
	require.NoError(t, repo.Create(t.Context(), entity.CreateEntityReq{
		Type: "t", Name: "root", Content: "", UserID: userID,
	}, root, time.Now().UTC()))
	child := uuid.New()
	require.NoError(t, repo.Create(t.Context(), entity.CreateEntityReq{
		Type: "t", Name: "child", Content: "", ParentID: &root, UserID: userID,
	}, child, time.Now().UTC()))

	// delete root → пометятся root и child
	delAt := time.Now().UTC()
	require.NoError(t, repo.Delete(t.Context(), root, delAt))

	var cnt int
	err := gdb.WithContext(t.Context()).
		Raw(`SELECT COUNT(*) FROM entities WHERE id IN ($1,$2) AND deleted_at IS NOT NULL`, root, child).
		Scan(&cnt).Error
	require.NoError(t, err)
	require.Equal(t, 2, cnt)

	// not found
	err = repo.Delete(t.Context(), uuid.New(), time.Now().UTC())
	require.ErrorIs(t, err, entity.ErrEntityNotFound())
}

func TestEntity_ValidateChangedParent_Cycle(t *testing.T) {
	t.Parallel()
	repo, gdb, clean := newEntityRepo(t, Config{MaxHierarchyDepth: 6})
	userID := createUserForEntity(t, gdb)

	a := uuid.New()
	require.NoError(t, repo.Create(t.Context(), entity.CreateEntityReq{
		Type: "t", Name: "A", Content: "", UserID: userID,
	}, a, time.Now().UTC()))
	b := uuid.New()
	require.NoError(t, repo.Create(t.Context(), entity.CreateEntityReq{
		Type: "t", Name: "B", Content: "", ParentID: &a, UserID: userID,
	}, b, time.Now().UTC()))

	// success
	err := repo.ValidateChangedParent(t.Context(), a, uuid.New())
	require.NoError(t, err)

	// try to set A as child of B → cycle
	err = repo.ValidateChangedParent(t.Context(), a, b)
	require.ErrorIs(t, err, entity.ErrParentCycle())

	// err
	clean()
	err = repo.ValidateChangedParent(t.Context(), a, uuid.New())
	require.Error(t, err)
}

func TestEntity_ValidateChangedParent_MaxDepth(t *testing.T) {
	t.Parallel()
	// MaxDepth=3
	repo, gdb, _ := newEntityRepo(t, Config{MaxHierarchyDepth: 3})
	userID := createUserForEntity(t, gdb)

	// parent chain: P0 <- P1 <- P2
	p0 := uuid.New()
	require.NoError(t, repo.Create(t.Context(), entity.CreateEntityReq{
		Type: "t", Name: "P0", Content: "", UserID: userID,
	}, p0, time.Now().UTC()))
	p1 := uuid.New()
	require.NoError(t, repo.Create(t.Context(), entity.CreateEntityReq{
		Type: "t", Name: "P1", Content: "", ParentID: &p0, UserID: userID,
	}, p1, time.Now().UTC()))
	p2 := uuid.New()
	require.NoError(t, repo.Create(t.Context(), entity.CreateEntityReq{
		Type: "t", Name: "P2", Content: "", ParentID: &p1, UserID: userID,
	}, p2, time.Now().UTC()))

	// X -> Xc
	x := uuid.New()
	require.NoError(t, repo.Create(t.Context(), entity.CreateEntityReq{
		Type: "t", Name: "X", Content: "", UserID: userID,
	}, x, time.Now().UTC()))
	xc := uuid.New()
	require.NoError(t, repo.Create(t.Context(), entity.CreateEntityReq{
		Type: "t", Name: "Xc", Content: "", ParentID: &x, UserID: userID,
	}, xc, time.Now().UTC()))

	err := repo.ValidateChangedParent(t.Context(), x, p1)
	require.ErrorIs(t, err, entity.ErrMaxHierarchyDepthExceeded(repo.cfg.MaxHierarchyDepth))
}

func TestEntity_CheckParentDepthLimit(t *testing.T) {
	t.Parallel()
	repo, gdb, cleanup := newEntityRepo(t, Config{MaxHierarchyDepth: 3})
	userID := createUserForEntity(t, gdb)

	// A <- B <- C
	a := uuid.New()
	require.NoError(t, repo.Create(t.Context(), entity.CreateEntityReq{
		Type: "t", Name: "A", Content: "", UserID: userID,
	}, a, time.Now().UTC()))
	b := uuid.New()
	require.NoError(t, repo.Create(t.Context(), entity.CreateEntityReq{
		Type: "t", Name: "B", Content: "", ParentID: &a, UserID: userID,
	}, b, time.Now().UTC()))
	c := uuid.New()
	require.NoError(t, repo.Create(t.Context(), entity.CreateEntityReq{
		Type: "t", Name: "C", Content: "", ParentID: &b, UserID: userID,
	}, c, time.Now().UTC()))

	// C: MAX(depth)=3 → 3+1 > 3 → err
	err := repo.CheckParentDepthLimit(t.Context(), c)
	require.ErrorIs(t, err, entity.ErrMaxHierarchyDepthExceeded(repo.cfg.MaxHierarchyDepth))

	// B: MAX(depth)=3 → 2+1 = 3 → ok
	require.NoError(t, repo.CheckParentDepthLimit(t.Context(), b))

	// err
	cleanup()
	err = repo.CheckParentDepthLimit(t.Context(), b)
	require.Error(t, err)
}

func TestNewRepository(t *testing.T) {
	t.Parallel()

	_, err := NewRepository(nil, Config{MaxHierarchyDepth: 3})
	require.Error(t, err)

	_, err = NewRepository(&gorm.DB{}, Config{MaxHierarchyDepth: 0})
	require.Error(t, err)
}
