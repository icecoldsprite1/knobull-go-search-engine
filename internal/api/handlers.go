package api

import (
	"context"
	"encoding/json"
	"log"
	"net/http"

	"github.com/icecoldsprite1/knobull-go-search-engine/internal/models"
)

// logEvent is the unit of work passed to background workers.
// It contains the necessary data to write a single log entry.
type logEvent struct {
	req          models.SearchRequest
	resultsCount int
}

// EngineServer manages the HTTP API and background task workers.
// It uses a bounded worker pool (logQueue) to handle search logging asynchronously,
// providing backpressure and load shedding if the database becomes slow.
type EngineServer struct {
	store    models.ResourceStore
	logQueue chan logEvent
}

// NewEngineServer initializes the API server and starts the background worker pool.
func NewEngineServer(store models.ResourceStore) *EngineServer {
	const numWorkers = 5
	const queueSize = 100

	s := &EngineServer{
		store:    store,
		logQueue: make(chan logEvent, queueSize), // buffered channel = our work queue
	}

	// Start the bounded pool of background workers.
	// They run for the server's lifetime and stop during graceful shutdown.
	for i := 0; i < numWorkers; i++ {
		go s.logWorker(i + 1)
	}

	log.Printf("Started %d background log workers with queue size %d", numWorkers, queueSize)
	return s
}

// logWorker processes events from the logQueue until the channel is closed.
func (e *EngineServer) logWorker(id int) {
	log.Printf("Log worker %d started", id)

	for event := range e.logQueue {
		// Use context.Background() since this background task outlives the original HTTP request.
		if err := e.store.LogSearch(context.Background(), event.req, event.resultsCount); err != nil {
			log.Printf("Worker %d: failed to log search: %v", id, err)
		}
	}

	// Reached when the channel is closed and drained during shutdown.
	log.Printf("Log worker %d shut down cleanly", id)
}

// Shutdown safely drains the log queue. It closes the log channel,
// ensuring all buffered events are processed before the workers exit.
func (e *EngineServer) Shutdown() {
	log.Println("Closing log queue — workers will drain remaining events...")
	close(e.logQueue)
}

// =============================================================================
// HTTP HANDLERS
// =============================================================================

// HandleGetResources returns all available resources.
func (e *EngineServer) HandleGetResources(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(e.store.GetResources())
}

// HandleRecommend processes search requests and asynchronously logs performance data.
func (e *EngineServer) HandleRecommend(w http.ResponseWriter, r *http.Request) {
	var req models.SearchRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request", http.StatusBadRequest)
		return
	}
	defer r.Body.Close()

	// Perform the search and handle operational errors (e.g., API/DB failures)
	// separately from valid empty results.
	matches, err := e.store.SearchResources(r.Context(), req)
	if err != nil {
		// Log the underlying error chain for debugging, but return a generic 500
		// to avoid leaking internal system details to the client.
		http.Error(w, "Search temporarily unavailable. Please try again.", http.StatusInternalServerError)
		return
	}

	// Enqueue the log event non-blockingly. If the queue is full, we shed the load
	// (drop the log) rather than degrading the user's search response time.
	select {
	case e.logQueue <- logEvent{req: req, resultsCount: len(matches)}:
		// Event enqueued successfully.
	default:
		log.Println("Warning: log queue full, dropping search log event")
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(matches)
}

