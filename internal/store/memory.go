package store

import (
	"context"
	"strings"

	"github.com/icecoldsprite1/knobull-go-search-engine/internal/models"
)

// InMemoryStore is our temporary database
type InMemoryStore struct {
	resources []models.Resource
}

// NewInMemoryStore initializes our fake data
func NewInMemoryStore() *InMemoryStore {
	return &InMemoryStore{
		resources: []models.Resource{
			{ID: "1", Title: "Distributed Systems in Go", Description: "Scalable backends", Category: "CS", Type: "external_course", Link: "https://example.com/go", Content: ""},
			{ID: "2", Title: "Sustainable Energy", Description: "Renewable tech", Category: "EnvSci", Type: "internal_article", Link: "", Content: "This is a full article about renewable tech."},
		},
	}
}

func (i *InMemoryStore) GetResources() []models.Resource {
	return i.resources
}

func (i *InMemoryStore) SearchResources(ctx context.Context, req models.SearchRequest, hybridEnabled bool, limit int) ([]models.Resource, error) {
	var matches []models.Resource
	userGoal := strings.ToLower(req.Goal)

	for _, resource := range i.resources {
		// Category Filter
		if req.Category != "" && resource.Category != req.Category {
			continue
		}
		// Type Filter
		if req.Type != "" && resource.Type != req.Type {
			continue
		}

		if strings.Contains(strings.ToLower(resource.Title), userGoal) || strings.Contains(strings.ToLower(resource.Description), userGoal) {
			matches = append(matches, resource)
			if len(matches) == limit {
				break
			}
		}
	}
	// InMemoryStore never fails — it has no external dependencies.
	// We return nil for the error to satisfy the ResourceStore interface.
	return matches, nil
}

func (i *InMemoryStore) LogSearch(ctx context.Context, req models.SearchRequest, resultsCount int) error {
	// Do nothing for in-memory store
	return nil
}
