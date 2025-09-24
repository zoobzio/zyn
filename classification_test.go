package zyn

import (
	"context"
	"strings"
	"testing"
	"time"
)

func TestClassificationBasic(t *testing.T) {
	provider := NewMockProviderWithResponse(`{
		"primary": "database",
		"secondary": "network",
		"confidence": 0.85,
		"reasoning": ["Port 5432 is PostgreSQL default", "Connection refused indicates database not running", "Network could be secondary issue"]
	}`)

	classifier := Classification(
		"What type of error is this?",
		[]string{"network", "database", "authentication", "validation", "unknown"},
		provider,
		WithTimeout(5*time.Second),
	)

	ctx := context.Background()

	// Test simple Fire
	category, err := classifier.Fire(ctx, "Connection refused on port 5432")
	if err != nil {
		t.Fatalf("Fire failed: %v", err)
	}
	if category != "database" {
		t.Errorf("Expected 'database', got '%s'", category)
	}

	// Test FireWithDetails
	response, err := classifier.FireWithDetails(ctx, "Connection refused on port 5432")
	if err != nil {
		t.Fatalf("FireWithDetails failed: %v", err)
	}
	if response.Primary != "database" {
		t.Errorf("Expected primary='database', got '%s'", response.Primary)
	}
	if response.Secondary != "network" {
		t.Errorf("Expected secondary='network', got '%s'", response.Secondary)
	}
	if response.Confidence != 0.85 {
		t.Errorf("Expected confidence=0.85, got %f", response.Confidence)
	}
	if len(response.Reasoning) != 3 {
		t.Errorf("Expected 3 reasoning steps, got %d", len(response.Reasoning))
	}
}

func TestClassificationWithExamples(t *testing.T) {
	provider := NewMockProviderWithResponse(`{
		"primary": "positive",
		"secondary": "",
		"confidence": 0.95,
		"reasoning": ["Matches positive examples", "Contains enthusiastic language"]
	}`)

	classifier := Classification(
		"What is the sentiment?",
		[]string{"positive", "negative", "neutral"},
		provider,
	)

	// Create input with examples
	input := ClassificationInput{
		Subject: "This is absolutely amazing!",
		Examples: map[string][]string{
			"positive": {"Great job!", "Love it!"},
			"negative": {"Terrible", "Hate it"},
			"neutral":  {"It's okay", "Fine"},
		},
	}

	ctx := context.Background()
	response, err := classifier.FireWithInput(ctx, input)
	if err != nil {
		t.Fatalf("FireWithInput failed: %v", err)
	}
	if response.Primary != "positive" {
		t.Errorf("Expected primary='positive', got '%s'", response.Primary)
	}
	if response.Secondary != "" {
		t.Errorf("Expected no secondary, got '%s'", response.Secondary)
	}
}

func TestClassificationPromptStructure(t *testing.T) {
	var capturedPrompt string
	provider := NewMockProviderWithCallback(func(prompt string, _ float32) (string, error) {
		capturedPrompt = prompt
		return `{"primary": "test", "secondary": "", "confidence": 1.0, "reasoning": ["test"]}`, nil
	})

	classifier := Classification(
		"Classify this",
		[]string{"cat1", "cat2", "cat3"},
		provider,
	)

	ctx := context.Background()
	_, err := classifier.Fire(ctx, "test input")
	if err != nil {
		t.Fatalf("Fire failed: %v", err)
	}

	// Check prompt structure - new format uses "Task:" and "Input:"
	t.Logf("Captured prompt:\n%s", capturedPrompt)
	if !strings.Contains(capturedPrompt, "Task: Classify this") {
		t.Error("Prompt missing task")
	}
	if !strings.Contains(capturedPrompt, "Input: test input") {
		t.Error("Prompt missing input")
	}
	if !strings.Contains(capturedPrompt, "Categories:") {
		t.Error("Prompt missing categories section")
	}
	if !strings.Contains(capturedPrompt, "1. cat1") {
		t.Error("Prompt missing category 1")
	}
	if !strings.Contains(capturedPrompt, "2. cat2") {
		t.Error("Prompt missing category 2")
	}
	if !strings.Contains(capturedPrompt, "3. cat3") {
		t.Error("Prompt missing category 3")
	}
	if !strings.Contains(capturedPrompt, "primary") {
		t.Error("Prompt missing JSON format instructions")
	}
}
