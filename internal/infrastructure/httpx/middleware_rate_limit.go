package httpx

import (
	"context"
	"fmt"
	"math"
	"net/http"
	"strconv"

	"github.com/66gu1/easygodocs/internal/infrastructure/apperr"
	"github.com/66gu1/easygodocs/internal/infrastructure/logger"
	"github.com/66gu1/easygodocs/internal/infrastructure/ratelimit"
)

type RateLimiter interface {
	Allow(ctx context.Context, key string) (ratelimit.Result, error)
}

// RateLimit rejects requests over the limit with 429 and a Retry-After header.
//
// It fails open: if the limiter is unavailable, the request is let through and the error is logged.
// The limit protects public endpoints from abuse, it isn't an access control check,
// so an outage of Redis shouldn't take login and registration down with it.
func RateLimit(limiter RateLimiter, keyFunc func(r *http.Request) (string, error)) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ctx := r.Context()

			key, err := keyFunc(r)
			if err != nil {
				ReturnError(ctx, w, fmt.Errorf("httpx.RateLimit: key: %w", err))
				return
			}

			res, err := limiter.Allow(ctx, key)
			if err != nil {
				logger.Error(ctx, err).Msg("rate limiter is unavailable, request is allowed")
				next.ServeHTTP(w, r)
				return
			}

			w.Header().Set("X-RateLimit-Limit", strconv.Itoa(res.Limit))
			w.Header().Set("X-RateLimit-Remaining", strconv.Itoa(res.Remaining))
			if !res.Allowed {
				// Retry-After is in whole seconds, round up so the client doesn't retry too early.
				w.Header().Set("Retry-After", strconv.Itoa(int(math.Ceil(res.RetryAfter.Seconds()))))
				ReturnError(ctx, w, fmt.Errorf("httpx.RateLimit: %w", apperr.ErrRateLimited()))
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}
