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
	"github.com/66gu1/easygodocs/internal/infrastructure/ratelimit"
	"github.com/66gu1/easygodocs/internal/infrastructure/secure"
	"github.com/gojuno/minimock/v3"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

//go:generate minimock -o ./mocks -s _mock.go

var dummyHash = []byte("dummy_hash")

type mock struct {
	core           *mocks.CoreMock
	userCore       *mocks.UserCoreMock
	passwordHasher *mocks.PasswordHasherMock
	loginAttempts  *mocks.LoginAttemptsMock
}

func newMock(t *testing.T) *mock {
	t.Helper()
	return &mock{
		core:           mocks.NewCoreMock(t),
		userCore:       mocks.NewUserCoreMock(t),
		passwordHasher: mocks.NewPasswordHasherMock(t),
		loginAttempts:  mocks.NewLoginAttemptsMock(t),
	}
}

func (m *mock) newService() *usecase.Service {
	return usecase.NewService(m.core, m.userCore, m.passwordHasher, m.loginAttempts, dummyHash)
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
			s := m.newService()
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
			s := m.newService()
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
			s := m.newService()
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
			s := m.newService()
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
			s := m.newService()
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
			s := m.newService()
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
			s := m.newService()
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
		ip             = "198.51.100.1"
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
		errExp  = fmt.Errorf("expired")
		allowed = ratelimit.AttemptResult{Allowed: true}
	)
	tests := []struct {
		name           string
		setup          func(m mock)
		err            error
		wantRetryAfter time.Duration
	}{
		{
			name: "ok: the attempt is released",
			setup: func(m mock) {
				m.loginAttempts.ReserveMock.Expect(ctx, email, ip).Return(allowed, nil)
				m.userCore.GetUserByEmailMock.Expect(ctx, email).Return(usr, hashedPassword, nil)
				m.passwordHasher.CheckPasswordHashMock.Expect([]byte(hashedPassword), []byte(password)).Return(nil)
				m.core.IssueTokensMock.Expect(ctx, userID, sessionVersion).Return(tokensExp, nil)
				m.loginAttempts.ReleaseMock.Expect(minimock.AnyContext, email, ip).Return(nil)
			},
		},
		{
			name: "ok: a release error doesn't fail the login",
			setup: func(m mock) {
				m.loginAttempts.ReserveMock.Expect(ctx, email, ip).Return(allowed, nil)
				m.userCore.GetUserByEmailMock.Expect(ctx, email).Return(usr, hashedPassword, nil)
				m.passwordHasher.CheckPasswordHashMock.Expect([]byte(hashedPassword), []byte(password)).Return(nil)
				m.core.IssueTokensMock.Expect(ctx, userID, sessionVersion).Return(tokensExp, nil)
				m.loginAttempts.ReleaseMock.Expect(minimock.AnyContext, email, ip).Return(errExp)
			},
		},
		{
			name: "ok: limiter unavailable, fails open and nothing is released",
			setup: func(m mock) {
				m.loginAttempts.ReserveMock.Expect(ctx, email, ip).Return(ratelimit.AttemptResult{}, errExp)
				m.userCore.GetUserByEmailMock.Expect(ctx, email).Return(usr, hashedPassword, nil)
				m.passwordHasher.CheckPasswordHashMock.Expect([]byte(hashedPassword), []byte(password)).Return(nil)
				m.core.IssueTokensMock.Expect(ctx, userID, sessionVersion).Return(tokensExp, nil)
			},
		},
		{
			name: "limiter unavailable and the login fails: nothing is refunded",
			setup: func(m mock) {
				m.loginAttempts.ReserveMock.Expect(ctx, email, ip).Return(ratelimit.AttemptResult{}, errExp)
				m.userCore.GetUserByEmailMock.Expect(ctx, email).Return(user.User{}, "", errExp)
			},
			err: errExp,
		},
		{
			name: "too many attempts: the user isn't looked up and the password isn't checked",
			setup: func(m mock) {
				m.loginAttempts.ReserveMock.Expect(ctx, email, ip).
					Return(ratelimit.AttemptResult{RetryAfter: 90 * time.Second}, nil)
			},
			err:            usecase.ErrTooManyLoginAttempts(),
			wantRetryAfter: 90 * time.Second,
		},
		{
			name: "error - core.IssueTokens: the attempt is refunded",
			setup: func(m mock) {
				m.loginAttempts.ReserveMock.Expect(ctx, email, ip).Return(allowed, nil)
				m.userCore.GetUserByEmailMock.Expect(ctx, email).Return(usr, hashedPassword, nil)
				m.passwordHasher.CheckPasswordHashMock.Expect([]byte(hashedPassword), []byte(password)).Return(nil)
				m.core.IssueTokensMock.Expect(ctx, userID, sessionVersion).Return(auth.Tokens{}, errExp)
				m.loginAttempts.RefundMock.Expect(minimock.AnyContext, email, ip).Return(nil)
			},
			err: errExp,
		},
		{
			name: "error - passwordHasher.CheckPasswordHash: the attempt is refunded",
			setup: func(m mock) {
				m.loginAttempts.ReserveMock.Expect(ctx, email, ip).Return(allowed, nil)
				m.userCore.GetUserByEmailMock.Expect(ctx, email).Return(usr, hashedPassword, nil)
				m.passwordHasher.CheckPasswordHashMock.Expect([]byte(hashedPassword), []byte(password)).Return(errExp)
				m.loginAttempts.RefundMock.Expect(minimock.AnyContext, email, ip).Return(nil)
			},
			err: errExp,
		},
		{
			name: "wrong password: the attempt is spent",
			setup: func(m mock) {
				m.loginAttempts.ReserveMock.Expect(ctx, email, ip).Return(allowed, nil)
				m.userCore.GetUserByEmailMock.Expect(ctx, email).Return(usr, hashedPassword, nil)
				m.passwordHasher.CheckPasswordHashMock.Expect([]byte(hashedPassword), []byte(password)).Return(secure.ErrMismatchedHashAndPassword)
			},
			err: usecase.ErrInvalidPasswordOrEmail(),
		},
		{
			name: "error - userCore.GetUserByEmail: the attempt is refunded",
			setup: func(m mock) {
				m.loginAttempts.ReserveMock.Expect(ctx, email, ip).Return(allowed, nil)
				m.userCore.GetUserByEmailMock.Expect(ctx, email).Return(user.User{}, "", errExp)
				m.loginAttempts.RefundMock.Expect(minimock.AnyContext, email, ip).Return(nil)
			},
			err: errExp,
		},
		{
			name: "error - a refund error doesn't change the result",
			setup: func(m mock) {
				m.loginAttempts.ReserveMock.Expect(ctx, email, ip).Return(allowed, nil)
				m.userCore.GetUserByEmailMock.Expect(ctx, email).Return(user.User{}, "", errExp)
				m.loginAttempts.RefundMock.Expect(minimock.AnyContext, email, ip).Return(fmt.Errorf("redis is down"))
			},
			err: errExp,
		},
		{
			name: "error - userCore.GetUserByEmail user not found: the attempt is spent",
			setup: func(m mock) {
				// the attempt is counted for unknown emails too, so the limit doesn't reveal which emails exist
				m.loginAttempts.ReserveMock.Expect(ctx, email, ip).Return(allowed, nil)
				m.userCore.GetUserByEmailMock.Expect(ctx, email).Return(user.User{}, "", user.ErrUserNotFound())
				// the dummy hash is still checked to keep response time constant
				m.passwordHasher.CheckPasswordHashMock.Expect(dummyHash, []byte(password)).Return(secure.ErrMismatchedHashAndPassword)
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
			got, err := m.newService().Login(ctx, usecase.LoginCmd{
				Email:    email,
				Password: []byte(password),
				ClientIP: ip,
			})
			if tt.err != nil {
				require.ErrorIs(t, err, tt.err)
				require.Equal(t, tt.wantRetryAfter, apperr.FromError(err).RetryAfter())
			} else {
				require.NoError(t, err)
				require.Equal(t, tokensExp, got)
			}
		})
	}
}

