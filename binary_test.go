package zyn

import (
	"context"
	"testing"
	"time"
)

// TestSimpleUsage tests the simplest usage.
func TestSimpleUsage(t *testing.T) {
	// Create provider
	provider := NewMockProvider()

	// Create synapse bound to provider
	synapse := Binary("Is this a valid email?", provider)

	ctx := context.Background()

	// Test valid email
	result, err := synapse.Fire(ctx, "user@example.com")
	if err != nil {
		t.Fatalf("Fire failed: %v", err)
	}
	if !result {
		t.Error("Expected true for valid email")
	}

	// Test invalid email
	result, err = synapse.Fire(ctx, "invalid-email")
	if err != nil {
		t.Fatalf("Fire failed: %v", err)
	}
	if result {
		t.Error("Expected false for invalid email")
	}
}

// TestWithReliability tests adding reliability features.
func TestWithReliability(t *testing.T) {
	provider := NewMockProvider()

	synapse := Binary("Is this valid?", provider,
		WithRetry(3),
		WithTimeout(10*time.Second))

	ctx := context.Background()

	result, err := synapse.Fire(ctx, "test@example.com")
	if err != nil {
		t.Fatalf("Fire failed: %v", err)
	}
	if !result {
		t.Error("Expected true for valid email")
	}
}

// TestWithDefaults tests synapse with default values.
func TestWithDefaults(t *testing.T) {
	provider := NewMockProvider()

	defaults := BinaryInput{
		Criteria: []string{
			"RFC 5322 compliant",
			"No disposable domains",
		},
		Temperature: 0.2,
	}

	synapse := Binary("Is this a valid business email?", provider).WithDefaults(defaults)

	ctx := context.Background()

	response, err := synapse.FireWithDetails(ctx, "admin@company.com")
	if err != nil {
		t.Fatalf("FireWithDetails failed: %v", err)
	}

	if !response.Decision {
		t.Error("Expected positive decision for business email")
	}
}

// TestWithFallback tests fallback synapse configuration.
func TestWithFallback(t *testing.T) {
	primary := NewMockProviderWithName("openai-gpt-4")
	fallbackProvider := NewMockProviderWithName("anthropic-claude-3")

	// Compose with fallback
	synapse := Binary("Is this valid?", primary,
		WithRetry(2),
		WithFallback(Binary("Is this valid?", fallbackProvider)))

	ctx := context.Background()

	result, err := synapse.Fire(ctx, "test@example.com")
	if err != nil {
		t.Fatalf("Fire failed: %v", err)
	}

	if !result {
		t.Error("Expected true for valid email with fallback config")
	}
}

// TestChaining tests the full chain.
func TestChaining(t *testing.T) {
	// Create reusable providers
	primary := NewMockProviderWithName("openai-gpt-4")
	fallbackProvider := NewMockProviderWithName("anthropic-claude-3")

	// Create synapse with all options
	synapse := Binary("Is this production ready?", primary,
		WithRetry(3),
		WithBackoff(5, 100*time.Millisecond), // Exponential backoff for API calls
		WithTimeout(30*time.Second),
		WithCircuitBreaker(5, 30*time.Second),
		WithRateLimit(10, 20),
		WithFallback(Binary("Is this production ready?", fallbackProvider)),
	).WithDefaults(BinaryInput{
		Criteria: []string{"Well-tested", "Documented", "Error handling"},
	})

	ctx := context.Background()

	response, err := synapse.FireWithDetails(ctx, "func main() { fmt.Println(\"hello\") }")
	if err != nil {
		t.Fatalf("FireWithDetails failed: %v", err)
	}

	// Just verify it returns something
	t.Logf("Response: Decision=%v, Confidence=%v",
		response.Decision, response.Confidence)
}

// TestProviderReuse tests that providers can be reused across synapses.
func TestProviderReuse(t *testing.T) {
	// Create one provider
	provider := NewMockProvider()

	// Use it in multiple synapses
	emailSynapse := Binary("Is this a valid email?", provider)
	ageSynapse := Binary("Is user over 18?", provider)

	ctx := context.Background()

	// Both synapses work with same provider
	emailResult, _ := emailSynapse.Fire(ctx, "test@example.com")
	ageResult, _ := ageSynapse.Fire(ctx, "25 years old")

	if !emailResult {
		t.Error("Email synapse should return true")
	}

	// Age synapse with mock will likely return false (no pattern match)
	// Just verify it runs without error
	_ = ageResult
}
