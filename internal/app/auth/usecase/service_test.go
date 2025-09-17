package usecase_test

import (
	"fmt"
	"testing"
	"time"

	"github.com/66gu1/easygodocs/internal/app/auth"
	"github.com/66gu1/easygodocs/internal/app/auth/usecase"
	"github.com/66gu1/easygodocs/internal/app/auth/usecase/mocks"
	"github.com/66gu1/easygodocs/internal/app/user"
	"github.com/66gu1/easygodocs/internal/infrastructure/apperr"
	"github.com/66gu1/easygodocs/internal/infrastructure/secure"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

//go:generate minimock -o ./mocks -s _mock.go

type mock struct {
	core           *mocks.CoreMock
	userCore       *mocks.UserCoreMock
	passwordHasher *mocks.PasswordHasherMock
}

func newMock(t *testing.T) *mock {
	t.Helper()
	return &mock{
		core:           mocks.NewCoreMock(t),
		userCore:       mocks.NewUserCoreMock(t),
		passwordHasher: mocks.NewPasswordHasherMock(t),
	}
}

func TestService_GetSessionsByUserID(t *testing.T) {
	t.Parallel()
	var (
		ctx      = t.Context()
		userID   = uuid.New()
		sessions = []auth.Session{
			{
				ID:             uuid.New(),
				UserID:         userID,
				CreatedAt:      time.Now(),
				ExpiresAt:      time.Now(),
				SessionVersion: 1,
			},
			{
				ID:             uuid.New(),
				UserID:         userID,
				CreatedAt:      time.Now(),
				ExpiresAt:      time.Now(),
				SessionVersion: 1,
			},
		}
		errExp = fmt.Errorf("expired")
	)
	tests := []struct {
		name  string
		setup func(m mock)
		err   error
	}{
		{
			name: "ok",
			setup: func(m mock) {
				m.core.CheckSelfOrAdminMock.Expect(ctx, userID).Return(nil)
				m.core.GetSessionsByUserIDMock.Expect(ctx, userID).Return(sessions, nil)
			},
		},
		{
			name: "error - core.CheckSelfOrAdmin",
			setup: func(m mock) {
				m.core.CheckSelfOrAdminMock.Expect(ctx, userID).Return(errExp)
			},
			err: errExp,
		},
		{
			name: "error - core.GetSessionsByUserID",
			setup: func(m mock) {
				m.core.CheckSelfOrAdminMock.Expect(ctx, userID).Return(nil)
				m.core.GetSessionsByUserIDMock.Expect(ctx, userID).Return(nil, errExp)
			},
			err: errExp,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			m := newMock(t)
			if tt.setup != nil {
				tt.setup(*m)
			}
			s := usecase.NewService(m.core, m.userCore, m.passwordHasher)
			got, err := s.GetSessionsByUserID(ctx, userID)
			if tt.err != nil {
				require.ErrorIs(t, err, tt.err)
			} else {
				require.NoError(t, err)
				require.Equal(t, sessions, got)
			}
		})
	}
}

