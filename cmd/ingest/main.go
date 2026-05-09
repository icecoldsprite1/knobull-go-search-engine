package main

import (
	"context"
	"encoding/csv"
	"fmt"
	"log"
	"os"
	"sync" // [cite: 13] Standard library for synchronization primitives like WaitGroups

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/joho/godotenv"
	pgxvec "github.com/pgvector/pgvector-go/pgx"

	"github.com/icecoldsprite1/knobull-go-search-engine/internal/ai"
)

// [cite: 12] A struct to represent a single row of data, making it easy to pass through channels
type Job struct {
	ID          string
	Title       string
	Description string
	Category    string
	Type        string
	Link        string
	Content     string
}

func main() {
	if err := godotenv.Load(); err != nil {
		log.Println("No .env file found")
	}

	connStr := os.Getenv("DATABASE_URL")
	if connStr == "" {
		log.Fatal("CRITICAL: DATABASE_URL is not set")
	}

	// Configure the database connection pool
	config, err := pgxpool.ParseConfig(connStr)
	if err != nil {
		log.Fatal(err)
	}

	// Register the pgvector type so the driver understands the 'vector' column
	config.AfterConnect = func(ctx context.Context, conn *pgx.Conn) error {
		return pgxvec.RegisterTypes(ctx, conn)
	}

	//  pgxpool is thread-safe, meaning multiple goroutines can use it at once
	pool, err := pgxpool.NewWithConfig(context.Background(), config)
	if err != nil {
		log.Fatal(err)
	}
	defer pool.Close()

	// Reset the database table for a fresh ingestion
	_, err = pool.Exec(context.Background(), `
        CREATE EXTENSION IF NOT EXISTS vector;
        DROP TABLE IF EXISTS resources;
        CREATE TABLE resources (
            id TEXT PRIMARY KEY,
            title TEXT,
            description TEXT,
            category TEXT,
            type TEXT,
            link TEXT,
            content TEXT,
            embedding vector(384)
        );
    `)
	if err != nil {
		log.Fatal("Failed to drop/create table: ", err)
	}

	// [cite: 43] Open and read the CSV file into memory
	f, err := os.Open("data/sample_courses.csv")
	if err != nil {
		log.Fatal(err)
	}
	defer f.Close()

	csvReader := csv.NewReader(f)
	records, err := csvReader.ReadAll() // Reads all rows at once
	if err != nil {
		log.Fatal(err)
	}

	// --- CONCURRENCY SETUP ---

	numWorkers := 3 // [cite: 9] We will run 3 tasks in parallel

	// [cite: 9] Create a buffered channel to hold Job structs.
	// Buffering it to the length of records prevents the main thread from blocking.
	jobs := make(chan Job, len(records))

	// [cite: 10] A WaitGroup acts as a counter to track how many workers are still running
	var wg sync.WaitGroup

	fmt.Printf("Starting %d workers...\n", numWorkers)

	// 1. Start the worker goroutines
	for w := 1; w <= numWorkers; w++ {
		wg.Add(1)                     // Increment the WaitGroup counter for each worker started
		go worker(w, jobs, pool, &wg) // Launch worker in its own lightweight thread
	}

	// 2. Feed the jobs into the channel
	for i, row := range records {
		if i == 0 {
			continue // Skip the CSV header row
		}

		// Push a Job struct into the channel for workers to pick up
		jobs <- Job{
			ID:          row[0],
			Title:       row[1],
			Description: row[2],
			Category:    row[3],
			Type:        row[4],
			Link:        row[5],
			Content:     row[6],
		}
	}

	// [cite: 11] Close the channel to tell workers "no more data is coming"
	close(jobs)

	// 3. Block main until the WaitGroup counter returns to zero
	wg.Wait()

	fmt.Println("Ingestion complete! All resources processed concurrently.")
}

// [cite: 15] worker logic is isolated for concurrent execution
func worker(id int, jobs <-chan Job, pool *pgxpool.Pool, wg *sync.WaitGroup) {
	// [cite: 10] Ensure we tell the WaitGroup we are done when this function exits
	defer wg.Done()

	// This loop pulls jobs from the channel one by one until the channel is closed
	for j := range jobs {
		fmt.Printf("Worker %d starting job %s: %s\n", id, j.ID, j.Title)

		// Combine fields for a rich semantic context [cite: 119, 120]
		textToEmbed := fmt.Sprintf("%s. %s %s", j.Title, j.Description, j.Content)

		// [cite: 56, 74] Call Hugging Face API to get the vector
		embedding, err := ai.GenerateEmbedding(textToEmbed)
		if err != nil {
			log.Printf("Worker %d failed to generate embedding for ID %s: %v\n", id, j.ID, err)
			continue // Move to the next job if embedding fails
		}

		//  Insert using parameterized queries to prevent SQL injection
		_, err = pool.Exec(context.Background(), `
            INSERT INTO resources (id, title, description, category, type, link, content, embedding)
            VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
        `, j.ID, j.Title, j.Description, j.Category, j.Type, j.Link, j.Content, embedding)

		if err != nil {
			log.Printf("Worker %d failed to insert ID %s: %v\n", id, j.ID, err)
		} else {
			fmt.Printf("Worker %d successfully inserted: %s\n", id, j.Title)
		}
	}
	fmt.Printf("Worker %d has finished and is exiting.\n", id)
}
