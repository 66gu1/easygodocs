package secure_test

import (
	"testing"
	"time"

	"github.com/66gu1/easygodocs/internal/app/auth"
	"github.com/66gu1/easygodocs/internal/infrastructure/apperr"
	"github.com/66gu1/easygodocs/internal/infrastructure/secure"
	"github.com/golang-jwt/jwt/v5"
	"github.com/stretchr/testify/require"
	"golang.org/x/crypto/bcrypt"
)

func TestPasswordHasher_HashPassword(t *testing.T) {
	t.Parallel()

	password := "password"
	tests := []struct {
		name     string
		password []byte
		wantErr  bool
	}{
		{
			name:     "ok",
			password: []byte(password),
		},
		{
			name:     "too long password",
			password: []byte("passwordasdasdasdsadsadsadsadsadasdasdsadasdgjhjhgjhagsdjgsajhdgjsahdgjasgdjgasdjhgsadasdsa"),
			wantErr:  true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			hasher := secure.NewPasswordHasher()
			hash, err := hasher.HashPassword(tt.password, 4)
			if tt.wantErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
				err = bcrypt.CompareHashAndPassword(hash, []byte(password))
				require.NoError(t, err)
			}
			for i := range tt.password {
				require.Zero(t, tt.password[i])
			}
		})
	}
}

func TestPasswordHasher_CheckPasswordHash(t *testing.T) {
	t.Parallel()
	password := "password"

	hash, err := bcrypt.GenerateFromPassword([]byte(password), 4)
	require.NoError(t, err)
	tests := []struct {
		name     string
		password []byte
		hash     []byte
		wantErr  bool
	}{
		{
			name:     "ok",
			password: []byte(password),
			hash:     hash,
		},
		{
			name:     "mismatched password",
			password: []byte("wrongpassword"),
			hash:     hash,
			wantErr:  true,
		},
		{
			name:     "empty hash",
			password: []byte(password),
			hash:     []byte(""),
			wantErr:  true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			hasher := secure.NewPasswordHasher()
			err := hasher.CheckPasswordHash(tt.hash, tt.password)
			if tt.wantErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}
			for i := range tt.password {
				require.Zero(t, tt.password[i])
			}
		})
	}
}

func TestTokenCodec_GenerateToken(t *testing.T) {
	t.Parallel()
	secret := []byte("mysecret")
	codec := secure.NewTokenCodec(secret)
	claims := auth.AccessTokenClaims{
		SID: "sid",
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer:    "Issuer",
			Subject:   "Subject",
			ExpiresAt: &jwt.NumericDate{Time: time.Now().Truncate(time.Second).Add(1 * time.Hour)},
			IssuedAt:  &jwt.NumericDate{Time: time.Now().Truncate(time.Second)},
			ID:        "ID",
		},
	}
	tokenStr, err := codec.GenerateToken(claims)
	require.NoError(t, err)
	require.NotEmpty(t, tokenStr)
	gotClaims := auth.AccessTokenClaims{}
	token, err := jwt.ParseWithClaims(tokenStr, &gotClaims, func(token *jwt.Token) (interface{}, error) { return secret, nil })
	require.NoError(t, err)
	require.True(t, token.Valid)
	require.Equal(t, claims, gotClaims)
}

func TestTokenCodec_ParseToken(t *testing.T) {
	t.Parallel()
	var (
		secret = []byte("mysecret")
		codec  = secure.NewTokenCodec(secret)
		claims = auth.AccessTokenClaims{
			SID: "sid",
			RegisteredClaims: jwt.RegisteredClaims{
				Issuer:    "Issuer",
				Subject:   "Subject",
				ExpiresAt: &jwt.NumericDate{Time: time.Now().Truncate(time.Second).Add(1 * time.Hour)},
				IssuedAt:  &jwt.NumericDate{Time: time.Now().Truncate(time.Second)},
				ID:        "ID",
			},
		}
	)
	validToken := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	validTokenStr, err := validToken.SignedString(secret)
	require.NoError(t, err)
	invalidMethodToken := jwt.NewWithClaims(jwt.SigningMethodHS384, claims)
	invalidMethodTokenStr, err := invalidMethodToken.SignedString(secret)
	require.NoError(t, err)
	invalidToken := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	invalidTokenStr, err := invalidToken.SignedString([]byte("wrongsecret"))
	require.NoError(t, err)

	tests := []struct {
		name     string
		tokenStr string
		err      error
	}{
		{
			name:     "valid token",
			tokenStr: validTokenStr,
		},
		{
			name:     "invalid signing method",
			tokenStr: invalidMethodTokenStr,
			err:      apperr.ErrUnauthorized(),
		},
		{
			name:     "invalid token",
			tokenStr: invalidTokenStr,
			err:      apperr.ErrUnauthorized(),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			gotClaims := auth.AccessTokenClaims{}
			err := codec.ParseToken(tt.tokenStr, &gotClaims)
			if tt.err != nil {
				require.ErrorIs(t, err, tt.err)
			} else {
				require.NoError(t, err)
				require.Equal(t, claims, gotClaims)
			}
		})
	}
}
