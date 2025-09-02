package logger

import (
	"context"
	"errors"

	"github.com/66gu1/easygodocs/internal/infrastructure/apperr"
	"github.com/66gu1/easygodocs/internal/infrastructure/contextx"
	"github.com/rs/zerolog"
)

func Error(ctx context.Context, loggingErr error) *zerolog.Event {
	return log(ctx, apperr.LogLevelOf(loggingErr), loggingErr)
}

func Warn(ctx context.Context, loggingErr error) *zerolog.Event {
	return log(ctx, apperr.LogLevelWarn, loggingErr)
}

func log(ctx context.Context, level apperr.LogLevel, loggingErr error) *zerolog.Event {
	ctx = context.WithoutCancel(ctx)
	event := zerolog.Ctx(ctx).WithLevel(toZerologLevel(level))

	currentUser, err := contextx.GetUserID(ctx)
	if err != nil {
		if !errors.Is(err, contextx.ErrNotFound) {
			zerolog.Ctx(ctx).Error().Err(err).Msg("logger.log: GetUserID")
		}
	} else {
		event = event.Str("current_user_id", currentUser.String())
	}

	sessionID, err := contextx.GetSessionID(ctx)
	if err != nil {
		if !errors.Is(err, contextx.ErrNotFound) {
			zerolog.Ctx(ctx).Error().Err(err).Msg("logger.log: GetSessionID")
		}
	} else {
		event = event.Str("session_id", sessionID.String())
	}

	if loggingErr != nil {
		event = event.Err(loggingErr)
	}

	return event
}

func toZerologLevel(level apperr.LogLevel) zerolog.Level {
	switch level {
	case apperr.LogLevelWarn:
		return zerolog.WarnLevel
	default:
		return zerolog.ErrorLevel
	}
}
