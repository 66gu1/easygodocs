package auth_test

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/66gu1/easygodocs/internal/app/auth"
	"github.com/66gu1/easygodocs/internal/app/auth/mocks"
	"github.com/66gu1/easygodocs/internal/infrastructure/apperr"
	"github.com/66gu1/easygodocs/internal/infrastructure/contextx"
	"github.com/66gu1/easygodocs/internal/infrastructure/secure"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

//go:generate minimock -o ./mocks -s _mock.go

type mock struct {
	repo       *mocks.RepositoryMock
	idGen      *mocks.UUIDGeneratorMock
	timeGen    *mocks.TimeGeneratorMock
	rndGen     *mocks.RNDGeneratorMock
	pswHasher  *mocks.PasswordHasherMock
	tokenCodec *mocks.TokenCodecMock
}

func setupMocks(t *testing.T) mock {
	return mock{
		repo:       mocks.NewRepositoryMock(t),
		idGen:      mocks.NewUUIDGeneratorMock(t),
		timeGen:    mocks.NewTimeGeneratorMock(t),
		rndGen:     mocks.NewRNDGeneratorMock(t),
		pswHasher:  mocks.NewPasswordHasherMock(t),
		tokenCodec: mocks.NewTokenCodecMock(t),
	}
}

func cfg() auth.Config {
	return auth.Config{
		SessionTTLMinutes:     1,
		AccessTokenTTLMinutes: 2,
	}
}

func TestCore_IssueTokens(t *testing.T) {
	t.Parallel()

	var (
		ctx            = context.Background()
		userID         = uuid.New()
		sessID         = uuid.New()
		now            = time.Now()
		sessionVersion = 1
		accessToken    = "access.token.value"
		refreshToken   = "refresh.token.value"
		rtHash         = []byte("refresh.token.hashed")
		claims         = auth.AccessTokenClaims{
			SID: sessID.String(),
			RegisteredClaims: jwt.RegisteredClaims{
				Subject:   userID.String(),
				IssuedAt:  jwt.NewNumericDate(now),
				ExpiresAt: jwt.NewNumericDate(now.Add(time.Duration(cfg().AccessTokenTTLMinutes) * time.Minute)),
			},
		}
		session = auth.Session{
			ID:             sessID,
			UserID:         userID,
			CreatedAt:      now,
			ExpiresAt:      now.Add(time.Duration(cfg().SessionTTLMinutes) * time.Minute),
			SessionVersion: sessionVersion,
		}
		errExp = fmt.Errorf("expected")
		want   = auth.Tokens{
			AccessToken: accessToken,
			RefreshToken: auth.RefreshToken{
				SessionID: sessID,
				Token:     refreshToken,
			},
		}
	)

	tests := []struct {
		name    string
		userID  uuid.UUID
		setup   func(mocks mock)
		wantErr bool
		err     error
	}{
		{
			name:   "ok",
			userID: userID,
			setup: func(mocks mock) {
				mocks.idGen.NewMock.Return(sessID, nil)
				mocks.timeGen.NowMock.Return(now)
				mocks.rndGen.NewMock.Expect(32).Return(refreshToken, nil)
				mocks.pswHasher.HashRefreshTokenMock.Expect([]byte(refreshToken)).Return(rtHash, nil)
				mocks.tokenCodec.GenerateTokenMock.Expect(claims).Return(accessToken, nil)
				mocks.repo.CreateSessionMock.Expect(ctx, session, string(rtHash)).Return(nil)
			},
		},
		{
			name:    "nil user id",
			userID:  uuid.Nil,
			setup:   func(mocks mock) {},
			wantErr: true,
		},
		{
			name:   "id gen error",
			userID: userID,
			setup: func(mocks mock) {
				mocks.idGen.NewMock.Return(uuid.Nil, errExp)
			},
			err: errExp,
		},
		{
			name:   "rnd gen error",
			userID: userID,
			setup: func(mocks mock) {
				mocks.idGen.NewMock.Return(sessID, nil)
				mocks.timeGen.NowMock.Return(now)
				mocks.rndGen.NewMock.Expect(32).Return("", errExp)
			},
			err: errExp,
		},
		{
			name:   "psw hasher error",
			userID: userID,
			setup: func(mocks mock) {
				mocks.idGen.NewMock.Return(sessID, nil)
				mocks.timeGen.NowMock.Return(now)
				mocks.rndGen.NewMock.Expect(32).Return(refreshToken, nil)
				mocks.pswHasher.HashRefreshTokenMock.Expect([]byte(refreshToken)).Return(nil, errExp)
			},
			err: errExp,
		},
		{
			name:   "token codec error",
			userID: userID,
			setup: func(mocks mock) {
				mocks.idGen.NewMock.Return(sessID, nil)
				mocks.timeGen.NowMock.Return(now)
				mocks.rndGen.NewMock.Expect(32).Return(refreshToken, nil)
				mocks.pswHasher.HashRefreshTokenMock.Expect([]byte(refreshToken)).Return(rtHash, nil)
				mocks.tokenCodec.GenerateTokenMock.Expect(claims).Return("", errExp)
			},
			err: errExp,
		},
		{
			name:   "repo error",
			userID: userID,
			setup: func(mocks mock) {
				mocks.idGen.NewMock.Return(sessID, nil)
				mocks.timeGen.NowMock.Return(now)
				mocks.rndGen.NewMock.Expect(32).Return(refreshToken, nil)
				mocks.pswHasher.HashRefreshTokenMock.Expect([]byte(refreshToken)).Return(rtHash, nil)
				mocks.tokenCodec.GenerateTokenMock.Expect(claims).Return(accessToken, nil)
				mocks.repo.CreateSessionMock.Expect(ctx, session, string(rtHash)).Return(errExp)
			},
			err: errExp,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			mocks := setupMocks(t)
			tt.setup(mocks)

			core, err := auth.NewCore(
				mocks.repo,
				mocks.tokenCodec,
				mocks.idGen,
				mocks.rndGen,
				mocks.timeGen,
				mocks.pswHasher,
				cfg(),
			)

			tokens, err := core.IssueTokens(ctx, tt.userID, sessionVersion)
			if tt.err != nil || tt.wantErr {
				require.Error(t, err)
				if tt.err != nil {
					require.ErrorIs(t, err, tt.err)
				}
				return
			}
			require.NoError(t, err)
			require.Equal(t, want, tokens)
		})
	}
}

