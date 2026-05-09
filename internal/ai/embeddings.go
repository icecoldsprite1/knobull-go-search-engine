package ai

import (
	"bytes"
	"encoding/json"
	"io"
	"log"
	"net/http"
	"os"

	"github.com/pgvector/pgvector-go"
)

var APIURL = "https://router.huggingface.co/hf-inference/models/sentence-transformers/all-MiniLM-L6-v2/pipeline/feature-extraction"

// GenerateEmbedding calls the Hugging Face API to turn text into an embedding vector.
func GenerateEmbedding(text string) (pgvector.Vector, error) {
	hfToken := os.Getenv("HUGGINGFACE_TOKEN")

	reqBody, _ := json.Marshal(map[string]string{"inputs": text})

	req, _ := http.NewRequest("POST", APIURL, bytes.NewBuffer(reqBody))
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
