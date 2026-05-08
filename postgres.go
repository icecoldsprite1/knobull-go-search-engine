package main

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"log"
	"net/http"
	"os"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/pgvector/pgvector-go"
	pgxvec "github.com/pgvector/pgvector-go/pgx"
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

func (p *PostgresStore) GetResources() []Resource {
	return []Resource{} // We will implement this later if needed for a frontend dashboard
}

// 🧠 THE REAL AI BRAIN (For Live User Searches)
func generateEmbedding(text string) (pgvector.Vector, error) {
	hfToken := os.Getenv("HUGGINGFACE_TOKEN")

	// The exact 2026 Router URL you found in the docs, but pointing to your specific model
	url := "https://router.huggingface.co/hf-inference/models/sentence-transformers/all-MiniLM-L6-v2/pipeline/feature-extraction"

	reqBody, _ := json.Marshal(map[string]string{"inputs": text})

	req, _ := http.NewRequest("POST", url, bytes.NewBuffer(reqBody))
	req.Header.Set("Authorization", "Bearer "+hfToken)
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return pgvector.Vector{}, err
	}
	defer resp.Body.Close()

	bodyBytes, _ := io.ReadAll(resp.Body)

	var embedding []float32
	if err := json.Unmarshal(bodyBytes, &embedding); err != nil {
		log.Println("Failed to parse AI response. Body:", string(bodyBytes))
		return pgvector.Vector{}, err
	}

	return pgvector.NewVector(embedding), nil
}

func (p *PostgresStore) SearchResources(goal string) []Resource {
	var matches []Resource

	// 1. Call the Hugging Face API to turn the user's career goal into math
	userVector, err := generateEmbedding(goal)
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
		var r Resource
		if err := rows.Scan(&r.ID, &r.Title, &r.Description, &r.Category); err != nil {
			log.Println("Row scan error:", err)
			continue
		}
		matches = append(matches, r)
	}

	return matches
}