func TestCore_RefreshTokens(t *testing.T) {
	t.Parallel()

	var (
		ctx             = context.Background()
		userID          = uuid.New()
		sessID          = uuid.New()
		now             = time.Now()
		sessionVersion  = 1
		accessToken     = "access.token.value"
		refreshToken    = "refresh.token.value"
		newRefreshToken = "new.refresh.token.value"
		rtHash          = "refresh.token.hashed"
		newRTHash       = "new.refresh.token.hashed"
		claims          = auth.AccessTokenClaims{
			SID: sessID.String(),
			RegisteredClaims: jwt.RegisteredClaims{
				Subject:   userID.String(),
				IssuedAt:  jwt.NewNumericDate(now),
				ExpiresAt: jwt.NewNumericDate(now.Add(time.Duration(cfg().AccessTokenTTLMinutes) * time.Minute)),
			},
		}
		session = auth.Session{
			ID:             sessID,
			UserID:         userID,
			CreatedAt:      now.Add(-time.Minute),
			ExpiresAt:      now.Add(time.Duration(cfg().SessionTTLMinutes) * time.Minute),
			SessionVersion: sessionVersion,
		}
		updateTokenReq = auth.UpdateTokenReq{
			SessionID:           sessID,
			UserID:              userID,
			RefreshTokenHash:    newRTHash,
			OldRefreshTokenHash: rtHash,
			ExpiresAt:           now.Add(time.Duration(cfg().SessionTTLMinutes) * time.Minute),
		}
		errExp = fmt.Errorf("expected")
		want   = auth.Tokens{
			AccessToken: accessToken,
			RefreshToken: auth.RefreshToken{
				SessionID: sessID,
				Token:     newRefreshToken,
			},
		}
	)

	tests := []struct {
		name    string
		session auth.Session
		setup   func(mocks mock)
		err     error
	}{
		{
			name:    "ok",
			session: session,
			setup: func(mocks mock) {
				mocks.timeGen.NowMock.Return(now)
				mocks.pswHasher.CheckPasswordHashMock.Expect([]byte(refreshToken), rtHash).Return(nil)
				mocks.rndGen.NewMock.Expect(32).Return(newRefreshToken, nil)
				mocks.pswHasher.HashRefreshTokenMock.Expect([]byte(newRefreshToken)).Return([]byte(newRTHash), nil)
				mocks.tokenCodec.GenerateTokenMock.Expect(claims).Return(accessToken, nil)
				mocks.repo.UpdateRefreshTokenMock.Expect(ctx, updateTokenReq).Return(nil)
			},
		},
		{
			name:    "session expired",
			session: auth.Session{ExpiresAt: now.Add(-time.Minute)},
			setup: func(mocks mock) {
				mocks.timeGen.NowMock.Return(now)
			},
			err: apperr.ErrUnauthorized(),
		},
		{
			name:    "invalid refresh token",
			session: session,
			setup: func(mocks mock) {
				mocks.timeGen.NowMock.Return(now)
				mocks.pswHasher.CheckPasswordHashMock.Expect([]byte(refreshToken), rtHash).Return(secure.ErrMismatchedHashAndPassword)
			},
			err: apperr.ErrUnauthorized(),
		},
		{
			name:    "check hash error",
			session: session,
			setup: func(mocks mock) {
				mocks.timeGen.NowMock.Return(now)
				mocks.pswHasher.CheckPasswordHashMock.Expect([]byte(refreshToken), rtHash).Return(errExp)
			},
			err: errExp,
		},
		{
			name:    "rnd gen error",
			session: session,
			setup: func(mocks mock) {
				mocks.timeGen.NowMock.Return(now)
				mocks.pswHasher.CheckPasswordHashMock.Expect([]byte(refreshToken), rtHash).Return(nil)
				mocks.rndGen.NewMock.Expect(32).Return("", errExp)
			},
			err: errExp,
		},
		{
			name:    "psw hasher error",
			session: session,
			setup: func(mocks mock) {
				mocks.timeGen.NowMock.Return(now)
				mocks.pswHasher.CheckPasswordHashMock.Expect([]byte(refreshToken), rtHash).Return(nil)
				mocks.rndGen.NewMock.Expect(32).Return(newRefreshToken, nil)
				mocks.pswHasher.HashRefreshTokenMock.Expect([]byte(newRefreshToken)).Return(nil, errExp)
			},
			err: errExp,
		},
		{
			name:    "token codec error",
			session: session,
			setup: func(mocks mock) {
				mocks.timeGen.NowMock.Return(now)
				mocks.pswHasher.CheckPasswordHashMock.Expect([]byte(refreshToken), rtHash).Return(nil)
				mocks.rndGen.NewMock.Expect(32).Return(newRefreshToken, nil)
				mocks.pswHasher.HashRefreshTokenMock.Expect([]byte(newRefreshToken)).Return([]byte(newRTHash), nil)
				mocks.tokenCodec.GenerateTokenMock.Expect(claims).Return("", errExp)
			},
			err: errExp,
		},
		{
			name:    "repo error",
			session: session,
			setup: func(mocks mock) {
				mocks.timeGen.NowMock.Return(now)
				mocks.pswHasher.CheckPasswordHashMock.Expect([]byte(refreshToken), rtHash).Return(nil)
				mocks.rndGen.NewMock.Expect(32).Return(newRefreshToken, nil)
				mocks.pswHasher.HashRefreshTokenMock.Expect([]byte(newRefreshToken)).Return([]byte(newRTHash), nil)
				mocks.tokenCodec.GenerateTokenMock.Expect(claims).Return(accessToken, nil)
				mocks.repo.UpdateRefreshTokenMock.Expect(ctx, updateTokenReq).Return(errExp)
			},
			err: errExp,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			mocks := setupMocks(t)
			if tt.setup != nil {
				tt.setup(mocks)
			}

			core, err := auth.NewCore(
				mocks.repo,
				mocks.tokenCodec,
				mocks.idGen,
				mocks.rndGen,
				mocks.timeGen,
				mocks.pswHasher,
				cfg(),
			)

			tokens, err := core.RefreshTokens(ctx, tt.session, refreshToken, rtHash)
			if tt.err != nil {
				require.ErrorIs(t, err, tt.err)
				return
			}
			require.NoError(t, err)
			require.Equal(t, want, tokens)
		})
	}
}

