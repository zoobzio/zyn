package zyn

import (
	"context"
	"strings"
	"testing"
	"time"
)

func TestRankingBasic(t *testing.T) {
	provider := NewMockProviderWithResponse(`{
		"ranked": ["Security patch", "Bug fix", "Add feature", "Fix typo"],
		"confidence": 0.9,
		"reasoning": ["Security issues have highest priority", "Bugs affect users", "Features are enhancements", "Typos are cosmetic"]
	}`)

	ranker := Ranking(
		"urgency and user impact",
		provider,
		WithTimeout(5*time.Second),
	)

	ctx := context.Background()

	// Test simple Fire
	items := []string{"Fix typo", "Security patch", "Add feature", "Bug fix"}
	ranked, err := ranker.Fire(ctx, items)
	if err != nil {
		t.Fatalf("Fire failed: %v", err)
	}
	
	if len(ranked) != 4 {
		t.Errorf("Expected 4 items, got %d", len(ranked))
	}
	if ranked[0] != "Security patch" {
		t.Errorf("Expected 'Security patch' first, got '%s'", ranked[0])
	}
	if ranked[3] != "Fix typo" {
		t.Errorf("Expected 'Fix typo' last, got '%s'", ranked[3])
	}

	// Test FireWithDetails
	response, err := ranker.FireWithDetails(ctx, items)
	if err != nil {
		t.Fatalf("FireWithDetails failed: %v", err)
	}
	if response.Confidence != 0.9 {
		t.Errorf("Expected confidence=0.9, got %f", response.Confidence)
	}
	if len(response.Reasoning) != 4 {
		t.Errorf("Expected 4 reasoning steps, got %d", len(response.Reasoning))
	}
}

func TestRankingWithTopN(t *testing.T) {
	provider := NewMockProviderWithResponse(`{
		"ranked": ["Critical bug", "Security issue"],
		"confidence": 0.85,
		"reasoning": ["Selected most critical items", "These require immediate attention"]
	}`)

	ranker := Ranking("severity", provider)

	input := RankingInput{
		Items: []string{"Typo", "Critical bug", "Feature", "Security issue", "Cleanup"},
		TopN:  2,
	}

	ctx := context.Background()
	response, err := ranker.FireWithInput(ctx, input)
	if err != nil {
		t.Fatalf("FireWithInput failed: %v", err)
	}

	if len(response.Ranked) != 2 {
		t.Errorf("Expected 2 items (TopN), got %d", len(response.Ranked))
	}
	if response.Ranked[0] != "Critical bug" {
		t.Errorf("Expected 'Critical bug' first, got '%s'", response.Ranked[0])
	}
}

func TestRankingValidation(t *testing.T) {
	tests := []struct {
		name      string
		items     []string
		response  string
		wantError bool
		errorMsg  string
	}{
		{
			name:  "Valid full ranking",
			items: []string{"A", "B", "C"},
			response: `{
				"ranked": ["B", "A", "C"],
				"confidence": 0.8,
				"reasoning": ["test"]
			}`,
			wantError: false,
		},
		{
			name:  "Missing item",
			items: []string{"A", "B", "C"},
			response: `{
				"ranked": ["B", "A"],
				"confidence": 0.8,
				"reasoning": ["test"]
			}`,
			wantError: true,
			errorMsg:  "expected 3 items, got 2",
		},
		{
			name:  "Duplicate item",
			items: []string{"A", "B", "C"},
			response: `{
				"ranked": ["B", "B", "C"],
				"confidence": 0.8,
				"reasoning": ["test"]
			}`,
			wantError: true,
			errorMsg:  "duplicate item",
		},
		{
			name:  "Item not in original list",
			items: []string{"A", "B", "C"},
			response: `{
				"ranked": ["B", "A", "D"],
				"confidence": 0.8,
				"reasoning": ["test"]
			}`,
			wantError: true,
			errorMsg:  "missing item in ranking: C",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			provider := NewMockProviderWithResponse(tt.response)
			ranker := Ranking("test", provider)

			ctx := context.Background()
			_, err := ranker.Fire(ctx, tt.items)

			if tt.wantError {
				if err == nil {
					t.Error("Expected error but got none")
				} else if !strings.Contains(err.Error(), tt.errorMsg) {
					t.Errorf("Expected error containing '%s', got '%s'", tt.errorMsg, err.Error())
				}
			} else {
				if err != nil {
					t.Errorf("Unexpected error: %v", err)
				}
			}
		})
	}
}

func TestRankingPromptStructure(t *testing.T) {
	var capturedPrompt string
	provider := NewMockProviderWithCallback(func(prompt string, temp float32) (string, error) {
		capturedPrompt = prompt
		return `{"ranked": ["Task A", "Task B"], "confidence": 1.0, "reasoning": ["test"]}`, nil
	})

	ranker := Ranking("complexity", provider)

	ctx := context.Background()
	_, err := ranker.Fire(ctx, []string{"Task A", "Task B"})
	if err != nil {
		t.Fatalf("Fire failed: %v", err)
	}

	// Check prompt structure
	if !strings.Contains(capturedPrompt, "Task: Rank by complexity") {
		t.Error("Prompt missing task description")
	}
	if !strings.Contains(capturedPrompt, "Items:") {
		t.Error("Prompt missing items section")
	}
	if !strings.Contains(capturedPrompt, "1. Task A") {
		t.Error("Prompt missing first item")
	}
	if !strings.Contains(capturedPrompt, "2. Task B") {
		t.Error("Prompt missing second item")
	}
	if !strings.Contains(capturedPrompt, "include every item exactly once") {
		t.Error("Prompt missing completeness instruction")
	}
	if !strings.Contains(capturedPrompt, "ranked") {
		t.Error("Prompt missing JSON format")
	}
}