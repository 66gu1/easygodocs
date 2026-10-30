package httpx_test

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/66gu1/easygodocs/internal/infrastructure/apperr"
	"github.com/66gu1/easygodocs/internal/infrastructure/httpx"
	"github.com/66gu1/easygodocs/internal/infrastructure/httpx/mocks"
	"github.com/66gu1/easygodocs/internal/infrastructure/ratelimit"
	"github.com/gojuno/minimock/v3"
	"github.com/stretchr/testify/require"
)

//go:generate minimock -o ./mocks -s _mock.go

func TestRateLimit(t *testing.T) {
	t.Parallel()

	const ip = "198.51.100.1"
	keyByIP := func(r *http.Request) (string, error) { return httpx.ClientIP(r), nil }

	tests := []struct {
		name        string
		keyFunc     func(r *http.Request) (string, error)
		setup       func(m *mocks.RateLimiterMock)
		wantStatus  int
		wantNext    bool
		wantHeaders map[string]string
		wantCode    apperr.Code
	}{
		{
			name:    "allowed",
			keyFunc: keyByIP,
			setup: func(m *mocks.RateLimiterMock) {
				m.AllowMock.Expect(minimock.AnyContext, ip).
					Return(ratelimit.Result{Allowed: true, Limit: 20, Remaining: 19}, nil)
			},
			wantStatus:  http.StatusOK,
			wantNext:    true,
			wantHeaders: map[string]string{"X-RateLimit-Limit": "20", "X-RateLimit-Remaining": "19", "Retry-After": ""},
		},
		{
			name:    "over the limit: 429 with Retry-After rounded up",
			keyFunc: keyByIP,
			setup: func(m *mocks.RateLimiterMock) {
				m.AllowMock.Expect(minimock.AnyContext, ip).
					Return(ratelimit.Result{Limit: 20, RetryAfter: 1500 * time.Millisecond}, nil)
			},
			wantStatus:  http.StatusTooManyRequests,
			wantHeaders: map[string]string{"X-RateLimit-Limit": "20", "X-RateLimit-Remaining": "0", "Retry-After": "2"},
			wantCode:    apperr.CodeRateLimited,
		},
		{
			name:    "limiter unavailable: fails open",
			keyFunc: keyByIP,
			setup: func(m *mocks.RateLimiterMock) {
				m.AllowMock.Expect(minimock.AnyContext, ip).Return(ratelimit.Result{}, errors.New("redis is down"))
			},
			wantStatus:  http.StatusOK,
			wantNext:    true,
			wantHeaders: map[string]string{"X-RateLimit-Limit": "", "Retry-After": ""},
		},
		{
			name:       "key error",
			keyFunc:    func(*http.Request) (string, error) { return "", errors.New("no key") },
			setup:      func(*mocks.RateLimiterMock) {},
			wantStatus: http.StatusInternalServerError,
			wantCode:   apperr.CodeInternal,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			limiter := mocks.NewRateLimiterMock(minimock.NewController(t))
			tt.setup(limiter)

			var nextCalled bool
			h := httpx.RateLimit(limiter, tt.keyFunc)(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				nextCalled = true
				w.WriteHeader(http.StatusOK)
			}))
			req := httptest.NewRequest(http.MethodPost, "/login", nil)
			req.RemoteAddr = ip + ":5000"
			rec := httptest.NewRecorder()
			h.ServeHTTP(rec, req)

			require.Equal(t, tt.wantStatus, rec.Code)
			require.Equal(t, tt.wantNext, nextCalled)
			for name, want := range tt.wantHeaders {
				require.Equal(t, want, rec.Header().Get(name), name)
			}
			if tt.wantCode != "" {
				var body apperr.AppError
				require.NoError(t, json.NewDecoder(rec.Body).Decode(&body))
				require.Equal(t, tt.wantCode, body.Code)
			}
		})
	}
}
