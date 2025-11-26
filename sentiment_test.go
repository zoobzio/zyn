package zyn

import (
	"context"
	"testing"
	"time"
)

func TestNewSentiment(t *testing.T) {
	t.Run("simple", func(t *testing.T) {
		provider := NewMockProvider()
		synapse, err := NewSentiment("basic sentiment", provider)
		if err != nil {
			t.Fatalf("failed to create synapse: %v", err)
		}

		if synapse == nil {
			t.Fatal("Expected synapse to be created")
		}
		if synapse.analysisType != "basic sentiment" {
			t.Errorf("Expected analysisType='basic sentiment', got '%s'", synapse.analysisType)
		}
	})

	t.Run("reliability", func(t *testing.T) {
		provider := NewMockProvider()
		synapse, err := NewSentiment("basic sentiment", provider,
			WithRetry(3),
			WithTimeout(10*time.Second))
		if err != nil {
			t.Fatalf("failed to create synapse: %v", err)
		}

		if synapse == nil {
			t.Fatal("Expected synapse with reliability options to be created")
		}
	})

	t.Run("chaining", func(t *testing.T) {
		primary := NewMockProviderWithName("primary")
		fallback := NewMockProviderWithName("fallback")
		fallbackSynapse, err := NewSentiment("basic sentiment", fallback)
		if err != nil {
			t.Fatalf("failed to create fallback synapse: %v", err)
		}

		synapse, err := NewSentiment("basic sentiment", primary,
			WithFallback(fallbackSynapse))
		if err != nil {
			t.Fatalf("failed to create synapse: %v", err)
		}

		if synapse == nil {
			t.Fatal("Expected synapse with fallback to be created")
		}
	})
}

func TestSentimentSynapse_GetPipeline(t *testing.T) {
	t.Run("simple", func(t *testing.T) {
		provider := NewMockProvider()
		synapse, err := NewSentiment("test", provider)
		if err != nil {
			t.Fatalf("failed to create synapse: %v", err)
		}

		pipeline := synapse.GetPipeline()
		if pipeline == nil {
			t.Error("GetPipeline returned nil")
		}
	})

	t.Run("reliability", func(t *testing.T) {
		provider := NewMockProvider()
		synapse, err := NewSentiment("test", provider, WithRetry(3))
		if err != nil {
			t.Fatalf("failed to create synapse: %v", err)
		}

		pipeline := synapse.GetPipeline()
		if pipeline == nil {
			t.Error("GetPipeline returned nil with retry option")
		}
	})

	t.Run("chaining", func(t *testing.T) {
		provider := NewMockProvider()
		synapse, err := NewSentiment("test", provider)
		if err != nil {
			t.Fatalf("failed to create synapse: %v", err)
		}

		pipeline := synapse.GetPipeline()
		if pipeline == nil {
			t.Fatal("GetPipeline returned nil")
		}

		ctx := context.Background()
		prompt := &Prompt{Task: "test", Input: "test", Schema: "{}"}
		req := &SynapseRequest{Prompt: prompt, Temperature: 0.5}
		_, err = pipeline.Process(ctx, req)
		if err != nil {
			t.Errorf("Pipeline processing failed: %v", err)
		}
	})
}

