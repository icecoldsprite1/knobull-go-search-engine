package api

import (
	"context"
	"encoding/json"
	"log"
	"net/http"

	"github.com/icecoldsprite1/knobull-go-search-engine/internal/models"
)

// EngineServer holds our database dependency
type EngineServer struct {
	store models.ResourceStore
}

// NewEngineServer creates a new API server
func NewEngineServer(store models.ResourceStore) *EngineServer {
	return &EngineServer{store: store}
}

func (e *EngineServer) HandleGetResources(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(e.store.GetResources())
}

func (e *EngineServer) HandleRecommend(w http.ResponseWriter, r *http.Request) {
	var req models.SearchRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request", http.StatusBadRequest)
		return
	}
	defer r.Body.Close()

	// Ask the database to do the searching with dynamic filters
	matches := e.store.SearchResources(r.Context(), req)

	// Log asynchronously in the background so we don't block the user's response
	go func() {
		if err := e.store.LogSearch(context.Background(), req, len(matches)); err != nil {
			log.Println("Failed to log search:", err)
		}
	}()

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(matches)
}