func TestService_DeleteSession(t *testing.T) {
	t.Parallel()
	var (
		ctx    = t.Context()
		userID = uuid.New()
		id     = uuid.New()
		errExp = fmt.Errorf("expired")
	)
	tests := []struct {
		name  string
		setup func(m mock)
		err   error
	}{
		{
			name: "ok",
			setup: func(m mock) {
				m.core.CheckSelfOrAdminMock.Expect(ctx, userID).Return(nil)
				m.core.DeleteSessionMock.Expect(ctx, id, userID).Return(nil)
			},
		},
		{
			name: "error - core.CheckSelfOrAdmin",
			setup: func(m mock) {
				m.core.CheckSelfOrAdminMock.Expect(ctx, userID).Return(errExp)
			},
			err: errExp,
		},
		{
			name: "error - core.DeleteSession",
			setup: func(m mock) {
				m.core.CheckSelfOrAdminMock.Expect(ctx, userID).Return(nil)
				m.core.DeleteSessionMock.Expect(ctx, id, userID).Return(errExp)
			},
			err: errExp,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			m := newMock(t)
			if tt.setup != nil {
				tt.setup(*m)
			}
			s := usecase.NewService(m.core, m.userCore, m.passwordHasher)
			err := s.DeleteSession(ctx, userID, id)
			if tt.err != nil {
				require.ErrorIs(t, err, tt.err)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestService_DeleteSessionsByUserID(t *testing.T) {
	t.Parallel()
	var (
		ctx    = t.Context()
		userID = uuid.New()
		errExp = fmt.Errorf("expired")
	)
	tests := []struct {
		name  string
		setup func(m mock)
		err   error
	}{
		{
			name: "ok",
			setup: func(m mock) {
				m.core.CheckSelfOrAdminMock.Expect(ctx, userID).Return(nil)
				m.core.DeleteSessionsByUserIDMock.Expect(ctx, userID).Return(nil)
			},
		},
		{
			name: "error - core.CheckSelfOrAdmin",
			setup: func(m mock) {
				m.core.CheckSelfOrAdminMock.Expect(ctx, userID).Return(errExp)
			},
			err: errExp,
		},
		{
			name: "error - core.DeleteSessionsByUserID",
			setup: func(m mock) {
				m.core.CheckSelfOrAdminMock.Expect(ctx, userID).Return(nil)
				m.core.DeleteSessionsByUserIDMock.Expect(ctx, userID).Return(errExp)
			},
			err: errExp,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			m := newMock(t)
			if tt.setup != nil {
				tt.setup(*m)
			}
			s := usecase.NewService(m.core, m.userCore, m.passwordHasher)
			err := s.DeleteSessionsByUserID(ctx, userID)
			if tt.err != nil {
				require.ErrorIs(t, err, tt.err)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestService_AddUserRole(t *testing.T) {
	t.Parallel()
	var (
		ctx      = t.Context()
		userRole = auth.UserRole{
			UserID: uuid.New(),
			Role:   auth.RoleAdmin,
		}
		errExp = fmt.Errorf("expired")
	)
	tests := []struct {
		name  string
		setup func(m mock)
		err   error
	}{
		{
			name: "ok",
			setup: func(m mock) {
				m.core.CheckIsAdminMock.Expect(ctx).Return(nil)
				m.core.AddUserRoleMock.Expect(ctx, userRole).Return(nil)
			},
		},
		{
			name: "error - userCore.AddRole",
			setup: func(m mock) {
				m.core.CheckIsAdminMock.Expect(ctx).Return(nil)
				m.core.AddUserRoleMock.Expect(ctx, userRole).Return(errExp)
			},
			err: errExp,
		},
		{
			name: "error - core.CheckIsAdmin",
			setup: func(m mock) {
				m.core.CheckIsAdminMock.Expect(ctx).Return(errExp)
			},
			err: errExp,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			m := newMock(t)
			if tt.setup != nil {
				tt.setup(*m)
			}
			s := usecase.NewService(m.core, m.userCore, m.passwordHasher)
			err := s.AddUserRole(ctx, userRole)
			if tt.err != nil {
				require.ErrorIs(t, err, tt.err)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestService_DeleteUserRole(t *testing.T) {
	t.Parallel()
	var (
		ctx      = t.Context()
		userRole = auth.UserRole{
			UserID: uuid.New(),
			Role:   auth.RoleAdmin,
		}
		errExp = fmt.Errorf("expired")
	)
	tests := []struct {
		name  string
		setup func(m mock)
		err   error
	}{
		{
			name: "ok",
			setup: func(m mock) {
				m.core.CheckIsAdminMock.Expect(ctx).Return(nil)
				m.core.DeleteUserRoleMock.Expect(ctx, userRole).Return(nil)
			},
		},
		{
			name: "error - core.DeleteUserRole",
			setup: func(m mock) {
				m.core.CheckIsAdminMock.Expect(ctx).Return(nil)
				m.core.DeleteUserRoleMock.Expect(ctx, userRole).Return(errExp)
			},
			err: errExp,
		},
		{
			name: "error - core.CheckIsAdmin",
			setup: func(m mock) {
				m.core.CheckIsAdminMock.Expect(ctx).Return(errExp)
			},
			err: errExp,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			m := newMock(t)
			if tt.setup != nil {
				tt.setup(*m)
			}
			s := usecase.NewService(m.core, m.userCore, m.passwordHasher)
			err := s.DeleteUserRole(ctx, userRole)
			if tt.err != nil {
				require.ErrorIs(t, err, tt.err)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestService_ListUserRoles(t *testing.T) {
	t.Parallel()
	var (
		ctx    = t.Context()
		userID = uuid.New()
		roles  = []auth.UserRole{
			{
				UserID: userID,
				Role:   auth.RoleAdmin,
			},
			{
				UserID: userID,
				Role:   auth.RoleWrite,
			},
		}
		errExp = fmt.Errorf("expired")
	)
	tests := []struct {
		name  string
		setup func(m mock)
		err   error
	}{
		{
			name: "ok",
			setup: func(m mock) {
				m.core.CheckSelfOrAdminMock.Expect(ctx, userID).Return(nil)
				m.core.ListUserRolesMock.Expect(ctx, userID).Return(roles, nil)
			},
		},
		{
			name: "error - core.ListUserRoles",
			setup: func(m mock) {
				m.core.CheckSelfOrAdminMock.Expect(ctx, userID).Return(nil)
				m.core.ListUserRolesMock.Expect(ctx, userID).Return(nil, errExp)
			},
			err: errExp,
		},
		{
			name: "error - core.CheckIsAdmin",
			setup: func(m mock) {
				m.core.CheckSelfOrAdminMock.Expect(ctx, userID).Return(errExp)
			},
			err: errExp,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			m := newMock(t)
			if tt.setup != nil {
				tt.setup(*m)
			}
			s := usecase.NewService(m.core, m.userCore, m.passwordHasher)
			got, err := s.ListUserRoles(ctx, userID)
			if tt.err != nil {
				require.ErrorIs(t, err, tt.err)
			} else {
				require.NoError(t, err)
				require.Equal(t, roles, got)
			}
		})
	}
}

func TestService_RefreshTokens(t *testing.T) {
	t.Parallel()
	var (
		ctx            = t.Context()
		sessionID      = uuid.New()
		userID         = uuid.New()
		sessionVersion = 1
		refreshToken   = auth.RefreshToken{
			SessionID: sessionID,
			Token:     "refresh_token",
		}
		rtHash  = "hashed_rt"
		session = auth.Session{
			ID:             sessionID,
			UserID:         userID,
			CreatedAt:      time.Now(),
			ExpiresAt:      time.Now().Add(24 * time.Hour),
			SessionVersion: sessionVersion,
		}
		usr = user.User{
			SessionVersion: sessionVersion,
		}
		tokensExp = auth.Tokens{
			AccessToken: "access_token",
			RefreshToken: auth.RefreshToken{
				SessionID: sessionID,
				Token:     "new_refresh_token",
			},
		}
		errExp = fmt.Errorf("expired")
	)
	tests := []struct {
		name  string
		req   auth.RefreshToken
		setup func(m mock)
		err   error
	}{
		{
			name: "ok",
			req:  refreshToken,
			setup: func(m mock) {
				m.core.GetSessionByIDMock.Expect(ctx, sessionID).Return(session, rtHash, nil)
				m.userCore.GetUserMock.Expect(ctx, userID).Return(usr, "", nil)
				m.core.RefreshTokensMock.Expect(ctx, session, refreshToken.Token, rtHash).Return(tokensExp, nil)
			},
		},
		{
			name: "error - core.RefreshTokens",
			req:  refreshToken,
			setup: func(m mock) {
				m.core.GetSessionByIDMock.Expect(ctx, sessionID).Return(session, rtHash, nil)
				m.userCore.GetUserMock.Expect(ctx, userID).Return(usr, "", nil)
				m.core.RefreshTokensMock.Expect(ctx, session, refreshToken.Token, rtHash).Return(auth.Tokens{}, errExp)
			},
			err: errExp,
		},
		{
			name: "session version mismatch",
			req:  refreshToken,
			setup: func(m mock) {
				m.core.GetSessionByIDMock.Expect(ctx, sessionID).Return(session, rtHash, nil)
				m.userCore.GetUserMock.Expect(ctx, userID).Return(user.User{SessionVersion: 2}, "", nil)
			},
			err: apperr.ErrUnauthorized(),
		},
		{
			name: "error - userCore.GetUser",
			req:  refreshToken,
			setup: func(m mock) {
				m.core.GetSessionByIDMock.Expect(ctx, sessionID).Return(session, rtHash, nil)
				m.userCore.GetUserMock.Expect(ctx, userID).Return(user.User{}, "", errExp)
			},
			err: errExp,
		},
		{
			name: "error - core.GetSessionByID",
			req:  refreshToken,
			setup: func(m mock) {
				m.core.GetSessionByIDMock.Expect(ctx, sessionID).Return(auth.Session{}, "", errExp)
			},
			err: errExp,
		},
		{
			name: "error - empty refresh token",
			req:  auth.RefreshToken{SessionID: sessionID, Token: ""},
			err:  apperr.ErrBadRequest(),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			m := newMock(t)
			if tt.setup != nil {
				tt.setup(*m)
			}
			s := usecase.NewService(m.core, m.userCore, m.passwordHasher)
			got, err := s.RefreshTokens(ctx, tt.req)
			if tt.err != nil {
				require.ErrorIs(t, err, tt.err)
			} else {
				require.NoError(t, err)
				require.Equal(t, tokensExp, got)
			}
		})
	}
}

func TestService_Login(t *testing.T) {
	t.Parallel()
	var (
		ctx            = t.Context()
		email          = "mail"
		password       = "password"
		hashedPassword = "hashed_password"
		userID         = uuid.New()
		sessionID      = uuid.New()
		sessionVersion = 1
		usr            = user.User{
			ID:             userID,
			Email:          email,
			SessionVersion: sessionVersion,
		}
		tokensExp = auth.Tokens{
			AccessToken: "access_token",
			RefreshToken: auth.RefreshToken{
				SessionID: sessionID,
				Token:     "refresh_token",
			},
		}
		errExp = fmt.Errorf("expired")
	)
	tests := []struct {
		name  string
		setup func(m mock)
		err   error
	}{
		{
			name: "ok",
			setup: func(m mock) {
				m.userCore.GetUserByEmailMock.Expect(ctx, email).Return(usr, hashedPassword, nil)
				m.passwordHasher.CheckPasswordHashMock.Expect([]byte(hashedPassword), []byte(password)).Return(nil)
				m.core.IssueTokensMock.Expect(ctx, userID, sessionVersion).Return(tokensExp, nil)
			},
		},
		{
			name: "error - core.IssueTokens",
			setup: func(m mock) {
				m.userCore.GetUserByEmailMock.Expect(ctx, email).Return(usr, hashedPassword, nil)
				m.passwordHasher.CheckPasswordHashMock.Expect([]byte(hashedPassword), []byte(password)).Return(nil)
				m.core.IssueTokensMock.Expect(ctx, userID, sessionVersion).Return(auth.Tokens{}, errExp)
			},
			err: errExp,
		},
		{
			name: "error - passwordHasher.CheckPasswordHash",
			setup: func(m mock) {
				m.userCore.GetUserByEmailMock.Expect(ctx, email).Return(usr, hashedPassword, nil)
				m.passwordHasher.CheckPasswordHashMock.Expect([]byte(hashedPassword), []byte(password)).Return(errExp)
			},
			err: errExp,
		},
		{
			name: "wrong password",
			setup: func(m mock) {
				m.userCore.GetUserByEmailMock.Expect(ctx, email).Return(usr, hashedPassword, nil)
				m.passwordHasher.CheckPasswordHashMock.Expect([]byte(hashedPassword), []byte(password)).Return(secure.ErrMismatchedHashAndPassword)
			},
			err: usecase.ErrInvalidPasswordOrEmail(),
		},
		{
			name: "error - userCore.GetUserByEmail",
			setup: func(m mock) {
				m.userCore.GetUserByEmailMock.Expect(ctx, email).Return(user.User{}, "", errExp)
			},
			err: errExp,
		},
		{
			name: "error - userCore.GetUserByEmail user not found",
			setup: func(m mock) {
				m.userCore.GetUserByEmailMock.Expect(ctx, email).Return(user.User{}, "", user.ErrUserNotFound())
			},
			err: usecase.ErrInvalidPasswordOrEmail(),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			m := newMock(t)
			if tt.setup != nil {
				tt.setup(*m)
			}
			s := usecase.NewService(m.core, m.userCore, m.passwordHasher)
			got, err := s.Login(ctx, usecase.LoginCmd{
				Email:    email,
				Password: []byte(password),
			})
			if tt.err != nil {
				require.ErrorIs(t, err, tt.err)
			} else {
				require.NoError(t, err)
				require.Equal(t, tokensExp, got)
			}
		})
	}
}
