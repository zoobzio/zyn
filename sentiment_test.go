package zyn

import (
	"context"
	"math"
	"strings"
	"testing"
	"time"
)

func TestSentimentBasic(t *testing.T) {
	provider := NewMockProviderWithResponse(`{
		"overall": "positive",
		"confidence": 0.92,
		"scores": {
			"positive": 0.85,
			"negative": 0.05,
			"neutral": 0.10
		},
		"aspects": {},
		"emotions": ["joy", "excitement"],
		"reasoning": ["Enthusiastic language", "Positive descriptors", "Exclamation marks indicate excitement"]
	}`)

	sentiment := Sentiment("emotional tone", provider, WithTimeout(5*time.Second))

	ctx := context.Background()

	// Test simple Fire
	result, err := sentiment.Fire(ctx, "This is absolutely amazing! I love it!")
	if err != nil {
		t.Fatalf("Fire failed: %v", err)
	}
	if result != "positive" {
		t.Errorf("Expected 'positive', got '%s'", result)
	}

	// Test FireWithDetails
	details, err := sentiment.FireWithDetails(ctx, "This is absolutely amazing! I love it!")
	if err != nil {
		t.Fatalf("FireWithDetails failed: %v", err)
	}
	if details.Overall != "positive" {
		t.Errorf("Expected overall 'positive', got '%s'", details.Overall)
	}
	if details.Confidence != 0.92 {
		t.Errorf("Expected confidence 0.92, got %f", details.Confidence)
	}
	if details.Scores.Positive != 0.85 {
		t.Errorf("Expected positive score 0.85, got %f", details.Scores.Positive)
	}
	if len(details.Emotions) != 2 {
		t.Errorf("Expected 2 emotions, got %d", len(details.Emotions))
	}
	if details.Emotions[0] != "joy" {
		t.Errorf("Expected first emotion 'joy', got '%s'", details.Emotions[0])
	}
}

func TestSentimentNegative(t *testing.T) {
	provider := NewMockProviderWithResponse(`{
		"overall": "negative",
		"confidence": 0.88,
		"scores": {
			"positive": 0.10,
			"negative": 0.75,
			"neutral": 0.15
		},
		"aspects": {},
		"emotions": ["frustration", "disappointment"],
		"reasoning": ["Complaint language", "Negative experience described"]
	}`)

	sentiment := Sentiment("customer feedback", provider)

	ctx := context.Background()
	result, err := sentiment.Fire(ctx, "This product is terrible and doesn't work at all")
	if err != nil {
		t.Fatalf("Fire failed: %v", err)
	}
	if result != "negative" {
		t.Errorf("Expected 'negative', got '%s'", result)
	}
}

func TestSentimentMixed(t *testing.T) {
	provider := NewMockProviderWithResponse(`{
		"overall": "mixed",
		"confidence": 0.70,
		"scores": {
			"positive": 0.40,
			"negative": 0.35,
			"neutral": 0.25
		},
		"aspects": {},
		"emotions": ["hope", "concern"],
		"reasoning": ["Both positive and negative elements", "Conflicting sentiments"]
	}`)

	sentiment := Sentiment("review analysis", provider)

	ctx := context.Background()
	details, err := sentiment.FireWithDetails(ctx, "The product quality is good but the service was terrible")
	if err != nil {
		t.Fatalf("FireWithDetails failed: %v", err)
	}
	if details.Overall != "mixed" {
		t.Errorf("Expected 'mixed', got '%s'", details.Overall)
	}
	// Check scores roughly sum to 1.0
	total := details.Scores.Positive + details.Scores.Negative + details.Scores.Neutral
	if math.Abs(total-1.0) > 0.01 {
		t.Errorf("Scores should sum to ~1.0, got %f", total)
	}
}

