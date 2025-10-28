package zyn

import (
	"context"
	"testing"
	"time"
)

func TestNewRanking(t *testing.T) {
	t.Run("simple", func(t *testing.T) {
		provider := NewMockProvider()
		synapse := NewRanking("quality", provider)

		if synapse == nil {
			t.Fatal("Expected synapse to be created")
		}
		if synapse.criteria != "quality" {
			t.Errorf("Expected criteria='quality', got '%s'", synapse.criteria)
		}
	})

	t.Run("reliability", func(t *testing.T) {
		provider := NewMockProvider()
		synapse := NewRanking("quality", provider,
			WithRetry(3),
			WithTimeout(10*time.Second))

		if synapse == nil {
			t.Fatal("Expected synapse with reliability options to be created")
		}
	})

	t.Run("chaining", func(t *testing.T) {
		primary := NewMockProviderWithName("primary")
		fallback := NewMockProviderWithName("fallback")
		fallbackSynapse := NewRanking("quality", fallback)

		synapse := NewRanking("quality", primary,
			WithFallback(fallbackSynapse))

		if synapse == nil {
			t.Fatal("Expected synapse with fallback to be created")
		}
	})
}

func TestRankingSynapse_GetPipeline(t *testing.T) {
	t.Run("simple", func(t *testing.T) {
		provider := NewMockProvider()
		synapse := NewRanking("test", provider)

		pipeline := synapse.GetPipeline()
		if pipeline == nil {
			t.Error("GetPipeline returned nil")
		}
	})

	t.Run("reliability", func(t *testing.T) {
		provider := NewMockProvider()
		synapse := NewRanking("test", provider, WithRetry(3))

		pipeline := synapse.GetPipeline()
		if pipeline == nil {
			t.Error("GetPipeline returned nil with retry option")
		}
	})

	t.Run("chaining", func(t *testing.T) {
		provider := NewMockProvider()
		synapse := NewRanking("test", provider)

		pipeline := synapse.GetPipeline()
		if pipeline == nil {
			t.Fatal("GetPipeline returned nil")
		}

		ctx := context.Background()
		prompt := &Prompt{Task: "test", Input: "test", Schema: "{}"}
		req := &SynapseRequest{Prompt: prompt, Temperature: 0.5}
		_, err := pipeline.Process(ctx, req)
		if err != nil {
			t.Errorf("Pipeline processing failed: %v", err)
		}
	})
}

