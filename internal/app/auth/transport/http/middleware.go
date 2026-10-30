package http

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/66gu1/easygodocs/internal/app/auth"
	"github.com/66gu1/easygodocs/internal/infrastructure/apperr"
	"github.com/66gu1/easygodocs/internal/infrastructure/contextx"
	"github.com/66gu1/easygodocs/internal/infrastructure/httpx"
	"github.com/66gu1/easygodocs/internal/infrastructure/logger"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

type TokenCodec interface {
	ParseToken(tokenStr string, claims jwt.Claims) error
}

type RevocationChecker interface {
	IsRevoked(ctx context.Context, userID, sessionID uuid.UUID, issuedAt time.Time) (bool, error)
}

type SessionValidator interface {
	ValidateSession(ctx context.Context, userID, sessionID uuid.UUID) error
}

// AuthMiddleware parses and validates JWT from Authorization header
// and checks that the token wasn't revoked, so logout, password change and user deletion
// take effect immediately.
//
// The check is a lookup in Redis instead of database queries on every request. If Redis is unavailable,
// the middleware falls back to checking the session in PostgreSQL, the source of truth: it is slower,
// but neither lets revoked tokens through nor rejects valid ones.
func AuthMiddleware(codec TokenCodec, revocations RevocationChecker, sessions SessionValidator) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ctx := r.Context()
			authHeader := r.Header.Get("Authorization")
			if !strings.HasPrefix(authHeader, "Bearer ") {
				err := apperr.ErrUnauthorized().WithDetail("missing or malformed Authorization header")
				httpx.ReturnError(ctx, w, fmt.Errorf("auth.AuthMiddleware: invalid Authorization header: %w", err))
				return
			}
			tokenStr := strings.TrimPrefix(authHeader, "Bearer ")
			claims := auth.AccessTokenClaims{}
			err := codec.ParseToken(tokenStr, &claims)
			if err != nil {
				httpx.ReturnError(ctx, w, fmt.Errorf("auth.AuthMiddleware: invalid token: %w", apperr.ErrUnauthorized().WithDetail(err.Error())))
				return
			}

			userID, err := uuid.Parse(claims.Subject)
			if err != nil {
				httpx.ReturnError(ctx, w, fmt.Errorf("auth.AuthMiddleware: invalid token claims.Subject: %w", apperr.ErrUnauthorized().WithDetail(err.Error())))
				return
			}
			if userID == uuid.Nil {
				err = apperr.ErrUnauthorized().WithDetail("invalid token claims.Subject")
				httpx.ReturnError(ctx, w, fmt.Errorf("auth.AuthMiddleware: invalid token claims.Subject: %w", err))
				return
			}
			sessionID, err := uuid.Parse(claims.SID)
			if err != nil {
				httpx.ReturnError(ctx, w, fmt.Errorf("auth.AuthMiddleware: invalid token claims.SID: %w", apperr.ErrUnauthorized().WithDetail(err.Error())))
				return
			}
			if sessionID == uuid.Nil {
				err = apperr.ErrUnauthorized().WithDetail("invalid token claims.SID")
				httpx.ReturnError(ctx, w, fmt.Errorf("auth.AuthMiddleware: invalid token claims.SID: %w", err))
				return
			}

			if claims.ExpiresAt == nil || !claims.ExpiresAt.After(time.Now().UTC()) {
				err = apperr.ErrUnauthorized().WithDetail("token is expired")
				httpx.ReturnError(ctx, w, fmt.Errorf("auth.AuthMiddleware: token is expired: %w", err))
				return
			}

			if err = checkNotRevoked(ctx, revocations, sessions, userID, sessionID, claims.IssuedAtMs); err != nil {
				httpx.ReturnError(ctx, w, fmt.Errorf("auth.AuthMiddleware: session %s: %w", sessionID, err))
				return
			}

			ctx = contextx.SetUserID(ctx, userID)
			ctx = contextx.SetSessionID(ctx, sessionID)

			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

func checkNotRevoked(ctx context.Context, revocations RevocationChecker, sessions SessionValidator,
	userID, sessionID uuid.UUID, issuedAtMs int64,
) error {
	revoked, err := revocations.IsRevoked(ctx, userID, sessionID, time.UnixMilli(issuedAtMs))
	if err == nil {
		if revoked {
			return fmt.Errorf("checkNotRevoked: %w", apperr.ErrUnauthorized().WithDetail("token is revoked"))
		}
		return nil
	}

	logger.Error(ctx, err).Msg("auth.AuthMiddleware: revocations are unavailable, checking the session in the database")
	if err = sessions.ValidateSession(ctx, userID, sessionID); err != nil {
		if apperr.ClassOf(err) != apperr.ClassInternal {
			// any client-side reason (not found, expired, ...) is reported as 401
			err = apperr.ErrUnauthorized().WithDetail(err.Error())
		}
		return fmt.Errorf("checkNotRevoked: session is not active: %w", err)
	}
	return nil
}
