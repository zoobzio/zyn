package anthropic

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"github.com/zoobzio/zyn"
)

func TestProviderCall(t *testing.T) {
	ctx := context.Background()
	// Create a test server that mimics Anthropic API
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify headers
		if r.Header.Get("x-api-key") != "test-key" {
			t.Errorf("Expected x-api-key header, got %s", r.Header.Get("x-api-key"))
		}
		if r.Header.Get("Content-Type") != "application/json" {
			t.Errorf("Expected Content-Type application/json, got %s", r.Header.Get("Content-Type"))
		}
		if r.Header.Get("anthropic-version") != "2023-06-01" {
			t.Errorf("Expected anthropic-version header, got %s", r.Header.Get("anthropic-version"))
		}

		// Verify request body
		var req messagesRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			t.Fatalf("Failed to decode request: %v", err)
		}

		if req.Model != "claude-sonnet-4-20250514" {
			t.Errorf("Expected model claude-sonnet-4-20250514, got %s", req.Model)
		}
		if req.Temperature != 0.7 {
			t.Errorf("Expected temperature 0.7, got %f", req.Temperature)
		}
		if len(req.Messages) != 1 || req.Messages[0].Content != "test prompt" {
			t.Errorf("Unexpected messages: %v", req.Messages)
		}

		// Send response
		resp := messagesResponse{
			ID:    "msg_test123",
			Type:  "message",
			Role:  "assistant",
			Model: "claude-sonnet-4-20250514",
			Content: []contentBlock{
				{
					Type: "text",
					Text: `{"decision": true, "confidence": 0.9, "reasoning": ["test"]}`,
				},
			},
			StopReason: "end_turn",
			Usage: usage{
				InputTokens:  10,
				OutputTokens: 5,
			},
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	// Create provider with test server URL
	provider := New(Config{
		APIKey:  "test-key",
		Model:   "claude-sonnet-4-20250514",
		BaseURL: server.URL,
	})

	// Make a call
	response, err := provider.Call(ctx, []zyn.Message{{Role: zyn.RoleUser, Content: "test prompt"}}, 0.7)
	if err != nil {
		t.Fatalf("Call failed: %v", err)
	}

	if !strings.Contains(response.Content, "decision") {
		t.Errorf("Expected JSON response with decision, got '%s'", response.Content)
	}

	if response.Usage.Prompt != 10 {
		t.Errorf("Expected 10 prompt tokens, got %d", response.Usage.Prompt)
	}
	if response.Usage.Completion != 5 {
		t.Errorf("Expected 5 completion tokens, got %d", response.Usage.Completion)
	}
}

func TestAnthropicIntegration(t *testing.T) {
	apiKey := os.Getenv("ANTHROPIC_API_KEY")
	if apiKey == "" {
		t.Skip("ANTHROPIC_API_KEY not set, skipping integration test")
	}

	ctx := context.Background()
	provider := New(Config{
		APIKey: apiKey,
		Model:  "claude-3-5-haiku-20241022",
	})

	response, err := provider.Call(ctx, []zyn.Message{{Role: zyn.RoleUser, Content: "Respond with exactly: {\"test\": true}"}}, 0.1)
	if err != nil {
		t.Fatalf("Call failed: %v", err)
	}

	if response.Content == "" {
		t.Error("Expected non-empty response")
	}

	t.Logf("Response: %s", response.Content)
}

func TestProviderErrorHandling(t *testing.T) {
	tests := []struct {
		name          string
		statusCode    int
		responseBody  string
		expectedError string
	}{
		{
			name:       "Rate limit error",
			statusCode: http.StatusTooManyRequests,
			responseBody: `{
				"type": "error",
				"error": {
					"type": "rate_limit_error",
					"message": "Rate limit exceeded"
				}
			}`,
			expectedError: "rate limit exceeded",
		},
		{
			name:       "Authentication error",
			statusCode: http.StatusUnauthorized,
			responseBody: `{
				"type": "error",
				"error": {
					"type": "authentication_error",
					"message": "Invalid API key"
				}
			}`,
			expectedError: "anthropic error (401): Invalid API key",
		},
		{
			name:          "Generic error",
			statusCode:    http.StatusInternalServerError,
			responseBody:  `not json`,
			expectedError: "anthropic error: status 500",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				w.WriteHeader(tt.statusCode)
				w.Write([]byte(tt.responseBody))
			}))
			defer server.Close()

			provider := New(Config{
				APIKey:  "test-key",
				BaseURL: server.URL,
			})

			_, err := provider.Call(ctx, []zyn.Message{{Role: zyn.RoleUser, Content: "test"}}, 0.7)
			if err == nil {
				t.Fatal("Expected error but got none")
			}

			if !strings.Contains(err.Error(), tt.expectedError) {
				t.Errorf("Expected error containing '%s', got '%s'", tt.expectedError, err.Error())
			}
		})
	}
}

