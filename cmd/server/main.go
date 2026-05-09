package main

import (
	"log"
	"net/http"
	"os"

	"github.com/joho/godotenv"

	"github.com/icecoldsprite1/knobull-go-search-engine/internal/api"
	"github.com/icecoldsprite1/knobull-go-search-engine/internal/store"
)

func main() {
	// 1. Load the .env file
	if err := godotenv.Load(); err != nil {
		log.Println("No .env file found, looking for system environment variables instead.")
	}

	// 2. Grab the connection string securely
	connStr := os.Getenv("DATABASE_URL")
	if connStr == "" {
		log.Fatal("CRITICAL: DATABASE_URL is not set in the environment")
	}

	// 3. Initialize the Postgres database
	postgresStore, err := store.NewPostgresStore(connStr)
	if err != nil {
		log.Fatal("Could not connect to database: ", err)
	}

	// 4. Inject the database into our server
	server := api.NewEngineServer(postgresStore)

	// 5. Setup the router
	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/resources", server.HandleGetResources)
	mux.HandleFunc("POST /api/recommend", server.HandleRecommend)
	// Serve static files from the "public" directory
	mux.Handle("/", http.FileServer(http.Dir("public")))

	log.Println("Knobull Engine started on :8080")
	if err := http.ListenAndServe(":8080", mux); err != nil {
		log.Fatal(err)
	}
}
