package models

type Resource struct {
	ID          string   `json:"id"`
	Title       string   `json:"title"`
	Description string   `json:"description"`
	Category    string   `json:"category"`
	Keywords    []string `json:"keywords"`
}

type SearchRequest struct {
	Goal string `json:"goal"`
}

// ResourceStore defines how our server interacts with ANY database
type ResourceStore interface {
	GetResources() []Resource
	SearchResources(goal string) []Resource
}