func TestSentimentSynapse_WithDefaults(t *testing.T) {
	t.Run("simple", func(t *testing.T) {
		provider := NewMockProvider()
		synapse, err := NewSentiment("test", provider)
		if err != nil {
			t.Fatalf("failed to create synapse: %v", err)
		}

		defaults := SentimentInput{
			Context:     "default context",
			Temperature: 0.5,
		}
		synapseWithDefaults := synapse.WithDefaults(defaults)

		if synapseWithDefaults == nil {
			t.Fatal("WithDefaults returned nil")
		}
		if synapseWithDefaults.defaults.Context != "default context" {
			t.Error("Defaults not set correctly")
		}
	})

	t.Run("reliability", func(t *testing.T) {
		provider := NewMockProvider()
		synapse, err := NewSentiment("test", provider, WithRetry(3))
		if err != nil {
			t.Fatalf("failed to create synapse: %v", err)
		}

		defaults := SentimentInput{Temperature: 0.7}
		synapseWithDefaults := synapse.WithDefaults(defaults)

		if synapseWithDefaults == nil {
			t.Error("WithDefaults returned nil with retry option")
		}
	})

	t.Run("chaining", func(t *testing.T) {
		provider := NewMockProviderWithResponse(`{"overall": "positive", "confidence": 0.9, "scores": {"positive": 0.9, "negative": 0.05, "neutral": 0.05}, "aspects": {}, "emotions": [], "reasoning": ["test"]}`)
		synapse, err := NewSentiment("test", provider)
		if err != nil {
			t.Fatalf("failed to create synapse: %v", err)
		}
		synapse = synapse.WithDefaults(SentimentInput{Context: "default", Temperature: 0.5})

		ctx := context.Background()
		result, err := synapse.Fire(ctx, NewSession(), "test text")
		if err != nil {
			t.Errorf("Fire failed with defaults: %v", err)
		}
		if result != "positive" {
			t.Errorf("Expected sentiment='positive', got '%s'", result)
		}
	})
}

func TestSentimentSynapse_Fire(t *testing.T) {
	t.Run("simple", func(t *testing.T) {
		provider := NewMockProviderWithResponse(`{"overall": "positive", "confidence": 0.9, "scores": {"positive": 0.9, "negative": 0.05, "neutral": 0.05}, "aspects": {}, "emotions": [], "reasoning": ["test"]}`)
		synapse, err := NewSentiment("basic sentiment", provider)
		if err != nil {
			t.Fatalf("failed to create synapse: %v", err)
		}

		ctx := context.Background()
		result, err := synapse.Fire(ctx, NewSession(), "I love this!")
		if err != nil {
			t.Fatalf("Fire failed: %v", err)
		}
		if result != "positive" {
			t.Errorf("Expected sentiment='positive', got '%s'", result)
		}
	})

	t.Run("reliability", func(t *testing.T) {
		provider := NewMockProviderWithResponse(`{"overall": "negative", "confidence": 0.8, "scores": {"positive": 0.1, "negative": 0.8, "neutral": 0.1}, "aspects": {}, "emotions": [], "reasoning": ["test"]}`)
		synapse, err := NewSentiment("test", provider,
			WithRetry(2),
			WithTimeout(5*time.Second))
		if err != nil {
			t.Fatalf("failed to create synapse: %v", err)
		}

		ctx := context.Background()
		result, err := synapse.Fire(ctx, NewSession(), "test input")
		if err != nil {
			t.Fatalf("Fire with reliability options failed: %v", err)
		}
		if result != "negative" {
			t.Errorf("Expected sentiment='negative', got '%s'", result)
		}
	})

	t.Run("chaining", func(t *testing.T) {
		failing := NewMockProviderWithError("primary failed")
		fallbackProvider := NewMockProviderWithResponse(`{"overall": "neutral", "confidence": 0.7, "scores": {"positive": 0.3, "negative": 0.3, "neutral": 0.4}, "aspects": {}, "emotions": [], "reasoning": ["test"]}`)
		fallbackSynapse, err := NewSentiment("test", fallbackProvider)
		if err != nil {
			t.Fatalf("failed to create fallback synapse: %v", err)
		}

		synapse, err := NewSentiment("test", failing,
			WithFallback(fallbackSynapse))
		if err != nil {
			t.Fatalf("failed to create synapse: %v", err)
		}

		ctx := context.Background()
		result, err := synapse.Fire(ctx, NewSession(), "test")
		if err != nil {
			t.Fatalf("Fire with fallback failed: %v", err)
		}
		if result != "neutral" {
			t.Error("Expected result from fallback")
		}
	})
}

