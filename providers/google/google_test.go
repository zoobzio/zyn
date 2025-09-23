package google

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestProviderCall(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify URL contains API key
		if !strings.Contains(r.URL.String(), "key=test-key") {
			t.Errorf("Expected API key in URL, got %s", r.URL.String())
		}

		// Send response
		resp := generateContentResponse{
			Candidates: []candidate{
				{
					Content: &content{
						Parts: []part{
							{Text: "test response"},
						},
					},
					FinishReason: "STOP",
				},
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
					"code": 429,
					"message": "Resource exhausted",
					"status": "RESOURCE_EXHAUSTED"
				}
			}`,
			expectedError: "rate limit exceeded",
		},
		{
			name:       "API error",
			statusCode: http.StatusBadRequest,
			responseBody: `{
				"error": {
					"code": 400,
					"message": "Invalid request",
					"status": "INVALID_ARGUMENT"
				}
			}`,
			expectedError: "google error (400)",
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
				t.Fatal("Expected error")
			}
			if !strings.Contains(err.Error(), tt.expectedError) {
				t.Errorf("Expected error containing '%s', got '%s'", tt.expectedError, err.Error())
			}
		})
	}
}