func TestRankingSynapse_WithDefaults(t *testing.T) {
	t.Run("simple", func(t *testing.T) {
		provider := NewMockProvider()
		synapse := NewRanking("test", provider)

		defaults := RankingInput{
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
		synapse := NewRanking("test", provider, WithRetry(3))

		defaults := RankingInput{Temperature: 0.7}
		synapseWithDefaults := synapse.WithDefaults(defaults)

		if synapseWithDefaults == nil {
			t.Error("WithDefaults returned nil with retry option")
		}
	})

	t.Run("chaining", func(t *testing.T) {
		provider := NewMockProviderWithResponse(`{"ranked": ["item1", "item2"], "confidence": 0.9, "reasoning": ["test"]}`)
		synapse := NewRanking("test", provider).
			WithDefaults(RankingInput{Context: "default", Temperature: 0.5})

		ctx := context.Background()
		result, err := synapse.Fire(ctx, []string{"item2", "item1"})
		if err != nil {
			t.Errorf("Fire failed with defaults: %v", err)
		}
		if len(result) != 2 {
			t.Errorf("Expected 2 items, got %d", len(result))
		}
	})
}

func TestRankingSynapse_Fire(t *testing.T) {
	t.Run("simple", func(t *testing.T) {
		provider := NewMockProviderWithResponse(`{"ranked": ["best", "good", "okay"], "confidence": 0.9, "reasoning": ["test"]}`)
		synapse := NewRanking("quality", provider)

		ctx := context.Background()
		result, err := synapse.Fire(ctx, []string{"okay", "best", "good"})
		if err != nil {
			t.Fatalf("Fire failed: %v", err)
		}
		if len(result) != 3 {
			t.Errorf("Expected 3 items, got %d", len(result))
		}
		if result[0] != "best" {
			t.Errorf("Expected first item 'best', got '%s'", result[0])
		}
	})

	t.Run("reliability", func(t *testing.T) {
		provider := NewMockProviderWithResponse(`{"ranked": ["a", "b"], "confidence": 0.9, "reasoning": ["test"]}`)
		synapse := NewRanking("test", provider,
			WithRetry(2),
			WithTimeout(5*time.Second))

		ctx := context.Background()
		result, err := synapse.Fire(ctx, []string{"b", "a"})
		if err != nil {
			t.Fatalf("Fire with reliability options failed: %v", err)
		}
		if len(result) != 2 {
			t.Errorf("Expected 2 items, got %d", len(result))
		}
	})

	t.Run("chaining", func(t *testing.T) {
		failing := NewMockProviderWithError("primary failed")
		fallbackProvider := NewMockProviderWithResponse(`{"ranked": ["fallback"], "confidence": 0.9, "reasoning": ["test"]}`)
		fallbackSynapse := NewRanking("test", fallbackProvider)

		synapse := NewRanking("test", failing,
			WithFallback(fallbackSynapse))

		ctx := context.Background()
		result, err := synapse.Fire(ctx, []string{"fallback"})
		if err != nil {
			t.Fatalf("Fire with fallback failed: %v", err)
		}
		if len(result) != 1 {
			t.Error("Expected result from fallback")
		}
	})
}

func TestRankingSynapse_FireWithDetails(t *testing.T) {
	t.Run("simple", func(t *testing.T) {
		provider := NewMockProviderWithResponse(`{"ranked": ["item1", "item2"], "confidence": 0.95, "reasoning": ["high quality", "good"]}`)
		synapse := NewRanking("quality", provider)

		ctx := context.Background()
		response, err := synapse.FireWithDetails(ctx, []string{"item2", "item1"})
		if err != nil {
			t.Fatalf("FireWithDetails failed: %v", err)
		}
		if len(response.Ranked) != 2 {
			t.Errorf("Expected 2 ranked items, got %d", len(response.Ranked))
		}
		if response.Confidence != 0.95 {
			t.Errorf("Expected confidence 0.95, got %f", response.Confidence)
		}
		if len(response.Reasoning) == 0 {
			t.Error("Expected reasoning to be set")
		}
	})

	t.Run("reliability", func(t *testing.T) {
		provider := NewMockProviderWithResponse(`{"ranked": ["a"], "confidence": 0.9, "reasoning": ["test"]}`)
		synapse := NewRanking("test", provider,
			WithRetry(3),
			WithBackoff(2, 10*time.Millisecond))

		ctx := context.Background()
		response, err := synapse.FireWithDetails(ctx, []string{"a"})
		if err != nil {
			t.Fatalf("FireWithDetails with backoff failed: %v", err)
		}
		if len(response.Ranked) != 1 {
			t.Error("Expected ranked items")
		}
	})

	t.Run("chaining", func(t *testing.T) {
		provider := NewMockProviderWithResponse(`{"ranked": ["x", "y"], "confidence": 0.9, "reasoning": ["test"]}`)
		synapse := NewRanking("test", provider).
			WithDefaults(RankingInput{Context: "test context"})

		ctx := context.Background()
		response, err := synapse.FireWithDetails(ctx, []string{"y", "x"})
		if err != nil {
			t.Fatalf("FireWithDetails with defaults failed: %v", err)
		}
		if len(response.Ranked) != 2 {
			t.Error("Expected 2 ranked items")
		}
	})
}

func TestRankingSynapse_FireWithInput(t *testing.T) {
	t.Run("simple", func(t *testing.T) {
		provider := NewMockProviderWithResponse(`{"ranked": ["item1", "item2"], "confidence": 0.9, "reasoning": ["test"]}`)
		synapse := NewRanking("quality", provider)

		ctx := context.Background()
		input := RankingInput{
			Items:   []string{"item2", "item1"},
			Context: "test context",
		}
		response, err := synapse.FireWithInput(ctx, input)
		if err != nil {
			t.Fatalf("FireWithInput failed: %v", err)
		}
		if len(response.Ranked) != 2 {
			t.Errorf("Expected 2 ranked items, got %d", len(response.Ranked))
		}
	})

	t.Run("reliability", func(t *testing.T) {
		provider := NewMockProviderWithResponse(`{"ranked": ["a"], "confidence": 0.9, "reasoning": ["test"]}`)
		synapse := NewRanking("test", provider,
			WithCircuitBreaker(5, 30*time.Second))

		ctx := context.Background()
		input := RankingInput{
			Items:       []string{"a"},
			Temperature: 0.3,
		}
		response, err := synapse.FireWithInput(ctx, input)
		if err != nil {
			t.Fatalf("FireWithInput with circuit breaker failed: %v", err)
		}
		if len(response.Ranked) != 1 {
			t.Error("Expected result")
		}
	})

	t.Run("chaining", func(t *testing.T) {
		provider := NewMockProviderWithResponse(`{"ranked": ["x", "y"], "confidence": 0.9, "reasoning": ["test"]}`)
		defaults := RankingInput{
			Context:  "default context",
			Examples: []string{"example1", "example2"},
		}
		synapse := NewRanking("test", provider).WithDefaults(defaults)

		ctx := context.Background()
		input := RankingInput{
			Items:    []string{"y", "x"},
			Examples: []string{"override"},
		}
		response, err := synapse.FireWithInput(ctx, input)
		if err != nil {
			t.Fatalf("FireWithInput with defaults merge failed: %v", err)
		}
		if len(response.Ranked) != 2 {
			t.Error("Expected result")
		}
	})
}

func TestRankingSynapse_mergeInputs(t *testing.T) {
	t.Run("simple", func(t *testing.T) {
		provider := NewMockProvider()
		synapse := NewRanking("test", provider)
		synapse.defaults = RankingInput{
			Context: "default context",
		}

		input := RankingInput{
			Items: []string{"item1", "item2"},
		}
		merged := synapse.mergeInputs(input)

		if len(merged.Items) != 2 {
			t.Error("Expected items to be set")
		}
		if merged.Context != "default context" {
			t.Errorf("Expected default context, got '%s'", merged.Context)
		}
	})

	t.Run("reliability", func(t *testing.T) {
		provider := NewMockProvider()
		synapse := NewRanking("test", provider)
		synapse.defaults = RankingInput{
			Context:     "default",
			Temperature: 0.5,
		}

		input := RankingInput{
			Items:       []string{"a"},
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
		synapse := NewRanking("test", provider)
		synapse.defaults = RankingInput{
			Context:  "default",
			Examples: []string{"example1"},
			TopN:     5,
		}

		input := RankingInput{
			Items:    []string{"x"},
			Examples: []string{"example2"},
			TopN:     3,
		}
		merged := synapse.mergeInputs(input)

		if len(merged.Examples) == 0 {
			t.Error("Expected examples to be set")
		}
		if merged.TopN != 3 {
			t.Error("Input should override default TopN")
		}
		if merged.Context != "default" {
			t.Error("Should keep default context when not overridden")
		}
	})
}

func TestRankingSynapse_buildPrompt(t *testing.T) {
	t.Run("simple", func(t *testing.T) {
		provider := NewMockProvider()
		synapse := NewRanking("quality", provider)

		input := RankingInput{
			Items: []string{"item1", "item2"},
		}
		prompt := synapse.buildPrompt(input)

		if prompt.Task != "Rank by quality" {
			t.Errorf("Expected task prefix, got '%s'", prompt.Task)
		}
		if len(prompt.Items) != 2 {
			t.Error("Expected items to be set")
		}
		if prompt.Schema == "" {
			t.Error("Expected schema to be set")
		}
	})

	t.Run("reliability", func(t *testing.T) {
		provider := NewMockProvider()
		synapse := NewRanking("test", provider)

		input := RankingInput{
			Items:    []string{"a", "b"},
			Context:  "ranking context",
			Examples: []string{"ex1", "ex2"},
			TopN:     1,
		}
		prompt := synapse.buildPrompt(input)

		if prompt.Context != "ranking context" {
			t.Error("Expected context to be set")
		}
		if len(prompt.Constraints) == 0 {
			t.Error("Expected constraints to be set")
		}
	})

	t.Run("chaining", func(t *testing.T) {
		provider := NewMockProvider()
		synapse := NewRanking("test", provider)

		input := RankingInput{
			Items: []string{"x", "y"},
		}
		prompt := synapse.buildPrompt(input)

		if err := prompt.Validate(); err != nil {
			t.Errorf("Built prompt failed validation: %v", err)
		}
	})
}

func TestRanking(t *testing.T) {
	t.Run("simple", func(t *testing.T) {
		provider := NewMockProvider()
		synapse := Ranking("quality", provider)

		if synapse == nil {
			t.Fatal("Ranking wrapper returned nil")
		}
	})

	t.Run("reliability", func(t *testing.T) {
		provider := NewMockProviderWithResponse(`{"ranked": ["a", "b"], "confidence": 0.9, "reasoning": ["test"]}`)
		synapse := Ranking("quality", provider,
			WithRetry(3),
			WithTimeout(10*time.Second))

		if synapse == nil {
			t.Fatal("Ranking wrapper with options returned nil")
		}

		ctx := context.Background()
		_, err := synapse.Fire(ctx, []string{"a", "b"})
		if err != nil {
			t.Errorf("Ranking synapse Fire failed: %v", err)
		}
	})

	t.Run("chaining", func(t *testing.T) {
		provider := NewMockProviderWithResponse(`{"ranked": ["x", "y"], "confidence": 0.9, "reasoning": ["test"]}`)
		synapse := Ranking("quality", provider).
			WithDefaults(RankingInput{Context: "test context"})

		ctx := context.Background()
		result, err := synapse.Fire(ctx, []string{"y", "x"})
		if err != nil {
			t.Fatalf("Ranking with chaining failed: %v", err)
		}
		if len(result) != 2 {
			t.Errorf("Expected 2 items, got %d", len(result))
		}
	})
}
