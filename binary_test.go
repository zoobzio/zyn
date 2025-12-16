package zyn

import (
	"context"
	"testing"
	"time"
)

func TestNewBinary(t *testing.T) {
	t.Run("basic", func(t *testing.T) {
		provider := NewMockProvider()
		synapse, err := NewBinary("Is this valid?", provider)
		if err != nil {
			t.Fatalf("failed to create synapse: %v", err)
		}

		if synapse == nil {
			t.Fatal("Expected synapse to be created")
		}
		if synapse.question != "Is this valid?" {
			t.Errorf("Expected question 'Is this valid?', got '%s'", synapse.question)
		}
	})

	t.Run("with_options", func(t *testing.T) {
		provider := NewMockProvider()
		synapse, err := NewBinary("Is this valid?", provider,
			WithRetry(3),
			WithTimeout(10*time.Second))
		if err != nil {
			t.Fatalf("failed to create synapse: %v", err)
		}

		if synapse == nil {
			t.Fatal("Expected synapse with options to be created")
		}
	})

	t.Run("with_fallback", func(t *testing.T) {
		primary := NewMockProviderWithName("primary")
		fallback := NewMockProviderWithName("fallback")
		fallbackSynapse, err := NewBinary("Is this valid?", fallback)
		if err != nil {
			t.Fatalf("failed to create fallback synapse: %v", err)
		}

		synapse, err := NewBinary("Is this valid?", primary,
			WithFallback(fallbackSynapse))
		if err != nil {
			t.Fatalf("failed to create synapse: %v", err)
		}

		if synapse == nil {
			t.Fatal("Expected synapse with fallback to be created")
		}
	})
}

func TestBinarySynapse_GetPipeline(t *testing.T) {
	provider := NewMockProvider()
	synapse, err := NewBinary("test", provider)
	if err != nil {
		t.Fatalf("failed to create synapse: %v", err)
	}

	pipeline := synapse.GetPipeline()
	if pipeline == nil {
		t.Error("GetPipeline returned nil")
	}
}

func TestBinarySynapse_WithDefaults(t *testing.T) {
	t.Run("sets_defaults", func(t *testing.T) {
		provider := NewMockProvider()
		synapse, err := NewBinary("test", provider)
		if err != nil {
			t.Fatalf("failed to create synapse: %v", err)
		}

		defaults := BinaryInput{
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

	t.Run("method_chaining", func(t *testing.T) {
		provider := NewMockProviderWithResponse(`{"decision": true, "confidence": 0.9, "reasoning": ["test"]}`)
		synapse, err := NewBinary("test", provider)
		if err != nil {
			t.Fatalf("failed to create synapse: %v", err)
		}
		synapse = synapse.WithDefaults(BinaryInput{Context: "default", Temperature: 0.5})

		ctx := context.Background()
		result, err := synapse.Fire(ctx, NewSession(), "test")
		if err != nil {
			t.Errorf("Fire failed with defaults: %v", err)
		}
		if !result {
			t.Error("Expected true result")
		}
	})
}

func TestBinarySynapse_Fire(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		provider := NewMockProviderWithResponse(`{"decision": true, "confidence": 0.9, "reasoning": ["valid"]}`)
		synapse, err := NewBinary("Is this valid?", provider)
		if err != nil {
			t.Fatalf("failed to create synapse: %v", err)
		}

		ctx := context.Background()
		result, err := synapse.Fire(ctx, NewSession(), "user@example.com")
		if err != nil {
			t.Fatalf("Fire failed: %v", err)
		}
		if !result {
			t.Error("Expected true for valid email")
		}
	})

	t.Run("with_retry", func(t *testing.T) {
		provider := NewMockProviderWithResponse(`{"decision": false, "confidence": 0.8, "reasoning": ["invalid"]}`)
		synapse, err := NewBinary("Is this valid?", provider,
			WithRetry(2),
			WithTimeout(5*time.Second))
		if err != nil {
			t.Fatalf("failed to create synapse: %v", err)
		}

		ctx := context.Background()
		result, err := synapse.Fire(ctx, NewSession(), "invalid")
		if err != nil {
			t.Fatalf("Fire with retry failed: %v", err)
		}
		if result {
			t.Error("Expected false for invalid input")
		}
	})

	t.Run("with_fallback", func(t *testing.T) {
		failing := NewMockProviderWithError("primary failed")
		fallbackProvider := NewMockProviderWithResponse(`{"decision": true, "confidence": 0.8, "reasoning": ["fallback"]}`)
		fallbackSynapse, err := NewBinary("Is this valid?", fallbackProvider)
		if err != nil {
			t.Fatalf("failed to create fallback synapse: %v", err)
		}

		synapse, err := NewBinary("Is this valid?", failing,
			WithFallback(fallbackSynapse))
		if err != nil {
			t.Fatalf("failed to create synapse: %v", err)
		}

		ctx := context.Background()
		result, err := synapse.Fire(ctx, NewSession(), "test")
		if err != nil {
			t.Fatalf("Fire with fallback failed: %v", err)
		}
		if !result {
			t.Error("Expected true from fallback")
		}
	})
}

func TestBinarySynapse_FireWithDetails(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		provider := NewMockProviderWithResponse(`{"decision": true, "confidence": 0.95, "reasoning": ["valid format", "known domain"]}`)
		synapse, err := NewBinary("Is this valid?", provider)
		if err != nil {
			t.Fatalf("failed to create synapse: %v", err)
		}

		ctx := context.Background()
		response, err := synapse.FireWithDetails(ctx, NewSession(), "test@example.com")
		if err != nil {
			t.Fatalf("FireWithDetails failed: %v", err)
		}
		if !response.Decision {
			t.Error("Expected decision to be true")
		}
		if response.Confidence != 0.95 {
			t.Errorf("Expected confidence 0.95, got %f", response.Confidence)
		}
		if len(response.Reasoning) != 2 {
			t.Error("Expected reasoning to be set")
		}
	})

	t.Run("with_retry", func(t *testing.T) {
		provider := NewMockProviderWithResponse(`{"decision": false, "confidence": 0.7, "reasoning": ["test"]}`)
		synapse, err := NewBinary("Is this valid?", provider,
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
		if response.Decision {
			t.Error("Expected false decision")
		}
	})
}

