package openai

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/zoobzio/capitan"
	"github.com/zoobzio/zyn"
)

// Provider implements the zyn Provider interface for OpenAI API.
type Provider struct {
	apiKey     string
	model      string
	baseURL    string
	httpClient *http.Client
	name       string
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
		name:    "openai",
		httpClient: &http.Client{
			Timeout: config.Timeout,
		},
	}
}

// Name returns the provider identifier.
func (p *Provider) Name() string {
	return p.name
}

// Call sends a prompt to OpenAI and returns the response.
func (p *Provider) Call(ctx context.Context, prompt string, temperature float32) (string, error) {
	startTime := time.Now()

	// Emit provider.call.started hook
	capitan.Emit(ctx, zyn.ProviderCallStarted,
		zyn.ProviderKey.Field(p.name),
		zyn.ModelKey.Field(p.model),
	)

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
		duration := time.Since(startTime)
		var errorResp errorResponse

		// Emit provider.call.failed hook
		fields := []capitan.Field{
			zyn.ProviderKey.Field(p.name),
			zyn.ModelKey.Field(p.model),
			zyn.HTTPStatusCodeKey.Field(resp.StatusCode),
			zyn.DurationMsKey.Field(int(duration.Milliseconds())),
		}

		if err := json.Unmarshal(body, &errorResp); err == nil && errorResp.Error.Message != "" {
			fields = append(fields,
				zyn.ErrorKey.Field(errorResp.Error.Message),
				zyn.APIErrorTypeKey.Field(errorResp.Error.Type),
			)
			if errorResp.Error.Code != "" {
				fields = append(fields, zyn.APIErrorCodeKey.Field(errorResp.Error.Code))
			}

			capitan.Emit(ctx, zyn.ProviderCallFailed, fields...)

			// Check for rate limit
			if resp.StatusCode == http.StatusTooManyRequests {
				return "", fmt.Errorf("rate limit exceeded: %s", errorResp.Error.Message)
			}
			return "", fmt.Errorf("openai error (%d): %s", resp.StatusCode, errorResp.Error.Message)
		}

		fields = append(fields, zyn.ErrorKey.Field(fmt.Sprintf("status %d", resp.StatusCode)))
		capitan.Emit(ctx, zyn.ProviderCallFailed, fields...)
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

	// Calculate duration
	duration := time.Since(startTime)

	// Emit provider.call.completed hook with token usage and metadata
	fields := []capitan.Field{
		zyn.ProviderKey.Field(p.name),
		zyn.ModelKey.Field(completionResp.Model),
		zyn.PromptTokensKey.Field(completionResp.Usage.PromptTokens),
		zyn.CompletionTokensKey.Field(completionResp.Usage.CompletionTokens),
		zyn.TotalTokensKey.Field(completionResp.Usage.TotalTokens),
		zyn.DurationMsKey.Field(int(duration.Milliseconds())),
		zyn.HTTPStatusCodeKey.Field(resp.StatusCode),
		zyn.ResponseIDKey.Field(completionResp.ID),
		zyn.ResponseCreatedKey.Field(int(completionResp.Created)),
	}

	if len(completionResp.Choices) > 0 && completionResp.Choices[0].FinishReason != "" {
		fields = append(fields, zyn.ResponseFinishReasonKey.Field(completionResp.Choices[0].FinishReason))
	}

	capitan.Emit(ctx, zyn.ProviderCallCompleted, fields...)

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
