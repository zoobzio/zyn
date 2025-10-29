package zyn

import (
	"context"
	"testing"
	"time"
)

type ExtractData struct {
	Name  string   `json:"name"`
	Value int      `json:"value"`
	Items []string `json:"items"`
}

func (ExtractData) Validate() error {
	return nil
}

func TestNewExtraction(t *testing.T) {
	t.Run("simple", func(t *testing.T) {
		provider := NewMockProvider()
		synapse := NewExtraction[ExtractData]("extract data", provider)

		if synapse == nil {
			t.Fatal("Expected synapse to be created")
		}
		if synapse.what != "extract data" {
			t.Errorf("Expected what='extract data', got '%s'", synapse.what)
		}
	})

	t.Run("reliability", func(t *testing.T) {
		provider := NewMockProvider()
		synapse := NewExtraction[ExtractData]("extract data", provider,
			WithRetry(3),
			WithTimeout(10*time.Second))

		if synapse == nil {
			t.Fatal("Expected synapse with reliability options to be created")
		}
	})

	t.Run("chaining", func(t *testing.T) {
		primary := NewMockProviderWithName("primary")
		fallback := NewMockProviderWithName("fallback")
		fallbackSynapse := NewExtraction[ExtractData]("extract data", fallback)

		synapse := NewExtraction[ExtractData]("extract data", primary,
			WithFallback(fallbackSynapse))

		if synapse == nil {
			t.Fatal("Expected synapse with fallback to be created")
		}
	})
}

