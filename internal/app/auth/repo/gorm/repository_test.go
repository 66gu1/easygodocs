package gorm

import (
	"os"
	"testing"
	"time"

	"github.com/66gu1/easygodocs/internal/app/auth"
	"github.com/66gu1/easygodocs/internal/infrastructure/apperr"
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

func newRepo(t *testing.T) (*gormRepo, *gorm.DB, func()) {
	gdb, _, cleanup := shared.CreateIsolatedDB(t)
	t.Cleanup(cleanup)
	repo, err := NewRepository(gdb)
	require.NoError(t, err)
	return repo, gdb, cleanup
}

func TestCreateAndGetSessionByID(t *testing.T) {
	t.Parallel()
	repo, gdb, cleanup := newRepo(t)

	uid := createUser(t, gdb)
	sid := uuid.New()

	now := time.Now().UTC().Truncate(time.Second)
	sess := auth.Session{
		ID:             sid,
		UserID:         uid,
		CreatedAt:      now,
		ExpiresAt:      now.Add(24 * time.Hour),
		SessionVersion: 1,
	}

	require.NoError(t, repo.CreateSession(t.Context(), sess, "hash-1"))

	got, rtHash, err := repo.GetSessionByID(t.Context(), sid)
	require.NoError(t, err)
	compareSessions(t, sess, got)
	require.Equal(t, "hash-1", rtHash)

	// pool closed error
	cleanup()
	_, _, err = repo.GetSessionByID(t.Context(), sid)
	require.Error(t, err)
	err = repo.CreateSession(t.Context(), sess, "hash-2")
	require.Error(t, err)
}

func TestGetSessionsByUserID(t *testing.T) {
	t.Parallel()
	repo, gdb, cleanup := newRepo(t)

	u1 := createUser(t, gdb)
	u2 := createUser(t, gdb)

	now := time.Now().UTC()

	s1 := auth.Session{
		ID:             uuid.New(),
		UserID:         u1,
		CreatedAt:      now,
		ExpiresAt:      now.Add(2 * time.Hour),
		SessionVersion: 1,
	}
	s2 := auth.Session{
		ID:             uuid.New(),
		UserID:         u1,
		CreatedAt:      now,
		ExpiresAt:      now.Add(3 * time.Hour),
		SessionVersion: 1,
	}
	s3 := auth.Session{
		ID:             uuid.New(),
		UserID:         u2,
		CreatedAt:      now,
		ExpiresAt:      now.Add(4 * time.Hour),
		SessionVersion: 1,
	}
	resMap := map[string]auth.Session{
		s1.ID.String(): s1,
		s2.ID.String(): s2,
	}

	require.NoError(t, repo.CreateSession(t.Context(), s1, "h1"))
	require.NoError(t, repo.CreateSession(t.Context(), s2, "h2"))
	require.NoError(t, repo.CreateSession(t.Context(), s3, "h3"))

	list, err := repo.GetSessionsByUserID(t.Context(), u1)
	require.NoError(t, err)
	require.Len(t, list, 2)
	for _, it := range list {
		if exp, ok := resMap[it.ID.String()]; ok {
			compareSessions(t, it, exp)
			delete(resMap, it.ID.String())
		} else {
			t.Errorf("unexpected session ID: %s", it.ID.String())
		}
	}

	cleanup()
	_, err = repo.GetSessionsByUserID(t.Context(), u1)
	require.Error(t, err)
}

func TestDeleteSessionByID(t *testing.T) {
	t.Parallel()
	repo, gdb, cleanup := newRepo(t)

	u := createUser(t, gdb)
	sid := uuid.New()
	now := time.Now().UTC()

	sess := auth.Session{
		ID:             sid,
		UserID:         u,
		CreatedAt:      now,
		ExpiresAt:      now.Add(time.Hour),
		SessionVersion: 1,
	}
	require.NoError(t, repo.CreateSession(t.Context(), sess, "h"))

	require.NoError(t, repo.DeleteSessionByID(t.Context(), sid))

	_, _, err := repo.GetSessionByID(t.Context(), sid)
	require.Error(t, err)
	require.ErrorIs(t, err, ErrSessionNotFound)

	err = repo.DeleteSessionByID(t.Context(), sid)
	require.Error(t, err)
	require.ErrorIs(t, err, ErrSessionNotFound)

	cleanup()
	err = repo.DeleteSessionByID(t.Context(), sid)
	require.Error(t, err)
}

func TestDeleteSessionByIDAndUser(t *testing.T) {
	t.Parallel()
	repo, gdb, cleanup := newRepo(t)

	userOK := createUser(t, gdb)
	userWrong := createUser(t, gdb)
	sid := uuid.New()
	now := time.Now().UTC()

	sess := auth.Session{
		ID:             sid,
		UserID:         userOK,
		CreatedAt:      now,
		ExpiresAt:      now.Add(time.Hour),
		SessionVersion: 1,
	}
	require.NoError(t, repo.CreateSession(t.Context(), sess, "h"))

	// user => NotFound
	err := repo.DeleteSessionByIDAndUser(t.Context(), sid, userWrong)
	require.Error(t, err)
	require.ErrorIs(t, err, ErrSessionNotFound)

	// success
	require.NoError(t, repo.DeleteSessionByIDAndUser(t.Context(), sid, userOK))

	_, _, err = repo.GetSessionByID(t.Context(), sid)
	require.Error(t, err)
	require.ErrorIs(t, err, ErrSessionNotFound)

	cleanup()
	err = repo.DeleteSessionByIDAndUser(t.Context(), sid, userOK)
	require.Error(t, err)
}

func TestDeleteSessionsByUserID(t *testing.T) {
	t.Parallel()

	repo, gdb, cleanup := newRepo(t)

	u := createUser(t, gdb)
	now := time.Now().UTC()

	for i := 0; i < 3; i++ {
		s := auth.Session{
			ID:             uuid.New(),
			UserID:         u,
			CreatedAt:      now,
			ExpiresAt:      now.Add(time.Duration(i+1) * time.Hour),
			SessionVersion: 1,
		}
		require.NoError(t, repo.CreateSession(t.Context(), s, "h"))
	}

	require.NoError(t, repo.DeleteSessionsByUserID(t.Context(), u))

	list, err := repo.GetSessionsByUserID(t.Context(), u)
	require.NoError(t, err)
	require.Len(t, list, 0)

	cleanup()
	err = repo.DeleteSessionsByUserID(t.Context(), u)
	require.Error(t, err)
}

func TestUpdateRefreshToken(t *testing.T) {
	t.Parallel()
	repo, gdb, cleanup := newRepo(t)

	u := createUser(t, gdb)
	sid := uuid.New()
	now := time.Now().UTC()

	s := auth.Session{
		ID:             sid,
		UserID:         u,
		CreatedAt:      now,
		ExpiresAt:      now.Add(30 * time.Minute),
		SessionVersion: 1,
	}
	require.NoError(t, repo.CreateSession(t.Context(), s, "old-hash"))

	// success
	req := auth.UpdateTokenReq{
		SessionID:           sid,
		OldRefreshTokenHash: "old-hash",
		RefreshTokenHash:    "new-hash",
		ExpiresAt:           now.Add(2 * time.Hour),
		UserID:              u,
	}
	require.NoError(t, repo.UpdateRefreshToken(t.Context(), req))

	got, rt, err := repo.GetSessionByID(t.Context(), sid)
	require.NoError(t, err)
	require.Equal(t, "new-hash", rt)
	require.WithinDuration(t, req.ExpiresAt, got.ExpiresAt, time.Second)

	// old hash -> NotFound
	req2 := auth.UpdateTokenReq{
		SessionID:           sid,
		OldRefreshTokenHash: "wrong",
		RefreshTokenHash:    "newer",
		ExpiresAt:           now.Add(3 * time.Hour),
		UserID:              u,
	}
	err = repo.UpdateRefreshToken(t.Context(), req2)
	require.Error(t, err)
	require.ErrorIs(t, err, ErrSessionNotFound)

	cleanup()
	err = repo.UpdateRefreshToken(t.Context(), req)
	require.Error(t, err)
}

func TestAddListDeleteUserRole_NullEntity(t *testing.T) {
	t.Parallel()
	repo, gdb, cleanup := newRepo(t)

	u := createUser(t, gdb)
	role := auth.Role("writer")
	item := auth.UserRole{UserID: u, Role: role}

	// add
	require.NoError(t, repo.AddUserRole(t.Context(), item))

	// duplicate -> conflict
	err := repo.AddUserRole(t.Context(), item)
	require.Error(t, err)
	require.True(t, apperr.CodeOf(err) == auth.CodeRoleDuplicate)

	// list
	got, err := repo.ListUserRoles(t.Context(), u)
	require.NoError(t, err)
	require.Len(t, got, 1)
	require.Equal(t, item, got[0])

	// delete (success)
	require.NoError(t, repo.DeleteUserRole(t.Context(), item))

	// list again -> empty
	got, err = repo.ListUserRoles(t.Context(), u)
	require.NoError(t, err)
	require.Len(t, got, 0)

	// delete again -> not found
	err = repo.DeleteUserRole(t.Context(), item)
	require.Error(t, err)
	require.ErrorIs(t, err, ErrRoleNotFound)

	// pool closed -> any op fails
	cleanup()
	err = repo.AddUserRole(t.Context(), item)
	require.Error(t, err)
	err = repo.DeleteUserRole(t.Context(), item)
	require.Error(t, err)
	_, err = repo.ListUserRoles(t.Context(), u)
	require.Error(t, err)
}

func TestAddUserRole_WithEntity_AndGetFilter(t *testing.T) {
	t.Parallel()
	repo, gdb, cleanup := newRepo(t)

	u := createUser(t, gdb)

	e1 := createEntity(t, gdb, u)
	ur1 := auth.UserRole{UserID: createUser(t, gdb), Role: auth.RoleWrite}
	ur2 := auth.UserRole{UserID: u, Role: auth.RoleRead}
	ur3 := auth.UserRole{UserID: u, Role: auth.RoleWrite, EntityID: &e1}
	ur4 := auth.UserRole{UserID: u, Role: auth.Role("admin")}
	urs := []auth.UserRole{ur1, ur2, ur3, ur4}

	for _, it := range urs {
		require.NoError(t, repo.AddUserRole(t.Context(), it))
	}

	exp := []auth.UserRole{ur2, ur3}
	got, err := repo.GetUserRoles(t.Context(), u, []auth.Role{auth.RoleWrite, auth.RoleRead})
	require.NoError(t, err)
	require.ElementsMatch(t, exp, got)

	// err on closed pool
	cleanup()
	_, err = repo.GetUserRoles(t.Context(), u, []auth.Role{auth.RoleWrite})
	require.Error(t, err)
}

func TestDeleteUserRole_WithAndWithoutEntity(t *testing.T) {
	t.Parallel()
	repo, gdb, _ := newRepo(t)

	u := createUser(t, gdb)
	e1 := createEntity(t, gdb, u)
	r := auth.Role("member")

	// добавим две записи: с entity и без
	withEnt := auth.UserRole{UserID: u, Role: r, EntityID: &e1}
	withoutEnt := auth.UserRole{UserID: u, Role: r, EntityID: nil}

	require.NoError(t, repo.AddUserRole(t.Context(), withEnt))
	require.NoError(t, repo.AddUserRole(t.Context(), withoutEnt))

	require.NoError(t, repo.DeleteUserRole(t.Context(), withEnt))

	all, err := repo.ListUserRoles(t.Context(), u)
	require.NoError(t, err)
	require.Len(t, all, 1)
	require.Equal(t, withoutEnt.Role, all[0].Role)
	require.Nil(t, all[0].EntityID)

	// удаляем оставшуюся
	require.NoError(t, repo.DeleteUserRole(t.Context(), withoutEnt))
}

func createUser(t *testing.T, gdb *gorm.DB) uuid.UUID {
	t.Helper()

	uid := uuid.New()
	email := uid.String() + "@example.com"
	err := gdb.WithContext(t.Context()).Exec(
		`INSERT INTO users(id,email,name,password_hash,created_at,updated_at,session_version)
         VALUES ($1,$2,$3,$4,NOW(),NOW(),$5)`,
		uid, email, "Test", "hash", 0,
	).Error
	require.NoError(t, err)

	return uid
}

func createEntity(t *testing.T, gdb *gorm.DB, userID uuid.UUID) uuid.UUID {
	t.Helper()

	eid := uuid.New()
	err := gdb.WithContext(t.Context()).Exec(
		`INSERT INTO entities(id,type,created_at,updated_at,name,content,created_by,updated_by)
		 VALUES ($1,'t',NOW(),NOW(),'name','',$2,$2)`,
		eid, userID,
	).Error
	require.NoError(t, err)

	return eid
}

func compareSessions(t *testing.T, exp, got auth.Session) {
	t.Helper()

	require.Equal(t, exp.ID, got.ID)
	require.Equal(t, exp.UserID, got.UserID)
	require.WithinDuration(t, exp.CreatedAt, got.CreatedAt, time.Second)
	require.WithinDuration(t, exp.ExpiresAt, got.ExpiresAt, time.Second)
	require.Equal(t, exp.SessionVersion, got.SessionVersion)
}

func TestNewRepository(t *testing.T) {
	t.Parallel()

	_, err := NewRepository(nil)
	require.Error(t, err)
}