func TestCore_GetSessionByID(t *testing.T) {
	t.Parallel()
	var (
		ctx     = context.Background()
		sessID  = uuid.New()
		session = auth.Session{
			ID:             sessID,
			UserID:         uuid.New(),
			CreatedAt:      time.Now(),
			ExpiresAt:      time.Now().Add(time.Hour),
			SessionVersion: 1,
		}
		rtHash = "refresh.token.hashed"
		errExp = fmt.Errorf("expected")
	)
	tests := []struct {
		name      string
		sessionID uuid.UUID
		setup     func(mocks mock)
		err       error
	}{
		{
			name:      "ok",
			sessionID: sessID,
			setup: func(mocks mock) {
				mocks.repo.GetSessionByIDMock.Expect(ctx, sessID).Return(session, rtHash, nil)
			},
		},
		{
			name:      "nil session id",
			sessionID: uuid.Nil,
			err:       apperr.ErrNilUUID(""),
		},
		{
			name:      "repo error",
			sessionID: sessID,
			setup: func(mocks mock) {
				mocks.repo.GetSessionByIDMock.Expect(ctx, sessID).Return(auth.Session{}, rtHash, errExp)
			},
			err: errExp,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			mocks := setupMocks(t)
			if tt.setup != nil {
				tt.setup(mocks)
			}

			core, err := auth.NewCore(
				mocks.repo,
				mocks.tokenCodec,
				mocks.idGen,
				mocks.rndGen,
				mocks.timeGen,
				mocks.pswHasher,
				cfg(),
			)
			require.NoError(t, err)

			gotSession, gotRTHash, err := core.GetSessionByID(ctx, tt.sessionID)
			if tt.err != nil {
				require.ErrorIs(t, err, tt.err)
				return
			}
			require.NoError(t, err)
			require.Equal(t, session, gotSession)
			require.Equal(t, rtHash, gotRTHash)
		})
	}
}

