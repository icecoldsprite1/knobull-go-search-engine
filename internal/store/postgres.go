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

func (p *PostgresStore) SearchResources(goal string) []models.Resource {
	var matches []models.Resource

	// 1. Call the Hugging Face API to turn the user's career goal into math
	userVector, err := ai.GenerateEmbedding(goal)
	if err != nil {
		log.Println("AI API Error:", err)
		return matches
	}

	// 2. Compare the user's math to the math you seeded in the database
	rows, err := p.pool.Query(context.Background(), `
		SELECT id, title, description, category 
		FROM resources 
		WHERE embedding <=> $1 < 0.65  -- The Threshold!
		ORDER BY embedding <=> $1 
		LIMIT 2
	`, userVector)

	if err != nil {
		log.Println("Database search error:", err)
		return matches
	}
	defer rows.Close()

	for rows.Next() {
		var r models.Resource
		if err := rows.Scan(&r.ID, &r.Title, &r.Description, &r.Category); err != nil {
			log.Println("Row scan error:", err)
			continue
		}
		matches = append(matches, r)
	}

	return matches
}