func TestBinarySynapse_FireWithInput(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		provider := NewMockProviderWithResponse(`{"decision": true, "confidence": 0.9, "reasoning": ["test"]}`)
		synapse, err := NewBinary("Is this valid?", provider)
		if err != nil {
			t.Fatalf("failed to create synapse: %v", err)
		}

		ctx := context.Background()
		input := BinaryInput{
			Subject: "test input",
			Context: "test context",
		}
		result, err := synapse.FireWithInput(ctx, NewSession(), input)
		if err != nil {
			t.Fatalf("FireWithInput failed: %v", err)
		}
		if !result.Decision {
			t.Error("Expected true result")
		}
	})

	t.Run("with_options", func(t *testing.T) {
		provider := NewMockProviderWithResponse(`{"decision": false, "confidence": 0.85, "reasoning": ["test"]}`)
		synapse, err := NewBinary("test", provider,
			WithCircuitBreaker(5, 30*time.Second))
		if err != nil {
			t.Fatalf("failed to create synapse: %v", err)
		}

		ctx := context.Background()
		input := BinaryInput{
			Subject:     "test",
			Temperature: 0.3,
		}
		result, err := synapse.FireWithInput(ctx, NewSession(), input)
		if err != nil {
			t.Fatalf("FireWithInput with circuit breaker failed: %v", err)
		}
		if result.Decision {
			t.Error("Expected false result")
		}
	})
}

func TestBinarySynapse_FireWithInput_FullResponse(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		provider := NewMockProviderWithResponse(`{"decision": true, "confidence": 0.9, "reasoning": ["r1"]}`)
		synapse, err := NewBinary("test", provider)
		if err != nil {
			t.Fatalf("failed to create synapse: %v", err)
		}

		ctx := context.Background()
		input := BinaryInput{
			Subject: "input",
		}
		response, err := synapse.FireWithInput(ctx, NewSession(), input)
		if err != nil {
			t.Fatalf("FireWithInput failed: %v", err)
		}
		if !response.Decision {
			t.Error("Expected true decision")
		}
	})

	t.Run("with_retry", func(t *testing.T) {
		provider := NewMockProviderWithResponse(`{"decision": false, "confidence": 0.8, "reasoning": ["test"]}`)
		synapse, err := NewBinary("test", provider, WithRetry(3))
		if err != nil {
			t.Fatalf("failed to create synapse: %v", err)
		}

		ctx := context.Background()
		input := BinaryInput{
			Subject:     "test",
			Temperature: 0.7,
		}
		response, err := synapse.FireWithInput(ctx, NewSession(), input)
		if err != nil {
			t.Fatalf("FireWithInput with retry failed: %v", err)
		}
		if response.Decision {
			t.Error("Expected false decision")
		}
	})
}