func TestCore_GetSessionsByUserID(t *testing.T) {
	t.Parallel()
	var (
		ctx      = context.Background()
		userID   = uuid.New()
		sessions = []auth.Session{
			{ID: uuid.New(), UserID: userID, CreatedAt: time.Now(), ExpiresAt: time.Now().Add(time.Hour), SessionVersion: 1},
			{ID: uuid.New(), UserID: userID, CreatedAt: time.Now(), ExpiresAt: time.Now().Add(time.Hour), SessionVersion: 1},
		}
		errExp = fmt.Errorf("expected")
	)
	tests := []struct {
		name  string
		setup func(mocks mock)
		err   error
	}{
		{
			name: "ok",
			setup: func(mocks mock) {
				mocks.repo.GetSessionsByUserIDMock.Expect(ctx, userID).Return(sessions, nil)
			},
		},
		{
			name: "repo error",
			setup: func(mocks mock) {
				mocks.repo.GetSessionsByUserIDMock.Expect(ctx, userID).Return(nil, errExp)
			},
			err: errExp,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			mocks := setupMocks(t)
			if tt.setup != nil {
				tt.setup(mocks)
			}

			core, err := auth.NewCore(
				mocks.repo,
				mocks.tokenCodec,
				mocks.idGen,
				mocks.rndGen,
				mocks.timeGen,
				mocks.pswHasher,
				cfg(),
			)
			require.NoError(t, err)

			gotSessions, err := core.GetSessionsByUserID(ctx, userID)
			if tt.err != nil {
				require.ErrorIs(t, err, tt.err)
				return
			}
			require.NoError(t, err)
			require.Equal(t, sessions, gotSessions)
		})
	}
}

