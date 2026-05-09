package store

import (
	"testing"
)

func TestInMemoryStore_SearchResources(t *testing.T) {
	store := NewInMemoryStore()

	// Test a match on Title
	results := store.SearchResources("distributed")
	if len(results) != 1 {
		t.Errorf("Expected 1 result for 'distributed', got %d", len(results))
	} else if results[0].ID != "1" {
		t.Errorf("Expected result ID 1, got %s", results[0].ID)
	}

	// Test a match on Description
	results = store.SearchResources("renewable")
	if len(results) != 1 {
		t.Errorf("Expected 1 result for 'renewable', got %d", len(results))
	} else if results[0].ID != "2" {
		t.Errorf("Expected result ID 2, got %s", results[0].ID)
	}

	// Test no matches
	results = store.SearchResources("nonexistent")
	if len(results) != 0 {
		t.Errorf("Expected 0 results for 'nonexistent', got %d", len(results))
	}
}
