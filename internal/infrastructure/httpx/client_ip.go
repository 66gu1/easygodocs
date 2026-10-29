package httpx

import (
	"net"
	"net/http"

	"github.com/go-chi/chi/v5/middleware"
)

// ClientIPMiddleware determines the client IP for logging and rate limiting.
// Without trusted proxies the IP of the TCP connection is used and forwarding headers are ignored,
// so clients can't spoof their IP. With trusted proxies (CIDR prefixes) the IP is taken from
// X-Forwarded-For, skipping the entries added by those proxies.
func ClientIPMiddleware(trustedProxies []string) func(http.Handler) http.Handler {
	if len(trustedProxies) == 0 {
		return middleware.ClientIPFromRemoteAddr
	}
	return middleware.ClientIPFromXFF(trustedProxies...)
}

// ClientIP returns the IP determined by ClientIPMiddleware,
// or the host of RemoteAddr if it couldn't be determined.
func ClientIP(r *http.Request) string {
	if ip := middleware.GetClientIP(r.Context()); ip != "" {
		return ip
	}
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}
	return host
}

// KeyByClientIP is a rate limiter key function that uses ClientIP.
func KeyByClientIP(r *http.Request) (string, error) {
	return ClientIP(r), nil
}
