package httpx_test

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/66gu1/easygodocs/internal/infrastructure/apperr"
	"github.com/66gu1/easygodocs/internal/infrastructure/httpx"
	"github.com/rs/zerolog"
	"github.com/stretchr/testify/require"
)

func TestReturnError(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		err         error
		wantStatus  int
		wantMessage string
		wantLevel   string
	}{
		{
			name:        "client error is logged as warning",
			err:         fmt.Errorf("handler: %w", apperr.ErrBadRequest().WithDetail("invalid entity ID")),
			wantStatus:  http.StatusBadRequest,
			wantMessage: apperr.BadRequestMsg,
			wantLevel:   "warn",
		},
		{
			name:        "internal error is logged as error and its details are hidden",
			err:         fmt.Errorf("service: %w", errors.New("connection refused")),
			wantStatus:  http.StatusInternalServerError,
			wantMessage: apperr.InternalMsg,
			wantLevel:   "error",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			var logs bytes.Buffer
			ctx := zerolog.New(&logs).WithContext(t.Context())
			rr := httptest.NewRecorder()

			httpx.ReturnError(ctx, rr, tt.err)

			require.Equal(t, tt.wantStatus, rr.Code)
			var body struct {
				Message string `json:"message"`
			}
			require.NoError(t, json.Unmarshal(rr.Body.Bytes(), &body))
			require.Equal(t, tt.wantMessage, body.Message)
			require.NotContains(t, rr.Body.String(), "connection refused")

			lines := strings.Split(strings.TrimSpace(logs.String()), "\n")
			require.Len(t, lines, 1, "the error must be logged exactly once")
			var entry map[string]any
			require.NoError(t, json.Unmarshal([]byte(lines[0]), &entry))
			require.Equal(t, tt.wantLevel, entry["level"])
			require.Equal(t, tt.err.Error(), entry["error"], "the log keeps the full wrap chain")
			require.EqualValues(t, tt.wantStatus, entry["status"])
		})
	}
}
