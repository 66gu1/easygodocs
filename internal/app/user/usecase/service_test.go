package usecase_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/66gu1/easygodocs/internal/app/user"
	"github.com/66gu1/easygodocs/internal/app/user/usecase"
	"github.com/66gu1/easygodocs/internal/app/user/usecase/mocks"
	"github.com/66gu1/easygodocs/internal/infrastructure/secure"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

//go:generate minimock -o ./mocks -s _mock.go

type mock struct {
	core           *mocks.CoreMock
	authService    *mocks.AuthServiceMock
	passwordHasher *mocks.PasswordHasherMock
}

func getMocks(t *testing.T) mock {
	t.Helper()
	return mock{
		core:           mocks.NewCoreMock(t),
		authService:    mocks.NewAuthServiceMock(t),
		passwordHasher: mocks.NewPasswordHasherMock(t),
	}
}

func TestService_CreateUser(t *testing.T) {
	t.Parallel()

	var (
		req = user.CreateUserReq{
			Email:    "user@mail.com",
			Name:     "name",
			Password: []byte("password"),
		}
		ctx    = t.Context()
		expErr = errors.New("user already exists")
	)

	tests := []struct {
		name  string
		setup func(mocks mock)
		err   error
	}{
		{
			name: "ok",
			setup: func(mocks mock) {
				mocks.core.CreateUserMock.Expect(ctx, req).Return(uuid.Nil, nil)
			},
		},
		{
			name: "core.CreateUser returns error",
			setup: func(mocks mock) {
				mocks.core.CreateUserMock.Expect(ctx, req).Return(uuid.Nil, expErr)
			},
			err: expErr,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			mocks := getMocks(t)
			if tt.setup != nil {
				tt.setup(mocks)
			}

			svc := usecase.NewService(mocks.core, mocks.authService, mocks.passwordHasher)
			err := svc.CreateUser(ctx, req)
			if tt.err != nil {
				require.ErrorIs(t, err, tt.err)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestService_GetUser(t *testing.T) {
	t.Parallel()

	var (
		userID = uuid.New()
		ctx    = t.Context()
		expErr = errors.New("user not found")
		user   = user.User{
			ID:             userID,
			Email:          "user@mail.com",
			Name:           "name",
			SessionVersion: 1,
			CreatedAt:      time.Now(),
			UpdatedAt:      time.Now(),
		}
	)

	tests := []struct {
		name  string
		setup func(mocks mock)
		err   error
	}{
		{
			name: "ok",
			setup: func(mocks mock) {
				mocks.authService.CheckSelfOrAdminMock.Expect(ctx, userID).Return(nil)
				mocks.core.GetUserMock.Expect(ctx, userID).Return(user, "", nil)
			},
		},
		{
			name: "authService.CheckSelfOrAdmin returns error",
			setup: func(mocks mock) {
				mocks.authService.CheckSelfOrAdminMock.Expect(ctx, userID).Return(expErr)
			},
			err: expErr,
		},
		{
			name: "core.GetUser returns error",
			setup: func(mocks mock) {
				mocks.authService.CheckSelfOrAdminMock.Expect(ctx, userID).Return(nil)
				mocks.core.GetUserMock.Expect(ctx, userID).Return(user, "", expErr)
			},
			err: expErr,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			mocks := getMocks(t)
			if tt.setup != nil {
				tt.setup(mocks)
			}

			svc := usecase.NewService(mocks.core, mocks.authService, mocks.passwordHasher)
			resp, err := svc.GetUser(ctx, userID)
			if tt.err != nil {
				require.ErrorIs(t, err, tt.err)
			} else {
				require.NoError(t, err)
				require.Equal(t, user, resp)
			}
		})
	}
}

func TestService_GetAllUsers(t *testing.T) {
	t.Parallel()

	var (
		ctx    = t.Context()
		expErr = errors.New("permission denied")
		users  = []user.User{
			{
				ID:             uuid.New(),
				Email:          "user@mail.com",
				Name:           "name",
				SessionVersion: 1,
				CreatedAt:      time.Now(),
				UpdatedAt:      time.Now(),
			},
			{
				ID:             uuid.New(),
				Email:          "user2@mail.com",
				Name:           "name2",
				SessionVersion: 1,
				CreatedAt:      time.Now(),
				UpdatedAt:      time.Now(),
			},
		}
	)

	tests := []struct {
		name  string
		setup func(mocks mock)
		err   error
	}{
		{
			name: "ok",
			setup: func(mocks mock) {
				mocks.authService.CheckIsAdminMock.Expect(ctx).Return(nil)
				mocks.core.GetAllUsersMock.Expect(ctx).Return(users, nil)
			},
		},
		{
			name: "authService.CheckIsAdmin returns error",
			setup: func(mocks mock) {
				mocks.authService.CheckIsAdminMock.Expect(ctx).Return(expErr)
			},
			err: expErr,
		},
		{
			name: "core.GetAllUsers returns error",
			setup: func(mocks mock) {
				mocks.authService.CheckIsAdminMock.Expect(ctx).Return(nil)
				mocks.core.GetAllUsersMock.Expect(ctx).Return(nil, expErr)
			},
			err: expErr,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			mocks := getMocks(t)
			if tt.setup != nil {
				tt.setup(mocks)
			}

			svc := usecase.NewService(mocks.core, mocks.authService, mocks.passwordHasher)
			resp, err := svc.GetAllUsers(ctx)
			if tt.err != nil {
				require.ErrorIs(t, err, tt.err)
			} else {
				require.NoError(t, err)
				require.Equal(t, users, resp)
			}
		})
	}
}

func TestService_UpdateUser(t *testing.T) {
	t.Parallel()

	var (
		req = user.UpdateUserReq{
			UserID: uuid.New(),
			Email:  "new_mail",
			Name:   "new_name",
		}
		ctx    = t.Context()
		expErr = errors.New("user not found")
	)

	tests := []struct {
		name  string
		setup func(mocks mock)
		err   error
	}{
		{
			name: "ok",
			setup: func(mocks mock) {
				mocks.authService.CheckSelfOrAdminMock.Expect(ctx, req.UserID).Return(nil)
				mocks.core.UpdateUserMock.Expect(ctx, req).Return(nil)
			},
		},
		{
			name: "authService.CheckSelfOrAdmin returns error",
			setup: func(mocks mock) {
				mocks.authService.CheckSelfOrAdminMock.Expect(ctx, req.UserID).Return(expErr)
			},
			err: expErr,
		},
		{
			name: "core.UpdateUser returns error",
			setup: func(mocks mock) {
				mocks.authService.CheckSelfOrAdminMock.Expect(ctx, req.UserID).Return(nil)
				mocks.core.UpdateUserMock.Expect(ctx, req).Return(expErr)
			},
			err: expErr,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			mocks := getMocks(t)
			if tt.setup != nil {
				tt.setup(mocks)
			}

			svc := usecase.NewService(mocks.core, mocks.authService, mocks.passwordHasher)
			err := svc.UpdateUser(ctx, req)
			if tt.err != nil {
				require.ErrorIs(t, err, tt.err)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestService_DeleteUser(t *testing.T) {
	t.Parallel()

	var (
		userID = uuid.New()
		ctx    = t.Context()
		expErr = errors.New("user not found")
	)

	tests := []struct {
		name  string
		setup func(mocks mock)
		err   error
	}{
		{
			name: "ok",
			setup: func(mocks mock) {
				mocks.authService.CheckIsAdminMock.Expect(ctx).Return(nil)
				mocks.core.DeleteUserMock.Expect(ctx, userID).Return(nil)
				mocks.authService.DeleteSessionsByUserIDMock.Expect(context.WithoutCancel(ctx), userID).Return(nil)
			},
		},
		{
			name: "authService.DeleteSessionsByUserID returns error",
			setup: func(mocks mock) {
				mocks.authService.CheckIsAdminMock.Expect(ctx).Return(nil)
				mocks.core.DeleteUserMock.Expect(ctx, userID).Return(nil)
				mocks.authService.DeleteSessionsByUserIDMock.Expect(context.WithoutCancel(ctx), userID).Return(expErr)
			},
		},
		{
			name: "authService.CheckSelfOrAdmin returns error",
			setup: func(mocks mock) {
				mocks.authService.CheckIsAdminMock.Expect(ctx).Return(expErr)
			},
			err: expErr,
		},
		{
			name: "core.DeleteUser returns error",
			setup: func(mocks mock) {
				mocks.authService.CheckIsAdminMock.Expect(ctx).Return(nil)
				mocks.core.DeleteUserMock.Expect(ctx, userID).Return(expErr)
			},
			err: expErr,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			mocks := getMocks(t)
			if tt.setup != nil {
				tt.setup(mocks)
			}

			svc := usecase.NewService(mocks.core, mocks.authService, mocks.passwordHasher)
			err := svc.DeleteUser(ctx, userID)
			if tt.err != nil {
				require.ErrorIs(t, err, tt.err)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestService_ChangePassword(t *testing.T) {
	t.Parallel()

	var (
		userID = uuid.New()
		req    = usecase.ChangePasswordCmd{
			ID:          userID,
			OldPassword: []byte("old"),
			NewPassword: []byte("new"),
		}
		ctx    = t.Context()
		hash   = "$2a$10$N9qo8uLOickgx2ZMRZoMyeIjZag0u.JxQ7a0h1yfof3f6K/3pG4b."
		expErr = errors.New("user not found")
	)

	tests := []struct {
		name  string
		req   usecase.ChangePasswordCmd
		setup func(mocks mock)
		err   error
	}{
		{
			name: "ok/admin",
			req:  req,
			setup: func(mocks mock) {
				mocks.authService.IsAdminMock.Expect(ctx).Return(true, nil)
				mocks.core.GetUserMock.Expect(ctx, userID).Return(user.User{}, hash, nil)
				mocks.passwordHasher.CheckPasswordHashMock.Expect([]byte(hash), req.NewPassword).Return(secure.ErrMismatchedHashAndPassword)
				mocks.core.ChangePasswordMock.Expect(ctx, userID, req.NewPassword).Return(nil)
				mocks.authService.DeleteSessionsByUserIDMock.Expect(context.WithoutCancel(ctx), userID).Return(nil)
			},
		},
		{
			name: "ok/self",
			req:  req,
			setup: func(mocks mock) {
				mocks.authService.IsAdminMock.Expect(ctx).Return(false, nil)
				mocks.authService.CheckSelfMock.Expect(ctx, userID).Return(nil)
				mocks.passwordHasher.CheckPasswordHashMock.When([]byte(hash), req.OldPassword).Then(nil)
				mocks.core.GetUserMock.Expect(ctx, userID).Return(user.User{}, hash, nil)
				mocks.passwordHasher.CheckPasswordHashMock.When([]byte(hash), req.NewPassword).Then(secure.ErrMismatchedHashAndPassword)
				mocks.core.ChangePasswordMock.Expect(ctx, userID, req.NewPassword).Return(nil)
				mocks.authService.DeleteSessionsByUserIDMock.Expect(context.WithoutCancel(ctx), userID).Return(nil)
			},
		},
		{
			name: "authService.DeleteSessionsByUserID returns error",
			req:  req,
			setup: func(mocks mock) {
				mocks.authService.IsAdminMock.Expect(ctx).Return(true, nil)
				mocks.core.GetUserMock.Expect(ctx, userID).Return(user.User{}, hash, nil)
				mocks.passwordHasher.CheckPasswordHashMock.Expect([]byte(hash), req.NewPassword).Return(secure.ErrMismatchedHashAndPassword)
				mocks.core.ChangePasswordMock.Expect(ctx, userID, req.NewPassword).Return(nil)
				mocks.authService.DeleteSessionsByUserIDMock.Expect(context.WithoutCancel(ctx), userID).Return(expErr)
			},
		},
		{
			name: "core.ChangePasswordMock returns error",
			req:  req,
			setup: func(mocks mock) {
				mocks.authService.IsAdminMock.Expect(ctx).Return(true, nil)
				mocks.core.GetUserMock.Expect(ctx, userID).Return(user.User{}, hash, nil)
				mocks.passwordHasher.CheckPasswordHashMock.Expect([]byte(hash), req.NewPassword).Return(secure.ErrMismatchedHashAndPassword)
				mocks.core.ChangePasswordMock.Expect(ctx, userID, req.NewPassword).Return(expErr)
			},
			err: expErr,
		},
		{
			name: "old password = new password error",
			req:  req,
			setup: func(mocks mock) {
				mocks.authService.IsAdminMock.Expect(ctx).Return(true, nil)
				mocks.core.GetUserMock.Expect(ctx, userID).Return(user.User{}, hash, nil)
				mocks.passwordHasher.CheckPasswordHashMock.Expect([]byte(hash), req.NewPassword).Return(nil)
			},
			err: usecase.ErrNewPasswordSameAsOld,
		},
		{
			name: "new password passwordHasher.CheckPasswordHashMock error",
			req:  req,
			setup: func(mocks mock) {
				mocks.authService.IsAdminMock.Expect(ctx).Return(true, nil)
				mocks.core.GetUserMock.Expect(ctx, userID).Return(user.User{}, hash, nil)
				mocks.passwordHasher.CheckPasswordHashMock.Expect([]byte(hash), req.NewPassword).Return(expErr)
			},
			err: expErr,
		},
		{
			name: "core.GetUser returns error",
			req:  req,
			setup: func(mocks mock) {
				mocks.authService.IsAdminMock.Expect(ctx).Return(true, nil)
				mocks.core.GetUserMock.Expect(ctx, userID).Return(user.User{}, "", expErr)
			},
			err: expErr,
		},
		{
			name: "authService.IsAdmin returns error",
			req:  req,
			setup: func(mocks mock) {
				mocks.authService.IsAdminMock.Expect(ctx).Return(false, expErr)
			},
			err: expErr,
		},
		{
			name: "old password incorrect",
			req:  req,
			setup: func(mocks mock) {
				mocks.authService.IsAdminMock.Expect(ctx).Return(false, nil)
				mocks.authService.CheckSelfMock.Expect(ctx, userID).Return(nil)
				mocks.core.GetUserMock.Expect(ctx, userID).Return(user.User{}, hash, nil)
				mocks.passwordHasher.CheckPasswordHashMock.Expect([]byte(hash), req.OldPassword).Return(secure.ErrMismatchedHashAndPassword)
			},
			err: usecase.ErrOldPasswordIncorrect,
		},
		{
			name: "authService.CheckSelf returns error",
			req:  req,
			setup: func(mocks mock) {
				mocks.authService.IsAdminMock.Expect(ctx).Return(false, nil)
				mocks.authService.CheckSelfMock.Expect(ctx, userID).Return(expErr)
			},
			err: expErr,
		},
		{
			name: "old password not provided",
			req:  usecase.ChangePasswordCmd{OldPassword: []byte("")},
			setup: func(mocks mock) {
				mocks.authService.IsAdminMock.Expect(ctx).Return(false, nil)
			},
			err: usecase.ErrOldPasswordRequired,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			mocks := getMocks(t)
			if tt.setup != nil {
				tt.setup(mocks)
			}

			svc := usecase.NewService(mocks.core, mocks.authService, mocks.passwordHasher)
			err := svc.ChangePassword(ctx, tt.req)
			if tt.err != nil {
				require.ErrorIs(t, err, tt.err)
			} else {
				require.NoError(t, err)
			}
		})
	}
}
