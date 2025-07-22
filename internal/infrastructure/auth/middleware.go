package auth

import (
	"errors"
	"github.com/66gu1/easygodocs/internal/infrastructure/apperror"
	"github.com/66gu1/easygodocs/internal/infrastructure/contextutil"
	"github.com/66gu1/easygodocs/internal/infrastructure/httputil"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"net/http"
	"strings"
)

type AccessTokenClaims struct {
	SID string `json:"sid"` // session_id
	jwt.RegisteredClaims
}

type authService struct {
	cfg *config
}

type config struct {
	JWTSecret string
}

func New(jwtSecret string) *authService {
	return &authService{
		cfg: &config{
			JWTSecret: jwtSecret,
		},
	}
}

// AuthMiddleware parses and validates JWT from Authorization header
func (s *authService) AuthMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		authHeader := r.Header.Get("Authorization")
		if !strings.HasPrefix(authHeader, "Bearer ") {
			err := &apperror.Error{
				Message:  "missing or invalid Authorization header",
				Code:     apperror.Unauthorized,
				LogLevel: apperror.LogLevelWarn,
			}
			httputil.ReturnError(ctx, w, err)
			return
		}
		tokenStr := strings.TrimPrefix(authHeader, "Bearer ")

		claims := &AccessTokenClaims{}
		token, err := jwt.ParseWithClaims(tokenStr, claims, func(t *jwt.Token) (interface{}, error) {
			if t.Method != jwt.SigningMethodHS256 {
				return nil, errors.New("unexpected signing method")
			}
			return []byte(s.cfg.JWTSecret), nil
		})
		if err != nil || !token.Valid {
			httputil.ReturnError(ctx, w, &apperror.Error{
				Message:  "invalid or expired access token",
				Code:     apperror.Unauthorized,
				LogLevel: apperror.LogLevelWarn,
			})
			return
		}

		userID, err := uuid.Parse(claims.Subject)
		if err != nil {
			err = &apperror.Error{
				Message:  "invalid user ID in token",
				Code:     apperror.Unauthorized,
				LogLevel: apperror.LogLevelWarn,
			}
			httputil.ReturnError(ctx, w, err)
			return
		}
		sessionID, err := uuid.Parse(claims.SID)
		if err != nil {
			err = &apperror.Error{
				Message:  "invalid session ID in token",
				Code:     apperror.Unauthorized,
				LogLevel: apperror.LogLevelWarn,
			}
			httputil.ReturnError(ctx, w, err)
			return
		}

		ctx = contextutil.SetToContext(ctx, contextutil.ContextKeyUserID, userID)
		ctx = contextutil.SetToContext(ctx, contextutil.ContextKeySessionID, sessionID)

		next.ServeHTTP(w, r.WithContext(ctx))
	})
}
