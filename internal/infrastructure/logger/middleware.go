package logger

import (
	"github.com/go-chi/chi/v5/middleware"
	"github.com/rs/zerolog/log"
	"net"
	"net/http"
	"time"
)

// Logger is a middleware that injects a zerolog.Logger into the context,
// and logs the request with method, path, status, and duration.
func Logger(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Start time
		start := time.Now()
		// Extract real IP
		remoteIP, _, err := net.SplitHostPort(r.RemoteAddr)
		if err != nil {
			remoteIP = r.RemoteAddr
		}
		// Get request ID and IP from context
		reqID := middleware.GetReqID(r.Context())

		// Create request-scoped logger
		l := log.With().
			Str("request_id", reqID).
			Str("remote_ip", remoteIP).
			Str("method", r.Method).
			Str("url", r.RequestURI).
			Logger()

		// Inject logger into context
		ctx := l.WithContext(r.Context())
		r = r.WithContext(ctx)

		// Wrap response writer to capture status and size
		ww := middleware.NewWrapResponseWriter(w, r.ProtoMajor)

		// Serve the request
		next.ServeHTTP(ww, r)

		// Log request
		l.Info().
			Int("status", ww.Status()).
			Int("bytes", ww.BytesWritten()).
			Dur("duration", time.Since(start)).
			Msg("request completed")
	})
}
