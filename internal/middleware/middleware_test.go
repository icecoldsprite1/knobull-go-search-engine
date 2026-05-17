package middleware

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
)

// TestLoggingMiddleware verifies that the Logging middleware correctly
// captures and passes through status codes from the inner handler.
// This tests the responseCapture wrapper — the most subtle part of the implementation.
func TestLoggingMiddleware(t *testing.T) {
	tests := []struct {
		name       string
		handler    http.HandlerFunc
		wantStatus int
	}{
		{
			name:       "passes through 200 from inner handler",
			handler:    func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(http.StatusOK) },
			wantStatus: http.StatusOK,
		},
		{
			name:       "captures 404 from inner handler",
			handler:    func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(http.StatusNotFound) },
			wantStatus: http.StatusNotFound,
		},
		{
			name:       "captures 500 from inner handler",
			handler:    func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(http.StatusInternalServerError) },
			wantStatus: http.StatusInternalServerError,
		},
		{
			name: "defaults to 200 when handler never calls WriteHeader",
			handler: func(w http.ResponseWriter, r *http.Request) {
				w.Write([]byte("hello")) // no explicit WriteHeader call
			},
			wantStatus: http.StatusOK,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/test", nil)
			rec := httptest.NewRecorder()

			Logging(tt.handler).ServeHTTP(rec, req)

			if rec.Code != tt.wantStatus {
				t.Errorf("got status %d, want %d", rec.Code, tt.wantStatus)
			}
		})
	}
}

// TestRecoveryMiddleware verifies that panics in handlers are caught and
// converted to 500 responses, rather than crashing the server process.
func TestRecoveryMiddleware(t *testing.T) {
	tests := []struct {
		name       string
		handler    http.HandlerFunc
		wantStatus int
	}{
		{
			name:       "healthy handler passes through normally",
			handler:    func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(http.StatusOK) },
			wantStatus: http.StatusOK,
		},
		{
			name:       "string panic is recovered and returns 500",
			handler:    func(w http.ResponseWriter, r *http.Request) { panic("something went very wrong") },
			wantStatus: http.StatusInternalServerError,
		},
		{
			// Tests that Recovery catches non-string panics (e.g. errors, runtime values).
			// Using errors.New rather than a nil dereference keeps static analysis happy
			// while covering the same code path in recovery.go.
			name:       "error-type panic is recovered and returns 500",
			handler:    func(w http.ResponseWriter, r *http.Request) { panic(errors.New("simulated error")) },
			wantStatus: http.StatusInternalServerError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/test", nil)
			rec := httptest.NewRecorder()

			Recovery(tt.handler).ServeHTTP(rec, req)

			if rec.Code != tt.wantStatus {
				t.Errorf("got status %d, want %d", rec.Code, tt.wantStatus)
			}
		})
	}
}

// TestRateLimitMiddleware verifies that the token bucket rate limiter
// allows requests within the burst limit and rejects requests beyond it.
func TestRateLimitMiddleware(t *testing.T) {
	okHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	tests := []struct {
		name       string
		burst      int
		reqCount   int // total requests to send
		want200    int // how many should succeed
		want429    int // how many should be rejected
	}{
		{
			name:     "all requests within burst succeed",
			burst:    5,
			reqCount: 5,
			want200:  5,
			want429:  0,
		},
		{
			name:     "requests beyond burst get 429",
			burst:    3,
			reqCount: 6,
			want200:  3,
			want429:  3,
		},
		{
			name:     "burst of 1 allows only 1",
			burst:    1,
			reqCount: 3,
			want200:  1,
			want429:  2,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Use a very low rps (0.001) so no tokens refill during the test.
			// This isolates the burst behavior we're testing.
			handler := RateLimit(0.001, tt.burst)(okHandler)

			got200, got429 := 0, 0
			for i := 0; i < tt.reqCount; i++ {
				req := httptest.NewRequest(http.MethodGet, "/test", nil)
				rec := httptest.NewRecorder()
				handler.ServeHTTP(rec, req)

				switch rec.Code {
				case http.StatusOK:
					got200++
				case http.StatusTooManyRequests:
					got429++
				default:
					t.Errorf("unexpected status %d on request %d", rec.Code, i+1)
				}
			}

			if got200 != tt.want200 {
				t.Errorf("got %d successful requests, want %d", got200, tt.want200)
			}
			if got429 != tt.want429 {
				t.Errorf("got %d rejected requests, want %d", got429, tt.want429)
			}
		})
	}
}
