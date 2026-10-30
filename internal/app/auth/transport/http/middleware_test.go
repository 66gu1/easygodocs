package http

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/66gu1/easygodocs/internal/app/auth"
	"github.com/66gu1/easygodocs/internal/app/auth/transport/http/mocks"
	"github.com/66gu1/easygodocs/internal/infrastructure/apperr"
	"github.com/66gu1/easygodocs/internal/infrastructure/contextx"
	"github.com/go-chi/chi/v5"
	"github.com/gojuno/minimock/v3"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

func TestAuthMiddleware(t *testing.T) {
	t.Parallel()

	userID := uuid.New()
	SID := uuid.New()
	token := "token"
	issuedAt := time.UnixMilli(time.Now().UnixMilli())
	validClaims := func(mock *mocks.TokenCodecMock) {
		mock.ParseTokenMock.Set(func(tokenStr string, claims jwt.Claims) error {
			c, ok := claims.(*auth.AccessTokenClaims)
			if !ok {
				return fmt.Errorf("unexpected claims type %T", claims)
			}
			c.Subject = userID.String()
			c.SID = SID.String()
			c.IssuedAtMs = issuedAt.UnixMilli()
			c.ExpiresAt = jwt.NewNumericDate(time.Now().Add(5 * time.Minute))
			return nil
		})
	}
	tests := []struct {
		name          string
		header        string
		setup         func(mock *mocks.TokenCodecMock)
		revokedSetup  func(mock *mocks.RevocationCheckerMock)
		sessionsSetup func(mock *mocks.SessionValidatorMock)
		wantStatus    int
	}{
		{
			name:       "missing Authorization -> 401",
			header:     "",
			wantStatus: http.StatusUnauthorized,
		},
		{
			name:       "malformed Authorization (no Bearer) -> 401",
			header:     "Token abc",
			wantStatus: http.StatusUnauthorized,
		},
		{
			name:   "ParseToken returns unauthorized -> 401",
			header: "Bearer token",
			setup: func(mock *mocks.TokenCodecMock) {
				mock.ParseTokenMock.Expect(token, &auth.AccessTokenClaims{}).Return(fmt.Errorf("badtoken"))
			},
			wantStatus: http.StatusUnauthorized,
		},
		{
			name:   "invalid Subject (not uuid) -> 401",
			header: "Bearer token",
			setup: func(mock *mocks.TokenCodecMock) {
				mock.ParseTokenMock.Set(func(tokenStr string, claims jwt.Claims) error {
					c, ok := claims.(*auth.AccessTokenClaims)
					if !ok {
						return fmt.Errorf("unexpected claims type %T", claims)
					}
					c.Subject = "not-uuid"
					c.SID = SID.String()
					return nil
				})
			},
			wantStatus: http.StatusUnauthorized,
		},
		{
			name:   "nil Subject (000..0) -> 401",
			header: "Bearer token",
			setup: func(mock *mocks.TokenCodecMock) {
				mock.ParseTokenMock.Set(func(tokenStr string, claims jwt.Claims) error {
					c, ok := claims.(*auth.AccessTokenClaims)
					if !ok {
						return fmt.Errorf("unexpected claims type %T", claims)
					}
					c.Subject = uuid.Nil.String()
					c.SID = SID.String()
					return nil
				})
			},
			wantStatus: http.StatusUnauthorized,
		},
		{
			name:   "invalid SID (not uuid) -> 401",
			header: "Bearer token",
			setup: func(mock *mocks.TokenCodecMock) {
				mock.ParseTokenMock.Set(func(tokenStr string, claims jwt.Claims) error {
					c, ok := claims.(*auth.AccessTokenClaims)
					if !ok {
						return fmt.Errorf("unexpected claims type %T", claims)
					}
					c.Subject = userID.String()
					c.SID = "not-uuid"
					return nil
				})
			},
			wantStatus: http.StatusUnauthorized,
		},
		{
			name:   "nil SID",
			header: "Bearer token",
			setup: func(mock *mocks.TokenCodecMock) {
				mock.ParseTokenMock.Set(func(tokenStr string, claims jwt.Claims) error {
					c, ok := claims.(*auth.AccessTokenClaims)
					if !ok {
						return fmt.Errorf("unexpected claims type %T", claims)
					}
					c.Subject = userID.String()
					c.SID = uuid.Nil.String()
					return nil
				})
			},
			wantStatus: http.StatusUnauthorized,
		},
		{
			name:   "expired token -> 401",
			header: "Bearer token",
			setup: func(mock *mocks.TokenCodecMock) {
				mock.ParseTokenMock.Set(func(tokenStr string, claims jwt.Claims) error {
					c, ok := claims.(*auth.AccessTokenClaims)
					if !ok {
						return fmt.Errorf("unexpected claims type %T", claims)
					}
					c.Subject = userID.String()
					c.SID = SID.String()
					c.ExpiresAt = jwt.NewNumericDate(time.Now().Add(-5 * time.Minute))
					return nil
				})
			},
			wantStatus: http.StatusUnauthorized,
		},
		{
			name:   "missing exp -> 401",
			header: "Bearer token",
			setup: func(mock *mocks.TokenCodecMock) {
				mock.ParseTokenMock.Set(func(tokenStr string, claims jwt.Claims) error {
					c, ok := claims.(*auth.AccessTokenClaims)
					if !ok {
						return fmt.Errorf("unexpected claims type %T", claims)
					}
					c.Subject = userID.String()
					c.SID = SID.String()
					return nil
				})
			},
			wantStatus: http.StatusUnauthorized,
		},
		{
			name:   "token revoked -> 401 without a database query",
			header: "Bearer token",
			setup:  validClaims,
			revokedSetup: func(mock *mocks.RevocationCheckerMock) {
				mock.IsRevokedMock.Expect(minimock.AnyContext, userID, SID, issuedAt).Return(true, nil)
			},
			wantStatus: http.StatusUnauthorized,
		},
		{
			name:   "ok -> next called without a database query, context has user_id & session_id",
			header: "Bearer token",
			setup:  validClaims,
			revokedSetup: func(mock *mocks.RevocationCheckerMock) {
				mock.IsRevokedMock.Expect(minimock.AnyContext, userID, SID, issuedAt).Return(false, nil)
			},
			wantStatus: http.StatusOK,
		},
		{
			name:   "revocations unavailable, session active in the database -> 200",
			header: "Bearer token",
			setup:  validClaims,
			revokedSetup: func(mock *mocks.RevocationCheckerMock) {
				mock.IsRevokedMock.Return(false, fmt.Errorf("redis is down"))
			},
			sessionsSetup: func(mock *mocks.SessionValidatorMock) {
				mock.ValidateSessionMock.ExpectUserIDParam2(userID).ExpectSessionIDParam3(SID).Return(nil)
			},
			wantStatus: http.StatusOK,
		},
		{
			name:   "revocations unavailable, session deleted in the database -> 401",
			header: "Bearer token",
			setup:  validClaims,
			revokedSetup: func(mock *mocks.RevocationCheckerMock) {
				mock.IsRevokedMock.Return(false, fmt.Errorf("redis is down"))
			},
			sessionsSetup: func(mock *mocks.SessionValidatorMock) {
				mock.ValidateSessionMock.ExpectUserIDParam2(userID).ExpectSessionIDParam3(SID).
					Return(apperr.New("session not found", auth.CodeSessionNotFound, apperr.ClassNotFound, apperr.LogLevelWarn))
			},
			wantStatus: http.StatusUnauthorized,
		},
		{
			name:   "revocations and database unavailable -> 500",
			header: "Bearer token",
			setup:  validClaims,
			revokedSetup: func(mock *mocks.RevocationCheckerMock) {
				mock.IsRevokedMock.Return(false, fmt.Errorf("redis is down"))
			},
			sessionsSetup: func(mock *mocks.SessionValidatorMock) {
				mock.ValidateSessionMock.ExpectUserIDParam2(userID).ExpectSessionIDParam3(SID).
					Return(fmt.Errorf("db is down"))
			},
			wantStatus: http.StatusInternalServerError,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				got, err := contextx.GetUserID(r.Context())
				require.NoError(t, err)
				require.Equal(t, userID, got)
				got, err = contextx.GetSessionID(r.Context())
				require.NoError(t, err)
				require.Equal(t, SID, got)
				w.WriteHeader(http.StatusOK)
			})

			mock := mocks.NewTokenCodecMock(t)
			if tc.setup != nil {
				tc.setup(mock)
			}
			revocations := mocks.NewRevocationCheckerMock(t)
			if tc.revokedSetup != nil {
				tc.revokedSetup(revocations)
			}
			sessions := mocks.NewSessionValidatorMock(t)
			if tc.sessionsSetup != nil {
				tc.sessionsSetup(sessions)
			}
			r := chi.NewRouter()
			r.Use(AuthMiddleware(mock, revocations, sessions))
			r.Get("/protected", next)

			req := httptest.NewRequest(http.MethodGet, "/protected", nil)
			if tc.header != "" {
				req.Header.Set("Authorization", tc.header)
			}
			rr := httptest.NewRecorder()

			r.ServeHTTP(rr, req)

			require.Equal(t, tc.wantStatus, rr.Code)

			if tc.wantStatus != http.StatusOK {
				require.NotEmpty(t, strings.TrimSpace(rr.Body.String()))
			}
		})
	}
}
