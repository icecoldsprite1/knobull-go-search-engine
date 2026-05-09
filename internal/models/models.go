package models

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
	Goal string `json:"goal"`
}

// ResourceStore defines how our server interacts with ANY database
type ResourceStore interface {
	GetResources() []Resource
	SearchResources(goal string) []Resource
}