func TestProviderName(t *testing.T) {
	provider := New(Config{
		APIKey: "test-key",
		Model:  "claude-sonnet-4-20250514",
	})

	name := provider.Name()
	if name != "anthropic" {
		t.Errorf("Expected 'anthropic', got '%s'", name)
	}
}

func TestProviderDefaults(t *testing.T) {
	provider := New(Config{
		APIKey: "test-key",
	})

	if provider.model != "claude-sonnet-4-20250514" {
		t.Errorf("Expected default model claude-sonnet-4-20250514, got %s", provider.model)
	}
	if provider.baseURL != "https://api.anthropic.com" {
		t.Errorf("Expected default baseURL, got %s", provider.baseURL)
	}
	if provider.maxTokens != 4096 {
		t.Errorf("Expected default maxTokens 4096, got %d", provider.maxTokens)
	}
}

func TestSystemMessage(t *testing.T) {
	ctx := context.Background()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req messagesRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			t.Fatalf("Failed to decode request: %v", err)
		}

		// Verify system message is extracted to separate field
		if req.System != "You are a helpful assistant.\n\nAlways respond in JSON." {
			t.Errorf("Expected combined system message, got %q", req.System)
		}

		// Verify only user/assistant messages in messages array
		if len(req.Messages) != 2 {
			t.Fatalf("Expected 2 messages (user + assistant), got %d", len(req.Messages))
		}
		if req.Messages[0].Role != "user" {
			t.Errorf("Expected first message role 'user', got %q", req.Messages[0].Role)
		}
		if req.Messages[1].Role != "assistant" {
			t.Errorf("Expected second message role 'assistant', got %q", req.Messages[1].Role)
		}

		resp := messagesResponse{
			ID:    "msg_test",
			Type:  "message",
			Role:  "assistant",
			Model: "claude-sonnet-4-20250514",
			Content: []contentBlock{
				{Type: "text", Text: `{"result": "ok"}`},
			},
			StopReason: "end_turn",
			Usage:      usage{InputTokens: 20, OutputTokens: 5},
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	provider := New(Config{
		APIKey:  "test-key",
		BaseURL: server.URL,
	})

	messages := []zyn.Message{
		{Role: zyn.RoleSystem, Content: "You are a helpful assistant."},
		{Role: zyn.RoleUser, Content: "Hello"},
		{Role: zyn.RoleSystem, Content: "Always respond in JSON."},
		{Role: zyn.RoleAssistant, Content: "Hi there"},
	}

	_, err := provider.Call(ctx, messages, 0.5)
	if err != nil {
		t.Fatalf("Call failed: %v", err)
	}
}

func TestNoSystemMessage(t *testing.T) {
	ctx := context.Background()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req messagesRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			t.Fatalf("Failed to decode request: %v", err)
		}

		// Verify no system field when no system messages
		if req.System != "" {
			t.Errorf("Expected empty system field, got %q", req.System)
		}

		resp := messagesResponse{
			ID:         "msg_test",
			Type:       "message",
			Role:       "assistant",
			Model:      "claude-sonnet-4-20250514",
			Content:    []contentBlock{{Type: "text", Text: `{"result": "ok"}`}},
			StopReason: "end_turn",
			Usage:      usage{InputTokens: 10, OutputTokens: 5},
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	provider := New(Config{
		APIKey:  "test-key",
		BaseURL: server.URL,
	})

	messages := []zyn.Message{
		{Role: zyn.RoleUser, Content: "Hello"},
	}

	_, err := provider.Call(ctx, messages, 0.5)
	if err != nil {
		t.Fatalf("Call failed: %v", err)
	}
}
