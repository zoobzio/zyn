package google

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

// Provider implements the zyn Provider interface for Google Gemini API.
type Provider struct {
	apiKey     string
	model      string
	baseURL    string
	httpClient *http.Client
}

// Config holds configuration for the Google provider.
type Config struct {
	APIKey  string
	Model   string        // e.g. "gemini-pro", "gemini-1.5-pro"
	BaseURL string        // Optional, defaults to Google AI API
	Timeout time.Duration // Optional, defaults to 30s
}

// New creates a new Google provider.
func New(config Config) *Provider {
	if config.Model == "" {
		config.Model = "gemini-pro"
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
		httpClient: &http.Client{
			Timeout: config.Timeout,
		},
	}
}

// Call sends a prompt to Google Gemini and returns the response.
func (p *Provider) Call(prompt string, temperature float32) (string, error) {
	// Build request body
	requestBody := generateContentRequest{
		Contents: []content{
			{
				Parts: []part{
					{
						Text: prompt,
					},
				},
			},
		},
		GenerationConfig: generationConfig{
			Temperature: temperature,
		},
	}

	jsonBody, err := json.Marshal(requestBody)
	if err != nil {
		return "", fmt.Errorf("failed to marshal request: %w", err)
	}

	// Create URL with API key
	url := fmt.Sprintf("%s/models/%s:generateContent?key=%s", p.baseURL, p.model, p.apiKey)

	// Create HTTP request
	req, err := http.NewRequest("POST", url, bytes.NewReader(jsonBody))
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")

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
			return "", fmt.Errorf("google error (%d): %s", resp.StatusCode, errorResp.Error.Message)
		}
		return "", fmt.Errorf("google error: status %d", resp.StatusCode)
	}

	// Parse successful response
	var generateResp generateContentResponse
	if err := json.Unmarshal(body, &generateResp); err != nil {
		return "", fmt.Errorf("failed to parse response: %w", err)
	}

	// Extract text from candidates
	if len(generateResp.Candidates) == 0 {
		return "", fmt.Errorf("no response candidates")
	}

	var result string
	for _, candidate := range generateResp.Candidates {
		if candidate.Content != nil {
			for _, part := range candidate.Content.Parts {
				result += part.Text
			}
		}
	}

	if result == "" {
		return "", fmt.Errorf("no text content in response")
	}

	return result, nil
}

// Request/Response types for Google API

type generateContentRequest struct {
	Contents         []content        `json:"contents"`
	GenerationConfig generationConfig `json:"generationConfig,omitempty"`
}

type content struct {
	Parts []part `json:"parts"`
	Role  string `json:"role,omitempty"`
}

type part struct {
	Text string `json:"text"`
}

type generationConfig struct {
	Temperature float32 `json:"temperature,omitempty"`
}

type generateContentResponse struct {
	Candidates []candidate `json:"candidates"`
}

type candidate struct {
	Content      *content     `json:"content"`
	FinishReason string       `json:"finishReason"`
	SafetyRatings []safetyRating `json:"safetyRatings,omitempty"`
}

type safetyRating struct {
	Category    string `json:"category"`
	Probability string `json:"probability"`
}

type errorResponse struct {
	Error struct {
		Code    int    `json:"code"`
		Message string `json:"message"`
		Status  string `json:"status"`
	} `json:"error"`
}