package middleware

import (
	"log/slog"
	"net/http"
)

// Recovery wraps an http.Handler and catches any panics that occur during
// request handling. Without this, a single nil-pointer dereference in any
// handler crashes the entire server process, dropping all in-flight requests.
// This is always the outermost middleware so it protects everything inside it.
func Recovery(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if err := recover(); err != nil {
				slog.Error("panic recovered",
					"error", err,
					"method", r.Method,
					"path", r.URL.Path,
				)
				http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			}
		}()
		next.ServeHTTP(w, r)
	})
}
