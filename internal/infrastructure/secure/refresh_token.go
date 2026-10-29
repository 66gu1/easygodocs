package secure

import (
	"crypto/sha256"
	"crypto/subtle"
	"encoding/hex"
	"errors"
	"fmt"
)

var ErrMismatchedRefreshToken = errors.New("mismatched refresh token")

// RefreshTokenHasher hashes refresh tokens with SHA-256.
// Refresh tokens are high-entropy random values, so a fast hash is enough:
// a slow KDF like bcrypt only protects low-entropy secrets such as passwords.
type RefreshTokenHasher struct{}

func NewRefreshTokenHasher() *RefreshTokenHasher {
	return &RefreshTokenHasher{}
}

// HashRefreshToken returns the hex-encoded SHA-256 digest of the token.
func (h *RefreshTokenHasher) HashRefreshToken(token []byte) []byte {
	sum := sha256.Sum256(token)
	dst := make([]byte, hex.EncodedLen(len(sum)))
	hex.Encode(dst, sum[:])
	return dst
}

// CheckRefreshTokenHash compares the token against the hash in constant time.
func (h *RefreshTokenHasher) CheckRefreshTokenHash(hash, token []byte) error {
	if subtle.ConstantTimeCompare(hash, h.HashRefreshToken(token)) != 1 {
		return fmt.Errorf("secure.CheckRefreshTokenHash: %w", ErrMismatchedRefreshToken)
	}
	return nil
}
