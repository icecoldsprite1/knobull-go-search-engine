package store

import (
	"context"
	"log"

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
		return nil, err
	}

	// This teaches the database connection how to read pgvector math arrays
	config.AfterConnect = func(ctx context.Context, conn *pgx.Conn) error {
		return pgxvec.RegisterTypes(ctx, conn)
	}

	pool, err := pgxpool.NewWithConfig(context.Background(), config)
	if err != nil {
		return nil, err
	}

	if err := pool.Ping(context.Background()); err != nil {
		return nil, err
	}

	log.Println("Successfully connected to Postgres and registered pgvector types!")
	return &PostgresStore{pool: pool}, nil
}

func (p *PostgresStore) GetResources() []models.Resource {
	return []models.Resource{} // We will implement this later if needed for a frontend dashboard
}

func (p *PostgresStore) SearchResources(ctx context.Context, req models.SearchRequest) []models.Resource {
	var matches []models.Resource

	// 1. Call the Hugging Face API to turn the user's career goal into math
	userVector, err := ai.GenerateEmbedding(ctx, req.Goal)
	if err != nil {
		log.Println("AI API Error:", err)
		return matches
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
		log.Println("Database search error:", err)
		return matches
	}
	defer rows.Close()

	for rows.Next() {
		var r models.Resource
		if err := rows.Scan(&r.ID, &r.Title, &r.Description, &r.Category, &r.Type, &r.Link, &r.Content); err != nil {
			log.Println("Row scan error:", err)
			continue
		}
		matches = append(matches, r)
	}

	return matches
}

func (p *PostgresStore) LogSearch(ctx context.Context, req models.SearchRequest, resultsCount int) error {
	_, err := p.pool.Exec(ctx, `
		INSERT INTO search_logs (goal, category_filter, type_filter, results_count)
		VALUES ($1, $2, $3, $4)
	`, req.Goal, req.Category, req.Type, resultsCount)
	return err
}
