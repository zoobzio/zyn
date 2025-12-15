// Package zyn provides type-safe LLM orchestration with composable reliability patterns.
//
// Zyn abstracts away prompt engineering complexity, response parsing, and error handling
// while enforcing strict type safety through Go generics. It provides eight synapse types
// covering the spectrum of LLM interaction patterns:
//
//   - Binary: Yes/No decisions with confidence scores
//   - Classification: Multi-class categorization
//   - Extraction: Structured data extraction from text
//   - Transform: Text transformation with instructions
//   - Analyze: Structured analysis of data
//   - Convert: Type-safe conversion between formats
//   - Ranking: Ordering items by criteria
//   - Sentiment: Sentiment analysis with aspect support
//
// All synapses support composable reliability options (retry, timeout, circuit breaker,
// rate limiting) and emit observability hooks for monitoring and debugging.
//
// Basic usage:
//
//	provider := openai.New(apiKey, "gpt-4")
//	synapse, _ := zyn.Binary("Is this a valid email address?", provider)
//	session := zyn.NewSession()
//	result, _ := synapse.Fire(ctx, session, "user@example.com")
//	fmt.Println(result.Decision, result.Confidence)
package zyn

import "context"

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

// Validator defines the interface for response validation.
// All response types must implement this to ensure LLM outputs are valid.
type Validator interface {
	Validate() error
}

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

// Message represents a single message in a conversation.
// Messages are exchanged between the user and the assistant (LLM).
type Message struct {
	Role    string // RoleUser, RoleAssistant, or RoleSystem
	Content string // The message content
}

// Role constants for message types.
const (
	RoleUser      = "user"
	RoleAssistant = "assistant"
	RoleSystem    = "system"
)

// Default temperature constants for different synapse types.
// Temperature controls the randomness/creativity of LLM responses.
// Lower values (0.1) produce more deterministic outputs.
// Higher values (0.3) allow more creative/varied responses.
const (
	// TemperatureUnset indicates that no temperature has been explicitly set.
	// When this value is encountered, the synapse will use its default temperature.
	// Note: A zero-value float32 (0.0) is also treated as unset for ergonomic struct initialization.
	TemperatureUnset float32 = -1

	// TemperatureZero provides an explicitly near-zero temperature for maximum determinism.
	// Use this instead of 0.0 since zero is treated as "unset".
	TemperatureZero float32 = 0.0001

	// DefaultTemperatureDeterministic is used for tasks requiring consistent,
	// precise outputs with minimal variation (binary decisions, extraction, conversion).
	DefaultTemperatureDeterministic float32 = 0.1

	// DefaultTemperatureAnalytical is used for tasks requiring consistent analysis
	// with some flexibility (sentiment analysis, ranking, data analysis).
	DefaultTemperatureAnalytical float32 = 0.2

	// DefaultTemperatureCreative is used for tasks benefiting from more varied
	// outputs (classification, text transformation).
	DefaultTemperatureCreative float32 = 0.3
)

// SynapseRequest flows through the pipz pipeline.
// It contains the prompt, parameters, session, and response data.
type SynapseRequest struct {
	// Input fields
	Prompt      *Prompt // The structured prompt to send to LLM
	Temperature float32 // Temperature parameter for response generation

	// Session fields
	SessionID string    // ID of the conversation session
	Messages  []Message // Message history from session

	// Metadata fields
	RequestID    string // Unique identifier for this request
	SynapseType  string // Type of synapse (binary, extraction, etc.)
	ProviderName string // Name of the provider being used

	// Output fields (populated by pipeline)
	Response string      // Raw text response from provider
	Usage    *TokenUsage // Token usage from provider response
	Error    error       // Any error that occurred during processing
}
