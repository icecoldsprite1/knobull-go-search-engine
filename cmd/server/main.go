package main

import (
	"context"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/joho/godotenv"

	"github.com/icecoldsprite1/knobull-go-search-engine/internal/api"
	"github.com/icecoldsprite1/knobull-go-search-engine/internal/flags"
	"github.com/icecoldsprite1/knobull-go-search-engine/internal/middleware"
	"github.com/icecoldsprite1/knobull-go-search-engine/internal/store"
)

// This package provides the entrypoint for the HTTP server.
// It implements graceful shutdown to ensure zero-downtime deployments
// by draining active requests and background workers upon receiving SIGTERM/SIGINT.

func main() {
	// 1. Initialize structured logging.
	// This redirects ALL log output — including the stdlib log package and the
	// LD SDK — through slog's JSON handler, making every line machine-readable.
	slog.SetDefault(slog.New(slog.NewJSONHandler(os.Stdout, nil)))

	// 2. Load environment variables
	if err := godotenv.Load(); err != nil {
		slog.Info("no .env file found, using system environment variables")
	}

	connStr := os.Getenv("DATABASE_URL")
	if connStr == "" {
		slog.Error("DATABASE_URL is not set in the environment")
		os.Exit(1)
	}

	// 3. Connect to Postgres
	postgresStore, err := store.NewPostgresStore(connStr)
	if err != nil {
		slog.Error("could not connect to database", "error", err)
		os.Exit(1)
	}

	// 4. Initialize feature flags.
	// If LD_SDK_KEY is not set, we use a stub with hardcoded defaults so the
	// app starts normally without a LaunchDarkly account.
	var flagProvider flags.Provider
	if ldKey := os.Getenv("LD_SDK_KEY"); ldKey != "" {
		ldProvider, err := flags.NewLaunchDarklyProvider(ldKey)
		if err != nil {
			slog.Error("LaunchDarkly initialization failed", "error", err)
			os.Exit(1)
		}
		flagProvider = ldProvider
		slog.Info("feature flags: using LaunchDarkly")
	} else {
		flagProvider = &flags.StubProvider{
			Bools: map[string]bool{"hybrid-search-enabled": true},
			Ints:  map[string]int{"search-results-limit": 5},
		}
		slog.Info("feature flags: using hardcoded defaults", "hybrid", true, "limit", 5)
	}

	// 5. Create the EngineServer.
	//    This also starts the background worker pool goroutines (see handlers.go).
	engineServer := api.NewEngineServer(postgresStore, flagProvider)

	// 6. Register routes on a new ServeMux to avoid polluting the global namespace.
	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/resources", engineServer.HandleGetResources)
	mux.HandleFunc("POST /api/recommend", engineServer.HandleRecommend)
	mux.Handle("/", http.FileServer(http.Dir("public")))

	// 7. Initialize the HTTP server.
	// The handler chain is: Recovery → RateLimit → Logging → mux → handler
	// Recovery is outermost so it catches panics from everything inside.
	// RateLimit is inside Recovery so rejected requests still get panic protection.
	// Logging is innermost so both 200s and 429s are logged with their status codes.
	httpServer := &http.Server{
		Addr:    ":8080",
		Handler: middleware.Recovery(middleware.RateLimit(10, 20)(middleware.Logging(mux))),

		// Explicit timeouts prevent resource exhaustion from slow or malicious clients.
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 30 * time.Second,
	}

	// 8. Start the server in a separate goroutine so the main thread can listen for signals.
	go func() {
		slog.Info("server started", "addr", ":8080")
		if err := httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			slog.Error("HTTP server error", "error", err)
		}
	}()

	// 9. Set up signal catching for graceful shutdown.
	// A buffered channel prevents dropped signals if the OS sends them concurrently.
	quit := make(chan os.Signal, 1)

	// Tell the OS signal package to forward SIGINT (Ctrl+C) and SIGTERM
	// (what Kubernetes sends) into our quit channel.
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	// Block until a signal is received.
	sig := <-quit
	slog.Info("shutdown signal received, beginning graceful shutdown", "signal", sig.String())

	// 10. Execute graceful shutdown with a 30-second timeout for in-flight requests.
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// 11. Shutdown the HTTP listener first so no new requests are accepted.
	if err := httpServer.Shutdown(ctx); err != nil {
		slog.Error("HTTP server forced shutdown", "error", err)
	} else {
		slog.Info("HTTP server shut down cleanly")
	}

	// 12. Drain the background worker pool after all HTTP handlers have finished.
	engineServer.Shutdown()

	// 13. Close the flag provider. If using LaunchDarkly, this flushes analytics.
	// If using the stub, this is a no-op.
	flagProvider.Close()

	slog.Info("all systems shut down")
}

