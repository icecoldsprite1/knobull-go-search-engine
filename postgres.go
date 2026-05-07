package main

import (
	"context"
	"log"
	"strings"

	"github.com/jackc/pgx/v5/pgxpool" // The modern standard
)

// PostgresStore holds our live database connection pool
type PostgresStore struct {
	pool *pgxpool.Pool
}

// NewPostgresStore connects to Supabase using pgx
func NewPostgresStore(connectionString string) (*PostgresStore, error) {
	// context.Background() is required by pgx to manage timeout limits
	pool, err := pgxpool.New(context.Background(), connectionString)
	if err != nil {
		return nil, err
	}

	// Ping the database
	if err := pool.Ping(context.Background()); err != nil {
		return nil, err
	}

	log.Println("Successfully connected to Supabase via pgx!")
	return &PostgresStore{pool: pool}, nil
}

func (p *PostgresStore) GetResources() []Resource {
	return []Resource{}
}

func (p *PostgresStore) SearchResources(goal string) []Resource {
	var matches []Resource
	searchQuery := "%" + strings.ToLower(goal) + "%"

	// Using the pgx pool to query
	rows, err := p.pool.Query(context.Background(), `
		SELECT id, title, description, category 
		FROM resources 
		WHERE LOWER(title) LIKE $1 OR LOWER(description) LIKE $1
	`, searchQuery)

	if err != nil {
		log.Println("Database search error:", err)
		return matches
	}
	defer rows.Close()

	// pgx has a slightly different syntax for looping rows, but it's very clean
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
