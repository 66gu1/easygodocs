package contextx

import (
	"context"
	"errors"
	"fmt"

	"github.com/66gu1/easygodocs/internal/infrastructure/apperr"
	"github.com/google/uuid"
)

var ErrNotFound = fmt.Errorf("not found in context")

type contextKey string

func (key contextKey) String() string {
	return string(key)
}

const (
	ContextKeyUserID    = contextKey("user_id")
	ContextKeySessionID = contextKey("session_id")
)

func getValue[T any](ctx context.Context, key contextKey) (T, error) {
	var zero T

	value := ctx.Value(key)
	if value == nil {
		return zero, fmt.Errorf("key %v: %w", key, ErrNotFound)
	}

	v, ok := value.(T)
	if !ok {
		return zero, fmt.Errorf("key %v: wrong format in context, got %T, want %T", key, value, zero)
	}

	return v, nil
}

func GetUserID(ctx context.Context) (uuid.UUID, error) {
	userID, err := getValue[uuid.UUID](ctx, ContextKeyUserID)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			err = apperr.ErrUnauthorized().WithDetail("current user ID not found in context")
		}
		return uuid.Nil, fmt.Errorf("contextx.GetUserID: %w", err)
	}
	if userID == uuid.Nil {
		return uuid.Nil, fmt.Errorf("contextx.GetUserID: user ID is nil")
	}

	return userID, nil
}

func GetSessionID(ctx context.Context) (uuid.UUID, error) {
	sessionID, err := getValue[uuid.UUID](ctx, ContextKeySessionID)
	if err != nil {
		return uuid.Nil, fmt.Errorf("contextx.GetSessionID: %w", err)
	}

	return sessionID, nil
}

func SetToContext[T any](ctx context.Context, key contextKey, value T) context.Context {
	return context.WithValue(ctx, key, value)
}
