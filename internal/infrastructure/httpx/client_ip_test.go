package httpx_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/66gu1/easygodocs/internal/infrastructure/httpx"
	"github.com/stretchr/testify/require"
)

func TestClientIP(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name           string
		trustedProxies []string
		remoteAddr     string
		xff            string
		want           string
	}{
		{
			name:       "no trusted proxies: forwarding headers are ignored",
			remoteAddr: "198.51.100.1:5000",
			xff:        "203.0.113.7",
			want:       "198.51.100.1",
		},
		{
			name:           "behind a trusted proxy: the client IP is taken from X-Forwarded-For",
			trustedProxies: []string{"10.0.0.0/8"},
			remoteAddr:     "10.0.0.5:5000",
			xff:            "203.0.113.7, 10.0.0.9",
			want:           "203.0.113.7",
		},
		{
			name:           "behind a trusted proxy: entries spoofed by the client are skipped",
			trustedProxies: []string{"10.0.0.0/8"},
			remoteAddr:     "10.0.0.5:5000",
			xff:            "1.1.1.1, 203.0.113.7",
			want:           "203.0.113.7",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			var got string
			h := httpx.ClientIPMiddleware(tt.trustedProxies)(http.HandlerFunc(func(_ http.ResponseWriter, r *http.Request) {
				got = httpx.ClientIP(r)
			}))
			req := httptest.NewRequest(http.MethodGet, "/", nil)
			req.RemoteAddr = tt.remoteAddr
			req.Header.Set("X-Forwarded-For", tt.xff)
			req.Header.Set("X-Real-IP", "192.0.2.1")
			h.ServeHTTP(httptest.NewRecorder(), req)

			require.Equal(t, tt.want, got)
		})
	}

	t.Run("without the middleware falls back to RemoteAddr", func(t *testing.T) {
		t.Parallel()
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		req.RemoteAddr = "198.51.100.1:5000"
		key, err := httpx.KeyByClientIP(req)
		require.NoError(t, err)
		require.Equal(t, "198.51.100.1", key)
	})
}
