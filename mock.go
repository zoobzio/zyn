package zyn

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
)

// MockProvider simulates LLM behavior for testing.
// It returns deterministic responses based on prompt patterns.
type MockProvider struct {
	name      string
	available bool
}

// NewMockProvider creates a new mock provider for testing.
func NewMockProvider() Provider {
	return &MockProvider{
		name:      "mock",
		available: true,
	}
}

// NewMockProviderWithName creates a new mock provider with a specific name.
func NewMockProviderWithName(name string) *MockProvider {
	return &MockProvider{
		name:      name,
		available: true,
	}
}

// Call simulates an LLM call with deterministic responses.
func (m *MockProvider) Call(_ context.Context, prompt string, _ float32) (string, error) {
	if !m.available {
		return "", fmt.Errorf("provider %s is unavailable", m.name)
	}

	// Generate response based on prompt patterns
	return m.generateResponse(prompt), nil
}

// SetAvailable sets the availability status (for testing failures).
func (m *MockProvider) SetAvailable(available bool) {
	m.available = available
}

// generateResponse creates a response based on prompt patterns.
func (m *MockProvider) generateResponse(prompt string) string {
	// Check for JSON response request
	if strings.Contains(prompt, "Response JSON Schema:") {
		// Binary decision pattern
		if strings.Contains(prompt, "valid email") || strings.Contains(prompt, "email") {
			return m.generateEmailValidationResponse(prompt)
		}

		// Default binary response
		return `{"decision": true, "confidence": 0.8, "reasoning": ["Mock response generated"]}`
	}

	// Default response
	return "Mock response"
}

// generateEmailValidationResponse creates email validation responses.
func (*MockProvider) generateEmailValidationResponse(prompt string) string {
	// Extract the subject from prompt
	subject := extractSubject(prompt)

	// Validate email format
	isValid := strings.Contains(subject, "@") &&
		strings.Contains(subject, ".") &&
		!strings.HasPrefix(subject, "@")

	response := struct {
		Decision   bool     `json:"decision"`
		Confidence float64  `json:"confidence"`
		Reasoning  []string `json:"reasoning"`
	}{
		Decision:   isValid,
		Confidence: 0.85,
		Reasoning: []string{
			"Checked for @ symbol",
			"Verified domain extension",
			"Validated format structure",
		},
	}

	if !isValid {
		response.Reasoning = append(response.Reasoning, "Invalid email format detected")
	}

	jsonBytes, err := json.Marshal(response)
	if err != nil {
		return "Mock response"
	}
	return string(jsonBytes)
}

// extractSubject extracts the subject from a prompt.
func extractSubject(prompt string) string {
	// Look for "Input: " pattern
	if idx := strings.Index(prompt, "Input: "); idx != -1 {
		start := idx + 7
		end := strings.Index(prompt[start:], "\n")
		if end == -1 {
			return strings.TrimSpace(prompt[start:])
		}
		return strings.TrimSpace(prompt[start : start+end])
	}

	return ""
}

// NewMockProviderWithResponse creates a mock that always returns a specific response.
func NewMockProviderWithResponse(response string) Provider {
	return &mockProviderFixed{response: response}
}

// NewMockProviderWithCallback creates a mock that calls a function to generate responses.
func NewMockProviderWithCallback(callback func(prompt string, temperature float32) (string, error)) Provider {
	return &mockProviderCallback{callback: callback}
}

// mockProviderFixed always returns a fixed response.
type mockProviderFixed struct {
	response string
}

func (m *mockProviderFixed) Call(_ context.Context, _ string, _ float32) (string, error) {
	return m.response, nil
}

// mockProviderCallback uses a callback to generate responses.
type mockProviderCallback struct {
	callback func(string, float32) (string, error)
}

func (m *mockProviderCallback) Call(_ context.Context, prompt string, temperature float32) (string, error) {
	return m.callback(prompt, temperature)
}
