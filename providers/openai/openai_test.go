package openai

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestProviderCall(t *testing.T) {
	// Create a test server that mimics OpenAI API
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify headers
		if r.Header.Get("Authorization") != "Bearer test-key" {
			t.Errorf("Expected Bearer token, got %s", r.Header.Get("Authorization"))
		}
		if r.Header.Get("Content-Type") != "application/json" {
			t.Errorf("Expected Content-Type application/json, got %s", r.Header.Get("Content-Type"))
		}

		// Verify request body
		var req chatCompletionRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			t.Fatalf("Failed to decode request: %v", err)
		}

		if req.Model != "gpt-3.5-turbo" {
			t.Errorf("Expected model gpt-3.5-turbo, got %s", req.Model)
		}
		if req.Temperature != 0.7 {
			t.Errorf("Expected temperature 0.7, got %f", req.Temperature)
		}
		if len(req.Messages) != 1 || req.Messages[0].Content != "test prompt" {
			t.Errorf("Unexpected prompt: %v", req.Messages)
		}

		// Send response
		resp := chatCompletionResponse{
			ID:      "test-id",
			Object:  "chat.completion",
			Created: 1234567890,
			Model:   "gpt-3.5-turbo",
			Choices: []choice{
				{
					Index: 0,
					Message: message{
						Role:    "assistant",
						Content: "test response",
					},
					FinishReason: "stop",
				},
			},
			Usage: usage{
				PromptTokens:     10,
				CompletionTokens: 5,
				TotalTokens:      15,
			},
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	// Create provider with test server URL
	provider := New(Config{
		APIKey:  "test-key",
		Model:   "gpt-3.5-turbo",
		BaseURL: server.URL,
	})

	// Make a call
	response, err := provider.Call("test prompt", 0.7)
	if err != nil {
		t.Fatalf("Call failed: %v", err)
	}

	if response != "test response" {
		t.Errorf("Expected 'test response', got '%s'", response)
	}
}

func TestProviderErrorHandling(t *testing.T) {
	tests := []struct {
		name           string
		statusCode     int
		responseBody   string
		expectedError  string
	}{
		{
			name:       "Rate limit error",
			statusCode: http.StatusTooManyRequests,
			responseBody: `{
				"error": {
					"message": "Rate limit exceeded",
					"type": "rate_limit_error",
					"code": "rate_limit"
				}
			}`,
			expectedError: "rate limit exceeded",
		},
		{
			name:       "API error",
			statusCode: http.StatusBadRequest,
			responseBody: `{
				"error": {
					"message": "Invalid request",
					"type": "invalid_request_error"
				}
			}`,
			expectedError: "openai error (400): Invalid request",
		},
		{
			name:          "Generic error",
			statusCode:    http.StatusInternalServerError,
			responseBody:  `not json`,
			expectedError: "openai error: status 500",
		},
		{
			name:         "Empty response",
			statusCode:   http.StatusOK,
			responseBody: `{"choices": []}`,
			expectedError: "no response choices returned",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(tt.statusCode)
				w.Write([]byte(tt.responseBody))
			}))
			defer server.Close()

			provider := New(Config{
				APIKey:  "test-key",
				BaseURL: server.URL,
			})

			_, err := provider.Call("test", 0.7)
			if err == nil {
				t.Fatal("Expected error but got none")
			}

			if !strings.Contains(err.Error(), tt.expectedError) {
				t.Errorf("Expected error containing '%s', got '%s'", tt.expectedError, err.Error())
			}
		})
	}
}