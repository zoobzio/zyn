package azure

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

// Provider implements the zyn Provider interface for Azure OpenAI Service.
type Provider struct {
	endpoint    string
	apiKey      string
	deployment  string
	apiVersion  string
	httpClient  *http.Client
}

// Config holds configuration for the Azure provider.
type Config struct {
	Endpoint   string        // Your Azure OpenAI endpoint (https://{your-resource}.openai.azure.com)
	APIKey     string        // Your Azure API key
	Deployment string        // Your deployment name
	APIVersion string        // API version, defaults to "2024-02-01"
	Timeout    time.Duration // Optional, defaults to 30s
}

// New creates a new Azure OpenAI provider.
func New(config Config) *Provider {
	if config.APIVersion == "" {
		config.APIVersion = "2024-02-01"
	}
	if config.Timeout == 0 {
		config.Timeout = 30 * time.Second
	}

	return &Provider{
		endpoint:   config.Endpoint,
		apiKey:     config.APIKey,
		deployment: config.Deployment,
		apiVersion: config.APIVersion,
		httpClient: &http.Client{
			Timeout: config.Timeout,
		},
	}
}

// Call sends a prompt to Azure OpenAI and returns the response.
func (p *Provider) Call(prompt string, temperature float32) (string, error) {
	// Build request body (same as OpenAI)
	requestBody := chatCompletionRequest{
		Messages: []message{
			{
				Role:    "user",
				Content: prompt,
			},
		},
		Temperature: temperature,
	}

	jsonBody, err := json.Marshal(requestBody)
	if err != nil {
		return "", fmt.Errorf("failed to marshal request: %w", err)
	}

	// Build Azure-specific URL
	url := fmt.Sprintf("%s/openai/deployments/%s/chat/completions?api-version=%s",
		p.endpoint, p.deployment, p.apiVersion)

	// Create HTTP request
	req, err := http.NewRequest("POST", url, bytes.NewReader(jsonBody))
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("api-key", p.apiKey)

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
			return "", fmt.Errorf("azure error (%d): %s", resp.StatusCode, errorResp.Error.Message)
		}
		return "", fmt.Errorf("azure error: status %d", resp.StatusCode)
	}

	// Parse successful response (same as OpenAI)
	var completionResp chatCompletionResponse
	if err := json.Unmarshal(body, &completionResp); err != nil {
		return "", fmt.Errorf("failed to parse response: %w", err)
	}

	if len(completionResp.Choices) == 0 {
		return "", fmt.Errorf("no response choices returned")
	}

	return completionResp.Choices[0].Message.Content, nil
}

// Request/Response types (compatible with OpenAI)

type chatCompletionRequest struct {
	Messages    []message `json:"messages"`
	Temperature float32   `json:"temperature"`
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
}

type choice struct {
	Index        int     `json:"index"`
	Message      message `json:"message"`
	FinishReason string  `json:"finish_reason"`
}

type errorResponse struct {
	Error struct {
		Message string `json:"message"`
		Type    string `json:"type"`
		Code    string `json:"code"`
	} `json:"error"`
}