func TestCore_DeleteSession(t *testing.T) {
	t.Parallel()
	var (
		ctx    = context.Background()
		sessID = uuid.New()
		userID = uuid.New()
		errExp = fmt.Errorf("expected")
	)
	tests := []struct {
		name    string
		isAdmin bool
		setup   func(mocks mock)
		err     error
	}{
		{
			name:    "ok/admin",
			isAdmin: true,
			setup: func(mocks mock) {
				mocks.repo.DeleteSessionByIDMock.Expect(ctx, sessID).Return(nil)
			},
		},
		{
			name:    "ok/user",
			isAdmin: false,
			setup: func(mocks mock) {
				mocks.repo.DeleteSessionByIDAndUserMock.Expect(ctx, sessID, userID).Return(nil)
			},
		},
		{
			name:    "err/repo/admin",
			isAdmin: true,
			setup: func(mocks mock) {
				mocks.repo.DeleteSessionByIDMock.Expect(ctx, sessID).Return(errExp)
			},
			err: errExp,
		},
		{
			name:    "err/repo/user",
			isAdmin: false,
			setup: func(mocks mock) {
				mocks.repo.DeleteSessionByIDAndUserMock.Expect(ctx, sessID, userID).Return(errExp)
			},
			err: errExp,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			mocks := setupMocks(t)
			if tt.setup != nil {
				tt.setup(mocks)
			}

			core, err := auth.NewCore(
				mocks.repo,
				mocks.tokenCodec,
				mocks.idGen,
				mocks.rndGen,
				mocks.timeGen,
				mocks.pswHasher,
				cfg(),
			)
			require.NoError(t, err)

			err = core.DeleteSession(ctx, sessID, userID, tt.isAdmin)
			if tt.err != nil {
				require.ErrorIs(t, err, tt.err)
				return
			}
			require.NoError(t, err)
		})
	}
}

func TestCore_DeleteSessionsByUserID(t *testing.T) {
	t.Parallel()
	var (
		ctx    = context.Background()
		userID = uuid.New()
		errExp = fmt.Errorf("expected")
	)
	tests := []struct {
		name  string
		setup func(mocks mock)
		err   error
	}{
		{
			name: "ok",
			setup: func(mocks mock) {
				mocks.repo.DeleteSessionsByUserIDMock.Expect(ctx, userID).Return(nil)
			},
		},
		{
			name: "err/repo",
			setup: func(mocks mock) {
				mocks.repo.DeleteSessionsByUserIDMock.Expect(ctx, userID).Return(errExp)
			},
			err: errExp,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			mocks := setupMocks(t)
			if tt.setup != nil {
				tt.setup(mocks)
			}

			core, err := auth.NewCore(
				mocks.repo,
				mocks.tokenCodec,
				mocks.idGen,
				mocks.rndGen,
				mocks.timeGen,
				mocks.pswHasher,
				cfg(),
			)
			require.NoError(t, err)

			err = core.DeleteSessionsByUserID(ctx, userID)
			if tt.err != nil {
				require.ErrorIs(t, err, tt.err)
				return
			}
			require.NoError(t, err)
		})
	}
}