func TestSentimentSynapse_FireWithDetails(t *testing.T) {
	t.Run("simple", func(t *testing.T) {
		provider := NewMockProviderWithResponse(`{"overall": "positive", "confidence": 0.95, "scores": {"positive": 0.9, "negative": 0.05, "neutral": 0.05}, "aspects": {"quality": "positive"}, "emotions": ["joy"], "reasoning": ["enthusiastic"]}`)
		synapse, err := NewSentiment("detailed sentiment", provider)
		if err != nil {
			t.Fatalf("failed to create synapse: %v", err)
		}

		ctx := context.Background()
		response, err := synapse.FireWithDetails(ctx, NewSession(), "This is amazing!")
		if err != nil {
			t.Fatalf("FireWithDetails failed: %v", err)
		}
		if response.Overall != "positive" {
			t.Errorf("Expected overall='positive', got '%s'", response.Overall)
		}
		if response.Confidence != 0.95 {
			t.Errorf("Expected confidence=0.95, got %f", response.Confidence)
		}
		if response.Scores.Positive != 0.9 {
			t.Errorf("Expected positive score=0.9, got %f", response.Scores.Positive)
		}
		if len(response.Reasoning) == 0 {
			t.Error("Expected reasoning to be set")
		}
	})

	t.Run("reliability", func(t *testing.T) {
		provider := NewMockProviderWithResponse(`{"overall": "negative", "confidence": 0.8, "scores": {"positive": 0.1, "negative": 0.8, "neutral": 0.1}, "aspects": {}, "emotions": [], "reasoning": ["test"]}`)
		synapse, err := NewSentiment("test", provider,
			WithRetry(3),
			WithBackoff(2, 10*time.Millisecond))
		if err != nil {
			t.Fatalf("failed to create synapse: %v", err)
		}

		ctx := context.Background()
		response, err := synapse.FireWithDetails(ctx, NewSession(), "test")
		if err != nil {
			t.Fatalf("FireWithDetails with backoff failed: %v", err)
		}
		if response.Overall != "negative" {
			t.Error("Expected sentiment")
		}
	})

	t.Run("chaining", func(t *testing.T) {
		provider := NewMockProviderWithResponse(`{"overall": "neutral", "confidence": 0.9, "scores": {"positive": 0.3, "negative": 0.3, "neutral": 0.4}, "aspects": {}, "emotions": [], "reasoning": ["test"]}`)
		synapse, err := NewSentiment("test", provider)
		if err != nil {
			t.Fatalf("failed to create synapse: %v", err)
		}
		synapse = synapse.WithDefaults(SentimentInput{Context: "test context"})

		ctx := context.Background()
		response, err := synapse.FireWithDetails(ctx, NewSession(), "test")
		if err != nil {
			t.Fatalf("FireWithDetails with defaults failed: %v", err)
		}
		if response.Overall != "neutral" {
			t.Error("Expected sentiment")
		}
	})
}

func TestSentimentSynapse_FireWithInput(t *testing.T) {
	t.Run("simple", func(t *testing.T) {
		provider := NewMockProviderWithResponse(`{"overall": "positive", "confidence": 0.9, "scores": {"positive": 0.9, "negative": 0.05, "neutral": 0.05}, "aspects": {}, "emotions": [], "reasoning": ["test"]}`)
		synapse, err := NewSentiment("test", provider)
		if err != nil {
			t.Fatalf("failed to create synapse: %v", err)
		}

		ctx := context.Background()
		input := SentimentInput{
			Text:    "Great product!",
			Context: "product review",
		}
		response, err := synapse.FireWithInput(ctx, NewSession(), input)
		if err != nil {
			t.Fatalf("FireWithInput failed: %v", err)
		}
		if response.Overall != "positive" {
			t.Errorf("Expected sentiment='positive', got '%s'", response.Overall)
		}
	})

	t.Run("reliability", func(t *testing.T) {
		provider := NewMockProviderWithResponse(`{"overall": "negative", "confidence": 0.85, "scores": {"positive": 0.1, "negative": 0.85, "neutral": 0.05}, "aspects": {}, "emotions": [], "reasoning": ["test"]}`)
		synapse, err := NewSentiment("test", provider,
			WithCircuitBreaker(5, 30*time.Second))
		if err != nil {
			t.Fatalf("failed to create synapse: %v", err)
		}

		ctx := context.Background()
		input := SentimentInput{
			Text:        "test",
			Temperature: 0.3,
		}
		response, err := synapse.FireWithInput(ctx, NewSession(), input)
		if err != nil {
			t.Fatalf("FireWithInput with circuit breaker failed: %v", err)
		}
		if response.Overall != "negative" {
			t.Error("Expected result")
		}
	})

	t.Run("chaining", func(t *testing.T) {
		provider := NewMockProviderWithResponse(`{"overall": "positive", "confidence": 0.9, "scores": {"positive": 0.9, "negative": 0.05, "neutral": 0.05}, "aspects": {}, "emotions": [], "reasoning": ["test"]}`)
		defaults := SentimentInput{
			Context: "default context",
			Aspects: []string{"quality", "service"},
		}
		synapse, err := NewSentiment("test", provider)
		if err != nil {
			t.Fatalf("failed to create synapse: %v", err)
		}
		synapse = synapse.WithDefaults(defaults)

		ctx := context.Background()
		input := SentimentInput{
			Text:    "test",
			Aspects: []string{"price"},
		}
		response, err := synapse.FireWithInput(ctx, NewSession(), input)
		if err != nil {
			t.Fatalf("FireWithInput with defaults merge failed: %v", err)
		}
		if response.Overall != "positive" {
			t.Error("Expected result")
		}
	})
}

