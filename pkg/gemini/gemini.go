package gemini

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

// Provider implements the zyn Provider interface for Google Gemini API.
type Provider struct {
	apiKey     string
	model      string
	baseURL    string
	httpClient *http.Client
	name       string
}

// Config holds configuration for the Gemini provider.
type Config struct {
	APIKey  string
	Model   string        // e.g. "gemini-1.5-flash", "gemini-1.5-pro"
	BaseURL string        // Optional, defaults to "https://generativelanguage.googleapis.com/v1beta"
	Timeout time.Duration // Optional, defaults to 30s
}

// New creates a new Gemini provider.
func New(config Config) *Provider {
	if config.Model == "" {
		config.Model = "gemini-1.5-flash"
	}
	if config.BaseURL == "" {
		config.BaseURL = "https://generativelanguage.googleapis.com/v1beta"
	}
	if config.Timeout == 0 {
		config.Timeout = 30 * time.Second
	}

	return &Provider{
		apiKey:  config.APIKey,
		model:   config.Model,
		baseURL: config.BaseURL,
		name:    "gemini",
		httpClient: &http.Client{
			Timeout: config.Timeout,
		},
	}
}

// Name returns the provider identifier.
func (p *Provider) Name() string {
	return p.name
}

// Call sends messages to Gemini and returns the response with usage stats.
func (p *Provider) Call(ctx context.Context, messages []zyn.Message, temperature float32) (*zyn.ProviderResponse, error) {
	startTime := time.Now()

	// Emit provider.call.started hook
	capitan.Info(ctx, zyn.ProviderCallStarted,
		zyn.ProviderKey.Field(p.name),
		zyn.ModelKey.Field(p.model),
	)

	// Extract system messages and conversation messages
	var systemParts []string
	var contents []content
	for _, msg := range messages {
		if msg.Role == zyn.RoleSystem {
			systemParts = append(systemParts, msg.Content)
		} else {
			role := msg.Role
			// Gemini uses "model" instead of "assistant"
			if role == zyn.RoleAssistant {
				role = "model"
			}
			contents = append(contents, content{
				Role: role,
				Parts: []part{
					{Text: msg.Content},
				},
			})
		}
	}

	// Build request body
	requestBody := generateContentRequest{
		Contents: contents,
		GenerationConfig: &generationConfig{
			Temperature:      temperature,
			ResponseMIMEType: "application/json",
		},
	}

	// Add system instruction if present
	if len(systemParts) > 0 {
		requestBody.SystemInstruction = &content{
			Parts: []part{
				{Text: strings.Join(systemParts, "\n\n")},
			},
		}
	}

	jsonBody, err := json.Marshal(requestBody)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	// Create HTTP request
	url := fmt.Sprintf("%s/models/%s:generateContent?key=%s", p.baseURL, p.model, p.apiKey)
	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(jsonBody))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")

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
				zyn.APIErrorTypeKey.Field(fmt.Sprintf("%d", errorResp.Error.Code)),
			)

			capitan.Error(ctx, zyn.ProviderCallFailed, fields...)

			// Check for rate limit
			if resp.StatusCode == http.StatusTooManyRequests {
				return nil, fmt.Errorf("rate limit exceeded: %s", errorResp.Error.Message)
			}
			return nil, fmt.Errorf("gemini error (%d): %s", resp.StatusCode, errorResp.Error.Message)
		}

		fields = append(fields, zyn.ErrorKey.Field(fmt.Sprintf("status %d", resp.StatusCode)))
		capitan.Error(ctx, zyn.ProviderCallFailed, fields...)
		return nil, fmt.Errorf("gemini error: status %d", resp.StatusCode)
	}

	// Parse successful response
	var generateResp generateContentResponse
	if err := json.Unmarshal(body, &generateResp); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	// Extract text content from response
	if len(generateResp.Candidates) == 0 {
		return nil, fmt.Errorf("no candidates in response")
	}

	candidate := generateResp.Candidates[0]
	var textContent string
	for _, part := range candidate.Content.Parts {
		if part.Text != "" {
			textContent = part.Text
			break
		}
	}

	if textContent == "" {
		return nil, fmt.Errorf("no text content in response")
	}

	// Calculate duration
	duration := time.Since(startTime)

	// Calculate token counts
	promptTokens := generateResp.UsageMetadata.PromptTokenCount
	completionTokens := generateResp.UsageMetadata.CandidatesTokenCount
	totalTokens := generateResp.UsageMetadata.TotalTokenCount

	// Emit provider.call.completed hook with token usage and metadata
	fields := []capitan.Field{
		zyn.ProviderKey.Field(p.name),
		zyn.ModelKey.Field(p.model),
		zyn.PromptTokensKey.Field(promptTokens),
		zyn.CompletionTokensKey.Field(completionTokens),
		zyn.TotalTokensKey.Field(totalTokens),
		zyn.DurationMsKey.Field(int(duration.Milliseconds())),
		zyn.HTTPStatusCodeKey.Field(resp.StatusCode),
	}

	if candidate.FinishReason != "" {
		fields = append(fields, zyn.ResponseFinishReasonKey.Field(candidate.FinishReason))
	}

	capitan.Info(ctx, zyn.ProviderCallCompleted, fields...)

	return &zyn.ProviderResponse{
		Content: textContent,
		Usage: zyn.TokenUsage{
			Prompt:     promptTokens,
			Completion: completionTokens,
			Total:      totalTokens,
		},
	}, nil
}

// Request/Response types for Gemini API

type generateContentRequest struct {
	Contents          []content         `json:"contents"`
	SystemInstruction *content          `json:"systemInstruction,omitempty"`
	GenerationConfig  *generationConfig `json:"generationConfig,omitempty"`
}

type content struct {
	Role  string `json:"role"`
	Parts []part `json:"parts"`
}

type part struct {
	Text string `json:"text"`
}

type generationConfig struct {
	Temperature      float32 `json:"temperature,omitempty"`
	TopP             float32 `json:"topP,omitempty"`
	TopK             int     `json:"topK,omitempty"`
	MaxOutputTokens  int     `json:"maxOutputTokens,omitempty"`
	ResponseMIMEType string  `json:"responseMimeType,omitempty"`
}

type generateContentResponse struct {
	Candidates    []candidate   `json:"candidates"`
	UsageMetadata usageMetadata `json:"usageMetadata"`
}

type candidate struct {
	Content      content `json:"content"`
	FinishReason string  `json:"finishReason"`
	Index        int     `json:"index"`
}

type usageMetadata struct {
	PromptTokenCount     int `json:"promptTokenCount"`
	CandidatesTokenCount int `json:"candidatesTokenCount"`
	TotalTokenCount      int `json:"totalTokenCount"`
}

type errorResponse struct {
	Error struct {
		Code    int    `json:"code"`
		Message string `json:"message"`
		Status  string `json:"status"`
	} `json:"error"`
}
