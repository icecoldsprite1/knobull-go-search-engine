package ai

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"time"

	"github.com/pgvector/pgvector-go"
)

var APIURL = "https://router.huggingface.co/hf-inference/models/sentence-transformers/all-MiniLM-L6-v2/pipeline/feature-extraction"

// GenerateEmbedding calls the Hugging Face API to turn text into a vector.
// It uses error wrapping (fmt.Errorf with %w) to preserve error context for callers,
// and enforces a strict timeout to prevent hung goroutines during API degradation.
func GenerateEmbedding(ctx context.Context, text string) (pgvector.Vector, error) {
	hfToken := os.Getenv("HUGGINGFACE_TOKEN")

	reqBody, err := json.Marshal(map[string]string{"inputs": text})
	if err != nil {
		return pgvector.Vector{}, fmt.Errorf("marshaling embedding request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", APIURL, bytes.NewBuffer(reqBody))
	if err != nil {
		return pgvector.Vector{}, fmt.Errorf("creating embedding request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+hfToken)
	req.Header.Set("Content-Type", "application/json")

	// Enforce a timeout on outbound HTTP calls to prevent goroutine leaks
	// if the external API is slow or unresponsive.
	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return pgvector.Vector{}, fmt.Errorf("calling hugging face API: %w", err)
	}
	defer resp.Body.Close()

	// Validate the HTTP status code. External APIs may return non-JSON error pages
	// (e.g., 503 Service Unavailable or 429 Too Many Requests).
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return pgvector.Vector{}, fmt.Errorf("hugging face API returned %d: %s", resp.StatusCode, string(body))
	}

	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return pgvector.Vector{}, fmt.Errorf("reading embedding response body: %w", err)
	}

	var embedding []float32
	if err := json.Unmarshal(bodyBytes, &embedding); err != nil {
		return pgvector.Vector{}, fmt.Errorf("parsing embedding response (body: %s): %w", string(bodyBytes), err)
	}

	return pgvector.NewVector(embedding), nil
}