func TestSentimentSynapse_mergeInputs(t *testing.T) {
	t.Run("simple", func(t *testing.T) {
		provider := NewMockProvider()
		synapse, err := NewSentiment("test", provider)
		if err != nil {
			t.Fatalf("failed to create synapse: %v", err)
		}
		synapse.defaults = SentimentInput{
			Context: "default context",
		}

		input := SentimentInput{
			Text: "test text",
		}
		merged := synapse.mergeInputs(input)

		if merged.Text != "test text" {
			t.Errorf("Expected text 'test text', got '%s'", merged.Text)
		}
		if merged.Context != "default context" {
			t.Errorf("Expected default context, got '%s'", merged.Context)
		}
	})

	t.Run("reliability", func(t *testing.T) {
		provider := NewMockProvider()
		synapse, err := NewSentiment("test", provider)
		if err != nil {
			t.Fatalf("failed to create synapse: %v", err)
		}
		synapse.defaults = SentimentInput{
			Context:     "default",
			Temperature: 0.5,
		}

		input := SentimentInput{
			Text:        "test",
			Context:     "override",
			Temperature: 0.7,
		}
		merged := synapse.mergeInputs(input)

		if merged.Context != "override" {
			t.Error("Input should override default context")
		}
		if merged.Temperature != 0.7 {
			t.Error("Input should override default temperature")
		}
	})

	t.Run("chaining", func(t *testing.T) {
		provider := NewMockProvider()
		synapse, err := NewSentiment("test", provider)
		if err != nil {
			t.Fatalf("failed to create synapse: %v", err)
		}
		synapse.defaults = SentimentInput{
			Context: "default",
			Aspects: []string{"aspect1"},
		}

		input := SentimentInput{
			Text:    "test",
			Aspects: []string{"aspect2"},
		}
		merged := synapse.mergeInputs(input)

		if len(merged.Aspects) == 0 {
			t.Error("Expected aspects to be set")
		}
		if merged.Context != "default" {
			t.Error("Should keep default context when not overridden")
		}
	})
}

