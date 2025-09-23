package azure

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestProviderCall(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify headers
		if r.Header.Get("api-key") != "test-key" {
			t.Errorf("Expected api-key header, got %s", r.Header.Get("api-key"))
		}

		// Verify URL structure
		expectedPath := "/openai/deployments/test-deployment/chat/completions"
		if !strings.Contains(r.URL.Path, expectedPath) {
			t.Errorf("Expected path to contain %s, got %s", expectedPath, r.URL.Path)
		}

		// Send response
		resp := chatCompletionResponse{
			ID:      "test-id",
			Object:  "chat.completion",
			Created: 1234567890,
			Model:   "gpt-4",
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
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	provider := New(Config{
		Endpoint:   server.URL,
		APIKey:     "test-key",
		Deployment: "test-deployment",
	})

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
			expectedError: "azure error (400)",
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
				Endpoint:   server.URL,
				APIKey:     "test-key",
				Deployment: "test",
			})

			_, err := provider.Call("test", 0.7)
			if err == nil {
				t.Fatal("Expected error")
			}
			if !strings.Contains(err.Error(), tt.expectedError) {
				t.Errorf("Expected error containing '%s', got '%s'", tt.expectedError, err.Error())
			}
		})
	}
}