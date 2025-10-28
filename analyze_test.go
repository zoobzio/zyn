package zyn

import (
	"context"
	"testing"
	"time"
)

type TestData struct {
	Value int    `json:"value"`
	Name  string `json:"name"`
}

func TestAnalyze(t *testing.T) {
	t.Run("simple", func(t *testing.T) {
		provider := NewMockProvider()
		synapse := Analyze[TestData]("data quality", provider)

		if synapse == nil {
			t.Fatal("Analyze wrapper returned nil")
		}
	})

	t.Run("reliability", func(t *testing.T) {
		provider := NewMockProvider()
		synapse := Analyze[TestData]("data quality", provider,
			WithRetry(3),
			WithTimeout(10*time.Second))

		if synapse == nil {
			t.Fatal("Analyze wrapper with options returned nil")
		}

		ctx := context.Background()
		_, err := synapse.Fire(ctx, TestData{Value: 42, Name: "test"})
		if err != nil {
			t.Errorf("Analyze synapse Fire failed: %v", err)
		}
	})

	t.Run("chaining", func(t *testing.T) {
		provider := NewMockProviderWithResponse(`{"analysis": "test result", "confidence": 0.9, "findings": [], "reasoning": ["test"]}`)
		synapse := Analyze[TestData]("data quality", provider)

		ctx := context.Background()
		result, err := synapse.Fire(ctx, TestData{Value: 42, Name: "test"})
		if err != nil {
			t.Fatalf("Analyze with chaining failed: %v", err)
		}
		if result == "" {
			t.Error("Expected non-empty result")
		}
	})
}

