package secure

import (
	"errors"
	"fmt"
	"runtime"

	"golang.org/x/crypto/bcrypt"
)

var ErrMismatchedHashAndPassword = errors.New("mismatched hash and password")

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

func (p *PasswordHasher) CheckPasswordHash(hash, password []byte) error {
	if err := bcrypt.CompareHashAndPassword(hash, password); err != nil {
		if errors.Is(err, bcrypt.ErrMismatchedHashAndPassword) {
			return fmt.Errorf("secure.CheckPasswordHash: %w", ErrMismatchedHashAndPassword)
		}
		return fmt.Errorf("secure.CheckPasswordHash: %w", err)
	}

	return nil
}
