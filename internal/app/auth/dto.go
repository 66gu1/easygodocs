package auth

import (
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

type Session struct {
	ID             uuid.UUID `json:"id"`
	UserID         uuid.UUID `json:"user_id"`
	CreatedAt      time.Time `json:"created_at"`
	ExpiresAt      time.Time `json:"expires_at"`
	SessionVersion int       `json:"session_version"`
}

type UserRole struct {
	UserID   uuid.UUID  `json:"user_id"`
	Role     Role       `json:"role"`
	EntityID *uuid.UUID `json:"entity_id"`
}

type UpdateTokenReq struct {
	SessionID           uuid.UUID `json:"session_id"`
	UserID              uuid.UUID `json:"user_id"`
	RefreshTokenHash    string    `json:"refresh_token_hash"`
	OldRefreshTokenHash string    `json:"old_refresh_token_hash"`
	ExpiresAt           time.Time `json:"expires_at"`
}

type RefreshToken struct {
	SessionID uuid.UUID `json:"session_id"`
	Token     string    `json:"token"`
}

type Tokens struct {
	AccessToken  string       `json:"access_token"`
	RefreshToken RefreshToken `json:"refresh_token"`
}

type AccessTokenClaims struct {
	SID string `json:"sid"` // session_id
	jwt.RegisteredClaims
}
