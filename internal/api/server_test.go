package api

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"testing"

	"github.com/icecoldsprite1/knobull-go-search-engine/internal/flags"
	"github.com/icecoldsprite1/knobull-go-search-engine/internal/models"
)

// StubStore is our fake database just for testing.
// It satisfies the models.ResourceStore interface without needing a real database.
type StubStore struct {
	resources []models.Resource
	// searchErr lets individual tests inject a fake error to test the error path.
	searchErr error
}

func (s *StubStore) GetResources() []models.Resource {
	return s.resources
}

// SearchResources now returns ([]models.Resource, error) to match the updated interface.
// If searchErr is set, we return that error to simulate an AI or DB failure.
func (s *StubStore) SearchResources(ctx context.Context, req models.SearchRequest, hybridEnabled bool, limit int) ([]models.Resource, error) {
	if s.searchErr != nil {
		return nil, s.searchErr
	}
	if req.Goal == "go" {
		return s.resources, nil
	}
	return nil, nil
}

func (s *StubStore) LogSearch(ctx context.Context, req models.SearchRequest, resultsCount int) error {
	return nil
}

func TestEngineServer(t *testing.T) {
	wantedResources := []models.Resource{
		{
			ID:      "99",
			Title:   "Test Course",
			Keywords: []string{"Test"},
			Type:    "internal_article",
			Link:    "https://example.com",
			Content: "This is a test article",
		},
	}
	store := &StubStore{resources: wantedResources}
	stubFlags := &flags.StubProvider{
		Bools: map[string]bool{"hybrid-search-enabled": true},
		Ints:  map[string]int{"search-results-limit": 5},
	}
	server := NewEngineServer(store, stubFlags)

	// Use t.Cleanup to ensure the server's background workers are cleanly
	// shut down after the test suite finishes, preventing goroutine leaks.
	t.Cleanup(server.Shutdown)

	t.Run("returns all resources as JSON", func(t *testing.T) {
		req, _ := http.NewRequest(http.MethodGet, "/api/resources", nil)
		res := httptest.NewRecorder()

		server.HandleGetResources(res, req)

		if res.Code != http.StatusOK {
			t.Errorf("got status %d want %d", res.Code, http.StatusOK)
		}

		var got []models.Resource
		if err := json.NewDecoder(res.Body).Decode(&got); err != nil {
			t.Fatalf("Unable to parse response: %v", err)
		}

		if !reflect.DeepEqual(got, wantedResources) {
			t.Errorf("got %v want %v", got, wantedResources)
		}
	})

	t.Run("returns matched resources for a specific goal", func(t *testing.T) {
		requestBody := strings.NewReader(`{"goal": "go"}`)
		req, _ := http.NewRequest(http.MethodPost, "/api/recommend", requestBody)
		res := httptest.NewRecorder()

		server.HandleRecommend(res, req)

		if res.Code != http.StatusOK {
			t.Errorf("got status %d want %d", res.Code, http.StatusOK)
		}

		var got []models.Resource
		if err := json.NewDecoder(res.Body).Decode(&got); err != nil {
			t.Fatalf("Unable to parse response: %v", err)
		}

		if len(got) != 1 {
			t.Errorf("Expected 1 match, got %d", len(got))
		}

		if !reflect.DeepEqual(got, wantedResources) {
			t.Errorf("got %v want %v", got, wantedResources)
		}
	})

	// Verify that operational errors from the data store properly propagate
	// up to the handler and result in a 500 Internal Server Error.
	t.Run("returns 500 when the store returns an error", func(t *testing.T) {
		// Inject a fake error into our stub store
		faultyStore := &StubStore{searchErr: errors.New("AI service unavailable")}
		stubFlags := &flags.StubProvider{} // defaults are fine for this test
		faultyServer := NewEngineServer(faultyStore, stubFlags)
		t.Cleanup(faultyServer.Shutdown)

		requestBody := strings.NewReader(`{"goal": "go"}`)
		req, _ := http.NewRequest(http.MethodPost, "/api/recommend", requestBody)
		res := httptest.NewRecorder()

		faultyServer.HandleRecommend(res, req)

		// The key assertion: a server-side failure MUST return 500, not 200.
		if res.Code != http.StatusInternalServerError {
			t.Errorf("expected status 500 on store error, got %d", res.Code)
		}
	})
}

