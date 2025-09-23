package bedrock

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestProviderCallClaude(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify headers
		if r.Header.Get("X-Amz-Date") == "" {
			t.Error("Expected X-Amz-Date header")
		}
		if r.Header.Get("Authorization") == "" {
			t.Error("Expected Authorization header")
		}

		// Verify request body
		var req claudeRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			t.Fatalf("Failed to decode request: %v", err)
		}

		if !strings.Contains(req.Prompt, "Human:") || !strings.Contains(req.Prompt, "Assistant:") {
			t.Errorf("Expected Claude prompt format, got: %s", req.Prompt)
		}

		// Send response
		resp := claudeResponse{
			Completion: "test response",
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	provider := &Provider{
		region:    "us-east-1",
		accessKey: "test-access",
		secretKey: "test-secret",
		model:     "anthropic.claude-v2",
		httpClient: http.DefaultClient,
	}

	// Override URL for testing
	provider.httpClient = &http.Client{
		Transport: &testTransport{
			server: server,
		},
	}

	response, err := provider.Call("test prompt", 0.7)
	if err != nil {
		t.Fatalf("Call failed: %v", err)
	}

	if response != "test response" {
		t.Errorf("Expected 'test response', got '%s'", response)
	}
}

func TestProviderCallTitan(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Send Titan response
		resp := titanResponse{
			Results: []struct {
				OutputText string `json:"outputText"`
			}{
				{OutputText: "test titan response"},
			},
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	provider := &Provider{
		region:    "us-east-1",
		accessKey: "test-access",
		secretKey: "test-secret",
		model:     "amazon.titan-text-express-v1",
		httpClient: &http.Client{
			Transport: &testTransport{
				server: server,
			},
		},
	}

	response, err := provider.Call("test prompt", 0.7)
	if err != nil {
		t.Fatalf("Call failed: %v", err)
	}

	if response != "test titan response" {
		t.Errorf("Expected 'test titan response', got '%s'", response)
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
			name:       "API error",
			statusCode: http.StatusBadRequest,
			responseBody: `{
				"message": "Invalid request"
			}`,
			expectedError: "bedrock error (400)",
		},
		{
			name:          "Generic error",
			statusCode:    http.StatusInternalServerError,
			responseBody:  `not json`,
			expectedError: "bedrock error: status 500",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(tt.statusCode)
				w.Write([]byte(tt.responseBody))
			}))
			defer server.Close()

			provider := &Provider{
				region:    "us-east-1",
				accessKey: "test",
				secretKey: "test",
				model:     "anthropic.claude-v2",
				httpClient: &http.Client{
					Transport: &testTransport{
						server: server,
					},
				},
			}

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

// testTransport redirects requests to test server
type testTransport struct {
	server *httptest.Server
}

func (t *testTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	// Redirect to test server
	req.URL.Scheme = "http"
	req.URL.Host = strings.TrimPrefix(t.server.URL, "http://")
	return http.DefaultTransport.RoundTrip(req)
}