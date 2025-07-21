package logger

import (
	"context"
	"errors"
	"github.com/66gu1/easygodocs/internal/infrastructure/apperror"
	"github.com/rs/zerolog"
)

func Error(ctx context.Context, err error) *zerolog.Event {
	l := zerolog.Ctx(ctx)
	if err == nil {
		return l.Error()
	}

	var appErr *apperror.Error
	if errors.As(err, &appErr) {
		if appErr.LogLevel == apperror.LogLevelWarn {
			return l.Warn().Err(err)
		}
	}

	return l.Error().Err(err)
}

func Warn(ctx context.Context, err error) *zerolog.Event {
	l := zerolog.Ctx(ctx)
	resp := l.Warn()
	if err != nil {
		return resp.Err(err)
	}

	return resp
}
