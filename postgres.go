package main

import (
	"context"
	"log"
	"strings"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/pgvector/pgvector-go"
	pgxvec "github.com/pgvector/pgvector-go/pgx" // The specific pgx translator
)

type PostgresStore struct {
	pool *pgxpool.Pool
}

func NewPostgresStore(connectionString string) (*PostgresStore, error) {
	// 1. Parse the connection string into a config object
	config, err := pgxpool.ParseConfig(connectionString)
	if err != nil {
		return nil, err
	}

	// 2. The crucial step from the docs!
	// Every time the pool creates a new database connection, teach it how to read Vectors.
	config.AfterConnect = func(ctx context.Context, conn *pgx.Conn) error {
		return pgxvec.RegisterTypes(ctx, conn)
	}

	// 3. Create the pool using our custom config
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

func (p *PostgresStore) GetResources() []Resource {
	return []Resource{}
}

// 🧠 THE MOCK AI BRAIN
// Later, we swap this to call HuggingFace/Transformers.js.
func generateEmbedding(text string) pgvector.Vector {
	vec := make([]float32, 384)

	text = strings.ToLower(text)
	if strings.Contains(text, "go") || strings.Contains(text, "backend") {
		vec[0] = 1.0
	} else if strings.Contains(text, "energy") || strings.Contains(text, "solar") {
		vec[1] = 1.0
	}

	return pgvector.NewVector(vec)
}

func (p *PostgresStore) SearchResources(goal string) []Resource {
	var matches []Resource

	// 1. Convert the user's messy sentence into clean AI math
	userVector := generateEmbedding(goal)

	// 2. Run the Semantic Math Query (<=> is Cosine Distance)
	// We want the lowest distance (closest meaning)
	rows, err := p.pool.Query(context.Background(), `
		SELECT id, title, description, category 
		FROM resources 
		ORDER BY embedding <=> $1 
		LIMIT 2
	`, userVector)

	if err != nil {
		log.Println("Database search error:", err)
		return matches
	}
	defer rows.Close()

	for rows.Next() {
		var r Resource
		if err := rows.Scan(&r.ID, &r.Title, &r.Description, &r.Category); err != nil {
			log.Println("Row scan error:", err)
			continue
		}
		matches = append(matches, r)
	}

	return matches
}
