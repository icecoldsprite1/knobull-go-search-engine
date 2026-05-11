package ai

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"reflect"
	"testing"
)

func TestGenerateEmbedding(t *testing.T) {
	// 1. Create a fake HTTP server that acts like Hugging Face
	fakeServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify the request method
		if r.Method != "POST" {
			t.Errorf("Expected POST request, got %s", r.Method)
		}

		// Verify the authorization header
		if r.Header.Get("Authorization") != "Bearer test-token" {
			t.Errorf("Expected correct Authorization header")
		}

		// Return a fake vector response
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode([]float32{0.1, 0.2, 0.3})
	}))
	defer fakeServer.Close()

	// 2. Setup environment variables and override APIURL
	os.Setenv("HUGGINGFACE_TOKEN", "test-token")
	defer os.Unsetenv("HUGGINGFACE_TOKEN")

	originalURL := APIURL
	APIURL = fakeServer.URL
	defer func() { APIURL = originalURL }()

	// 3. Call the function
	vector, err := GenerateEmbedding(context.Background(), "hello world")

	// 4. Assert results
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	expected := []float32{0.1, 0.2, 0.3}
	if !reflect.DeepEqual(vector.Slice(), expected) {
		t.Errorf("Expected %v, got %v", expected, vector.Slice())
	}
}
