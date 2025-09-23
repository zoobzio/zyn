package zyn

// Provider defines the interface for LLM providers.
// Simple adapter that sends prompts to LLMs and returns text responses.
type Provider interface {
	// Call sends a prompt to the LLM and returns the text response
	Call(prompt string, temperature float32) (string, error)
}

// NewOpenAIProvider creates an OpenAI provider.
// Model examples: "gpt-4", "gpt-3.5-turbo", "gpt-4-turbo-preview"
func NewOpenAIProvider(apiKey, model string) Provider {
	// TODO: Implement real OpenAI provider
	// For now, return mock
	return NewMockProviderWithName("openai-" + model)
}

// NewAnthropicProvider creates an Anthropic provider.
// Model examples: "claude-3-opus", "claude-3-sonnet", "claude-2.1"
func NewAnthropicProvider(apiKey, model string) Provider {
	// TODO: Implement real Anthropic provider
	// For now, return mock
	return NewMockProviderWithName("anthropic-" + model)
}