func TestCore_AddUserRole(t *testing.T) {
	t.Parallel()
	var (
		ctx      = context.Background()
		entityID = uuid.New()
		userRole = auth.UserRole{
			UserID:   uuid.New(),
			Role:     auth.RoleRead,
			EntityID: &entityID,
		}
		errExp = fmt.Errorf("expected")
	)
	tests := []struct {
		name     string
		userRole auth.UserRole
		setup    func(mocks mock)
		err      error
	}{
		{
			name:     "ok",
			userRole: userRole,
			setup: func(mocks mock) {
				mocks.repo.AddUserRoleMock.Expect(ctx, userRole).Return(nil)
			},
		},
		{
			name: "nil user id",
			userRole: auth.UserRole{
				UserID:   uuid.Nil,
				Role:     auth.RoleRead,
				EntityID: &entityID,
			},
			err: apperr.ErrNilUUID("user id"),
		},
		{
			name: "invalid role",
			userRole: auth.UserRole{
				UserID:   uuid.New(),
				Role:     "invalid",
				EntityID: &entityID,
			},
			err: auth.ErrInvalidRole,
		},
		{
			name: "missing entity for role that requires it",
			userRole: auth.UserRole{
				UserID:   uuid.New(),
				Role:     auth.RoleRead,
				EntityID: nil,
			},
			err: auth.ErrRoleRequiresEntity(),
		},
		{
			name: "entity provided for role that forbids it",
			userRole: auth.UserRole{
				UserID:   uuid.New(),
				Role:     auth.RoleAdmin,
				EntityID: &entityID,
			},
			err: auth.ErrRoleForbidsEntity(),
		},
		{
			name:     "repo error",
			userRole: userRole,
			setup: func(mocks mock) {
				mocks.repo.AddUserRoleMock.Expect(ctx, userRole).Return(errExp)
			},
			err: errExp,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			mocks := setupMocks(t)
			if tt.setup != nil {
				tt.setup(mocks)
			}

			core, err := auth.NewCore(
				mocks.repo,
				mocks.tokenCodec,
				mocks.idGen,
				mocks.rndGen,
				mocks.timeGen,
				mocks.pswHasher,
				cfg(),
			)
			require.NoError(t, err)

			err = core.AddUserRole(ctx, tt.userRole)
			if tt.err != nil {
				require.ErrorIs(t, err, tt.err)
				return
			}
			require.NoError(t, err)
		})
	}
}

func TestCore_DeleteUserRole(t *testing.T) {
	t.Parallel()
	var (
		ctx      = context.Background()
		entityID = uuid.New()
		userRole = auth.UserRole{
			UserID:   uuid.New(),
			Role:     auth.RoleRead,
			EntityID: &entityID,
		}
		errExp = fmt.Errorf("expected")
	)
	tests := []struct {
		name  string
		setup func(mocks mock)
		err   error
	}{
		{
			name: "ok",
			setup: func(mocks mock) {
				mocks.repo.DeleteUserRoleMock.Expect(ctx, userRole).Return(nil)
			},
		},
		{
			name: "repo error",
			setup: func(mocks mock) {
				mocks.repo.DeleteUserRoleMock.Expect(ctx, userRole).Return(errExp)
			},
			err: errExp,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			mocks := setupMocks(t)
			if tt.setup != nil {
				tt.setup(mocks)
			}
			core, err := auth.NewCore(
				mocks.repo,
				mocks.tokenCodec,
				mocks.idGen,
				mocks.rndGen,
				mocks.timeGen,
				mocks.pswHasher,
				cfg(),
			)
			require.NoError(t, err)
			err = core.DeleteUserRole(ctx, userRole)
			if tt.err != nil {
				require.ErrorIs(t, err, tt.err)
				return
			}
			require.NoError(t, err)
		})
	}
}

func TestCore_ListUserRoles(t *testing.T) {
	t.Parallel()
	var (
		ctx       = context.Background()
		userID    = uuid.New()
		entityID  = uuid.New()
		errExp    = fmt.Errorf("expected")
		userRoles = []auth.UserRole{
			{UserID: userID, Role: auth.RoleAdmin, EntityID: nil},
			{UserID: userID, Role: auth.RoleRead, EntityID: &entityID},
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
				mocks.repo.ListUserRolesMock.Expect(ctx, userID).Return(userRoles, nil)
			},
		},
		{
			name: "repo error",
			setup: func(mocks mock) {
				mocks.repo.ListUserRolesMock.Expect(ctx, userID).Return(nil, errExp)
			},
			err: errExp,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			mocks := setupMocks(t)
			if tt.setup != nil {
				tt.setup(mocks)
			}
			core, err := auth.NewCore(
				mocks.repo,
				mocks.tokenCodec,
				mocks.idGen,
				mocks.rndGen,
				mocks.timeGen,
				mocks.pswHasher,
				cfg(),
			)
			require.NoError(t, err)
			gotUserRoles, err := core.ListUserRoles(ctx, userID)
			if tt.err != nil {
				require.ErrorIs(t, err, tt.err)
				return
			}
			require.NoError(t, err)
			require.Equal(t, userRoles, gotUserRoles)
		})
	}
}

