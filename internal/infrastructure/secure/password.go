package secure

import (
	"errors"
	"fmt"
	"runtime"

	"golang.org/x/crypto/bcrypt"
)

var ErrMismatchedHashAndPassword = fmt.Errorf("mismatched hash and password")

func ZeroBytes(b []byte) {
	for i := range b {
		b[i] = 0
	}
	runtime.KeepAlive(b)
}

type PasswordHasher struct{}

func NewPasswordHasher() *PasswordHasher {
	return &PasswordHasher{}
}

func (p *PasswordHasher) HashPassword(password []byte, cost int) ([]byte, error) {
	defer ZeroBytes(password)
	hash, err := bcrypt.GenerateFromPassword(password, cost)
	if err != nil {
		return nil, fmt.Errorf("secure.HashPassword: %w", err)
	}

	return hash, nil
}

func (p *PasswordHasher) HashRefreshToken(token []byte) ([]byte, error) {
	hash, err := p.HashPassword(token, bcrypt.MinCost)
	if err != nil {
		return nil, fmt.Errorf("secure.HashRefreshToken: %w", err)
	}

	return hash, nil
}

func (p *PasswordHasher) CheckPasswordHash(password []byte, hash string) error {
	defer ZeroBytes(password)
	if err := bcrypt.CompareHashAndPassword([]byte(hash), password); err != nil {
		if errors.Is(err, bcrypt.ErrMismatchedHashAndPassword) {
			return fmt.Errorf("secure.CheckPasswordHash: %w", ErrMismatchedHashAndPassword)
		}
		return fmt.Errorf("secure.CheckPasswordHash: %w", err)
	}

	return nil
}
