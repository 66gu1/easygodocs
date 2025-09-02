package httpx

import (
	"context"
	"encoding/json"
	"net/http"

	"github.com/66gu1/easygodocs/internal/infrastructure/apperr"
	"github.com/66gu1/easygodocs/internal/infrastructure/logger"
)

func ReturnError(ctx context.Context, w http.ResponseWriter, returningErr error) {
	appError := apperr.FromError(returningErr)
	code := toHTTPCode(apperr.ClassOf(appError))
	if code == 0 {
		logger.Error(ctx, returningErr).Int("error_code", code).Msg("incorrect error code")
		code = http.StatusInternalServerError
	}

	if w.Header().Get("Content-Type") == "" {
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
	}
	w.WriteHeader(code)

	err := json.NewEncoder(w).Encode(map[string]any{
		"error": appError,
	})
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
