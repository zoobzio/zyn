package zyn

import (
	"context"
	"strings"
	"testing"
	"time"
)

func TestTransformSummary(t *testing.T) {
	provider := NewMockProviderWithResponse(`{
		"output": "AI systems process data through neural networks. Key applications: NLP, computer vision, robotics.",
		"confidence": 0.85,
		"changes": ["Condensed from 500 words to 15 words", "Extracted key concepts", "Removed redundant details"],
		"reasoning": ["Identified main topic", "Listed primary applications", "Preserved technical accuracy"]
	}`)

	summarizer := Transform("summarize into key points", provider, WithTimeout(5*time.Second))

	ctx := context.Background()
	
	longText := "Artificial intelligence systems are revolutionizing how we process information. These systems use complex neural networks to analyze patterns in data. Natural language processing allows computers to understand human language. Computer vision enables machines to interpret visual information. Robotics combines these technologies for physical automation."
	
	// Test simple Fire
	summary, err := summarizer.Fire(ctx, longText)
	if err != nil {
		t.Fatalf("Fire failed: %v", err)
	}
	if !strings.Contains(summary, "neural networks") {
		t.Errorf("Summary missing key concept 'neural networks'")
	}

	// Test FireWithDetails
	details, err := summarizer.FireWithDetails(ctx, longText)
	if err != nil {
		t.Fatalf("FireWithDetails failed: %v", err)
	}
	if details.Confidence != 0.85 {
		t.Errorf("Expected confidence 0.85, got %f", details.Confidence)
	}
	if len(details.Changes) != 3 {
		t.Errorf("Expected 3 changes, got %d", len(details.Changes))
	}
}

func TestTransformTranslation(t *testing.T) {
	provider := NewMockProviderWithResponse(`{
		"output": "Hola, ¿cómo estás hoy?",
		"confidence": 0.95,
		"changes": ["Translated from English to Spanish", "Maintained informal tone"],
		"reasoning": ["Direct translation", "Preserved greeting context"]
	}`)

	translator := Transform("translate to Spanish", provider)

	ctx := context.Background()
	result, err := translator.Fire(ctx, "Hello, how are you today?")
	if err != nil {
		t.Fatalf("Fire failed: %v", err)
	}
	if result != "Hola, ¿cómo estás hoy?" {
		t.Errorf("Expected Spanish translation, got '%s'", result)
	}
}

func TestTransformFormatting(t *testing.T) {
	provider := NewMockProviderWithResponse(`{
		"output": "• First point\n• Second point\n• Third point",
		"confidence": 0.90,
		"changes": ["Converted to bullet points", "Added line breaks"],
		"reasoning": ["Identified list items", "Applied bullet formatting"]
	}`)

	formatter := Transform("convert to bullet points", provider)

	ctx := context.Background()
	input := "First point, second point, and third point"
	result, err := formatter.Fire(ctx, input)
	if err != nil {
		t.Fatalf("Fire failed: %v", err)
	}
	if !strings.Contains(result, "•") {
		t.Error("Expected bullet points in output")
	}
}

func TestTransformWithStyle(t *testing.T) {
	provider := NewMockProviderWithResponse(`{
		"output": "I would be delighted to inform you that the meeting has been rescheduled to Tuesday.",
		"confidence": 0.88,
		"changes": ["Elevated vocabulary", "Added formal expressions", "Restructured sentence"],
		"reasoning": ["Applied formal tone", "Enhanced politeness"]
	}`)

	transformer := Transform("make more formal", provider)

	input := TransformInput{
		Text:  "Hey, just wanted to let you know the meeting got moved to Tuesday",
		Style: "professional business communication",
	}

	ctx := context.Background()
	details, err := transformer.FireWithInputDetails(ctx, input)
	if err != nil {
		t.Fatalf("FireWithInputDetails failed: %v", err)
	}
	if !strings.Contains(details.Output, "delighted") {
		t.Error("Expected formal language in output")
	}
	if details.Confidence != 0.88 {
		t.Errorf("Expected confidence 0.88, got %f", details.Confidence)
	}
}

