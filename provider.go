package zyn

import "context"

// TokenUsage contains token counts from a provider response.
type TokenUsage struct {
	Prompt     int // Tokens used by the prompt/messages
	Completion int // Tokens used by the completion/response
	Total      int // Total tokens used
}

// ProviderResponse contains the response from an LLM provider.
type ProviderResponse struct {
	Content string     // The text response content
	Usage   TokenUsage // Token usage statistics
}

// Provider defines the interface for LLM providers.
// Providers accept conversation messages and return responses with usage stats.
// Providers are responsible for handling prompt caching internally.
type Provider interface {
	// Call sends messages to the LLM and returns the response with usage stats.
	// Messages should be in chronological order (oldest first).
	// Providers automatically handle prompt caching when supported.
	Call(ctx context.Context, messages []Message, temperature float32) (*ProviderResponse, error)

	// Name returns the provider identifier (e.g., "openai", "anthropic")
	Name() string
}
