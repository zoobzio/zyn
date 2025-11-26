package zyn

import (
	"context"
	"testing"
	"time"
)

func TestNewClassification(t *testing.T) {
	t.Run("simple", func(t *testing.T) {
		provider := NewMockProvider()
		categories := []string{"cat1", "cat2", "cat3"}
		synapse, err := NewClassification("What is this?", categories, provider)
		if err != nil {
			t.Fatalf("failed to create synapse: %v", err)
		}

		if synapse == nil {
			t.Fatal("Expected synapse to be created")
		}
		if synapse.question != "What is this?" {
			t.Errorf("Expected question 'What is this?', got '%s'", synapse.question)
		}
		if len(synapse.categories) != 3 {
			t.Errorf("Expected 3 categories, got %d", len(synapse.categories))
		}
	})

	t.Run("reliability", func(t *testing.T) {
		provider := NewMockProvider()
		categories := []string{"cat1", "cat2"}
		synapse, err := NewClassification("Classify", categories, provider,
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
		categories := []string{"cat1", "cat2"}
		fallbackSynapse, err := NewClassification("Classify", categories, fallback)
		if err != nil {
			t.Fatalf("failed to create fallback synapse: %v", err)
		}

		synapse, err := NewClassification("Classify", categories, primary,
			WithFallback(fallbackSynapse))
		if err != nil {
			t.Fatalf("failed to create synapse: %v", err)
		}

		if synapse == nil {
			t.Fatal("Expected synapse with fallback to be created")
		}
	})
}

func TestClassificationSynapse_GetPipeline(t *testing.T) {
	t.Run("simple", func(t *testing.T) {
		provider := NewMockProvider()
		synapse, err := NewClassification("test", []string{"a", "b"}, provider)
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
		synapse, err := NewClassification("test", []string{"a", "b"}, provider, WithRetry(3))
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
		synapse, err := NewClassification("test", []string{"a", "b"}, provider)
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

func TestClassificationSynapse_WithDefaults(t *testing.T) {
	t.Run("simple", func(t *testing.T) {
		provider := NewMockProvider()
		synapse, err := NewClassification("test", []string{"a", "b"}, provider)
		if err != nil {
			t.Fatalf("failed to create synapse: %v", err)
		}

		defaults := ClassificationInput{
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
		synapse, err := NewClassification("test", []string{"a", "b"}, provider, WithRetry(3))
		if err != nil {
			t.Fatalf("failed to create synapse: %v", err)
		}

		defaults := ClassificationInput{Temperature: 0.7}
		synapseWithDefaults := synapse.WithDefaults(defaults)

		if synapseWithDefaults == nil {
			t.Error("WithDefaults returned nil with retry option")
		}
	})

	t.Run("chaining", func(t *testing.T) {
		provider := NewMockProviderWithResponse(`{"primary": "cat1", "secondary": "", "confidence": 0.9, "reasoning": ["test"]}`)
		synapse, err := NewClassification("test", []string{"cat1", "cat2"}, provider)
		if err != nil {
			t.Fatalf("failed to create synapse: %v", err)
		}
		synapse = synapse.WithDefaults(ClassificationInput{Context: "default", Temperature: 0.5})

		ctx := context.Background()
		result, err := synapse.Fire(ctx, NewSession(), "test")
		if err != nil {
			t.Errorf("Fire failed with defaults: %v", err)
		}
		if result != "cat1" {
			t.Errorf("Expected 'cat1', got '%s'", result)
		}
	})
}

func TestClassificationSynapse_Fire(t *testing.T) {
	t.Run("simple", func(t *testing.T) {
		provider := NewMockProviderWithResponse(`{"primary": "database", "secondary": "", "confidence": 0.9, "reasoning": ["test"]}`)
		categories := []string{"network", "database", "auth"}
		synapse, err := NewClassification("What type of error?", categories, provider)
		if err != nil {
			t.Fatalf("failed to create synapse: %v", err)
		}

		ctx := context.Background()
		result, err := synapse.Fire(ctx, NewSession(), "Connection refused on port 5432")
		if err != nil {
			t.Fatalf("Fire failed: %v", err)
		}
		if result != "database" {
			t.Errorf("Expected 'database', got '%s'", result)
		}
	})

	t.Run("reliability", func(t *testing.T) {
		provider := NewMockProviderWithResponse(`{"primary": "cat1", "secondary": "", "confidence": 0.9, "reasoning": ["test"]}`)
		synapse, err := NewClassification("Classify", []string{"cat1", "cat2"}, provider,
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
		if result != "cat1" {
			t.Errorf("Expected 'cat1', got '%s'", result)
		}
	})

	t.Run("chaining", func(t *testing.T) {
		failing := NewMockProviderWithError("primary failed")
		fallbackProvider := NewMockProviderWithResponse(`{"primary": "cat2", "secondary": "", "confidence": 0.8, "reasoning": ["fallback"]}`)
		fallbackSynapse, err := NewClassification("Classify", []string{"cat1", "cat2"}, fallbackProvider)
		if err != nil {
			t.Fatalf("failed to create fallback synapse: %v", err)
		}

		synapse, err := NewClassification("Classify", []string{"cat1", "cat2"}, failing,
			WithFallback(fallbackSynapse))
		if err != nil {
			t.Fatalf("failed to create synapse: %v", err)
		}

		ctx := context.Background()
		result, err := synapse.Fire(ctx, NewSession(), "test")
		if err != nil {
			t.Fatalf("Fire with fallback failed: %v", err)
		}
		if result != "cat2" {
			t.Errorf("Expected 'cat2' from fallback, got '%s'", result)
		}
	})
}

func TestClassificationSynapse_FireWithDetails(t *testing.T) {
	t.Run("simple", func(t *testing.T) {
		provider := NewMockProviderWithResponse(`{"primary": "positive", "secondary": "neutral", "confidence": 0.85, "reasoning": ["enthusiastic language"]}`)
		synapse, err := NewClassification("Sentiment", []string{"positive", "negative", "neutral"}, provider)
		if err != nil {
			t.Fatalf("failed to create synapse: %v", err)
		}

		ctx := context.Background()
		response, err := synapse.FireWithDetails(ctx, NewSession(), "This is great!")
		if err != nil {
			t.Fatalf("FireWithDetails failed: %v", err)
		}
		if response.Primary != "positive" {
			t.Errorf("Expected primary='positive', got '%s'", response.Primary)
		}
		if response.Secondary != "neutral" {
			t.Errorf("Expected secondary='neutral', got '%s'", response.Secondary)
		}
		if response.Confidence != 0.85 {
			t.Errorf("Expected confidence=0.85, got %f", response.Confidence)
		}
	})

	t.Run("reliability", func(t *testing.T) {
		provider := NewMockProviderWithResponse(`{"primary": "cat1", "secondary": "", "confidence": 0.8, "reasoning": ["test"]}`)
		synapse, err := NewClassification("Classify", []string{"cat1", "cat2"}, provider,
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
		if response.Primary != "cat1" {
			t.Errorf("Expected primary='cat1', got '%s'", response.Primary)
		}
	})

	t.Run("chaining", func(t *testing.T) {
		provider := NewMockProviderWithResponse(`{"primary": "cat1", "secondary": "", "confidence": 0.9, "reasoning": ["test"]}`)
		synapse, err := NewClassification("Classify", []string{"cat1", "cat2"}, provider)
		if err != nil {
			t.Fatalf("failed to create synapse: %v", err)
		}
		synapse = synapse.WithDefaults(ClassificationInput{Context: "test context"})

		ctx := context.Background()
		response, err := synapse.FireWithDetails(ctx, NewSession(), "test")
		if err != nil {
			t.Fatalf("FireWithDetails with defaults failed: %v", err)
		}
		if response.Primary != "cat1" {
			t.Error("Expected 'cat1'")
		}
	})
}

func TestClassificationSynapse_FireWithInput(t *testing.T) {
	t.Run("simple", func(t *testing.T) {
		provider := NewMockProviderWithResponse(`{"primary": "positive", "secondary": "", "confidence": 0.9, "reasoning": ["test"]}`)
		synapse, err := NewClassification("Sentiment", []string{"positive", "negative", "neutral"}, provider)
		if err != nil {
			t.Fatalf("failed to create synapse: %v", err)
		}

		ctx := context.Background()
		input := ClassificationInput{
			Subject: "This is amazing!",
			Context: "customer review",
		}
		response, err := synapse.FireWithInput(ctx, NewSession(), input)
		if err != nil {
			t.Fatalf("FireWithInput failed: %v", err)
		}
		if response.Primary != "positive" {
			t.Errorf("Expected primary='positive', got '%s'", response.Primary)
		}
	})

	t.Run("reliability", func(t *testing.T) {
		provider := NewMockProviderWithResponse(`{"primary": "cat1", "secondary": "", "confidence": 0.9, "reasoning": ["test"]}`)
		synapse, err := NewClassification("Classify", []string{"cat1", "cat2"}, provider,
			WithCircuitBreaker(5, 30*time.Second))
		if err != nil {
			t.Fatalf("failed to create synapse: %v", err)
		}

		ctx := context.Background()
		input := ClassificationInput{
			Subject:     "test",
			Temperature: 0.3,
		}
		response, err := synapse.FireWithInput(ctx, NewSession(), input)
		if err != nil {
			t.Fatalf("FireWithInput with circuit breaker failed: %v", err)
		}
		if response.Primary != "cat1" {
			t.Error("Expected 'cat1'")
		}
	})

	t.Run("chaining", func(t *testing.T) {
		provider := NewMockProviderWithResponse(`{"primary": "cat1", "secondary": "", "confidence": 0.9, "reasoning": ["test"]}`)
		defaults := ClassificationInput{
			Context: "default context",
			Examples: map[string][]string{
				"cat1": {"example1"},
			},
			Temperature: 0.5,
		}
		synapse, err := NewClassification("Classify", []string{"cat1", "cat2"}, provider)
		if err != nil {
			t.Fatalf("failed to create synapse: %v", err)
		}
		synapse = synapse.WithDefaults(defaults)

		ctx := context.Background()
		input := ClassificationInput{
			Subject: "test",
			Examples: map[string][]string{
				"cat2": {"example2"},
			},
		}
		response, err := synapse.FireWithInput(ctx, NewSession(), input)
		if err != nil {
			t.Fatalf("FireWithInput with defaults merge failed: %v", err)
		}
		if response.Primary != "cat1" {
			t.Error("Expected 'cat1'")
		}
	})
}

func TestClassificationSynapse_mergeInputs(t *testing.T) {
	t.Run("simple", func(t *testing.T) {
		synapse := &ClassificationSynapse{
			defaults: ClassificationInput{
				Context: "default context",
			},
		}

		input := ClassificationInput{
			Subject: "test subject",
		}
		merged := synapse.mergeInputs(input)

		if merged.Subject != "test subject" {
			t.Errorf("Expected subject 'test subject', got '%s'", merged.Subject)
		}
		if merged.Context != "default context" {
			t.Errorf("Expected default context, got '%s'", merged.Context)
		}
	})

	t.Run("reliability", func(t *testing.T) {
		synapse := &ClassificationSynapse{
			defaults: ClassificationInput{
				Context:     "default",
				Temperature: 0.5,
			},
		}

		input := ClassificationInput{
			Subject:     "test",
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
		synapse := &ClassificationSynapse{
			defaults: ClassificationInput{
				Context: "default",
				Examples: map[string][]string{
					"cat1": {"e1"},
				},
			},
		}

		input := ClassificationInput{
			Subject: "test",
			Examples: map[string][]string{
				"cat1": {"e2"},
				"cat2": {"e3"},
			},
		}
		merged := synapse.mergeInputs(input)

		if len(merged.Examples["cat1"]) != 2 {
			t.Errorf("Expected 2 examples for cat1, got %d", len(merged.Examples["cat1"]))
		}
		if len(merged.Examples["cat2"]) != 1 {
			t.Errorf("Expected 1 example for cat2, got %d", len(merged.Examples["cat2"]))
		}
	})
}

func TestClassificationSynapse_buildPrompt(t *testing.T) {
	t.Run("simple", func(t *testing.T) {
		schema, err := generateJSONSchema[ClassificationResponse]()
		if err != nil {
			t.Fatalf("failed to generate schema: %v", err)
		}
		synapse := &ClassificationSynapse{
			question:   "What is this?",
			categories: []string{"cat1", "cat2"},
			schema:     schema,
		}

		input := ClassificationInput{
			Subject: "test subject",
		}
		prompt := synapse.buildPrompt(input)

		if prompt.Task != "What is this?" {
			t.Errorf("Expected task 'What is this?', got '%s'", prompt.Task)
		}
		if prompt.Input != "test subject" {
			t.Errorf("Expected input 'test subject', got '%s'", prompt.Input)
		}
		if len(prompt.Categories) != 2 {
			t.Errorf("Expected 2 categories, got %d", len(prompt.Categories))
		}
		if prompt.Schema == "" {
			t.Error("Expected schema to be set")
		}
	})

	t.Run("reliability", func(t *testing.T) {
		schema, err := generateJSONSchema[ClassificationResponse]()
		if err != nil {
			t.Fatalf("failed to generate schema: %v", err)
		}
		synapse := &ClassificationSynapse{
			question:   "Classify",
			categories: []string{"cat1", "cat2"},
			schema:     schema,
		}

		input := ClassificationInput{
			Subject: "test",
			Context: "classification context",
			Examples: map[string][]string{
				"cat1": {"e1", "e2"},
			},
		}
		prompt := synapse.buildPrompt(input)

		if prompt.Context != "classification context" {
			t.Error("Expected context to be set")
		}
		if len(prompt.Examples["cat1"]) != 2 {
			t.Error("Expected examples to be set")
		}
		if len(prompt.Constraints) == 0 {
			t.Error("Expected constraints to be set")
		}
	})

	t.Run("chaining", func(t *testing.T) {
		schema, err := generateJSONSchema[ClassificationResponse]()
		if err != nil {
			t.Fatalf("failed to generate schema: %v", err)
		}
		synapse := &ClassificationSynapse{
			question:   "Classify",
			categories: []string{"cat1", "cat2"},
			schema:     schema,
		}

		input := ClassificationInput{
			Subject: "test",
		}
		prompt := synapse.buildPrompt(input)

		if err := prompt.Validate(); err != nil {
			t.Errorf("Built prompt failed validation: %v", err)
		}
	})
}

func TestClassification(t *testing.T) {
	t.Run("simple", func(t *testing.T) {
		provider := NewMockProvider()
		synapse, err := Classification("Classify", []string{"cat1", "cat2"}, provider)
		if err != nil {
			t.Fatalf("failed to create synapse: %v", err)
		}

		if synapse == nil {
			t.Fatal("Classification wrapper returned nil")
		}
	})

	t.Run("reliability", func(t *testing.T) {
		provider := NewMockProvider()
		synapse, err := Classification("Classify", []string{"cat1", "cat2"}, provider,
			WithRetry(3),
			WithTimeout(10*time.Second))
		if err != nil {
			t.Fatalf("failed to create synapse: %v", err)
		}

		if synapse == nil {
			t.Fatal("Classification wrapper with options returned nil")
		}

		ctx := context.Background()
		_, err = synapse.Fire(ctx, NewSession(), "test")
		if err != nil {
			t.Errorf("Classification synapse Fire failed: %v", err)
		}
	})

	t.Run("chaining", func(t *testing.T) {
		provider := NewMockProviderWithResponse(`{"primary": "cat1", "secondary": "", "confidence": 0.9, "reasoning": ["test"]}`)
		synapse, err := Classification("Classify", []string{"cat1", "cat2"}, provider)
		if err != nil {
			t.Fatalf("failed to create synapse: %v", err)
		}
		synapse = synapse.WithDefaults(ClassificationInput{Context: "test context"})

		ctx := context.Background()
		result, err := synapse.Fire(ctx, NewSession(), "test")
		if err != nil {
			t.Fatalf("Classification with chaining failed: %v", err)
		}
		if result != "cat1" {
			t.Errorf("Expected 'cat1', got '%s'", result)
		}
	})
}
