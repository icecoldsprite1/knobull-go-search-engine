package ai

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"time"

	"github.com/pgvector/pgvector-go"
)

var APIURL = "https://router.huggingface.co/hf-inference/models/sentence-transformers/all-MiniLM-L6-v2/pipeline/feature-extraction"

// embeddingCache stores recently computed vectors to avoid redundant API calls.
// Entries expire after 1 hour to prevent stale data if the model updates.
var embeddingCache = NewEmbeddingCache(1 * time.Hour)

// GenerateEmbedding returns a vector representation of the given text.
// It checks the in-memory cache first (microsecond RLock read) and only
// calls the HuggingFace API on a cache miss (~300ms network call).
func GenerateEmbedding(ctx context.Context, text string) (pgvector.Vector, error) {
	// 1. Check cache (fast path — RLock, concurrent readers allowed)
	if vec, ok := embeddingCache.Get(text); ok {
		slog.Info("embedding cache hit", "text", text)
		return vec, nil
	}

	// 2. Cache miss — call HuggingFace (slow path)
	vec, err := callHuggingFace(ctx, text)
	if err != nil {
		return pgvector.Vector{}, err
	}

	// 3. Store in cache for next time (Lock, exclusive write)
	embeddingCache.Set(text, vec)
	slog.Info("embedding cached", "text", text)
	return vec, nil
}

// callHuggingFace performs the actual HTTP call to the HuggingFace API.
// Separated from GenerateEmbedding so the caching logic stays clean.
func callHuggingFace(ctx context.Context, text string) (pgvector.Vector, error) {
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

