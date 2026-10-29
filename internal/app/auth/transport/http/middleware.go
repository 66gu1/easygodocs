package http

import (
	"context"
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

type SessionValidator interface {
	ValidateSession(ctx context.Context, userID, sessionID uuid.UUID) error
}

// AuthMiddleware parses and validates JWT from Authorization header
// and checks that the session it was issued for is still active,
// so revoked sessions and deleted users lose access immediately.
func AuthMiddleware(codec TokenCodec, sessions SessionValidator) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ctx := r.Context()
			authHeader := r.Header.Get("Authorization")
			if !strings.HasPrefix(authHeader, "Bearer ") {
				err := apperr.ErrUnauthorized().WithDetail("missing or malformed Authorization header")
				logger.Error(ctx, err).
					Msg("auth.AuthMiddleware: invalid Authorization header")
				httpx.ReturnError(ctx, w, err)
				return
			}
			tokenStr := strings.TrimPrefix(authHeader, "Bearer ")
			claims := auth.AccessTokenClaims{}
			err := codec.ParseToken(tokenStr, &claims)
			if err != nil {
				logger.Error(ctx, err).
					Msg("auth.AuthMiddleware: invalid token")
				httpx.ReturnError(ctx, w, apperr.ErrUnauthorized())
				return
			}

			userID, err := uuid.Parse(claims.Subject)
			if err != nil {
				logger.Warn(ctx, err).
					Str("subject", claims.Subject).
					Msg("auth.AuthMiddleware: invalid token claims.Subject")
				httpx.ReturnError(ctx, w, apperr.ErrUnauthorized())
				return
			}
			if userID == uuid.Nil {
				err = apperr.ErrUnauthorized().WithDetail("invalid token claims.Subject")
				logger.Error(ctx, err).
					Msg("auth.AuthMiddleware: invalid token claims.Subject")
				httpx.ReturnError(ctx, w, err)
				return
			}
			sessionID, err := uuid.Parse(claims.SID)
			if err != nil {
				logger.Warn(ctx, err).
					Str("sid", claims.SID).
					Msg("auth.AuthMiddleware: invalid token claims.SID")
				httpx.ReturnError(ctx, w, apperr.ErrUnauthorized())
				return
			}
			if sessionID == uuid.Nil {
				err = apperr.ErrUnauthorized().WithDetail("invalid token claims.SID")
				logger.Error(ctx, err).
					Msg("auth.AuthMiddleware: invalid token claims.SID")
				httpx.ReturnError(ctx, w, err)
				return
			}

			if claims.ExpiresAt == nil || !claims.ExpiresAt.After(time.Now().UTC()) {
				err = apperr.ErrUnauthorized().WithDetail("token is expired")
				logger.Error(ctx, err).
					Msg("auth.AuthMiddleware: token is expired")
				httpx.ReturnError(ctx, w, err)
				return
			}

			if err = sessions.ValidateSession(ctx, userID, sessionID); err != nil {
				logger.Error(ctx, err).
					Str(auth.FieldUserID.String(), userID.String()).
					Str(auth.FieldSessionID.String(), sessionID.String()).
					Msg("auth.AuthMiddleware: session is not active")
				if apperr.ClassOf(err) == apperr.ClassInternal {
					httpx.ReturnError(ctx, w, err)
					return
				}
				httpx.ReturnError(ctx, w, apperr.ErrUnauthorized())
				return
			}

			ctx = contextx.SetUserID(ctx, userID)
			ctx = contextx.SetSessionID(ctx, sessionID)

			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}
