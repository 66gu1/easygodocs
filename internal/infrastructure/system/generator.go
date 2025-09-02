package system

import (
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"time"

	"github.com/google/uuid"
)

type UUIDv7Generator struct{}

func (g *UUIDv7Generator) New() (uuid.UUID, error) {
	id, err := uuid.NewV7()
	if err != nil {
		return uuid.Nil, fmt.Errorf("UUIDv7Generator.New: %w", err)
	}

	return id, nil
}

type RNDGenerator struct{}

func (g *RNDGenerator) New(n int) (string, error) {
	bytes := make([]byte, n)
	if _, err := rand.Read(bytes); err != nil {
		return "", fmt.Errorf("RNDGenerator.New: %w", err)
	}
	return base64.URLEncoding.EncodeToString(bytes), nil
}

type TimeGenerator struct{}

func (g *TimeGenerator) Now() time.Time {
	return time.Now().UTC()
}
