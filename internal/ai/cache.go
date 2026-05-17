package ai

import (
	"sync"
	"time"

	"github.com/pgvector/pgvector-go"
)

// cacheEntry stores a single cached embedding alongside its creation time.
// The creation time is used to implement TTL-based expiration.
type cacheEntry struct {
	vector    pgvector.Vector
	createdAt time.Time
}

// EmbeddingCache is a thread-safe in-memory cache for embedding vectors.
//
// It uses sync.RWMutex to allow many concurrent readers (RLock) while
// ensuring exclusive access for writes (Lock). This is the same pattern
// used inside the LaunchDarkly SDK's internal flag store — a map that is
// read thousands of times per second but only written to occasionally.
type EmbeddingCache struct {
	mu      sync.RWMutex
	entries map[string]cacheEntry
	ttl     time.Duration
}

// NewEmbeddingCache creates a cache where entries expire after the given TTL.
// Without a TTL, the cache would grow forever and return stale vectors
// if the embedding model is updated upstream.
func NewEmbeddingCache(ttl time.Duration) *EmbeddingCache {
	return &EmbeddingCache{
		entries: make(map[string]cacheEntry),
		ttl:     ttl,
	}
}

// Get retrieves a cached embedding vector for the given text.
// Returns the vector and true if a valid (non-expired) entry exists.
// Returns a zero vector and false on cache miss or expiration.
//
// Uses RLock — multiple goroutines can call Get simultaneously without
// blocking each other. Only a concurrent Set call will block readers.
func (c *EmbeddingCache) Get(text string) (pgvector.Vector, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	entry, ok := c.entries[text]
	if !ok {
		return pgvector.Vector{}, false
	}

	// Check TTL — expired entries are treated as cache misses.
	if time.Since(entry.createdAt) > c.ttl {
		return pgvector.Vector{}, false
	}

	return entry.vector, true
}

// Set stores an embedding vector in the cache.
//
// Uses Lock (not RLock) — this acquires exclusive write access.
// While Set is running, all Get calls on other goroutines will block
// until the write completes. This is safe because writes are infrequent
// (only on cache misses) and extremely fast (a single map assignment).
func (c *EmbeddingCache) Set(text string, vec pgvector.Vector) {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.entries[text] = cacheEntry{
		vector:    vec,
		createdAt: time.Now(),
	}
}
