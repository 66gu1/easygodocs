package httpx

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"mime"
	"net/http"

	"github.com/66gu1/easygodocs/internal/infrastructure/apperr"
	"github.com/66gu1/easygodocs/internal/infrastructure/logger"
)

func WriteJSON(ctx context.Context, w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(v); err != nil {
		logger.Error(ctx, err).Msg("httpx.WriteJSON: failed to encode JSON")
	}
}

func DecodeJSON(r *http.Request, v any) error {
	ct := r.Header.Get("Content-Type")
	if ct == "" {
		return fmt.Errorf("httpx.DecodeJSON: %w",
			apperr.ErrBadRequest().WithDetail("Content-Type required"))
	}

	mt, _, err := mime.ParseMediaType(ct)
	if err != nil {
		return fmt.Errorf("httpx.DecodeJSON: %w",
			apperr.ErrBadRequest().WithDetail("invalid Content-Type header"))
	}

	switch mt {
	case "application/json", "application/problem+json", "application/vnd.api+json":
		// ok
	default:
		return fmt.Errorf("httpx.DecodeJSON: %w",
			apperr.ErrBadRequest().WithDetail("unsupported Content-Type; allowed: application/json, application/problem+json, application/vnd.api+json"))
	}

	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()
	dec.UseNumber()

	if err = dec.Decode(v); err != nil {
		return fmt.Errorf("httpx.DecodeJSON: %w",
			apperr.ErrBadRequest().WithDetail(err.Error()))
	}
	// Disallows a second JSON object
	if err = dec.Decode(&struct{}{}); !errors.Is(err, io.EOF) {
		return fmt.Errorf("httpx.DecodeJSON: %w",
			apperr.ErrBadRequest().WithDetail(err.Error()))
	}
	return nil
}
