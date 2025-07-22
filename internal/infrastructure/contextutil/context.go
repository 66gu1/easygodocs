package contextutil

import (
	"context"
)

type contextKey string

const (
	ContextKeyUserID    = contextKey("user_id")
	ContextKeySessionID = contextKey("session_id")
)

func GetFromContext[T any](ctx context.Context, key contextKey) (T, bool) {
	value := ctx.Value(key)
	if value == nil {
		return *new(T), false
	}

	if v, ok := value.(T); ok {
		return v, true
	}

	return *new(T), false
}

func SetToContext[T any](ctx context.Context, key contextKey, value T) context.Context {
	return context.WithValue(ctx, key, value)
}
