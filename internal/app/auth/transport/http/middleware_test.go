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
	"github.com/66gu1/easygodocs/internal/infrastructure/contextx"
	"github.com/go-chi/chi/v5"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

func TestAuthMiddleware(t *testing.T) {
	t.Parallel()

	userID := uuid.New()
	SID := uuid.New()
	token := "token"
	tests := []struct {
		name       string
		header     string
		setup      func(mock *mocks.TokenCodecMock)
		wantStatus int
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
			name:   "ok -> next called, context has user_id & session_id",
			header: "Bearer token",
			setup: func(mock *mocks.TokenCodecMock) {
				mock.ParseTokenMock.Set(func(tokenStr string, claims jwt.Claims) error {
					c, ok := claims.(*auth.AccessTokenClaims)
					if !ok {
						return fmt.Errorf("unexpected claims type %T", claims)
					}
					c.Subject = userID.String()
					c.SID = SID.String()
					c.ExpiresAt = jwt.NewNumericDate(time.Now().Add(5 * time.Minute))
					return nil
				})
			},
			wantStatus: http.StatusOK,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			// next–handler-спай: проверяем, что дошли и что в контексте лежат правильные значения
			var gotUserID, gotSID uuid.UUID

			next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				// достаём значения напрямую по ключам, которые использует middleware
				if v := r.Context().Value(contextx.UserIDKey); v != nil {
					if id, ok := v.(uuid.UUID); ok {
						gotUserID = id
					}
				}
				if v := r.Context().Value(contextx.SessionIDKey); v != nil {
					if id, ok := v.(uuid.UUID); ok {
						gotSID = id
					}
				}
				w.WriteHeader(http.StatusOK)
			})

			mock := mocks.NewTokenCodecMock(t)
			if tc.setup != nil {
				tc.setup(mock)
			}
			r := chi.NewRouter()
			r.Use(AuthMiddleware(mock))
			r.Get("/protected", next)

			req := httptest.NewRequest(http.MethodGet, "/protected", nil)
			if tc.header != "" {
				req.Header.Set("Authorization", tc.header)
			}
			rr := httptest.NewRecorder()

			r.ServeHTTP(rr, req)

			require.Equal(t, tc.wantStatus, rr.Code)

			if tc.wantStatus == http.StatusOK {
				require.Equal(t, userID, gotUserID, "userID in context mismatch")
				require.Equal(t, SID, gotSID, "sessionID in context mismatch")
			} else {
				require.NotEmpty(t, strings.TrimSpace(rr.Body.String()))
			}
		})
	}
}