func TestExtractionSynapse_GetPipeline(t *testing.T) {
	t.Run("simple", func(t *testing.T) {
		provider := NewMockProvider()
		synapse := NewExtraction[ExtractData]("test", provider)

		pipeline := synapse.GetPipeline()
		if pipeline == nil {
			t.Error("GetPipeline returned nil")
		}
	})

	t.Run("reliability", func(t *testing.T) {
		provider := NewMockProvider()
		synapse := NewExtraction[ExtractData]("test", provider, WithRetry(3))

		pipeline := synapse.GetPipeline()
		if pipeline == nil {
			t.Error("GetPipeline returned nil with retry option")
		}
	})

	t.Run("chaining", func(t *testing.T) {
		provider := NewMockProvider()
		synapse := NewExtraction[ExtractData]("test", provider)

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

func TestExtractionSynapse_WithDefaults(t *testing.T) {
	t.Run("simple", func(t *testing.T) {
		provider := NewMockProvider()
		synapse := NewExtraction[ExtractData]("test", provider)

		defaults := ExtractionInput{
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
		synapse := NewExtraction[ExtractData]("test", provider, WithRetry(3))

		defaults := ExtractionInput{Temperature: 0.7}
		synapseWithDefaults := synapse.WithDefaults(defaults)

		if synapseWithDefaults == nil {
			t.Error("WithDefaults returned nil with retry option")
		}
	})

	t.Run("chaining", func(t *testing.T) {
		provider := NewMockProviderWithResponse(`{"name": "test", "value": 42, "items": ["a", "b"]}`)
		synapse := NewExtraction[ExtractData]("test", provider).
			WithDefaults(ExtractionInput{Context: "default", Temperature: 0.5})

		ctx := context.Background()
		result, err := synapse.Fire(ctx, "test text")
		if err != nil {
			t.Errorf("Fire failed with defaults: %v", err)
		}
		if result.Name != "test" {
			t.Errorf("Expected name='test', got '%s'", result.Name)
		}
	})
}

func TestExtractionSynapse_Fire(t *testing.T) {
	t.Run("simple", func(t *testing.T) {
		provider := NewMockProviderWithResponse(`{"name": "extracted", "value": 100, "items": ["item1"]}`)
		synapse := NewExtraction[ExtractData]("extract data", provider)

		ctx := context.Background()
		result, err := synapse.Fire(ctx, "Some text with data to extract")
		if err != nil {
			t.Fatalf("Fire failed: %v", err)
		}
		if result.Name != "extracted" {
			t.Errorf("Expected name='extracted', got '%s'", result.Name)
		}
	})

	t.Run("reliability", func(t *testing.T) {
		provider := NewMockProviderWithResponse(`{"name": "test", "value": 1, "items": []}`)
		synapse := NewExtraction[ExtractData]("test", provider,
			WithRetry(2),
			WithTimeout(5*time.Second))

		ctx := context.Background()
		result, err := synapse.Fire(ctx, "test input")
		if err != nil {
			t.Fatalf("Fire with reliability options failed: %v", err)
		}
		if result.Name != "test" {
			t.Errorf("Expected name='test', got '%s'", result.Name)
		}
	})

	t.Run("chaining", func(t *testing.T) {
		failing := NewMockProviderWithError("primary failed")
		fallbackProvider := NewMockProviderWithResponse(`{"name": "fallback", "value": 99, "items": ["fb"]}`)
		fallbackSynapse := NewExtraction[ExtractData]("test", fallbackProvider)

		synapse := NewExtraction[ExtractData]("test", failing,
			WithFallback(fallbackSynapse))

		ctx := context.Background()
		result, err := synapse.Fire(ctx, "test")
		if err != nil {
			t.Fatalf("Fire with fallback failed: %v", err)
		}
		if result.Name != "fallback" {
			t.Error("Expected result from fallback")
		}
	})
}

func TestExtractionSynapse_FireWithInput(t *testing.T) {
	t.Run("simple", func(t *testing.T) {
		provider := NewMockProviderWithResponse(`{"name": "extracted", "value": 50, "items": ["a", "b"]}`)
		synapse := NewExtraction[ExtractData]("test", provider)

		ctx := context.Background()
		input := ExtractionInput{
			Text:    "Text to extract from",
			Context: "test context",
		}
		result, err := synapse.FireWithInput(ctx, input)
		if err != nil {
			t.Fatalf("FireWithInput failed: %v", err)
		}
		if result.Name != "extracted" {
			t.Errorf("Expected name='extracted', got '%s'", result.Name)
		}
	})

	t.Run("reliability", func(t *testing.T) {
		provider := NewMockProviderWithResponse(`{"name": "test", "value": 1, "items": []}`)
		synapse := NewExtraction[ExtractData]("test", provider,
			WithCircuitBreaker(5, 30*time.Second))

		ctx := context.Background()
		input := ExtractionInput{
			Text:        "test",
			Temperature: 0.3,
		}
		result, err := synapse.FireWithInput(ctx, input)
		if err != nil {
			t.Fatalf("FireWithInput with circuit breaker failed: %v", err)
		}
		if result.Name != "test" {
			t.Error("Expected result")
		}
	})

	t.Run("chaining", func(t *testing.T) {
		provider := NewMockProviderWithResponse(`{"name": "test", "value": 42, "items": ["x"]}`)
		defaults := ExtractionInput{
			Context:  "default context",
			Examples: "default examples",
		}
		synapse := NewExtraction[ExtractData]("test", provider).WithDefaults(defaults)

		ctx := context.Background()
		input := ExtractionInput{
			Text:     "test text",
			Examples: "override examples",
		}
		result, err := synapse.FireWithInput(ctx, input)
		if err != nil {
			t.Fatalf("FireWithInput with defaults merge failed: %v", err)
		}
		if result.Name != "test" {
			t.Error("Expected result")
		}
	})
}

func TestExtractionSynapse_mergeInputs(t *testing.T) {
	t.Run("simple", func(t *testing.T) {
		provider := NewMockProvider()
		synapse := NewExtraction[ExtractData]("test", provider)
		synapse.defaults = ExtractionInput{
			Context: "default context",
		}

		input := ExtractionInput{
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
		synapse := NewExtraction[ExtractData]("test", provider)
		synapse.defaults = ExtractionInput{
			Context:     "default",
			Temperature: 0.5,
		}

		input := ExtractionInput{
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
		synapse := NewExtraction[ExtractData]("test", provider)
		synapse.defaults = ExtractionInput{
			Context:  "default",
			Examples: "default examples",
		}

		input := ExtractionInput{
			Text:     "test",
			Examples: "override examples",
		}
		merged := synapse.mergeInputs(input)

		if merged.Examples != "override examples" {
			t.Error("Input should override default examples")
		}
		if merged.Context != "default" {
			t.Error("Should keep default context when not overridden")
		}
	})
}

func TestExtractionSynapse_buildPrompt(t *testing.T) {
	t.Run("simple", func(t *testing.T) {
		provider := NewMockProvider()
		synapse := NewExtraction[ExtractData]("extract data", provider)

		input := ExtractionInput{
			Text: "text to extract from",
		}
		prompt := synapse.buildPrompt(input)

		if prompt.Task != "Extract extract data" {
			t.Errorf("Expected task prefix, got '%s'", prompt.Task)
		}
		if prompt.Input != "text to extract from" {
			t.Errorf("Expected input to be set, got '%s'", prompt.Input)
		}
		if prompt.Schema == "" {
			t.Error("Expected schema to be set")
		}
	})

	t.Run("reliability", func(t *testing.T) {
		provider := NewMockProvider()
		synapse := NewExtraction[ExtractData]("test", provider)

		input := ExtractionInput{
			Text:     "test",
			Context:  "extraction context",
			Examples: "example1\nexample2",
		}
		prompt := synapse.buildPrompt(input)

		if prompt.Context != "extraction context" {
			t.Error("Expected context to be set")
		}
		if len(prompt.Examples["examples"]) == 0 {
			t.Error("Expected examples to be set")
		}
		if len(prompt.Constraints) == 0 {
			t.Error("Expected constraints to be set")
		}
	})

	t.Run("chaining", func(t *testing.T) {
		provider := NewMockProvider()
		synapse := NewExtraction[ExtractData]("test", provider)

		input := ExtractionInput{
			Text: "test",
		}
		prompt := synapse.buildPrompt(input)

		if err := prompt.Validate(); err != nil {
			t.Errorf("Built prompt failed validation: %v", err)
		}
	})
}

func TestExtract(t *testing.T) {
	t.Run("simple", func(t *testing.T) {
		provider := NewMockProvider()
		synapse := Extract[ExtractData]("extract data", provider)

		if synapse == nil {
			t.Fatal("Extract wrapper returned nil")
		}
	})

	t.Run("reliability", func(t *testing.T) {
		provider := NewMockProvider()
		synapse := Extract[ExtractData]("extract data", provider,
			WithRetry(3),
			WithTimeout(10*time.Second))

		if synapse == nil {
			t.Fatal("Extract wrapper with options returned nil")
		}

		ctx := context.Background()
		_, err := synapse.Fire(ctx, "test text")
		if err != nil {
			t.Errorf("Extract synapse Fire failed: %v", err)
		}
	})

	t.Run("chaining", func(t *testing.T) {
		provider := NewMockProviderWithResponse(`{"name": "test", "value": 42, "items": ["x"]}`)
		synapse := Extract[ExtractData]("extract data", provider).
			WithDefaults(ExtractionInput{Context: "test context"})

		ctx := context.Background()
		result, err := synapse.Fire(ctx, "test text")
		if err != nil {
			t.Fatalf("Extract with chaining failed: %v", err)
		}
		if result.Name != "test" {
			t.Errorf("Expected name='test', got '%s'", result.Name)
		}
	})
}
