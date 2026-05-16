package store

import (
	"context"
	"testing"

	"github.com/icecoldsprite1/knobull-go-search-engine/internal/models"
)

func TestInMemoryStore_SearchResources(t *testing.T) {
	store := NewInMemoryStore()

	// Test a match on Title
	results, err := store.SearchResources(context.Background(), models.SearchRequest{Goal: "distributed"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(results) != 1 {
		t.Errorf("Expected 1 result for 'distributed', got %d", len(results))
	} else if results[0].ID != "1" {
		t.Errorf("Expected result ID 1, got %s", results[0].ID)
	}

	// Test a match on Description
	results, err = store.SearchResources(context.Background(), models.SearchRequest{Goal: "renewable"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(results) != 1 {
		t.Errorf("Expected 1 result for 'renewable', got %d", len(results))
	} else if results[0].ID != "2" {
		t.Errorf("Expected result ID 2, got %s", results[0].ID)
	}

	// Test no matches
	results, err = store.SearchResources(context.Background(), models.SearchRequest{Goal: "nonexistent"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(results) != 0 {
		t.Errorf("Expected 0 results for 'nonexistent', got %d", len(results))
	}
}

