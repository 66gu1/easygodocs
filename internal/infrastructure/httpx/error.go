package httpx

import (
	"context"
	"encoding/json"
	"net/http"

	"github.com/66gu1/easygodocs/internal/infrastructure/apperr"
	"github.com/66gu1/easygodocs/internal/infrastructure/logger"
)

// ReturnError writes the error response and logs the error.
// This is the only place where request errors are logged: lower layers just wrap them,
// so every error is logged once, with the full wrap chain and request context
// (request ID, URL, current user). Client errors are logged as warnings, internal ones as errors.
func ReturnError(ctx context.Context, w http.ResponseWriter, returningErr error) {
	appError := apperr.FromError(returningErr)
	code := toHTTPCode(apperr.ClassOf(appError))
	if code == 0 {
		logger.Error(ctx, returningErr).Int("error_code", code).Msg("incorrect error code")
		code = http.StatusInternalServerError
	}
	logger.Error(ctx, returningErr).Int("status", code).Msg("request failed")

	if w.Header().Get("Content-Type") == "" {
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
	}
	w.WriteHeader(code)

	err := json.NewEncoder(w).Encode(appError)
	if err != nil {
		logger.Error(ctx, err).Str("returning_error", returningErr.Error()).Msg("error encode failed")
	}
}

func toHTTPCode(code apperr.Class) int {
	switch code {
	case apperr.ClassBadRequest:
		return http.StatusBadRequest
	case apperr.ClassNotFound:
		return http.StatusNotFound
	case apperr.ClassUnauthorized:
		return http.StatusUnauthorized
	case apperr.ClassForbidden:
		return http.StatusForbidden
	case apperr.ClassInternal:
		return http.StatusInternalServerError
	case apperr.ClassConflict:
		return http.StatusConflict
	}

	return 0
}
