package gorm

import (
	"os"
	"testing"

	uapp "github.com/66gu1/easygodocs/internal/app/user"
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

/* helpers */

func compareUsersDTO(t *testing.T, u uapp.User, name, email string, id uuid.UUID) {
	t.Helper()
	require.Equal(t, email, u.Email)
	require.Equal(t, name, u.Name)
	require.Equal(t, 0, u.SessionVersion)
	require.NotZero(t, u.CreatedAt)
	require.NotZero(t, u.UpdatedAt)
	require.Nil(t, u.DeletedAt)
	require.Equal(t, id, u.ID)
}

type testDataForCreate struct {
	req  uapp.CreateUserReq
	hash string
	id   uuid.UUID
}

/* tests */

func TestUser_CreateAndGet_ByID_And_ByEmail(t *testing.T) {
	t.Parallel()
	repo, _, cleanup := newRepo(t)

	req := uapp.CreateUserReq{
		Email: uuid.New().String() + "@ex.com",
		Name:  "John",
	}
	expHash := "phash"
	id := uuid.New()

	// create
	err := repo.CreateUser(t.Context(), req, id, expHash)
	require.NoError(t, err)

	// duplicate
	err = repo.CreateUser(t.Context(), req, uuid.New(), expHash)
	require.ErrorIs(t, err, uapp.ErrUserWithEmailAlreadyExists())

	// by ID
	u, ph, err := repo.GetUser(t.Context(), id)
	require.NoError(t, err)
	compareUsersDTO(t, u, req.Name, req.Email, id)
	require.Equal(t, expHash, ph)

	// by Email
	u, ph, err = repo.GetUserByEmail(t.Context(), req.Email)
	require.NoError(t, err)
	compareUsersDTO(t, u, req.Name, req.Email, id)
	require.Equal(t, expHash, ph)

	// not found
	_, _, err = repo.GetUserByEmail(t.Context(), "ggg@ex.com")
	require.ErrorIs(t, err, uapp.ErrUserNotFound())

	// err
	cleanup()
	_, _, err = repo.GetUser(t.Context(), id)
	require.Error(t, err)
	_, _, err = repo.GetUserByEmail(t.Context(), req.Email)
	require.Error(t, err)
	err = repo.CreateUser(t.Context(), req, uuid.New(), expHash)
	require.Error(t, err)
}

func TestUser_GetAllUsers(t *testing.T) {
	t.Parallel()
	repo, _, cleanup := newRepo(t)

	var (
		data1  = testDataForCreate{req: uapp.CreateUserReq{Email: uuid.New().String() + "@ex.com", Name: "John"}, hash: "h1", id: uuid.New()}
		data2  = testDataForCreate{req: uapp.CreateUserReq{Email: uuid.New().String() + "@ex.com", Name: "Mike"}, hash: "h2", id: uuid.New()}
		data3  = testDataForCreate{req: uapp.CreateUserReq{Email: uuid.New().String() + "@ex.com", Name: "Alice"}, hash: "h3", id: uuid.New()}
		expMap = map[uuid.UUID]testDataForCreate{
			data1.id: data1,
			data2.id: data2,
			data3.id: data3,
		}
	)
	// create
	for _, d := range expMap {
		err := repo.CreateUser(t.Context(), d.req, d.id, d.hash)
		require.NoError(t, err)
	}

	// get all
	list, err := repo.GetAllUsers(t.Context())
	require.NoError(t, err)
	require.Len(t, list, 3)
	for _, d := range list {
		exp, ok := expMap[d.ID]
		require.True(t, ok)
		compareUsersDTO(t, d, exp.req.Name, exp.req.Email, exp.id)
		delete(expMap, d.ID)
	}

	// err
	cleanup()
	_, err = repo.GetAllUsers(t.Context())
	require.Error(t, err)
}

func TestUser_UpdateUser_Success_Duplicate_NotFound(t *testing.T) {
	t.Parallel()
	repo, _, cleanup := newRepo(t)

	var (
		data1 = testDataForCreate{req: uapp.CreateUserReq{Email: uuid.New().String() + "@ex.com", Name: "John"}, hash: "h1", id: uuid.New()}
		data2 = testDataForCreate{req: uapp.CreateUserReq{Email: uuid.New().String() + "@ex.com", Name: "Mike"}, hash: "h2", id: uuid.New()}
	)

	// create
	for _, d := range []testDataForCreate{data1, data2} {
		err := repo.CreateUser(t.Context(), d.req, d.id, d.hash)
		require.NoError(t, err)
	}

	// success
	newEmail := uuid.New().String() + "@ex.com"
	err := repo.UpdateUser(t.Context(), uapp.UpdateUserReq{
		UserID: data1.id, Name: "John Doe", Email: newEmail,
	})
	require.NoError(t, err)

	u, _, err := repo.GetUser(t.Context(), data1.id)
	require.NoError(t, err)
	require.Equal(t, "John Doe", u.Name)
	require.Equal(t, newEmail, u.Email)
	require.True(t, u.UpdatedAt.After(u.CreatedAt))

	// duplicate email
	err = repo.UpdateUser(t.Context(), uapp.UpdateUserReq{
		UserID: data1.id, Name: "No matter", Email: data2.req.Email,
	})
	require.Error(t, err)
	require.ErrorIs(t, err, uapp.ErrUserWithEmailAlreadyExists())

	// not found
	err = repo.UpdateUser(t.Context(), uapp.UpdateUserReq{
		UserID: uuid.New(), Name: "X", Email: "x@example.com",
	})
	require.Error(t, err)
	require.Equal(t, apperr.CodeOf(uapp.ErrUserNotFound()), apperr.CodeOf(err))

	// err
	cleanup()
	err = repo.UpdateUser(t.Context(), uapp.UpdateUserReq{
		UserID: data2.id, Name: "Y", Email: "y@example.com",
	})
	require.Error(t, err)
}

func TestUser_DeleteUser_Success_And_NotFound(t *testing.T) {
	t.Parallel()
	repo, _, cleanup := newRepo(t)

	id := uuid.New()
	// create
	err := repo.CreateUser(t.Context(), uapp.CreateUserReq{Email: uuid.New().String() + "@ex.com", Name: "ToDelete"}, id, "hash")
	require.NoError(t, err)

	require.NoError(t, repo.DeleteUser(t.Context(), id))

	// not found
	_, _, err = repo.GetUser(t.Context(), id)
	require.Error(t, err)
	require.ErrorIs(t, err, uapp.ErrUserNotFound())

	// not found
	err = repo.DeleteUser(t.Context(), id)
	require.Error(t, err)
	require.ErrorIs(t, err, uapp.ErrUserNotFound())

	// err
	cleanup()
	err = repo.DeleteUser(t.Context(), uuid.New())
	require.Error(t, err)
}

func TestUser_ChangePassword_IncrementsSessionVersion(t *testing.T) {
	t.Parallel()
	repo, _, cleanup := newRepo(t)

	id := uuid.New()
	require.NoError(t, repo.CreateUser(t.Context(), uapp.CreateUserReq{Email: uuid.New().String() + "@ex.com", Name: "Changeme"}, id, "oldhash"))

	// success
	require.NoError(t, repo.ChangePassword(t.Context(), id, "newhash"))
	u2, ph2, err := repo.GetUser(t.Context(), id)
	require.NoError(t, err)
	require.Equal(t, "newhash", ph2)
	require.Equal(t, 1, u2.SessionVersion)

	// not found
	errNF := repo.ChangePassword(t.Context(), uuid.New(), "zzz")
	require.Error(t, errNF)
	require.Equal(t, apperr.CodeOf(uapp.ErrUserNotFound()), apperr.CodeOf(errNF))

	// err
	cleanup()
	err = repo.ChangePassword(t.Context(), id, "xxx")
	require.Error(t, err)
}

func TestNewRepository(t *testing.T) {
	t.Parallel()

	_, err := NewRepository(nil)
	require.Error(t, err)
}
