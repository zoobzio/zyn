package zyn

import "context"

// Provider defines the interface for LLM providers.
// Simple adapter that sends prompts to LLMs and returns text responses.
type Provider interface {
	// Call sends a prompt to the LLM and returns the text response
	Call(ctx context.Context, prompt string, temperature float32) (string, error)
}
