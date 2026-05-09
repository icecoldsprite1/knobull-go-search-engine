package store

import (
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

func (i *InMemoryStore) SearchResources(goal string) []models.Resource {
	var matches []models.Resource
	userGoal := strings.ToLower(goal)

	for _, resource := range i.resources {
		if strings.Contains(strings.ToLower(resource.Title), userGoal) || strings.Contains(strings.ToLower(resource.Description), userGoal) {
			matches = append(matches, resource)
		}
	}
	return matches
}
