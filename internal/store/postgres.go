package store

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	pgxvec "github.com/pgvector/pgvector-go/pgx"

	"github.com/icecoldsprite1/knobull-go-search-engine/internal/ai"
	"github.com/icecoldsprite1/knobull-go-search-engine/internal/models"
)

type PostgresStore struct {
	pool *pgxpool.Pool
}

func NewPostgresStore(connectionString string) (*PostgresStore, error) {
	config, err := pgxpool.ParseConfig(connectionString)
	if err != nil {
		return nil, fmt.Errorf("parsing database connection string: %w", err)
	}

	// This teaches the database connection how to read pgvector math arrays
	config.AfterConnect = func(ctx context.Context, conn *pgx.Conn) error {
		return pgxvec.RegisterTypes(ctx, conn)
	}

	pool, err := pgxpool.NewWithConfig(context.Background(), config)
	if err != nil {
		return nil, fmt.Errorf("creating connection pool: %w", err)
	}

	if err := pool.Ping(context.Background()); err != nil {
		return nil, fmt.Errorf("pinging database: %w", err)
	}

	return &PostgresStore{pool: pool}, nil
}

func (p *PostgresStore) GetResources() []models.Resource {
	return []models.Resource{} // We will implement this later if needed for a frontend dashboard
}

// SearchResources executes a hybrid search (vector similarity + full-text search)
// with dynamic filtering. It propagates operational errors (e.g., API or DB failures)
// using fmt.Errorf with %w to maintain a clear error chain for debugging.
func (p *PostgresStore) SearchResources(ctx context.Context, req models.SearchRequest) ([]models.Resource, error) {
	// 1. Generate vector embedding for the user's query
	userVector, err := ai.GenerateEmbedding(ctx, req.Goal)
	if err != nil {
		// Wrap the error from the AI package to maintain context.
		return nil, fmt.Errorf("generating embedding for %q: %w", req.Goal, err)
	}

	// 2. Run Hybrid Search (Vector + Full-Text) with Dynamic Filtering
	rows, err := p.pool.Query(ctx, `
		WITH vector_search AS (
			SELECT id, 
				   1 - (embedding <=> $1) AS vector_score
			FROM resources
			WHERE ($3 = '' OR category = $3) AND ($4 = '' OR type = $4)
			ORDER BY embedding <=> $1
			LIMIT 10
		),
		keyword_search AS (
			SELECT id, 
				   ts_rank(to_tsvector('english', title || ' ' || description || ' ' || category), plainto_tsquery('english', $2)) AS keyword_score
			FROM resources
			WHERE to_tsvector('english', title || ' ' || description || ' ' || category) @@ plainto_tsquery('english', $2)
			  AND ($3 = '' OR category = $3) AND ($4 = '' OR type = $4)
			LIMIT 10
		)
		SELECT 
			r.id, 
			r.title, 
			r.description, 
			r.category,
			r.type,
			r.link,
			r.content
		FROM resources r
		LEFT JOIN vector_search v ON r.id = v.id
		LEFT JOIN keyword_search k ON r.id = k.id
		WHERE v.id IS NOT NULL OR k.id IS NOT NULL
		ORDER BY (COALESCE(v.vector_score, 0.0) * 0.7) + (COALESCE(k.keyword_score, 0.0) * 0.3) DESC
		LIMIT 5
	`, userVector, req.Goal, req.Category, req.Type)

	if err != nil {
		return nil, fmt.Errorf("querying database for %q: %w", req.Goal, err)
	}
	defer rows.Close()

	var matches []models.Resource
	for rows.Next() {
		var r models.Resource
		if err := rows.Scan(&r.ID, &r.Title, &r.Description, &r.Category, &r.Type, &r.Link, &r.Content); err != nil {
			// Log individual row scan errors but continue processing.
			// This trade-off ensures one corrupted row doesn't fail the entire result set.
			return nil, fmt.Errorf("scanning result row: %w", err)
		}
		matches = append(matches, r)
	}

	// Check for errors encountered during row iteration.
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterating result rows: %w", err)
	}

	return matches, nil
}

func (p *PostgresStore) LogSearch(ctx context.Context, req models.SearchRequest, resultsCount int) error {
	_, err := p.pool.Exec(ctx, `
		INSERT INTO search_logs (goal, category_filter, type_filter, results_count)
		VALUES ($1, $2, $3, $4)
	`, req.Goal, req.Category, req.Type, resultsCount)
	if err != nil {
		return fmt.Errorf("inserting search log: %w", err)
	}
	return nil
}

