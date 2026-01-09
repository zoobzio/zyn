package tendo

import (
	"os"
	"testing"

	"github.com/zoobzio/zyn"
)

func TestLlama3Template(t *testing.T) {
	template := Llama3Template{}

	tests := []struct {
		name     string
		messages []zyn.Message
		expected string
	}{
		{
			name: "single user message",
			messages: []zyn.Message{
				{Role: zyn.RoleUser, Content: "Hello"},
			},
			expected: "<|begin_of_text|><|start_header_id|>user<|end_header_id|>\n\nHello<|eot_id|><|start_header_id|>assistant<|end_header_id|>\n\n",
		},
		{
			name: "system and user message",
			messages: []zyn.Message{
				{Role: zyn.RoleSystem, Content: "You are helpful."},
				{Role: zyn.RoleUser, Content: "Hi"},
			},
			expected: "<|begin_of_text|><|start_header_id|>system<|end_header_id|>\n\nYou are helpful.<|eot_id|><|start_header_id|>user<|end_header_id|>\n\nHi<|eot_id|><|start_header_id|>assistant<|end_header_id|>\n\n",
		},
		{
			name: "multi-turn conversation",
			messages: []zyn.Message{
				{Role: zyn.RoleUser, Content: "What is 2+2?"},
				{Role: zyn.RoleAssistant, Content: "4"},
				{Role: zyn.RoleUser, Content: "And 3+3?"},
			},
			expected: "<|begin_of_text|><|start_header_id|>user<|end_header_id|>\n\nWhat is 2+2?<|eot_id|><|start_header_id|>assistant<|end_header_id|>\n\n4<|eot_id|><|start_header_id|>user<|end_header_id|>\n\nAnd 3+3?<|eot_id|><|start_header_id|>assistant<|end_header_id|>\n\n",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := template.Apply(tt.messages)
			if result != tt.expected {
				t.Errorf("Expected:\n%q\nGot:\n%q", tt.expected, result)
			}
		})
	}
}

func TestChatMLTemplate(t *testing.T) {
	template := ChatMLTemplate{}

	tests := []struct {
		name     string
		messages []zyn.Message
		expected string
	}{
		{
			name: "single user message",
			messages: []zyn.Message{
				{Role: zyn.RoleUser, Content: "Hello"},
			},
			expected: "<|im_start|>user\nHello<|im_end|>\n<|im_start|>assistant\n",
		},
		{
			name: "system and user message",
			messages: []zyn.Message{
				{Role: zyn.RoleSystem, Content: "You are helpful."},
				{Role: zyn.RoleUser, Content: "Hi"},
			},
			expected: "<|im_start|>system\nYou are helpful.<|im_end|>\n<|im_start|>user\nHi<|im_end|>\n<|im_start|>assistant\n",
		},
		{
			name: "multi-turn conversation",
			messages: []zyn.Message{
				{Role: zyn.RoleUser, Content: "What is 2+2?"},
				{Role: zyn.RoleAssistant, Content: "4"},
				{Role: zyn.RoleUser, Content: "And 3+3?"},
			},
			expected: "<|im_start|>user\nWhat is 2+2?<|im_end|>\n<|im_start|>assistant\n4<|im_end|>\n<|im_start|>user\nAnd 3+3?<|im_end|>\n<|im_start|>assistant\n",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := template.Apply(tt.messages)
			if result != tt.expected {
				t.Errorf("Expected:\n%q\nGot:\n%q", tt.expected, result)
			}
		})
	}
}

func TestRawTemplate(t *testing.T) {
	template := RawTemplate{}

	messages := []zyn.Message{
		{Role: zyn.RoleUser, Content: "Hello"},
		{Role: zyn.RoleAssistant, Content: "Hi there"},
	}

	expected := "Hello\nHi there"
	result := template.Apply(messages)

	if result != expected {
		t.Errorf("Expected %q, got %q", expected, result)
	}
}

func TestTemplateForModel(t *testing.T) {
	tests := []struct {
		modelConfig  string
		expectedType string
	}{
		{"llama3.2-1b", "Llama3Template"},
		{"llama3.2-3b", "Llama3Template"},
		{"tinyllama", "Llama3Template"},
		{"smollm-135m", "ChatMLTemplate"},
		{"smollm-360m", "ChatMLTemplate"},
		{"qwen2-0.5b", "ChatMLTemplate"},
		{"unknown", "ChatMLTemplate"}, // default
	}

	for _, tt := range tests {
		t.Run(tt.modelConfig, func(t *testing.T) {
			template := templateForModel(tt.modelConfig)
			typeName := getTypeName(template)
			if typeName != tt.expectedType {
				t.Errorf("Expected %s for %s, got %s", tt.expectedType, tt.modelConfig, typeName)
			}
		})
	}
}

func TestGetModelConfig(t *testing.T) {
	validConfigs := []string{
		"smollm-135m",
		"smollm-360m",
		"qwen2-0.5b",
		"llama3.2-1b",
		"llama3.2-3b",
		"tinyllama",
	}

	for _, name := range validConfigs {
		t.Run(name, func(t *testing.T) {
			_, err := getModelConfig(name)
			if err != nil {
				t.Errorf("Expected valid config for %s, got error: %v", name, err)
			}
		})
	}

	// Test invalid config
	_, err := getModelConfig("invalid-model")
	if err == nil {
		t.Error("Expected error for invalid model config")
	}
}

func TestProviderName(t *testing.T) {
	// We can't create a real provider without model files,
	// but we can verify the name would be set correctly
	p := &Provider{name: "tendo"}
	if p.Name() != "tendo" {
		t.Errorf("Expected 'tendo', got '%s'", p.Name())
	}
}

func TestTendoIntegration(t *testing.T) {
	modelPath := os.Getenv("TENDO_MODEL_PATH")
	tokenizerPath := os.Getenv("TENDO_TOKENIZER_PATH")
	modelConfig := os.Getenv("TENDO_MODEL_CONFIG")

	if modelPath == "" || tokenizerPath == "" {
		t.Skip("TENDO_MODEL_PATH and TENDO_TOKENIZER_PATH not set, skipping integration test")
	}

	if modelConfig == "" {
		modelConfig = "smollm-135m"
	}

	provider, err := New(Config{
		ModelPath:     modelPath,
		TokenizerPath: tokenizerPath,
		ModelConfig:   modelConfig,
		Device:        DeviceCPU,
		MaxTokens:     20,
	})
	if err != nil {
		t.Fatalf("Failed to create provider: %v", err)
	}
	defer provider.Close()

	messages := []zyn.Message{
		{Role: zyn.RoleUser, Content: "Say hello"},
	}

	response, err := provider.Call(t.Context(), messages, 0.7)
	if err != nil {
		t.Fatalf("Call failed: %v", err)
	}

	if response.Content == "" {
		t.Error("Expected non-empty response")
	}

	t.Logf("Response: %s", response.Content)
	t.Logf("Usage: prompt=%d, completion=%d, total=%d",
		response.Usage.Prompt, response.Usage.Completion, response.Usage.Total)
}

func getTypeName(t Template) string {
	switch t.(type) {
	case Llama3Template:
		return "Llama3Template"
	case ChatMLTemplate:
		return "ChatMLTemplate"
	case RawTemplate:
		return "RawTemplate"
	default:
		return "unknown"
	}
}
