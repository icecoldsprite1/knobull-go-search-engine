package flags

import (
	"fmt"
	"log"
	"time"

	"github.com/launchdarkly/go-sdk-common/v3/ldcontext"
	ld "github.com/launchdarkly/go-server-sdk/v7"
)

// Provider is the interface your server uses to evaluate feature flags.
// By depending on this interface instead of the LD SDK directly,
// tests can inject a stub without needing a real LaunchDarkly connection.
type Provider interface {
	BoolVariation(key string, defaultVal bool) bool
	IntVariation(key string, defaultVal int) int
	Close()
}

// LaunchDarklyProvider wraps the real LD SDK client.
type LaunchDarklyProvider struct {
	client *ld.LDClient
	// context represents "who is asking?" for flag targeting.
	// In this server-side use case, we use a single server context.
	// In production, you'd build this per-request from user identity.
	context ldcontext.Context
}

// NewLaunchDarklyProvider initializes the SDK and blocks until it has
// downloaded the full flag ruleset (up to 5 seconds).
//
// If the SDK key is wrong or LD is unreachable, it returns an error —
// rather than silently using defaults, which could mask misconfiguration.
func NewLaunchDarklyProvider(sdkKey string) (*LaunchDarklyProvider, error) {
	client, err := ld.MakeClient(sdkKey, 5*time.Second)
	if err != nil {
		return nil, fmt.Errorf("initializing LaunchDarkly client: %w", err)
	}

	if !client.Initialized() {
		return nil, fmt.Errorf("LaunchDarkly client failed to initialize within timeout")
	}

	log.Println("LaunchDarkly SDK initialized and flag ruleset synced.")

	return &LaunchDarklyProvider{
		client: client,
		// "server" context: a single, stable context for all server-side evaluations.
		// No PII — we're not identifying individual users here.
		context: ldcontext.New("knobull-search-server"),
	}, nil
}

func (p *LaunchDarklyProvider) BoolVariation(key string, defaultVal bool) bool {
	val, err := p.client.BoolVariation(key, p.context, defaultVal)
	if err != nil {
		log.Printf("flag evaluation error for %q: %v — using default %v", key, err, defaultVal)
		return defaultVal
	}
	return val
}

func (p *LaunchDarklyProvider) IntVariation(key string, defaultVal int) int {
	val, err := p.client.IntVariation(key, p.context, defaultVal)
	if err != nil {
		log.Printf("flag evaluation error for %q: %v — using default %d", key, err, defaultVal)
		return defaultVal
    }
	return val
}

// Close flushes pending analytics events and closes the SDK connection.
// This must be called during graceful shutdown — same as closing a DB connection.
func (p *LaunchDarklyProvider) Close() {
	p.client.Close()
}
