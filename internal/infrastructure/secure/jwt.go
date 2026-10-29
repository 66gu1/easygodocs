package secure

import (
	"fmt"
	"time"

	"github.com/66gu1/easygodocs/internal/infrastructure/apperr"
	"github.com/golang-jwt/jwt/v5"
)

// MinJWTSecretLength is the minimum HS256 secret size recommended by RFC 7518 (256 bits).
const MinJWTSecretLength = 32

const tokenLeeway = 30 * time.Second

type TokenCodec struct {
	JWTSecret []byte
	parser    *jwt.Parser
}

// NewTokenCodec creates a codec that signs tokens with HS256 and only accepts
// tokens with the given issuer and an expiration time.
func NewTokenCodec(secret []byte, issuer string) *TokenCodec {
	if len(secret) == 0 {
		panic("TokenCodec: secret is empty")
	}
	if issuer == "" {
		panic("TokenCodec: issuer is empty")
	}
	return &TokenCodec{
		JWTSecret: secret,
		parser: jwt.NewParser(
			jwt.WithValidMethods([]string{jwt.SigningMethodHS256.Alg()}),
			jwt.WithIssuer(issuer),
			jwt.WithExpirationRequired(),
			jwt.WithIssuedAt(),
			jwt.WithLeeway(tokenLeeway),
		),
	}
}

func (c *TokenCodec) ParseToken(tokenStr string, claims jwt.Claims) error {
	token, err := c.parser.ParseWithClaims(tokenStr, claims, func(_ *jwt.Token) (interface{}, error) {
		return c.JWTSecret, nil
	})
	if err != nil {
		return fmt.Errorf("TokenCodec.ParseToken: %w", apperr.ErrUnauthorized().WithDetail(err.Error()))
	}
	if !token.Valid {
		return fmt.Errorf("TokenCodec.ParseToken: %w", apperr.ErrUnauthorized().WithDetail("invalid token"))
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
