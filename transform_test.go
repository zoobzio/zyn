package zyn

import (
	"context"
	"testing"
	"time"
)

func TestTransform(t *testing.T) {
	t.Run("simple", func(t *testing.T) {
		provider := NewMockProvider()
		synapse := Transform("summarize", provider)

		if synapse == nil {
			t.Fatal("Transform wrapper returned nil")
		}
	})

	t.Run("reliability", func(t *testing.T) {
		provider := NewMockProviderWithResponse(`{"output": "transformed text", "confidence": 0.9, "changes": [], "reasoning": ["test"]}`)
		synapse := Transform("summarize", provider,
			WithRetry(3),
			WithTimeout(10*time.Second))

		if synapse == nil {
			t.Fatal("Transform wrapper with options returned nil")
		}

		ctx := context.Background()
		_, err := synapse.Fire(ctx, "test text")
		if err != nil {
			t.Errorf("Transform synapse Fire failed: %v", err)
		}
	})

	t.Run("chaining", func(t *testing.T) {
		provider := NewMockProviderWithResponse(`{"output": "transformed", "confidence": 0.9, "changes": [], "reasoning": ["test"]}`)
		synapse := Transform("summarize", provider)

		ctx := context.Background()
		result, err := synapse.Fire(ctx, "test text")
		if err != nil {
			t.Fatalf("Transform with chaining failed: %v", err)
		}
		if result != "transformed" {
			t.Errorf("Expected output='transformed', got '%s'", result)
		}
	})
}

