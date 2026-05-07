package main

import (
	"encoding/json"
	"net/http"
)

// EngineServer holds our database dependency
type EngineServer struct {
	store ResourceStore
}

func (e *EngineServer) HandleGetResources(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(e.store.GetResources())
}

func (e *EngineServer) HandleRecommend(w http.ResponseWriter, r *http.Request) {
	var req SearchRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request", http.StatusBadRequest)
		return
	}
	defer r.Body.Close()

	// Ask the database to do the searching
	matches := e.store.SearchResources(req.Goal)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(matches)
}