func TestBinarySynapse_mergeInputs(t *testing.T) {
	t.Run("defaults", func(t *testing.T) {
		synapse := &BinarySynapse{
			defaults: BinaryInput{
				Context: "default context",
			},
		}

		input := BinaryInput{
			Subject: "test text",
		}
		merged := synapse.mergeInputs(input)

		if merged.Subject != "test text" {
			t.Errorf("Expected subject 'test text', got '%s'", merged.Subject)
		}
		if merged.Context != "default context" {
			t.Errorf("Expected default context, got '%s'", merged.Context)
		}
	})

	t.Run("overrides", func(t *testing.T) {
		synapse := &BinarySynapse{
			defaults: BinaryInput{
				Context:     "default",
				Temperature: 0.5,
			},
		}

		input := BinaryInput{
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
}

func TestBinarySynapse_buildPrompt(t *testing.T) {
	t.Run("basic", func(t *testing.T) {
		schema, err := generateJSONSchema[BinaryResponse]()
		if err != nil {
			t.Fatalf("failed to generate schema: %v", err)
		}
		synapse := &BinarySynapse{
			question: "Is this valid?",
			schema:   schema,
		}

		input := BinaryInput{
			Subject: "test input",
		}
		prompt := synapse.buildPrompt(input)

		if prompt.Task != "Determine if Is this valid?" {
			t.Errorf("Expected task prefix, got '%s'", prompt.Task)
		}
		if prompt.Input != "test input" {
			t.Errorf("Expected input to be set, got '%s'", prompt.Input)
		}
		if prompt.Schema == "" {
			t.Error("Expected schema to be set")
		}
	})

	t.Run("with_context", func(t *testing.T) {
		schema, err := generateJSONSchema[BinaryResponse]()
		if err != nil {
			t.Fatalf("failed to generate schema: %v", err)
		}
		synapse := &BinarySynapse{
			question: "test",
			schema:   schema,
		}

		input := BinaryInput{
			Subject:  "test",
			Context:  "additional context",
			Examples: []string{"example1", "example2"},
		}
		prompt := synapse.buildPrompt(input)

		if prompt.Context != "additional context" {
			t.Error("Expected context to be set")
		}
		if len(prompt.Examples) == 0 {
			t.Error("Expected examples to be set")
		}
		if err := prompt.Validate(); err != nil {
			t.Errorf("Built prompt failed validation: %v", err)
		}
	})
}

func TestBinary(t *testing.T) {
	t.Run("basic", func(t *testing.T) {
		provider := NewMockProvider()
		synapse, err := Binary("Is this valid?", provider)
		if err != nil {
			t.Fatalf("failed to create synapse: %v", err)
		}

		if synapse == nil {
			t.Fatal("Binary wrapper returned nil")
		}
	})

	t.Run("with_options", func(t *testing.T) {
		provider := NewMockProviderWithResponse(`{"decision": true, "confidence": 0.9, "reasoning": ["test"]}`)
		synapse, err := Binary("Is this valid?", provider,
			WithRetry(3),
			WithTimeout(10*time.Second))
		if err != nil {
			t.Fatalf("failed to create synapse: %v", err)
		}

		if synapse == nil {
			t.Fatal("Binary wrapper with options returned nil")
		}

		ctx := context.Background()
		_, err = synapse.Fire(ctx, NewSession(), "test text")
		if err != nil {
			t.Errorf("Binary synapse Fire failed: %v", err)
		}
	})
}

func TestBinaryResponse_Validate(t *testing.T) {
	t.Run("valid_response", func(t *testing.T) {
		r := BinaryResponse{
			Decision:   true,
			Confidence: 0.9,
			Reasoning:  []string{"valid reason"},
		}
		if err := r.Validate(); err != nil {
			t.Errorf("expected valid response, got error: %v", err)
		}
	})

	t.Run("confidence_too_low", func(t *testing.T) {
		r := BinaryResponse{
			Decision:   true,
			Confidence: -0.1,
			Reasoning:  []string{"reason"},
		}
		err := r.Validate()
		if err == nil {
			t.Error("expected error for negative confidence")
		}
	})

	t.Run("confidence_too_high", func(t *testing.T) {
		r := BinaryResponse{
			Decision:   true,
			Confidence: 1.5,
			Reasoning:  []string{"reason"},
		}
		err := r.Validate()
		if err == nil {
			t.Error("expected error for confidence > 1")
		}
	})

	t.Run("empty_reasoning", func(t *testing.T) {
		r := BinaryResponse{
			Decision:   true,
			Confidence: 0.9,
			Reasoning:  []string{},
		}
		err := r.Validate()
		if err == nil {
			t.Error("expected error for empty reasoning")
		}
	})
}