func TestSentimentWithAspects(t *testing.T) {
	provider := NewMockProviderWithResponse(`{
		"overall": "mixed",
		"confidence": 0.75,
		"scores": {
			"positive": 0.50,
			"negative": 0.30,
			"neutral": 0.20
		},
		"aspects": {
			"food quality": "positive",
			"service": "negative",
			"ambiance": "positive",
			"price": "negative"
		},
		"emotions": ["satisfaction", "frustration"],
		"reasoning": ["Mixed review with varying aspects", "Good food but poor service"]
	}`)

	sentiment := Sentiment("restaurant review", provider)

	input := SentimentInput{
		Text:    "The food was delicious and the ambiance was lovely, but service was slow and prices too high",
		Aspects: []string{"food quality", "service", "ambiance", "price"},
	}

	ctx := context.Background()
	details, err := sentiment.FireWithInput(ctx, input)
	if err != nil {
		t.Fatalf("FireWithInput failed: %v", err)
	}

	if len(details.Aspects) != 4 {
		t.Errorf("Expected 4 aspects, got %d", len(details.Aspects))
	}
	if details.Aspects["food quality"] != "positive" {
		t.Errorf("Expected food quality 'positive', got '%s'", details.Aspects["food quality"])
	}
	if details.Aspects["service"] != "negative" {
		t.Errorf("Expected service 'negative', got '%s'", details.Aspects["service"])
	}
}

func TestSentimentNormalization(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"Positive", "positive"},
		{"POSITIVE", "positive"},
		{"pos", "positive"},
		{"Negative", "negative"},
		{"neg", "negative"},
		{"Neutral", "neutral"},
		{"neu", "neutral"},
		{"Mixed", "mixed"},
		{"mix", "mixed"},
		{"unknown", "unknown"}, // Unrecognized returns as-is
	}

	for _, tt := range tests {
		result := normalizeSentiment(tt.input)
		if result != tt.expected {
			t.Errorf("normalizeSentiment(%s) = %s, want %s", tt.input, result, tt.expected)
		}
	}
}

func TestSentimentPromptStructure(t *testing.T) {
	var capturedPrompt string
	provider := NewMockProviderWithCallback(func(prompt string, _ float32) (string, error) {
		capturedPrompt = prompt
		return `{
			"overall": "neutral",
			"confidence": 0.5,
			"scores": {"positive": 0.33, "negative": 0.33, "neutral": 0.34},
			"aspects": {},
			"emotions": [],
			"reasoning": ["test"]
		}`, nil
	})

	sentiment := Sentiment("social media post", provider)

	ctx := context.Background()
	_, err := sentiment.Fire(ctx, "test input")
	if err != nil {
		t.Fatalf("Fire failed: %v", err)
	}

	// Check prompt structure
	if !strings.Contains(capturedPrompt, "Task: Analyze social media post sentiment") {
		t.Error("Prompt missing task description")
	}
	if !strings.Contains(capturedPrompt, "Input: test input") {
		t.Error("Prompt missing input")
	}
	if !strings.Contains(capturedPrompt, "positive/negative/neutral/mixed") {
		t.Error("Prompt missing sentiment options")
	}
	if !strings.Contains(capturedPrompt, `"emotions"`) {
		t.Error("Prompt missing emotions field")
	}
	if !strings.Contains(capturedPrompt, `"scores"`) {
		t.Error("Prompt missing scores field")
	}
}

func TestSentimentWithContext(t *testing.T) {
	provider := NewMockProviderWithResponse(`{
		"overall": "positive",
		"confidence": 0.80,
		"scores": {
			"positive": 0.70,
			"negative": 0.10,
			"neutral": 0.20
		},
		"aspects": {},
		"emotions": ["sarcasm", "humor"],
		"reasoning": ["Context indicates sarcastic tone", "Actually positive despite negative words"]
	}`)

	sentiment := Sentiment("message tone", provider)

	input := SentimentInput{
		Text:    "Oh great, another meeting. Just what I needed!",
		Context: "Said jokingly to a colleague who also dislikes meetings",
	}

	ctx := context.Background()
	details, err := sentiment.FireWithInput(ctx, input)
	if err != nil {
		t.Fatalf("FireWithInput failed: %v", err)
	}

	// With context, sarcasm might be detected as actually positive/humorous
	if details.Overall != "positive" {
		t.Errorf("Expected 'positive' (sarcasm with context), got '%s'", details.Overall)
	}

	hasHumorOrSarcasm := false
	for _, emotion := range details.Emotions {
		if emotion == "sarcasm" || emotion == "humor" {
			hasHumorOrSarcasm = true
			break
		}
	}
	if !hasHumorOrSarcasm {
		t.Error("Expected sarcasm or humor in emotions")
	}
}