func TestSentimentSynapse_buildPrompt(t *testing.T) {
	t.Run("simple", func(t *testing.T) {
		provider := NewMockProvider()
		synapse, err := NewSentiment("basic sentiment", provider)
		if err != nil {
			t.Fatalf("failed to create synapse: %v", err)
		}

		input := SentimentInput{
			Text: "text to analyze",
		}
		prompt := synapse.buildPrompt(input)

		if prompt.Task != "Analyze basic sentiment sentiment" {
			t.Errorf("Expected task prefix, got '%s'", prompt.Task)
		}
		if prompt.Input != "text to analyze" {
			t.Errorf("Expected input to be set, got '%s'", prompt.Input)
		}
		if prompt.Schema == "" {
			t.Error("Expected schema to be set")
		}
	})

	t.Run("reliability", func(t *testing.T) {
		provider := NewMockProvider()
		synapse, err := NewSentiment("test", provider)
		if err != nil {
			t.Fatalf("failed to create synapse: %v", err)
		}

		input := SentimentInput{
			Text:    "test",
			Context: "sentiment context",
			Aspects: []string{"quality", "price"},
		}
		prompt := synapse.buildPrompt(input)

		if prompt.Context != "sentiment context" {
			t.Error("Expected context to be set")
		}
		if len(prompt.Constraints) == 0 {
			t.Error("Expected constraints to be set")
		}
	})

	t.Run("chaining", func(t *testing.T) {
		provider := NewMockProvider()
		synapse, err := NewSentiment("test", provider)
		if err != nil {
			t.Fatalf("failed to create synapse: %v", err)
		}

		input := SentimentInput{
			Text: "test",
		}
		prompt := synapse.buildPrompt(input)

		if err := prompt.Validate(); err != nil {
			t.Errorf("Built prompt failed validation: %v", err)
		}
	})
}

func TestNormalizeSentiment(t *testing.T) {
	t.Run("simple", func(t *testing.T) {
		result := normalizeSentiment("positive")
		if result != "positive" {
			t.Errorf("Expected 'positive', got '%s'", result)
		}
	})

	t.Run("reliability", func(t *testing.T) {
		tests := []struct {
			input    string
			expected string
		}{
			{"POSITIVE", "positive"},
			{"Negative", "negative"},
			{"NEUTRAL", "neutral"},
			{"mixed", "mixed"},
			{"unknown", "unknown"},
		}

		for _, tt := range tests {
			result := normalizeSentiment(tt.input)
			if result != tt.expected {
				t.Errorf("normalizeSentiment(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		}
	})

	t.Run("chaining", func(t *testing.T) {
		// Test normalization works in response flow
		normalized := normalizeSentiment("PoSiTiVe")
		if normalized != "positive" {
			t.Error("Case normalization should work")
		}
	})
}

func TestSentiment(t *testing.T) {
	t.Run("simple", func(t *testing.T) {
		provider := NewMockProvider()
		synapse, err := Sentiment("basic sentiment", provider)
		if err != nil {
			t.Fatalf("failed to create synapse: %v", err)
		}

		if synapse == nil {
			t.Fatal("Sentiment wrapper returned nil")
		}
	})

	t.Run("reliability", func(t *testing.T) {
		provider := NewMockProviderWithResponse(`{"overall": "positive", "confidence": 0.9, "scores": {"positive": 0.9, "negative": 0.05, "neutral": 0.05}, "aspects": {}, "emotions": [], "reasoning": ["test"]}`)
		synapse, err := Sentiment("basic sentiment", provider,
			WithRetry(3),
			WithTimeout(10*time.Second))
		if err != nil {
			t.Fatalf("failed to create synapse: %v", err)
		}

		if synapse == nil {
			t.Fatal("Sentiment wrapper with options returned nil")
		}

		ctx := context.Background()
		_, err = synapse.Fire(ctx, NewSession(), "test text")
		if err != nil {
			t.Errorf("Sentiment synapse Fire failed: %v", err)
		}
	})

	t.Run("chaining", func(t *testing.T) {
		provider := NewMockProviderWithResponse(`{"overall": "positive", "confidence": 0.9, "scores": {"positive": 0.9, "negative": 0.05, "neutral": 0.05}, "aspects": {}, "emotions": [], "reasoning": ["test"]}`)
		synapse, err := Sentiment("basic sentiment", provider)
		if err != nil {
			t.Fatalf("failed to create synapse: %v", err)
		}
		synapse = synapse.WithDefaults(SentimentInput{Context: "test context"})

		ctx := context.Background()
		result, err := synapse.Fire(ctx, NewSession(), "test text")
		if err != nil {
			t.Fatalf("Sentiment with chaining failed: %v", err)
		}
		if result != "positive" {
			t.Errorf("Expected sentiment='positive', got '%s'", result)
		}
	})
}
