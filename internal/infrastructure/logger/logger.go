package logger

import (
	"context"
	"errors"
	"github.com/66gu1/easygodocs/internal/infrastructure/apperror"
	"github.com/66gu1/easygodocs/internal/infrastructure/contextutil"
	"github.com/google/uuid"
	"github.com/rs/zerolog"
)

func Error(ctx context.Context, err error) *zerolog.Event {
	l := zerolog.Ctx(ctx)

	var (
		appErr *apperror.Error
		event  = l.Error()
	)

	if errors.As(err, &appErr) && appErr.LogLevel == apperror.LogLevelWarn {
		event = l.Warn()
	}

	if currentUser, ok := contextutil.GetFromContext[uuid.UUID](ctx, contextutil.ContextKeyUserID); ok {
		event = event.Str("current_user_id", currentUser.String())
	}

	return event.Err(err)
}

func Warn(ctx context.Context, err error) *zerolog.Event {
	l := zerolog.Ctx(ctx)
	resp := l.Warn()
	if err != nil {
		return resp.Err(err)
	}

	return resp
}
