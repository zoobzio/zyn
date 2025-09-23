package anthropic

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

// Provider implements the zyn Provider interface for Anthropic Claude API.
type Provider struct {
	apiKey     string
	model      string
	version    string
	baseURL    string
	httpClient *http.Client
}

// Config holds configuration for the Anthropic provider.
type Config struct {
	APIKey  string
	Model   string        // e.g. "claude-3-opus-20240229", "claude-3-sonnet-20240229"
	Version string        // API version, defaults to "2023-06-01"
	BaseURL string        // Optional, defaults to "https://api.anthropic.com/v1"
	Timeout time.Duration // Optional, defaults to 30s
}

// New creates a new Anthropic provider.
func New(config Config) *Provider {
	if config.Model == "" {
		config.Model = "claude-3-sonnet-20240229"
	}
	if config.Version == "" {
		config.Version = "2023-06-01"
	}
	if config.BaseURL == "" {
		config.BaseURL = "https://api.anthropic.com/v1"
	}
	if config.Timeout == 0 {
		config.Timeout = 30 * time.Second
	}

	return &Provider{
		apiKey:  config.APIKey,
		model:   config.Model,
		version: config.Version,
		baseURL: config.BaseURL,
		httpClient: &http.Client{
			Timeout: config.Timeout,
		},
	}
}

// Call sends a prompt to Anthropic and returns the response.
func (p *Provider) Call(prompt string, temperature float32) (string, error) {
	// Build request body
	requestBody := messagesRequest{
		Model:       p.model,
		MaxTokens:   4096,
		Temperature: temperature,
		Messages: []message{
			{
				Role:    "user",
				Content: prompt,
			},
		},
	}

	jsonBody, err := json.Marshal(requestBody)
	if err != nil {
		return "", fmt.Errorf("failed to marshal request: %w", err)
	}

	// Create HTTP request
	req, err := http.NewRequest("POST", p.baseURL+"/messages", bytes.NewReader(jsonBody))
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("x-api-key", p.apiKey)
	req.Header.Set("anthropic-version", p.version)

	// Make the request
	resp, err := p.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	// Read response body
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read response: %w", err)
	}

	// Handle errors
	if resp.StatusCode != http.StatusOK {
		var errorResp errorResponse
		if err := json.Unmarshal(body, &errorResp); err == nil && errorResp.Error.Message != "" {
			// Check for rate limit
			if resp.StatusCode == http.StatusTooManyRequests {
				return "", fmt.Errorf("rate limit exceeded: %s", errorResp.Error.Message)
			}
			return "", fmt.Errorf("anthropic error (%d): %s", resp.StatusCode, errorResp.Error.Message)
		}
		return "", fmt.Errorf("anthropic error: status %d", resp.StatusCode)
	}

	// Parse successful response
	var messagesResp messagesResponse
	if err := json.Unmarshal(body, &messagesResp); err != nil {
		return "", fmt.Errorf("failed to parse response: %w", err)
	}

	// Extract text from content blocks
	var result string
	for _, content := range messagesResp.Content {
		if content.Type == "text" {
			result += content.Text
		}
	}

	if result == "" {
		return "", fmt.Errorf("no text content in response")
	}

	return result, nil
}

// Request/Response types for Anthropic API

type messagesRequest struct {
	Model       string    `json:"model"`
	MaxTokens   int       `json:"max_tokens"`
	Temperature float32   `json:"temperature"`
	Messages    []message `json:"messages"`
}

type message struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type messagesResponse struct {
	ID      string    `json:"id"`
	Type    string    `json:"type"`
	Role    string    `json:"role"`
	Content []content `json:"content"`
	Model   string    `json:"model"`
	Usage   usage     `json:"usage"`
}

type content struct {
	Type string `json:"type"`
	Text string `json:"text,omitempty"`
}

type usage struct {
	InputTokens  int `json:"input_tokens"`
	OutputTokens int `json:"output_tokens"`
}

type errorResponse struct {
	Error struct {
		Type    string `json:"type"`
		Message string `json:"message"`
	} `json:"error"`
}