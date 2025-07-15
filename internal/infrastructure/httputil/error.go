package httputil

import (
	"context"
	"encoding/json"
	"errors"
	"github.com/66gu1/easygodocs/internal/infrastructure/apperror"
	"github.com/66gu1/easygodocs/internal/infrastructure/logger"
	"net/http"
)

func returnInternalError(ctx context.Context, w http.ResponseWriter) {
	returnError(ctx, w, http.StatusInternalServerError, "internal server error")
}

func returnError(ctx context.Context, w http.ResponseWriter, status int, message string) {

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)

	err := json.NewEncoder(w).Encode(map[string]string{
		"error": message,
	})
	if err != nil {
		logger.Error(ctx, err).Msg("error encode failed")
	}
}

func ReturnError(ctx context.Context, w http.ResponseWriter, err error) {
	var appErr *apperror.Error
	if !errors.As(err, &appErr) || appErr.Code == apperror.InternalError {
		returnInternalError(ctx, w)
		return
	}

	code := getCode(appErr)
	if code == 0 {
		logger.Error(ctx, err).Int("error_code", int(appErr.Code)).Msg("incorrect error code")
		returnInternalError(ctx, w)
	}

	returnError(ctx, w, code, appErr.Message)
}

func getCode(err *apperror.Error) int {
	switch err.Code {
	case apperror.InternalError:
		return http.StatusInternalServerError
	case apperror.BadRequest:
		return http.StatusBadRequest
	}

	return 0
}
