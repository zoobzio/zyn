package bedrock

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

// Provider implements the zyn Provider interface for AWS Bedrock.
type Provider struct {
	region     string
	accessKey  string
	secretKey  string
	model      string
	httpClient *http.Client
}

// Config holds configuration for the Bedrock provider.
type Config struct {
	Region    string        // AWS region (e.g. "us-east-1")
	AccessKey string        // AWS access key
	SecretKey string        // AWS secret key
	Model     string        // Model ID (e.g. "anthropic.claude-v2", "amazon.titan-text-express-v1")
	Timeout   time.Duration // Optional, defaults to 30s
}

// New creates a new Bedrock provider.
func New(config Config) *Provider {
	if config.Model == "" {
		config.Model = "anthropic.claude-v2"
	}
	if config.Timeout == 0 {
		config.Timeout = 30 * time.Second
	}

	return &Provider{
		region:    config.Region,
		accessKey: config.AccessKey,
		secretKey: config.SecretKey,
		model:     config.Model,
		httpClient: &http.Client{
			Timeout: config.Timeout,
		},
	}
}

// Call sends a prompt to Bedrock and returns the response.
func (p *Provider) Call(ctx context.Context, prompt string, temperature float32) (string, error) {
	// Build request body based on model
	var requestBody interface{}
	var contentType string

	switch {
	case contains(p.model, "claude"):
		// Anthropic Claude format
		requestBody = claudeRequest{
			Prompt:      fmt.Sprintf("\n\nHuman: %s\n\nAssistant:", prompt),
			MaxTokens:   4096,
			Temperature: temperature,
		}
		contentType = "application/json"
	case contains(p.model, "titan"):
		// Amazon Titan format
		requestBody = titanRequest{
			InputText: prompt,
			TextGenerationConfig: titanConfig{
				Temperature: temperature,
				MaxTokens:   4096,
			},
		}
		contentType = "application/json"
	default:
		return "", fmt.Errorf("unsupported model: %s", p.model)
	}

	jsonBody, err := json.Marshal(requestBody)
	if err != nil {
		return "", fmt.Errorf("failed to marshal request: %w", err)
	}

	// Build URL
	url := fmt.Sprintf("https://bedrock-runtime.%s.amazonaws.com/model/%s/invoke",
		p.region, p.model)

	// Create request
	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(jsonBody))
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}

	// Add headers
	req.Header.Set("Content-Type", contentType)
	req.Header.Set("Accept", "application/json")

	// Sign request with AWS Signature V4 (simplified for demo)
	p.signRequest(req, jsonBody)

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
		var errorResp bedrockError
		if err := json.Unmarshal(body, &errorResp); err == nil && errorResp.Message != "" {
			return "", fmt.Errorf("bedrock error (%d): %s", resp.StatusCode, errorResp.Message)
		}
		return "", fmt.Errorf("bedrock error: status %d", resp.StatusCode)
	}

	// Parse response based on model
	if contains(p.model, "claude") {
		var claudeResp claudeResponse
		if err := json.Unmarshal(body, &claudeResp); err != nil {
			return "", fmt.Errorf("failed to parse claude response: %w", err)
		}
		return claudeResp.Completion, nil
	} else if contains(p.model, "titan") {
		var titanResp titanResponse
		if err := json.Unmarshal(body, &titanResp); err != nil {
			return "", fmt.Errorf("failed to parse titan response: %w", err)
		}
		if len(titanResp.Results) == 0 {
			return "", fmt.Errorf("no results in response")
		}
		return titanResp.Results[0].OutputText, nil
	}

	return "", fmt.Errorf("failed to parse response for model: %s", p.model)
}

// signRequest adds simplified AWS Signature V4 headers.
func (p *Provider) signRequest(req *http.Request, _ []byte) {
	now := time.Now().UTC()
	dateStr := now.Format("20060102T150405Z")

	// Add required headers
	req.Header.Set("X-Amz-Date", dateStr)

	// Simplified signature (in production, use AWS SDK for proper signing)
	h := hmac.New(sha256.New, []byte("AWS4"+p.secretKey))
	h.Write([]byte(dateStr[:8]))
	h.Write([]byte(p.region))
	h.Write([]byte("bedrock"))
	h.Write([]byte("aws4_request"))
	signature := hex.EncodeToString(h.Sum(nil))

	authHeader := fmt.Sprintf("AWS4-HMAC-SHA256 Credential=%s/%s/%s/bedrock/aws4_request, SignedHeaders=content-type;host;x-amz-date, Signature=%s",
		p.accessKey, dateStr[:8], p.region, signature)
	req.Header.Set("Authorization", authHeader)
}

// Request types for different models

type claudeRequest struct {
	Prompt      string  `json:"prompt"`
	MaxTokens   int     `json:"max_tokens_to_sample"`
	Temperature float32 `json:"temperature"`
}

type claudeResponse struct {
	Completion string `json:"completion"`
}

type titanRequest struct {
	InputText            string      `json:"inputText"`
	TextGenerationConfig titanConfig `json:"textGenerationConfig"`
}

type titanConfig struct {
	Temperature float32 `json:"temperature"`
	MaxTokens   int     `json:"maxTokenCount"`
}

type titanResponse struct {
	Results []struct {
		OutputText string `json:"outputText"`
	} `json:"results"`
}

type bedrockError struct {
	Message string `json:"message"`
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) &&
		bytes.Contains([]byte(s), []byte(substr))
}
