package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"testing"
)

// StubStore is our fake database just for testing
type StubStore struct {
	resources []Resource
}

func (s *StubStore) GetResources() []Resource {
	return s.resources
}

func (s *StubStore) SearchResources(goal string) []Resource {
	// If the test asks for "go", return our fake resource. Otherwise, return nothing.
	if goal == "go" {
		return s.resources
	}
	return nil
}

func TestEngineServer(t *testing.T) {
	// Setup our fake environment
	wantedResources := []Resource{
		{ID: "99", Title: "Test Course", Keywords: []string{"Test"}},
	}
	store := &StubStore{resources: wantedResources}
	server := &EngineServer{store: store}

	t.Run("returns all resources as JSON", func(t *testing.T) {
		req, _ := http.NewRequest(http.MethodGet, "/api/resources", nil)
		res := httptest.NewRecorder()

		// Call the handler
		server.HandleGetResources(res, req)

		// Assert Status Code
		if res.Code != http.StatusOK {
			t.Errorf("got status %d want %d", res.Code, http.StatusOK)
		}

		// Decode the response
		var got []Resource
		err := json.NewDecoder(res.Body).Decode(&got)
		if err != nil {
			t.Fatalf("Unable to parse response: %v", err)
		}

		// Assert Data matches
		if !reflect.DeepEqual(got, wantedResources) {
			t.Errorf("got %v want %v", got, wantedResources)
		}
	})
	t.Run("returns matched resources for a specific goal", func(t *testing.T) {
		// 1. Create a fake JSON payload {"goal": "go"}
		requestBody := strings.NewReader(`{"goal": "go"}`)

		// 2. Create the POST request
		req, _ := http.NewRequest(http.MethodPost, "/api/recommend", requestBody)
		res := httptest.NewRecorder()

		// 3. Fire it at the server
		server.HandleRecommend(res, req)

		// 4. Decode what the server sent back
		var got []Resource
		err := json.NewDecoder(res.Body).Decode(&got)
		if err != nil {
			t.Fatalf("Unable to parse response: %v", err)
		}

		// 5. Assert that we got exactly 1 result back
		if len(got) != 1 {
			t.Errorf("Expected 1 match, got %d", len(got))
		}

		// 6. Assert that it is the exact resource we expected
		if !reflect.DeepEqual(got, wantedResources) {
			t.Errorf("got %v want %v", got, wantedResources)
		}
	})
}
