package models

import "context"

type Resource struct {
	ID          string   `json:"id"`
	Title       string   `json:"title"`
	Description string   `json:"description"`
	Category    string   `json:"category"`
	Keywords    []string `json:"keywords"`
	Type        string   `json:"type"`    // e.g., "external_course", "internal_article"
	Link        string   `json:"link"`    // URL for external resources
	Content     string   `json:"content"` // Full text for internal reading
}

type SearchRequest struct {
	Goal     string `json:"goal"`
	Category string `json:"category"`
	Type     string `json:"type"`
}

// ResourceStore defines how our server interacts with ANY database
type ResourceStore interface {
	GetResources() []Resource
	SearchResources(ctx context.Context, req SearchRequest) []Resource
	LogSearch(ctx context.Context, req SearchRequest, resultsCount int) error
}