func TestTransformWithExamples(t *testing.T) {
	provider := NewMockProviderWithResponse(`{
		"output": "The Process Is Beginning",
		"confidence": 0.92,
		"changes": ["Capitalized each word"],
		"reasoning": ["Followed example pattern", "Applied title case"]
	}`)

	transformer := Transform("apply same style", provider)

	input := TransformInput{
		Text: "the process is beginning",
		Examples: map[string]string{
			"hello world":    "Hello World",
			"testing this":   "Testing This",
		},
	}

	ctx := context.Background()
	result, err := transformer.FireWithInput(ctx, input)
	if err != nil {
		t.Fatalf("FireWithInput failed: %v", err)
	}
	if result != "The Process Is Beginning" {
		t.Errorf("Expected title case, got '%s'", result)
	}
}

func TestTransformWithMaxLength(t *testing.T) {
	provider := NewMockProviderWithResponse(`{
		"output": "Quick summary here",
		"confidence": 0.80,
		"changes": ["Truncated to fit length limit", "Preserved key information"],
		"reasoning": ["Respected 20 character limit", "Prioritized essential content"]
	}`)

	transformer := Transform("summarize", provider)

	input := TransformInput{
		Text:      "This is a very long text that needs to be summarized into something much shorter",
		MaxLength: 20,
	}

	ctx := context.Background()
	details, err := transformer.FireWithInputDetails(ctx, input)
	if err != nil {
		t.Fatalf("FireWithInputDetails failed: %v", err)
	}
	if len(details.Output) > 20 {
		t.Errorf("Output exceeded max length: %d > 20", len(details.Output))
	}
}

func TestTransformPromptStructure(t *testing.T) {
	var capturedPrompt string
	provider := NewMockProviderWithCallback(func(prompt string, temp float32) (string, error) {
		capturedPrompt = prompt
		return `{"output": "test", "confidence": 1.0, "changes": [], "reasoning": ["test"]}`, nil
	})

	transformer := Transform("simplify", provider)

	ctx := context.Background()
	_, err := transformer.Fire(ctx, "test input")
	if err != nil {
		t.Fatalf("Fire failed: %v", err)
	}

	// Check prompt structure
	if !strings.Contains(capturedPrompt, "Task: Transform: simplify") {
		t.Error("Prompt missing task description")
	}
	if !strings.Contains(capturedPrompt, "Input: test input") {
		t.Error("Prompt missing input")
	}
	if !strings.Contains(capturedPrompt, "Return JSON:") {
		t.Error("Prompt missing JSON structure")
	}
	if !strings.Contains(capturedPrompt, `"output"`) {
		t.Error("Schema missing output field")
	}
	if !strings.Contains(capturedPrompt, `"changes"`) {
		t.Error("Schema missing changes field")
	}
	if !strings.Contains(capturedPrompt, `"reasoning"`) {
		t.Error("Schema missing reasoning field")
	}
}

func TestTransformVariousUseCases(t *testing.T) {
	tests := []struct {
		name        string
		instruction string
		input       string
		response    string
		checkOutput func(string) bool
	}{
		{
			name:        "Code formatting",
			instruction: "format as Python code",
			input:       "print hello world",
			response:    `{"output": "print(\"hello world\")", "confidence": 0.95, "changes": ["Added parentheses", "Added quotes"], "reasoning": ["Python 3 syntax"]}`,
			checkOutput: func(s string) bool { return strings.Contains(s, "print(") },
		},
		{
			name:        "Tone adjustment",
			instruction: "make it friendly",
			input:       "Send me the report.",
			response:    `{"output": "Hey! Could you please send me the report when you get a chance? Thanks!", "confidence": 0.85, "changes": ["Added greeting", "Made request polite"], "reasoning": ["Friendly tone"]}`,
			checkOutput: func(s string) bool { return strings.Contains(s, "please") },
		},
		{
			name:        "Simplification",
			instruction: "explain like I'm five",
			input:       "Photosynthesis is the process by which plants convert light energy into chemical energy",
			response:    `{"output": "Plants eat sunlight to make food, just like you eat breakfast for energy!", "confidence": 0.82, "changes": ["Simplified vocabulary", "Added analogy"], "reasoning": ["Age-appropriate language"]}`,
			checkOutput: func(s string) bool { return strings.Contains(s, "eat") || strings.Contains(s, "food") },
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			provider := NewMockProviderWithResponse(tt.response)
			transformer := Transform(tt.instruction, provider)
			
			ctx := context.Background()
			result, err := transformer.Fire(ctx, tt.input)
			if err != nil {
				t.Fatalf("Fire failed: %v", err)
			}
			
			if !tt.checkOutput(result) {
				t.Errorf("Unexpected output for %s: %s", tt.name, result)
			}
		})
	}
}