package anthropic

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/zoobzio/capitan"
	"github.com/zoobzio/zyn"
)

// Provider implements the zyn Provider interface for Anthropic API.
type Provider struct {
	apiKey     string
	model      string
	baseURL    string
	maxTokens  int
	httpClient *http.Client
	name       string
}

// Config holds configuration for the Anthropic provider.
type Config struct {
	APIKey    string
	Model     string        // e.g. "claude-sonnet-4-20250514", "claude-3-5-haiku-20241022"
	BaseURL   string        // Optional, defaults to "https://api.anthropic.com"
	MaxTokens int           // Optional, defaults to 4096
	Timeout   time.Duration // Optional, defaults to 30s
}

// New creates a new Anthropic provider.
func New(config Config) *Provider {
	if config.Model == "" {
		config.Model = "claude-sonnet-4-20250514"
	}
	if config.BaseURL == "" {
		config.BaseURL = "https://api.anthropic.com"
	}
	if config.MaxTokens == 0 {
		config.MaxTokens = 4096
	}
	if config.Timeout == 0 {
		config.Timeout = 30 * time.Second
	}

	return &Provider{
		apiKey:    config.APIKey,
		model:     config.Model,
		baseURL:   config.BaseURL,
		maxTokens: config.MaxTokens,
		name:      "anthropic",
		httpClient: &http.Client{
			Timeout: config.Timeout,
		},
	}
}

// Name returns the provider identifier.
func (p *Provider) Name() string {
	return p.name
}

// Call sends messages to Anthropic and returns the response with usage stats.
func (p *Provider) Call(ctx context.Context, messages []zyn.Message, temperature float32) (*zyn.ProviderResponse, error) {
	startTime := time.Now()

	// Emit provider.call.started hook
	capitan.Info(ctx, zyn.ProviderCallStarted,
		zyn.ProviderKey.Field(p.name),
		zyn.ModelKey.Field(p.model),
	)

	// Extract system messages and conversation messages
	var systemParts []string
	var apiMessages []message
	for _, msg := range messages {
		if msg.Role == zyn.RoleSystem {
			systemParts = append(systemParts, msg.Content)
		} else {
			apiMessages = append(apiMessages, message{
				Role:    msg.Role,
				Content: msg.Content,
			})
		}
	}

	// Build request body
	requestBody := messagesRequest{
		Model:       p.model,
		Messages:    apiMessages,
		MaxTokens:   p.maxTokens,
		Temperature: temperature,
	}

	// Add system message if present
	if len(systemParts) > 0 {
		requestBody.System = strings.Join(systemParts, "\n\n")
	}

	jsonBody, err := json.Marshal(requestBody)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	// Create HTTP request
	req, err := http.NewRequestWithContext(ctx, "POST", p.baseURL+"/v1/messages", bytes.NewReader(jsonBody))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("x-api-key", p.apiKey)
	req.Header.Set("anthropic-version", "2023-06-01")

	// Make the request
	resp, err := p.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	// Read response body
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
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

			capitan.Error(ctx, zyn.ProviderCallFailed, fields...)

			// Check for rate limit
			if resp.StatusCode == http.StatusTooManyRequests {
				return nil, fmt.Errorf("rate limit exceeded: %s", errorResp.Error.Message)
			}
			return nil, fmt.Errorf("anthropic error (%d): %s", resp.StatusCode, errorResp.Error.Message)
		}

		fields = append(fields, zyn.ErrorKey.Field(fmt.Sprintf("status %d", resp.StatusCode)))
		capitan.Error(ctx, zyn.ProviderCallFailed, fields...)
		return nil, fmt.Errorf("anthropic error: status %d", resp.StatusCode)
	}

	// Parse successful response
	var messagesResp messagesResponse
	if err := json.Unmarshal(body, &messagesResp); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	// Extract text content from response
	var content string
	for _, block := range messagesResp.Content {
		if block.Type == "text" {
			content = block.Text
			break
		}
	}

	if content == "" {
		return nil, fmt.Errorf("no text content in response")
	}

	// Calculate duration
	duration := time.Since(startTime)

	// Emit provider.call.completed hook with token usage and metadata
	fields := []capitan.Field{
		zyn.ProviderKey.Field(p.name),
		zyn.ModelKey.Field(messagesResp.Model),
		zyn.PromptTokensKey.Field(messagesResp.Usage.InputTokens),
		zyn.CompletionTokensKey.Field(messagesResp.Usage.OutputTokens),
		zyn.TotalTokensKey.Field(messagesResp.Usage.InputTokens + messagesResp.Usage.OutputTokens),
		zyn.DurationMsKey.Field(int(duration.Milliseconds())),
		zyn.HTTPStatusCodeKey.Field(resp.StatusCode),
		zyn.ResponseIDKey.Field(messagesResp.ID),
	}

	if messagesResp.StopReason != "" {
		fields = append(fields, zyn.ResponseFinishReasonKey.Field(messagesResp.StopReason))
	}

	capitan.Info(ctx, zyn.ProviderCallCompleted, fields...)

	return &zyn.ProviderResponse{
		Content: content,
		Usage: zyn.TokenUsage{
			Prompt:     messagesResp.Usage.InputTokens,
			Completion: messagesResp.Usage.OutputTokens,
			Total:      messagesResp.Usage.InputTokens + messagesResp.Usage.OutputTokens,
		},
	}, nil
}

// Request/Response types for Anthropic API

type messagesRequest struct {
	Model       string    `json:"model"`
	Messages    []message `json:"messages"`
	MaxTokens   int       `json:"max_tokens"`
	Temperature float32   `json:"temperature,omitempty"`
	System      string    `json:"system,omitempty"`
}

type message struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type messagesResponse struct {
	ID         string         `json:"id"`
	Type       string         `json:"type"`
	Role       string         `json:"role"`
	Model      string         `json:"model"`
	Content    []contentBlock `json:"content"`
	StopReason string         `json:"stop_reason"`
	Usage      usage          `json:"usage"`
}

type contentBlock struct {
	Type string `json:"type"`
	Text string `json:"text,omitempty"`
}

type usage struct {
	InputTokens  int `json:"input_tokens"`
	OutputTokens int `json:"output_tokens"`
}

type errorResponse struct {
	Type  string `json:"type"`
	Error struct {
		Type    string `json:"type"`
		Message string `json:"message"`
	} `json:"error"`
}
