package secure

import (
	"fmt"
	"runtime"

	"golang.org/x/crypto/bcrypt"
)

func ZeroBytes(b []byte) {
	for i := range b {
		b[i] = 0
	}
	runtime.KeepAlive(b)
}

func HashPassword(password []byte, cost int) ([]byte, error) {
	defer ZeroBytes(password)
	hash, err := bcrypt.GenerateFromPassword(password, cost)
	if err != nil {
		return nil, fmt.Errorf("secure.HashPassword: %w", err)
	}

	return hash, nil
}

func HashRefreshToken(token []byte) ([]byte, error) {
	hash, err := HashPassword(token, bcrypt.MinCost)
	if err != nil {
		return nil, fmt.Errorf("secure.HashRefreshToken: %w", err)
	}

	return hash, nil
}
