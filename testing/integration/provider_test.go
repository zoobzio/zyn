package integration

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/zoobzio/zyn"
	"github.com/zoobzio/zyn/pkg/openai"
)

// skipWithoutAPIKey skips the test if OPENAI_API_KEY is not set.
func skipWithoutAPIKey(t *testing.T) {
	t.Helper()
	if os.Getenv("OPENAI_API_KEY") == "" {
		t.Skip("OPENAI_API_KEY not set, skipping provider test")
	}
}

// newTestProvider creates an OpenAI provider for testing.
// Uses gpt-4o-mini for cost efficiency.
func newTestProvider(t *testing.T) zyn.Provider {
	t.Helper()
	apiKey := os.Getenv("OPENAI_API_KEY")
	return openai.New(openai.Config{
		APIKey: apiKey,
		Model:  "gpt-4o-mini",
	})
}

func TestProvider_Binary(t *testing.T) {
	skipWithoutAPIKey(t)
	provider := newTestProvider(t)

	synapse, err := zyn.Binary("Is this a valid email address?", provider,
		zyn.WithTimeout(30*time.Second),
	)
	if err != nil {
		t.Fatalf("failed to create synapse: %v", err)
	}

	session := zyn.NewSession()
	ctx := context.Background()

	// Valid email
	result, err := synapse.Fire(ctx, session, "user@example.com")
	if err != nil {
		t.Fatalf("call failed: %v", err)
	}

	if !result {
		t.Error("expected valid email to return true")
	}

	// Verify token usage was tracked
	usage := session.LastUsage()
	if usage == nil {
		t.Error("expected usage to be tracked")
	} else {
		if usage.Total == 0 {
			t.Error("expected non-zero token usage")
		}
		t.Logf("Token usage: prompt=%d, completion=%d, total=%d",
			usage.Prompt, usage.Completion, usage.Total)
	}
}

func TestProvider_BinaryInvalid(t *testing.T) {
	skipWithoutAPIKey(t)
	provider := newTestProvider(t)

	synapse, err := zyn.Binary("Is this a valid email address?", provider,
		zyn.WithTimeout(30*time.Second),
	)
	if err != nil {
		t.Fatalf("failed to create synapse: %v", err)
	}

	session := zyn.NewSession()
	ctx := context.Background()

	// Invalid email
	result, err := synapse.Fire(ctx, session, "not-an-email")
	if err != nil {
		t.Fatalf("call failed: %v", err)
	}

	if result {
		t.Error("expected invalid email to return false")
	}
}

func TestProvider_Classification(t *testing.T) {
	skipWithoutAPIKey(t)
	provider := newTestProvider(t)

	synapse, err := zyn.Classification(
		"What type of message is this?",
		[]string{"spam", "urgent", "newsletter", "personal"},
		provider,
		zyn.WithTimeout(30*time.Second),
	)
	if err != nil {
		t.Fatalf("failed to create synapse: %v", err)
	}

	session := zyn.NewSession()
	ctx := context.Background()

	result, err := synapse.Fire(ctx, session, "URGENT: Your account will be suspended!")
	if err != nil {
		t.Fatalf("call failed: %v", err)
	}

	// Should classify as urgent or spam
	if result != "urgent" && result != "spam" {
		t.Logf("Classification result: %s (expected urgent or spam)", result)
	}

	t.Logf("Classification result: %s", result)
}

func TestProvider_Sentiment(t *testing.T) {
	skipWithoutAPIKey(t)
	provider := newTestProvider(t)

	synapse, err := zyn.Sentiment("Analyze the sentiment", provider,
		zyn.WithTimeout(30*time.Second),
	)
	if err != nil {
		t.Fatalf("failed to create synapse: %v", err)
	}

	session := zyn.NewSession()
	ctx := context.Background()

	result, err := synapse.FireWithDetails(ctx, session, "I absolutely love this product! Best purchase ever!")
	if err != nil {
		t.Fatalf("call failed: %v", err)
	}

	if result.Overall != "positive" {
		t.Errorf("expected positive sentiment, got %s", result.Overall)
	}

	t.Logf("Sentiment: %s (confidence: %.2f)", result.Overall, result.Confidence)
}

func TestProvider_Transform(t *testing.T) {
	skipWithoutAPIKey(t)
	provider := newTestProvider(t)

	synapse, err := zyn.Transform("Translate to French", provider,
		zyn.WithTimeout(30*time.Second),
	)
	if err != nil {
		t.Fatalf("failed to create synapse: %v", err)
	}

	session := zyn.NewSession()
	ctx := context.Background()

	result, err := synapse.Fire(ctx, session, "Hello, how are you?")
	if err != nil {
		t.Fatalf("call failed: %v", err)
	}

	if result == "" {
		t.Error("expected non-empty translation")
	}

	t.Logf("Translation: %s", result)
}

func TestProvider_Ranking(t *testing.T) {
	skipWithoutAPIKey(t)
	provider := newTestProvider(t)

	synapse, err := zyn.Ranking(
		"most healthy to least healthy",
		provider,
		zyn.WithTimeout(30*time.Second),
	)
	if err != nil {
		t.Fatalf("failed to create synapse: %v", err)
	}

	session := zyn.NewSession()
	ctx := context.Background()

	items := []string{"apple", "candy bar", "salad", "soda"}
	result, err := synapse.FireWithDetails(ctx, session, items)
	if err != nil {
		t.Fatalf("call failed: %v", err)
	}

	if len(result.Ranked) != 4 {
		t.Errorf("expected 4 ranked items, got %d", len(result.Ranked))
	}

	t.Logf("Ranking: %v", result.Ranked)
}

func TestProvider_SessionContext(t *testing.T) {
	skipWithoutAPIKey(t)
	provider := newTestProvider(t)

	synapse, err := zyn.Binary("Based on the previous context, is this related?", provider,
		zyn.WithTimeout(30*time.Second),
	)
	if err != nil {
		t.Fatalf("failed to create synapse: %v", err)
	}

	session := zyn.NewSession()
	ctx := context.Background()

	// First call establishes context
	firstSynapse, _ := zyn.Transform("Remember: we are discussing programming languages", provider,
		zyn.WithTimeout(30*time.Second),
	)
	_, _ = firstSynapse.Fire(ctx, session, "The topic is Python")

	// Second call should see the context
	result, err := synapse.Fire(ctx, session, "Is Java relevant to our discussion?")
	if err != nil {
		t.Fatalf("call failed: %v", err)
	}

	// Java is a programming language, so should be related
	t.Logf("Related to programming context: %v", result)
}

func TestProvider_TokenUsageAccumulation(t *testing.T) {
	skipWithoutAPIKey(t)
	provider := newTestProvider(t)

	synapse, err := zyn.Binary("Is this true?", provider,
		zyn.WithTimeout(30*time.Second),
	)
	if err != nil {
		t.Fatalf("failed to create synapse: %v", err)
	}

	session := zyn.NewSession()
	ctx := context.Background()

	var totalTokens int

	// Make multiple calls and track usage
	for i := 0; i < 3; i++ {
		_, err := synapse.Fire(ctx, session, "The sky is blue")
		if err != nil {
			t.Fatalf("call %d failed: %v", i, err)
		}

		usage := session.LastUsage()
		if usage != nil {
			totalTokens += usage.Total
			t.Logf("Call %d: %d tokens", i+1, usage.Total)
		}
	}

	t.Logf("Total tokens across 3 calls: %d", totalTokens)

	// Session should have 6 messages (3 user + 3 assistant)
	if session.Len() != 6 {
		t.Errorf("expected 6 messages, got %d", session.Len())
	}
}