func TestService_ValidateSession(t *testing.T) {
	t.Parallel()
	var (
		ctx            = t.Context()
		sessionID      = uuid.New()
		userID         = uuid.New()
		sessionVersion = 1
		session        = auth.Session{
			ID:             sessionID,
			UserID:         userID,
			ExpiresAt:      time.Now().Add(time.Hour),
			SessionVersion: sessionVersion,
		}
		usr    = user.User{ID: userID, SessionVersion: sessionVersion}
		errExp = fmt.Errorf("expected")
	)
	tests := []struct {
		name  string
		setup func(m mock)
		err   error
	}{
		{
			name: "ok",
			setup: func(m mock) {
				m.core.GetSessionByIDMock.Expect(ctx, sessionID).Return(session, "hash", nil)
				m.core.CheckSessionActiveMock.Expect(session).Return(nil)
				m.userCore.GetUserMock.Expect(ctx, userID).Return(usr, "", nil)
			},
		},
		{
			name: "error - session not found",
			setup: func(m mock) {
				m.core.GetSessionByIDMock.Expect(ctx, sessionID).Return(auth.Session{}, "", errExp)
			},
			err: errExp,
		},
		{
			name: "session belongs to another user",
			setup: func(m mock) {
				other := session
				other.UserID = uuid.New()
				m.core.GetSessionByIDMock.Expect(ctx, sessionID).Return(other, "hash", nil)
			},
			err: apperr.ErrUnauthorized(),
		},
		{
			name: "session expired",
			setup: func(m mock) {
				m.core.GetSessionByIDMock.Expect(ctx, sessionID).Return(session, "hash", nil)
				m.core.CheckSessionActiveMock.Expect(session).Return(apperr.ErrUnauthorized())
			},
			err: apperr.ErrUnauthorized(),
		},
		{
			name: "user deleted",
			setup: func(m mock) {
				m.core.GetSessionByIDMock.Expect(ctx, sessionID).Return(session, "hash", nil)
				m.core.CheckSessionActiveMock.Expect(session).Return(nil)
				m.userCore.GetUserMock.Expect(ctx, userID).Return(user.User{}, "", user.ErrUserNotFound())
			},
			err: user.ErrUserNotFound(),
		},
		{
			name: "session version mismatch (password changed)",
			setup: func(m mock) {
				m.core.GetSessionByIDMock.Expect(ctx, sessionID).Return(session, "hash", nil)
				m.core.CheckSessionActiveMock.Expect(session).Return(nil)
				m.userCore.GetUserMock.Expect(ctx, userID).Return(user.User{ID: userID, SessionVersion: 2}, "", nil)
			},
			err: apperr.ErrUnauthorized(),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			m := newMock(t)
			tt.setup(*m)
			s := m.newService()
			err := s.ValidateSession(ctx, userID, sessionID)
			if tt.err != nil {
				require.ErrorIs(t, err, tt.err)
				return
			}
			require.NoError(t, err)
		})
	}
}