func TestTransformSynapse_GetPipeline(t *testing.T) {
	t.Run("simple", func(t *testing.T) {
		provider := NewMockProvider()
		synapse := Transform("test", provider)

		pipeline := synapse.GetPipeline()
		if pipeline == nil {
			t.Error("GetPipeline returned nil")
		}
	})

	t.Run("reliability", func(t *testing.T) {
		provider := NewMockProvider()
		synapse := Transform("test", provider, WithRetry(3))

		pipeline := synapse.GetPipeline()
		if pipeline == nil {
			t.Error("GetPipeline returned nil with retry option")
		}
	})

	t.Run("chaining", func(t *testing.T) {
		provider := NewMockProvider()
		synapse := Transform("test", provider)

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

func TestTransformSynapse_Fire(t *testing.T) {
	t.Run("simple", func(t *testing.T) {
		provider := NewMockProviderWithResponse(`{"output": "summary text", "confidence": 0.9, "changes": [], "reasoning": ["test"]}`)
		synapse := Transform("summarize", provider)

		ctx := context.Background()
		result, err := synapse.Fire(ctx, "long text to summarize")
		if err != nil {
			t.Fatalf("Fire failed: %v", err)
		}
		if result != "summary text" {
			t.Errorf("Expected output='summary text', got '%s'", result)
		}
	})

	t.Run("reliability", func(t *testing.T) {
		provider := NewMockProviderWithResponse(`{"output": "test output", "confidence": 0.8, "changes": [], "reasoning": ["test"]}`)
		synapse := Transform("test", provider,
			WithRetry(2),
			WithTimeout(5*time.Second))

		ctx := context.Background()
		result, err := synapse.Fire(ctx, "test input")
		if err != nil {
			t.Fatalf("Fire with reliability options failed: %v", err)
		}
		if result != "test output" {
			t.Errorf("Expected output='test output', got '%s'", result)
		}
	})

	t.Run("chaining", func(t *testing.T) {
		failing := NewMockProviderWithError("primary failed")
		fallbackProvider := NewMockProviderWithResponse(`{"output": "fallback output", "confidence": 0.7, "changes": [], "reasoning": ["test"]}`)
		fallbackSynapse := Transform("test", fallbackProvider)

		synapse := Transform("test", failing,
			WithFallback(fallbackSynapse))

		ctx := context.Background()
		result, err := synapse.Fire(ctx, "test")
		if err != nil {
			t.Fatalf("Fire with fallback failed: %v", err)
		}
		if result != "fallback output" {
			t.Error("Expected result from fallback")
		}
	})
}

func TestTransformSynapse_FireWithDetails(t *testing.T) {
	t.Run("simple", func(t *testing.T) {
		provider := NewMockProviderWithResponse(`{"output": "transformed", "confidence": 0.95, "changes": ["change1"], "reasoning": ["reason1"]}`)
		synapse := Transform("test", provider)

		ctx := context.Background()
		response, err := synapse.FireWithDetails(ctx, "input text")
		if err != nil {
			t.Fatalf("FireWithDetails failed: %v", err)
		}
		if response.Output != "transformed" {
			t.Errorf("Expected output='transformed', got '%s'", response.Output)
		}
		if response.Confidence != 0.95 {
			t.Errorf("Expected confidence=0.95, got %f", response.Confidence)
		}
		if len(response.Reasoning) == 0 {
			t.Error("Expected reasoning to be set")
		}
	})

	t.Run("reliability", func(t *testing.T) {
		provider := NewMockProviderWithResponse(`{"output": "test", "confidence": 0.8, "changes": [], "reasoning": ["test"]}`)
		synapse := Transform("test", provider,
			WithRetry(3),
			WithBackoff(2, 10*time.Millisecond))

		ctx := context.Background()
		response, err := synapse.FireWithDetails(ctx, "test")
		if err != nil {
			t.Fatalf("FireWithDetails with backoff failed: %v", err)
		}
		if response.Output != "test" {
			t.Error("Expected output")
		}
	})

	t.Run("chaining", func(t *testing.T) {
		provider := NewMockProviderWithResponse(`{"output": "test", "confidence": 0.9, "changes": [], "reasoning": ["test"]}`)
		synapse := Transform("test", provider)

		ctx := context.Background()
		response, err := synapse.FireWithDetails(ctx, "test")
		if err != nil {
			t.Fatalf("FireWithDetails failed: %v", err)
		}
		if response.Output != "test" {
			t.Error("Expected output")
		}
	})
}

func TestTransformSynapse_FireWithInput(t *testing.T) {
	t.Run("simple", func(t *testing.T) {
		provider := NewMockProviderWithResponse(`{"output": "transformed text", "confidence": 0.9, "changes": [], "reasoning": ["test"]}`)
		synapse := Transform("test", provider)

		ctx := context.Background()
		input := TransformInput{
			Text:    "input text",
			Context: "test context",
		}
		result, err := synapse.FireWithInput(ctx, input)
		if err != nil {
			t.Fatalf("FireWithInput failed: %v", err)
		}
		if result != "transformed text" {
			t.Errorf("Expected output='transformed text', got '%s'", result)
		}
	})

	t.Run("reliability", func(t *testing.T) {
		provider := NewMockProviderWithResponse(`{"output": "test", "confidence": 0.85, "changes": [], "reasoning": ["test"]}`)
		synapse := Transform("test", provider,
			WithCircuitBreaker(5, 30*time.Second))

		ctx := context.Background()
		input := TransformInput{
			Text:        "test",
			Temperature: 0.3,
		}
		result, err := synapse.FireWithInput(ctx, input)
		if err != nil {
			t.Fatalf("FireWithInput with circuit breaker failed: %v", err)
		}
		if result != "test" {
			t.Error("Expected result")
		}
	})

	t.Run("chaining", func(t *testing.T) {
		provider := NewMockProviderWithResponse(`{"output": "test", "confidence": 0.9, "changes": [], "reasoning": ["test"]}`)
		synapse := Transform("test", provider)

		ctx := context.Background()
		input := TransformInput{
			Text:    "test",
			Context: "context",
		}
		result, err := synapse.FireWithInput(ctx, input)
		if err != nil {
			t.Fatalf("FireWithInput failed: %v", err)
		}
		if result != "test" {
			t.Error("Expected result")
		}
	})
}

func TestTransformSynapse_FireWithInputDetails(t *testing.T) {
	t.Run("simple", func(t *testing.T) {
		provider := NewMockProviderWithResponse(`{"output": "detailed", "confidence": 0.9, "changes": ["c1"], "reasoning": ["r1"]}`)
		synapse := Transform("test", provider)

		ctx := context.Background()
		input := TransformInput{
			Text: "input",
		}
		response, err := synapse.FireWithInputDetails(ctx, input)
		if err != nil {
			t.Fatalf("FireWithInputDetails failed: %v", err)
		}
		if response.Output != "detailed" {
			t.Error("Expected output")
		}
	})

	t.Run("reliability", func(t *testing.T) {
		provider := NewMockProviderWithResponse(`{"output": "test", "confidence": 0.8, "changes": [], "reasoning": ["test"]}`)
		synapse := Transform("test", provider, WithRetry(3))

		ctx := context.Background()
		input := TransformInput{
			Text:        "test",
			Temperature: 0.7,
		}
		response, err := synapse.FireWithInputDetails(ctx, input)
		if err != nil {
			t.Fatalf("FireWithInputDetails with retry failed: %v", err)
		}
		if response.Output != "test" {
			t.Error("Expected output")
		}
	})

	t.Run("chaining", func(t *testing.T) {
		provider := NewMockProviderWithResponse(`{"output": "test", "confidence": 0.9, "changes": [], "reasoning": ["test"]}`)
		synapse := Transform("test", provider)

		ctx := context.Background()
		input := TransformInput{
			Text:    "test",
			Context: "context",
		}
		response, err := synapse.FireWithInputDetails(ctx, input)
		if err != nil {
			t.Fatalf("FireWithInputDetails failed: %v", err)
		}
		if response.Output != "test" {
			t.Error("Expected output")
		}
	})
}

func TestTransformSynapse_mergeInputs(t *testing.T) {
	t.Run("simple", func(t *testing.T) {
		synapse := &TransformSynapse{
			defaults: TransformInput{
				Context: "default context",
			},
		}

		input := TransformInput{
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
		synapse := &TransformSynapse{
			defaults: TransformInput{
				Context:     "default",
				Temperature: 0.5,
			},
		}

		input := TransformInput{
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
		synapse := &TransformSynapse{
			defaults: TransformInput{
				Context: "default",
				Style:   "formal",
			},
		}

		input := TransformInput{
			Text:  "test",
			Style: "casual",
		}
		merged := synapse.mergeInputs(input)

		if merged.Style != "casual" {
			t.Error("Input should override default style")
		}
		if merged.Context != "default" {
			t.Error("Should keep default context when not overridden")
		}
	})
}

func TestTransformSynapse_buildPrompt(t *testing.T) {
	t.Run("simple", func(t *testing.T) {
		synapse := &TransformSynapse{
			instruction: "summarize",
		}

		input := TransformInput{
			Text: "text to transform",
		}
		prompt := synapse.buildPrompt(input)

		if prompt.Task != "Transform: summarize" {
			t.Errorf("Expected task prefix, got '%s'", prompt.Task)
		}
		if prompt.Input != "text to transform" {
			t.Errorf("Expected input to be set, got '%s'", prompt.Input)
		}
		if prompt.Schema == "" {
			t.Error("Expected schema to be set")
		}
	})

	t.Run("reliability", func(t *testing.T) {
		synapse := &TransformSynapse{
			instruction: "test",
		}

		input := TransformInput{
			Text:    "test",
			Context: "transform context",
			Style:   "concise",
			Examples: map[string]string{
				"verbose": "brief",
			},
		}
		prompt := synapse.buildPrompt(input)

		if prompt.Context != "transform context" {
			t.Error("Expected context to be set")
		}
		if len(prompt.Examples) == 0 {
			t.Error("Expected examples to be set")
		}
	})

	t.Run("chaining", func(t *testing.T) {
		synapse := &TransformSynapse{
			instruction: "test",
		}

		input := TransformInput{
			Text: "test",
		}
		prompt := synapse.buildPrompt(input)

		if err := prompt.Validate(); err != nil {
			t.Errorf("Built prompt failed validation: %v", err)
		}
	})
}
