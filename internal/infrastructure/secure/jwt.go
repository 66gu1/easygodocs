package secure

import (
	"fmt"

	"github.com/66gu1/easygodocs/internal/infrastructure/apperr"
	"github.com/golang-jwt/jwt/v5"
)

type TokenCodec struct {
	JWTSecret []byte
}

func NewTokenCodec(secret []byte) *TokenCodec {
	if len(secret) == 0 {
		panic("TokenCodec: secret is empty")
	}
	return &TokenCodec{
		JWTSecret: secret,
	}
}

func (c *TokenCodec) ParseToken(tokenStr string, claims jwt.Claims) error {
	token, err := jwt.ParseWithClaims(tokenStr, claims, func(t *jwt.Token) (interface{}, error) {
		method, ok := t.Method.(*jwt.SigningMethodHMAC)
		if !ok || method != jwt.SigningMethodHS256 {
			return nil, fmt.Errorf("NewTokenCodec.ParseToken: %w", apperr.ErrUnauthorized().WithDetail("unexpected signing method"))
		}
		return c.JWTSecret, nil
	})
	if err != nil {
		return fmt.Errorf("NewTokenCodec.ParseToken: %w", apperr.ErrUnauthorized().WithDetail(err.Error()))
	}
	if !token.Valid {
		return fmt.Errorf("NewTokenCodec.ParseToken: %w", apperr.ErrUnauthorized().WithDetail("invalid token"))
	}

	return nil
}

func (c *TokenCodec) GenerateToken(claims jwt.Claims) (string, error) {
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenStr, err := token.SignedString(c.JWTSecret)
	if err != nil {
		return "", fmt.Errorf("TokenCodec.GenerateToken: %w", err)
	}
	return tokenStr, nil
}
