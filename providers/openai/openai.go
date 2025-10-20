package openai

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

// Provider implements the zyn Provider interface for OpenAI API.
type Provider struct {
	apiKey     string
	model      string
	baseURL    string
	httpClient *http.Client
}

// Config holds configuration for the OpenAI provider.
type Config struct {
	APIKey  string
	Model   string        // e.g. "gpt-4", "gpt-3.5-turbo"
	BaseURL string        // Optional, defaults to "https://api.openai.com/v1"
	Timeout time.Duration // Optional, defaults to 30s
}

// New creates a new OpenAI provider.
func New(config Config) *Provider {
	if config.Model == "" {
		config.Model = "gpt-3.5-turbo"
	}
	if config.BaseURL == "" {
		config.BaseURL = "https://api.openai.com/v1"
	}
	if config.Timeout == 0 {
		config.Timeout = 30 * time.Second
	}

	return &Provider{
		apiKey:  config.APIKey,
		model:   config.Model,
		baseURL: config.BaseURL,
		httpClient: &http.Client{
			Timeout: config.Timeout,
		},
	}
}

// Call sends a prompt to OpenAI and returns the response.
func (p *Provider) Call(ctx context.Context, prompt string, temperature float32) (string, error) {
	// Build request body with JSON mode enabled
	requestBody := chatCompletionRequest{
		Model: p.model,
		Messages: []message{
			{
				Role:    "user",
				Content: prompt,
			},
		},
		Temperature: temperature,
		ResponseFormat: &responseFormat{
			Type: "json_object",
		},
	}

	jsonBody, err := json.Marshal(requestBody)
	if err != nil {
		return "", fmt.Errorf("failed to marshal request: %w", err)
	}

	// Create HTTP request
	req, err := http.NewRequestWithContext(ctx, "POST", p.baseURL+"/chat/completions", bytes.NewReader(jsonBody))
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+p.apiKey)

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
			return "", fmt.Errorf("openai error (%d): %s", resp.StatusCode, errorResp.Error.Message)
		}
		return "", fmt.Errorf("openai error: status %d", resp.StatusCode)
	}

	// Parse successful response
	var completionResp chatCompletionResponse
	if err := json.Unmarshal(body, &completionResp); err != nil {
		return "", fmt.Errorf("failed to parse response: %w", err)
	}

	if len(completionResp.Choices) == 0 {
		return "", fmt.Errorf("no response choices returned")
	}

	return completionResp.Choices[0].Message.Content, nil
}

// Request/Response types for OpenAI API

type responseFormat struct {
	Type string `json:"type"`
}

type chatCompletionRequest struct {
	Model          string          `json:"model"`
	Messages       []message       `json:"messages"`
	Temperature    float32         `json:"temperature"`
	ResponseFormat *responseFormat `json:"response_format,omitempty"`
}

type message struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type chatCompletionResponse struct {
	ID      string   `json:"id"`
	Object  string   `json:"object"`
	Created int64    `json:"created"`
	Model   string   `json:"model"`
	Choices []choice `json:"choices"`
	Usage   usage    `json:"usage"`
}

type choice struct {
	Index        int     `json:"index"`
	Message      message `json:"message"`
	FinishReason string  `json:"finish_reason"`
}

type usage struct {
	PromptTokens     int `json:"prompt_tokens"`
	CompletionTokens int `json:"completion_tokens"`
	TotalTokens      int `json:"total_tokens"`
}

type errorResponse struct {
	Error struct {
		Message string `json:"message"`
		Type    string `json:"type"`
		Code    string `json:"code"`
	} `json:"error"`
}
