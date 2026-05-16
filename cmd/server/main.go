package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/joho/godotenv"

	"github.com/icecoldsprite1/knobull-go-search-engine/internal/api"
	"github.com/icecoldsprite1/knobull-go-search-engine/internal/store"
)

// This package provides the entrypoint for the HTTP server.
// It implements graceful shutdown to ensure zero-downtime deployments
// by draining active requests and background workers upon receiving SIGTERM/SIGINT.

func main() {
	// 1. Load environment variables
	if err := godotenv.Load(); err != nil {
		log.Println("No .env file found, using system environment variables.")
	}

	connStr := os.Getenv("DATABASE_URL")
	if connStr == "" {
		log.Fatal("CRITICAL: DATABASE_URL is not set in the environment")
	}

	// 2. Connect to Postgres
	postgresStore, err := store.NewPostgresStore(connStr)
	if err != nil {
		log.Fatal("Could not connect to database: ", err)
	}

	// 3. Create the EngineServer.
	//    This also starts the background worker pool goroutines (see handlers.go).
	engineServer := api.NewEngineServer(postgresStore)

	// 4. Register routes on a new ServeMux to avoid polluting the global namespace.
	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/resources", engineServer.HandleGetResources)
	mux.HandleFunc("POST /api/recommend", engineServer.HandleRecommend)
	mux.Handle("/", http.FileServer(http.Dir("public")))

	// 5. Initialize the HTTP server with explicit timeouts to mitigate Slowloris attacks.
	httpServer := &http.Server{
		Addr:    ":8080",
		Handler: mux,

		// Explicit timeouts prevent resource exhaustion from slow or malicious clients.
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 30 * time.Second,
	}

	// 6. Start the server in a separate goroutine so the main thread can listen for signals.
	go func() {
		log.Println("Knobull Engine started on :8080")
		if err := httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("HTTP server error: %v", err)
		}
	}()

	// 7. Set up signal catching for graceful shutdown.
	// A buffered channel prevents dropped signals if the OS sends them concurrently.
	quit := make(chan os.Signal, 1)

	// Tell the OS signal package to forward SIGINT (Ctrl+C) and SIGTERM
	// (what Kubernetes sends) into our quit channel.
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	// Block until a signal is received.
	sig := <-quit
	log.Printf("Received signal: %s. Beginning graceful shutdown...", sig)

	// 8. Execute graceful shutdown with a 30-second timeout for in-flight requests.
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// 9. Shutdown the HTTP listener first so no new requests are accepted.
	if err := httpServer.Shutdown(ctx); err != nil {
		log.Printf("HTTP server forced to shut down: %v", err)
	} else {
		log.Println("HTTP server shut down cleanly.")
	}

	// 10. Drain the background worker pool after all HTTP handlers have finished.
	engineServer.Shutdown()

	log.Println("All systems shut down. Goodbye.")
}

