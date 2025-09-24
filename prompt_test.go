package zyn

import (
	"context"
	"strings"
	"testing"
)

func TestPromptConsistency(t *testing.T) {
	var capturedPrompts []string

	// Capture prompts from each synapse
	provider := NewMockProviderWithCallback(func(prompt string, _ float32) (string, error) {
		capturedPrompts = append(capturedPrompts, prompt)

		// Return appropriate response based on prompt content
		if strings.Contains(prompt, "Return JSON:") {
			if strings.Contains(prompt, "decision") {
				return `{"decision": true, "confidence": 0.9, "reasoning": ["test"]}`, nil
			}
			if strings.Contains(prompt, "primary") {
				return `{"primary": "cat1", "secondary": "", "confidence": 0.9, "reasoning": ["test"]}`, nil
			}
			if strings.Contains(prompt, "ranked") {
				return `{"ranked": ["item1", "item2"], "confidence": 0.9, "reasoning": ["test"]}`, nil
			}
			if strings.Contains(prompt, "overall") {
				return `{"overall": "positive", "confidence": 0.9, "scores": {"positive": 0.8, "negative": 0.1, "neutral": 0.1}, "aspects": {}, "emotions": [], "reasoning": ["test"]}`, nil
			}
			if strings.Contains(prompt, "name") {
				return `{"name": "test", "email": "test@example.com", "phone": "123"}`, nil
			}
			if strings.Contains(prompt, "output") {
				return `{"output": "transformed", "confidence": 0.9, "changes": [], "reasoning": ["test"]}`, nil
			}
		}
		return "test", nil
	})

	ctx := context.Background()

	// Test Binary
	binary := Binary("Is this valid?", provider)
	_, _ = binary.Fire(ctx, "test input")

	// Test Classification
	classifier := Classification("Classify this", []string{"cat1", "cat2"}, provider)
	_, _ = classifier.Fire(ctx, "test input")

	// Test Ranking
	ranker := Ranking("importance", provider)
	_, _ = ranker.Fire(ctx, []string{"item1", "item2"})

	// Test Sentiment
	sentiment := Sentiment("tone", provider)
	_, _ = sentiment.Fire(ctx, "test input")

	// Test Extraction
	type TestStruct struct {
		Name  string `json:"name"`
		Email string `json:"email"`
		Phone string `json:"phone"`
	}
	extractor := Extract[TestStruct]("contact info", provider)
	_, _ = extractor.Fire(ctx, "test input")

	// Test Transform
	transformer := Transform("simplify text", provider)
	_, _ = transformer.Fire(ctx, "test input")

	// Verify all prompts have consistent structure
	if len(capturedPrompts) != 6 {
		t.Fatalf("Expected 6 prompts, got %d", len(capturedPrompts))
	}

	for i, prompt := range capturedPrompts {
		synapseName := []string{"Binary", "Classification", "Ranking", "Sentiment", "Extraction", "Transform"}[i]

		// All prompts should have these sections in order
		requiredSections := []string{
			"Task:",
			"Return JSON:",
		}

		for _, section := range requiredSections {
			if !strings.Contains(prompt, section) {
				t.Errorf("%s prompt missing required section: %s", synapseName, section)
			}
		}

		// Check ordering - Task should come before Return JSON
		taskIdx := strings.Index(prompt, "Task:")
		jsonIdx := strings.Index(prompt, "Return JSON:")

		if taskIdx == -1 || jsonIdx == -1 {
			t.Errorf("%s prompt missing core sections", synapseName)
		} else if taskIdx > jsonIdx {
			t.Errorf("%s prompt has incorrect section ordering", synapseName)
		}

		// Input section (except Ranking which uses Items)
		if synapseName != "Ranking" {
			if !strings.Contains(prompt, "Input:") {
				t.Errorf("%s prompt missing Input section", synapseName)
			}
		} else {
			if !strings.Contains(prompt, "Items:") {
				t.Errorf("%s prompt missing Items section", synapseName)
			}
		}

		// All should have Constraints
		if !strings.Contains(prompt, "Constraints:") {
			t.Errorf("%s prompt missing Constraints section", synapseName)
		}
	}

	t.Log("All synapses using consistent prompt structure ✓")
}
