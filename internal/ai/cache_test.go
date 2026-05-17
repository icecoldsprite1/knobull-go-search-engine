package ai

import (
	"sync"
	"testing"
	"time"

	"github.com/pgvector/pgvector-go"
)

func TestEmbeddingCache(t *testing.T) {
	tests := []struct {
		name      string
		setup     func(c *EmbeddingCache)
		key       string
		wantHit   bool
		wantSlice []float32
	}{
		{
			name:    "cache miss on empty cache",
			setup:   func(c *EmbeddingCache) {},
			key:     "hello",
			wantHit: false,
		},
		{
			name: "cache hit after Set",
			setup: func(c *EmbeddingCache) {
				c.Set("hello", pgvector.NewVector([]float32{0.1, 0.2, 0.3}))
			},
			key:       "hello",
			wantHit:   true,
			wantSlice: []float32{0.1, 0.2, 0.3},
		},
		{
			name: "cache miss for different key",
			setup: func(c *EmbeddingCache) {
				c.Set("hello", pgvector.NewVector([]float32{0.1, 0.2, 0.3}))
			},
			key:     "world",
			wantHit: false,
		},
		{
			name: "expired entry returns miss",
			setup: func(c *EmbeddingCache) {
				c.Set("hello", pgvector.NewVector([]float32{0.1, 0.2, 0.3}))
				// Manually backdate the entry so it appears expired
				c.mu.Lock()
				e := c.entries["hello"]
				e.createdAt = time.Now().Add(-2 * time.Hour)
				c.entries["hello"] = e
				c.mu.Unlock()
			},
			key:     "hello",
			wantHit: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cache := NewEmbeddingCache(1 * time.Hour)
			tt.setup(cache)

			vec, hit := cache.Get(tt.key)

			if hit != tt.wantHit {
				t.Errorf("Get(%q) hit = %v, want %v", tt.key, hit, tt.wantHit)
			}

			if tt.wantHit && tt.wantSlice != nil {
				got := vec.Slice()
				if len(got) != len(tt.wantSlice) {
					t.Fatalf("Get(%q) returned %d floats, want %d", tt.key, len(got), len(tt.wantSlice))
				}
				for i := range got {
					if got[i] != tt.wantSlice[i] {
						t.Errorf("Get(%q)[%d] = %f, want %f", tt.key, i, got[i], tt.wantSlice[i])
					}
				}
			}
		})
	}
}

// TestEmbeddingCacheConcurrency verifies that concurrent reads and writes
// do not race. This test is only meaningful when run with `go test -race`.
func TestEmbeddingCacheConcurrency(t *testing.T) {
	cache := NewEmbeddingCache(1 * time.Hour)
	vec := pgvector.NewVector([]float32{0.1, 0.2, 0.3})

	var wg sync.WaitGroup

	// Spawn 50 writers
	for i := 0; i < 50; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			cache.Set("concurrent-key", vec)
		}()
	}

	// Spawn 50 readers
	for i := 0; i < 50; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			cache.Get("concurrent-key")
		}()
	}

	wg.Wait()
}
