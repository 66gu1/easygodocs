package httputil

import (
	"context"
	"encoding/json"
	"github.com/66gu1/easygodocs/internal/infrastructure/logger"
	"net/http"
)

func WriteJSON(ctx context.Context, w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(v); err != nil {
		logger.Error(ctx, err).Msg("httputil.WriteJSON: failed to encode JSON")
	}
}
