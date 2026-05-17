package middleware

import (
	"log/slog"
	"net/http"

	"golang.org/x/time/rate"
)

// RateLimit returns a middleware that enforces a token bucket rate limit.
//
// The token bucket algorithm works like this:
//   - A bucket holds up to `burst` tokens
//   - Tokens refill at `rps` (requests per second) rate
//   - Each request consumes 1 token
//   - If the bucket is empty, the request is rejected with 429 Too Many Requests
//
// This naturally allows short bursts (up to `burst` requests at once) while
// enforcing a sustained rate of `rps` requests per second.
//
// The LaunchDarkly SDK uses the same algorithm for rate-limiting analytics
// event flushes to prevent overwhelming the LD backend.
//
// This function is a "middleware factory" — it returns a middleware function.
// Usage: RateLimit(10, 20)(nextHandler)
//   - 10 requests/second sustained rate
//   - 20 requests allowed in a single burst
func RateLimit(rps float64, burst int) func(http.Handler) http.Handler {
	// The limiter is created once when the middleware is initialized.
	// It is safe for concurrent use — rate.Limiter uses sync.Mutex internally.
	limiter := rate.NewLimiter(rate.Limit(rps), burst)

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Allow() consumes one token and returns false if none are available.
			if !limiter.Allow() {
				slog.Warn("rate limit exceeded",
					"method", r.Method,
					"path", r.URL.Path,
				)
				http.Error(w, "Too Many Requests", http.StatusTooManyRequests)
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}
