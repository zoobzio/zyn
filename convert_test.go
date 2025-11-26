package zyn

import (
	"context"
	"testing"
	"time"
)

type SimpleInput struct {
	Value int    `json:"value"`
	Name  string `json:"name"`
}

type SimpleOutput struct {
	Count  int    `json:"count"`
	Label  string `json:"label"`
	Active bool   `json:"active"`
}

func (SimpleOutput) Validate() error {
	return nil
}

func TestConvert(t *testing.T) {
	t.Run("simple", func(t *testing.T) {
		provider := NewMockProvider()
		synapse, err := Convert[SimpleInput, SimpleOutput]("convert data", provider)
		if err != nil {
			t.Fatalf("failed to create synapse: %v", err)
		}

		if synapse == nil {
			t.Fatal("Convert wrapper returned nil")
		}
	})

	t.Run("reliability", func(t *testing.T) {
		provider := NewMockProvider()
		synapse, err := Convert[SimpleInput, SimpleOutput]("convert data", provider,
			WithRetry(3),
			WithTimeout(10*time.Second))
		if err != nil {
			t.Fatalf("failed to create synapse: %v", err)
		}

		if synapse == nil {
			t.Fatal("Convert wrapper with options returned nil")
		}

		ctx := context.Background()
		_, err = synapse.Fire(ctx, NewSession(), SimpleInput{Value: 42, Name: "test"})
		if err != nil {
			t.Errorf("Convert synapse Fire failed: %v", err)
		}
	})

	t.Run("chaining", func(t *testing.T) {
		provider := NewMockProviderWithResponse(`{"count": 100, "label": "converted", "active": true}`)
		synapse, err := Convert[SimpleInput, SimpleOutput]("convert data", provider)
		if err != nil {
			t.Fatalf("failed to create synapse: %v", err)
		}

		ctx := context.Background()
		result, err := synapse.Fire(ctx, NewSession(), SimpleInput{Value: 42, Name: "test"})
		if err != nil {
			t.Fatalf("Convert with chaining failed: %v", err)
		}
		if result.Count != 100 {
			t.Errorf("Expected count=100, got %d", result.Count)
		}
	})
}