func TestAnalyzeSynapse_GetPipeline(t *testing.T) {
	t.Run("simple", func(t *testing.T) {
		provider := NewMockProvider()
		synapse := Analyze[TestData]("test", provider)

		pipeline := synapse.GetPipeline()
		if pipeline == nil {
			t.Error("GetPipeline returned nil")
		}
	})

	t.Run("reliability", func(t *testing.T) {
		provider := NewMockProvider()
		synapse := Analyze[TestData]("test", provider, WithRetry(3))

		pipeline := synapse.GetPipeline()
		if pipeline == nil {
			t.Error("GetPipeline returned nil with retry option")
		}
	})

	t.Run("chaining", func(t *testing.T) {
		provider := NewMockProvider()
		synapse := Analyze[TestData]("test", provider)

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

func TestAnalyzeSynapse_Fire(t *testing.T) {
	t.Run("simple", func(t *testing.T) {
		provider := NewMockProviderWithResponse(`{"analysis": "valid data", "confidence": 0.9, "findings": [], "reasoning": ["test"]}`)
		synapse := Analyze[TestData]("data quality", provider)

		ctx := context.Background()
		result, err := synapse.Fire(ctx, TestData{Value: 42, Name: "test"})
		if err != nil {
			t.Fatalf("Fire failed: %v", err)
		}
		if result == "" {
			t.Error("Expected non-empty analysis")
		}
	})

	t.Run("reliability", func(t *testing.T) {
		provider := NewMockProviderWithResponse(`{"analysis": "test", "confidence": 0.9, "findings": [], "reasoning": ["test"]}`)
		synapse := Analyze[TestData]("test", provider,
			WithRetry(2),
			WithTimeout(5*time.Second))

		ctx := context.Background()
		result, err := synapse.Fire(ctx, TestData{Value: 1, Name: "test"})
		if err != nil {
			t.Fatalf("Fire with reliability options failed: %v", err)
		}
		if result == "" {
			t.Error("Expected non-empty result")
		}
	})

	t.Run("chaining", func(t *testing.T) {
		failing := NewMockProviderWithError("primary failed")
		fallbackProvider := NewMockProviderWithResponse(`{"analysis": "fallback analysis", "confidence": 0.8, "findings": [], "reasoning": ["fallback"]}`)
		fallbackSynapse := Analyze[TestData]("test", fallbackProvider)

		synapse := Analyze[TestData]("test", failing,
			WithFallback(fallbackSynapse))

		ctx := context.Background()
		result, err := synapse.Fire(ctx, TestData{Value: 1, Name: "test"})
		if err != nil {
			t.Fatalf("Fire with fallback failed: %v", err)
		}
		if result == "" {
			t.Error("Expected result from fallback")
		}
	})
}

func TestAnalyzeSynapse_FireWithDetails(t *testing.T) {
	t.Run("simple", func(t *testing.T) {
		provider := NewMockProviderWithResponse(`{"analysis": "detailed analysis", "confidence": 0.95, "findings": ["finding1"], "reasoning": ["reason1"]}`)
		synapse := Analyze[TestData]("test", provider)

		ctx := context.Background()
		response, err := synapse.FireWithDetails(ctx, TestData{Value: 42, Name: "test"})
		if err != nil {
			t.Fatalf("FireWithDetails failed: %v", err)
		}
		if response.Analysis == "" {
			t.Error("Expected analysis to be set")
		}
		if response.Confidence != 0.95 {
			t.Errorf("Expected confidence 0.95, got %f", response.Confidence)
		}
	})

	t.Run("reliability", func(t *testing.T) {
		provider := NewMockProviderWithResponse(`{"analysis": "test", "confidence": 0.8, "findings": [], "reasoning": ["test"]}`)
		synapse := Analyze[TestData]("test", provider,
			WithRetry(3),
			WithBackoff(2, 10*time.Millisecond))

		ctx := context.Background()
		response, err := synapse.FireWithDetails(ctx, TestData{Value: 1, Name: "test"})
		if err != nil {
			t.Fatalf("FireWithDetails with backoff failed: %v", err)
		}
		if response.Analysis == "" {
			t.Error("Expected analysis")
		}
	})

	t.Run("chaining", func(t *testing.T) {
		provider := NewMockProviderWithResponse(`{"analysis": "test", "confidence": 0.9, "findings": [], "reasoning": ["test"]}`)
		synapse := Analyze[TestData]("test", provider)

		ctx := context.Background()
		response, err := synapse.FireWithDetails(ctx, TestData{Value: 42, Name: "test"})
		if err != nil {
			t.Fatalf("FireWithDetails failed: %v", err)
		}
		if response.Analysis == "" {
			t.Error("Expected analysis")
		}
	})
}

func TestAnalyzeSynapse_FireWithInput(t *testing.T) {
	t.Run("simple", func(t *testing.T) {
		provider := NewMockProviderWithResponse(`{"analysis": "test analysis", "confidence": 0.9, "findings": [], "reasoning": ["test"]}`)
		synapse := Analyze[TestData]("test", provider)

		ctx := context.Background()
		input := AnalyzeInput[TestData]{
			Data:    TestData{Value: 42, Name: "test"},
			Context: "test context",
		}
		result, err := synapse.FireWithInput(ctx, input)
		if err != nil {
			t.Fatalf("FireWithInput failed: %v", err)
		}
		if result == "" {
			t.Error("Expected non-empty result")
		}
	})

	t.Run("reliability", func(t *testing.T) {
		provider := NewMockProviderWithResponse(`{"analysis": "test", "confidence": 0.9, "findings": [], "reasoning": ["test"]}`)
		synapse := Analyze[TestData]("test", provider,
			WithCircuitBreaker(5, 30*time.Second))

		ctx := context.Background()
		input := AnalyzeInput[TestData]{
			Data:        TestData{Value: 1, Name: "test"},
			Temperature: 0.3,
		}
		result, err := synapse.FireWithInput(ctx, input)
		if err != nil {
			t.Fatalf("FireWithInput with circuit breaker failed: %v", err)
		}
		if result == "" {
			t.Error("Expected result")
		}
	})

	t.Run("chaining", func(t *testing.T) {
		provider := NewMockProviderWithResponse(`{"analysis": "test", "confidence": 0.9, "findings": [], "reasoning": ["test"]}`)
		synapse := Analyze[TestData]("test", provider)

		ctx := context.Background()
		input := AnalyzeInput[TestData]{
			Data:  TestData{Value: 42, Name: "test"},
			Focus: "specific aspect",
		}
		result, err := synapse.FireWithInput(ctx, input)
		if err != nil {
			t.Fatalf("FireWithInput with focus failed: %v", err)
		}
		if result == "" {
			t.Error("Expected result")
		}
	})
}

func TestAnalyzeSynapse_FireWithInputDetails(t *testing.T) {
	t.Run("simple", func(t *testing.T) {
		provider := NewMockProviderWithResponse(`{"analysis": "detailed", "confidence": 0.9, "findings": ["f1"], "reasoning": ["r1"]}`)
		synapse := Analyze[TestData]("test", provider)

		ctx := context.Background()
		input := AnalyzeInput[TestData]{
			Data: TestData{Value: 42, Name: "test"},
		}
		response, err := synapse.FireWithInputDetails(ctx, input)
		if err != nil {
			t.Fatalf("FireWithInputDetails failed: %v", err)
		}
		if response.Analysis == "" {
			t.Error("Expected analysis")
		}
	})

	t.Run("reliability", func(t *testing.T) {
		provider := NewMockProviderWithResponse(`{"analysis": "test", "confidence": 0.8, "findings": [], "reasoning": ["test"]}`)
		synapse := Analyze[TestData]("test", provider, WithRetry(3))

		ctx := context.Background()
		input := AnalyzeInput[TestData]{
			Data:        TestData{Value: 1, Name: "test"},
			Temperature: 0.7,
		}
		response, err := synapse.FireWithInputDetails(ctx, input)
		if err != nil {
			t.Fatalf("FireWithInputDetails with retry failed: %v", err)
		}
		if response.Analysis == "" {
			t.Error("Expected analysis")
		}
	})

	t.Run("chaining", func(t *testing.T) {
		provider := NewMockProviderWithResponse(`{"analysis": "test", "confidence": 0.9, "findings": [], "reasoning": ["test"]}`)
		synapse := Analyze[TestData]("test", provider)

		ctx := context.Background()
		input := AnalyzeInput[TestData]{
			Data:    TestData{Value: 42, Name: "test"},
			Context: "test context",
			Focus:   "specific focus",
		}
		response, err := synapse.FireWithInputDetails(ctx, input)
		if err != nil {
			t.Fatalf("FireWithInputDetails with context and focus failed: %v", err)
		}
		if response.Analysis == "" {
			t.Error("Expected analysis")
		}
	})
}

func TestAnalyzeSynapse_mergeInputs(t *testing.T) {
	t.Run("simple", func(t *testing.T) {
		synapse := &AnalyzeSynapse[TestData]{
			defaults: AnalyzeInput[TestData]{
				Context: "default context",
			},
		}

		input := AnalyzeInput[TestData]{
			Data: TestData{Value: 42, Name: "test"},
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
		synapse := &AnalyzeSynapse[TestData]{
			defaults: AnalyzeInput[TestData]{
				Context:     "default",
				Temperature: 0.5,
			},
		}

		input := AnalyzeInput[TestData]{
			Data:        TestData{Value: 1, Name: "test"},
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
		synapse := &AnalyzeSynapse[TestData]{
			defaults: AnalyzeInput[TestData]{
				Context: "default",
				Focus:   "default focus",
			},
		}

		input := AnalyzeInput[TestData]{
			Data:  TestData{Value: 42, Name: "test"},
			Focus: "override focus",
		}
		merged := synapse.mergeInputs(input)

		if merged.Focus != "override focus" {
			t.Error("Input should override default focus")
		}
		if merged.Context != "default" {
			t.Error("Should keep default context when not overridden")
		}
	})
}

func TestAnalyzeSynapse_buildPrompt(t *testing.T) {
	t.Run("simple", func(t *testing.T) {
		synapse := &AnalyzeSynapse[TestData]{
			what: "data quality",
		}

		input := AnalyzeInput[TestData]{
			Data: TestData{Value: 42, Name: "test"},
		}
		prompt := synapse.buildPrompt(input)

		if prompt.Task != "Analyze: data quality" {
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
		synapse := &AnalyzeSynapse[TestData]{
			what: "test",
		}

		input := AnalyzeInput[TestData]{
			Data:    TestData{Value: 1, Name: "test"},
			Context: "analysis context",
			Focus:   "specific focus",
		}
		prompt := synapse.buildPrompt(input)

		if prompt.Context != "analysis context" {
			t.Error("Expected context to be set")
		}
		if len(prompt.Constraints) == 0 {
			t.Error("Expected constraints to be set")
		}
	})

	t.Run("chaining", func(t *testing.T) {
		synapse := &AnalyzeSynapse[TestData]{
			what: "test",
		}

		input := AnalyzeInput[TestData]{
			Data: TestData{Value: 42, Name: "test"},
		}
		prompt := synapse.buildPrompt(input)

		if err := prompt.Validate(); err != nil {
			t.Errorf("Built prompt failed validation: %v", err)
		}
	})
}
