package middleware

import (
	"log/slog"
	"net/http"
	"time"
)

// responseCapture wraps http.ResponseWriter to capture the HTTP status code
// written by the handler. The stdlib's ResponseWriter doesn't expose the status
// code after WriteHeader is called, so we intercept and store it here.
type responseCapture struct {
	http.ResponseWriter
	statusCode int
}

func (rc *responseCapture) WriteHeader(code int) {
	rc.statusCode = code
	rc.ResponseWriter.WriteHeader(code)
}

// Logging wraps an http.Handler and emits a structured log line after every
// request completes, including method, path, status code, and latency.
// It wraps the entire mux, so every route — including static file serving —
// is automatically covered without touching any individual handler.
func Logging(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()

		// Default to 200 because if the handler never calls WriteHeader explicitly
		// (e.g. it only calls w.Write), the stdlib defaults to 200.
		wrapped := &responseCapture{ResponseWriter: w, statusCode: http.StatusOK}

		next.ServeHTTP(wrapped, r)

		slog.Info("request",
			"method", r.Method,
			"path", r.URL.Path,
			"status", wrapped.statusCode,
			"duration_ms", time.Since(start).Milliseconds(),
		)
	})
}