func TestConvertSynapse_GetPipeline(t *testing.T) {
	t.Run("simple", func(t *testing.T) {
		provider := NewMockProvider()
		synapse, err := Convert[SimpleInput, SimpleOutput]("test", provider)
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
		synapse, err := Convert[SimpleInput, SimpleOutput]("test", provider, WithRetry(3))
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
		synapse, err := Convert[SimpleInput, SimpleOutput]("test", provider)
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

func TestConvertSynapse_Fire(t *testing.T) {
	t.Run("simple", func(t *testing.T) {
		provider := NewMockProviderWithResponse(`{"count": 42, "label": "test", "active": true}`)
		synapse, err := Convert[SimpleInput, SimpleOutput]("convert data", provider)
		if err != nil {
			t.Fatalf("failed to create synapse: %v", err)
		}

		ctx := context.Background()
		result, err := synapse.Fire(ctx, NewSession(), SimpleInput{Value: 10, Name: "input"})
		if err != nil {
			t.Fatalf("Fire failed: %v", err)
		}
		if result.Count != 42 {
			t.Errorf("Expected count=42, got %d", result.Count)
		}
	})

	t.Run("reliability", func(t *testing.T) {
		provider := NewMockProviderWithResponse(`{"count": 1, "label": "test", "active": false}`)
		synapse, err := Convert[SimpleInput, SimpleOutput]("test", provider,
			WithRetry(2),
			WithTimeout(5*time.Second))
		if err != nil {
			t.Fatalf("failed to create synapse: %v", err)
		}

		ctx := context.Background()
		result, err := synapse.Fire(ctx, NewSession(), SimpleInput{Value: 1, Name: "test"})
		if err != nil {
			t.Fatalf("Fire with reliability options failed: %v", err)
		}
		if result.Count != 1 {
			t.Errorf("Expected count=1, got %d", result.Count)
		}
	})

	t.Run("chaining", func(t *testing.T) {
		failing := NewMockProviderWithError("primary failed")
		fallbackProvider := NewMockProviderWithResponse(`{"count": 99, "label": "fallback", "active": true}`)
		fallbackSynapse, err := Convert[SimpleInput, SimpleOutput]("test", fallbackProvider)
		if err != nil {
			t.Fatalf("failed to create fallback synapse: %v", err)
		}

		synapse, err := Convert[SimpleInput, SimpleOutput]("test", failing,
			WithFallback(fallbackSynapse))
		if err != nil {
			t.Fatalf("failed to create synapse: %v", err)
		}

		ctx := context.Background()
		result, err := synapse.Fire(ctx, NewSession(), SimpleInput{Value: 1, Name: "test"})
		if err != nil {
			t.Fatalf("Fire with fallback failed: %v", err)
		}
		if result.Count != 99 {
			t.Error("Expected result from fallback")
		}
	})
}

func TestConvertSynapse_FireWithInput(t *testing.T) {
	t.Run("simple", func(t *testing.T) {
		provider := NewMockProviderWithResponse(`{"count": 50, "label": "converted", "active": true}`)
		synapse, err := Convert[SimpleInput, SimpleOutput]("test", provider)
		if err != nil {
			t.Fatalf("failed to create synapse: %v", err)
		}

		ctx := context.Background()
		input := ConvertInput[SimpleInput]{
			Data:    SimpleInput{Value: 10, Name: "test"},
			Context: "test context",
		}
		result, err := synapse.FireWithInput(ctx, NewSession(), input)
		if err != nil {
			t.Fatalf("FireWithInput failed: %v", err)
		}
		if result.Count != 50 {
			t.Errorf("Expected count=50, got %d", result.Count)
		}
	})

	t.Run("reliability", func(t *testing.T) {
		provider := NewMockProviderWithResponse(`{"count": 1, "label": "test", "active": true}`)
		synapse, err := Convert[SimpleInput, SimpleOutput]("test", provider,
			WithCircuitBreaker(5, 30*time.Second))
		if err != nil {
			t.Fatalf("failed to create synapse: %v", err)
		}

		ctx := context.Background()
		input := ConvertInput[SimpleInput]{
			Data:        SimpleInput{Value: 1, Name: "test"},
			Temperature: 0.3,
		}
		result, err := synapse.FireWithInput(ctx, NewSession(), input)
		if err != nil {
			t.Fatalf("FireWithInput with circuit breaker failed: %v", err)
		}
		if result.Count != 1 {
			t.Error("Expected result")
		}
	})

	t.Run("chaining", func(t *testing.T) {
		provider := NewMockProviderWithResponse(`{"count": 42, "label": "test", "active": true}`)
		defaults := ConvertInput[SimpleInput]{
			Context: "default context",
			Rules:   "default rules",
		}
		synapse, err := Convert[SimpleInput, SimpleOutput]("test", provider)
		if err != nil {
			t.Fatalf("failed to create synapse: %v", err)
		}
		synapse.defaults = defaults

		ctx := context.Background()
		input := ConvertInput[SimpleInput]{
			Data:  SimpleInput{Value: 42, Name: "test"},
			Rules: "override rules",
		}
		result, err := synapse.FireWithInput(ctx, NewSession(), input)
		if err != nil {
			t.Fatalf("FireWithInput with defaults merge failed: %v", err)
		}
		if result.Count != 42 {
			t.Error("Expected result")
		}
	})
}

func TestConvertSynapse_mergeInputs(t *testing.T) {
	t.Run("simple", func(t *testing.T) {
		provider := NewMockProvider()
		synapse, err := Convert[SimpleInput, SimpleOutput]("test", provider)
		if err != nil {
			t.Fatalf("failed to create synapse: %v", err)
		}
		synapse.defaults = ConvertInput[SimpleInput]{
			Context: "default context",
		}

		input := ConvertInput[SimpleInput]{
			Data: SimpleInput{Value: 42, Name: "test"},
		}
		merged := synapse.mergeInputs(input)

		if merged.Data.Value != 42 {
			t.Error("Expected data to be set")
		}
		if merged.Context != "default context" {
			t.Errorf("Expected default context, got '%s'", merged.Context)
		}
	})

	t.Run("reliability", func(t *testing.T) {
		provider := NewMockProvider()
		synapse, err := Convert[SimpleInput, SimpleOutput]("test", provider)
		if err != nil {
			t.Fatalf("failed to create synapse: %v", err)
		}
		synapse.defaults = ConvertInput[SimpleInput]{
			Context:     "default",
			Temperature: 0.5,
		}

		input := ConvertInput[SimpleInput]{
			Data:        SimpleInput{Value: 1, Name: "test"},
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
		synapse, err := Convert[SimpleInput, SimpleOutput]("test", provider)
		if err != nil {
			t.Fatalf("failed to create synapse: %v", err)
		}
		synapse.defaults = ConvertInput[SimpleInput]{
			Context: "default",
			Rules:   "default rules",
		}

		input := ConvertInput[SimpleInput]{
			Data:  SimpleInput{Value: 42, Name: "test"},
			Rules: "override rules",
		}
		merged := synapse.mergeInputs(input)

		if merged.Rules != "override rules" {
			t.Error("Input should override default rules")
		}
		if merged.Context != "default" {
			t.Error("Should keep default context when not overridden")
		}
	})
}

func TestConvertSynapse_buildPrompt(t *testing.T) {
	t.Run("simple", func(t *testing.T) {
		provider := NewMockProvider()
		synapse, err := Convert[SimpleInput, SimpleOutput]("convert data", provider)
		if err != nil {
			t.Fatalf("failed to create synapse: %v", err)
		}

		input := ConvertInput[SimpleInput]{
			Data: SimpleInput{Value: 42, Name: "test"},
		}
		prompt := synapse.buildPrompt(input)

		if prompt.Task != "Convert: convert data" {
			t.Errorf("Expected task prefix, got '%s'", prompt.Task)
		}
		if prompt.Input == "" {
			t.Error("Expected input to be set")
		}
		if prompt.Schema == "" {
			t.Error("Expected schema to be set")
		}
	})

	t.Run("reliability", func(t *testing.T) {
		provider := NewMockProvider()
		synapse, err := Convert[SimpleInput, SimpleOutput]("test", provider)
		if err != nil {
			t.Fatalf("failed to create synapse: %v", err)
		}

		input := ConvertInput[SimpleInput]{
			Data:    SimpleInput{Value: 1, Name: "test"},
			Context: "conversion context",
			Rules:   "apply rules",
		}
		prompt := synapse.buildPrompt(input)

		if prompt.Context != "conversion context" {
			t.Error("Expected context to be set")
		}
		if len(prompt.Constraints) == 0 {
			t.Error("Expected constraints to be set")
		}
	})

	t.Run("chaining", func(t *testing.T) {
		provider := NewMockProvider()
		synapse, err := Convert[SimpleInput, SimpleOutput]("test", provider)
		if err != nil {
			t.Fatalf("failed to create synapse: %v", err)
		}

		input := ConvertInput[SimpleInput]{
			Data: SimpleInput{Value: 42, Name: "test"},
		}
		prompt := synapse.buildPrompt(input)

		if err := prompt.Validate(); err != nil {
			t.Errorf("Built prompt failed validation: %v", err)
		}
	})
}
