package anthropic

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestProviderCall(t *testing.T) {
	ctx := context.Background()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify headers
		if r.Header.Get("x-api-key") != "test-key" {
			t.Errorf("Expected x-api-key header, got %s", r.Header.Get("x-api-key"))
		}
		if r.Header.Get("anthropic-version") != "2023-06-01" {
			t.Errorf("Expected anthropic-version header, got %s", r.Header.Get("anthropic-version"))
		}

		// Send response
		resp := messagesResponse{
			ID:   "msg_123",
			Type: "message",
			Role: "assistant",
			Content: []content{
				{
					Type: "text",
					Text: "test response",
				},
			},
			Model: "claude-3-sonnet",
			Usage: usage{
				InputTokens:  10,
				OutputTokens: 5,
			},
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	provider := New(Config{
		APIKey:  "test-key",
		BaseURL: server.URL,
	})

	response, err := provider.Call(ctx, "test prompt", 0.7)
	if err != nil {
		t.Fatalf("Call failed: %v", err)
	}

	if response != "test response" {
		t.Errorf("Expected 'test response', got '%s'", response)
	}
}

func TestProviderErrorHandling(t *testing.T) {
	tests := []struct {
		name          string
		statusCode    int
		responseBody  string
		expectedError string
	}{
		{
			name:       "Rate limit",
			statusCode: http.StatusTooManyRequests,
			responseBody: `{
				"error": {
					"type": "rate_limit_error",
					"message": "Rate limit exceeded"
				}
			}`,
			expectedError: "rate limit exceeded",
		},
		{
			name:       "API error",
			statusCode: http.StatusBadRequest,
			responseBody: `{
				"error": {
					"type": "invalid_request_error",
					"message": "Invalid request"
				}
			}`,
			expectedError: "anthropic error (400)",
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

			_, err := provider.Call(ctx, "test", 0.7)
			if err == nil {
				t.Fatal("Expected error")
			}
			if !strings.Contains(err.Error(), tt.expectedError) {
				t.Errorf("Expected error containing '%s', got '%s'", tt.expectedError, err.Error())
			}
		})
	}
}
