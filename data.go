package main

import "strings"

// InMemoryStore is our temporary database
type InMemoryStore struct {
	resources []Resource
}

// NewInMemoryStore initializes our fake data
func NewInMemoryStore() *InMemoryStore {
	return &InMemoryStore{
		resources: []Resource{
			{ID: "1", Title: "Distributed Systems in Go", Description: "Scalable backends", Category: "CS", Keywords: []string{"Go", "Backend", "Systems"}},
			{ID: "2", Title: "Sustainable Energy", Description: "Renewable tech", Category: "EnvSci", Keywords: []string{"Energy", "Green", "Environment"}},
		},
	}
}

func (i *InMemoryStore) GetResources() []Resource {
	return i.resources
}

func (i *InMemoryStore) SearchResources(goal string) []Resource {
	var matches []Resource
	userGoal := strings.ToLower(goal)

	for _, resource := range i.resources {
		for _, keyword := range resource.Keywords {
			if strings.Contains(userGoal, strings.ToLower(keyword)) {
				matches = append(matches, resource)
				break
			}
		}
	}
	return matches
}