func TestCore_GetCurrentUserDirectPermissions(t *testing.T) {
	t.Parallel()
	var (
		ctx        = context.Background()
		userID     = uuid.New()
		entityID1  = uuid.New()
		entityID2  = uuid.New()
		adminRoles = []auth.UserRole{
			{UserID: userID, Role: auth.RoleAdmin, EntityID: nil},
		}
		roles = []auth.UserRole{
			{UserID: userID, Role: auth.RoleRead, EntityID: &entityID1},
			{UserID: userID, Role: auth.RoleWrite, EntityID: &entityID2},
		}
		ids    = []uuid.UUID{entityID1, entityID2}
		errExp = fmt.Errorf("expected")
	)
	ctx = contextx.SetToContext(ctx, contextx.ContextKeyUserID, userID)
	tests := []struct {
		name    string
		ctx     context.Context
		role    auth.Role
		isAdmin bool
		setup   func(mocks mock)
		wantErr bool
		err     error
	}{
		{
			name:    "ok/admin",
			ctx:     ctx,
			role:    auth.RoleRead,
			isAdmin: true,
			setup: func(mocks mock) {
				mocks.repo.GetUserRolesMock.Expect(ctx, userID, auth.RoleRead.GetHierarchy()).Return(adminRoles, nil)
			},
		},
		{
			name:    "ok/user",
			ctx:     ctx,
			role:    auth.RoleRead,
			isAdmin: false,
			setup: func(mocks mock) {
				mocks.repo.GetUserRolesMock.Expect(ctx, userID, auth.RoleRead.GetHierarchy()).Return(roles, nil)
			},
		},
		{
			name:    "no user in context",
			ctx:     context.Background(),
			role:    auth.RoleRead,
			wantErr: true,
		},
		{
			name: "invalid role",
			ctx:  ctx,
			role: "invalid",
			err:  auth.ErrInvalidRole,
		},
		{
			name: "repo error",
			ctx:  ctx,
			role: auth.RoleRead,
			setup: func(mocks mock) {
				mocks.repo.GetUserRolesMock.Expect(ctx, userID, auth.RoleRead.GetHierarchy()).Return(nil, errExp)
			},
			err: errExp,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			mocks := setupMocks(t)
			if tt.setup != nil {
				tt.setup(mocks)
			}
			core, err := auth.NewCore(
				mocks.repo,
				mocks.tokenCodec,
				mocks.idGen,
				mocks.rndGen,
				mocks.timeGen,
				mocks.pswHasher,
				cfg(),
			)
			require.NoError(t, err)
			gotRoles, gotAdmin, err := core.GetCurrentUserDirectPermissions(tt.ctx, tt.role)
			if tt.err != nil || tt.wantErr {
				require.Error(t, err)
				if tt.err != nil {
					require.ErrorIs(t, err, tt.err)
				}
				return
			}
			require.NoError(t, err)
			if tt.isAdmin {
				require.True(t, gotAdmin)
			} else {
				require.False(t, gotAdmin)
				require.Equal(t, ids, gotRoles)
			}
		})
	}
}

func TestCore_IsSelf(t *testing.T) {
	t.Parallel()
	var (
		userID = uuid.New()
	)
	ctx := contextx.SetToContext(context.Background(), contextx.ContextKeyUserID, userID)
	tests := []struct {
		name    string
		ctx     context.Context
		userID  uuid.UUID
		isSelf  bool
		wantErr bool
	}{
		{
			name:   "is self",
			ctx:    ctx,
			userID: userID,
			isSelf: true,
		},
		{
			name:   "is not self",
			ctx:    ctx,
			userID: uuid.New(),
			isSelf: false,
		},
		{
			name:    "no user in context",
			ctx:     context.Background(),
			userID:  userID,
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			mocks := setupMocks(t)

			core, err := auth.NewCore(
				mocks.repo,
				mocks.tokenCodec,
				mocks.idGen,
				mocks.rndGen,
				mocks.timeGen,
				mocks.pswHasher,
				cfg(),
			)
			require.NoError(t, err)

			isSelf, err := core.IsSelf(tt.ctx, tt.userID)
			if tt.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			require.Equal(t, tt.isSelf, isSelf)
		})
	}
}
