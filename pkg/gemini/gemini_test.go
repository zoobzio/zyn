package gemini

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
	// Create a test server that mimics Gemini API
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify API key in query params
		if r.URL.Query().Get("key") != "test-key" {
			t.Errorf("Expected key query param, got %s", r.URL.Query().Get("key"))
		}
		if r.Header.Get("Content-Type") != "application/json" {
			t.Errorf("Expected Content-Type application/json, got %s", r.Header.Get("Content-Type"))
		}

		// Verify request body
		var req generateContentRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			t.Fatalf("Failed to decode request: %v", err)
		}

		if len(req.Contents) != 1 {
			t.Errorf("Expected 1 content, got %d", len(req.Contents))
		}
		if req.Contents[0].Role != "user" {
			t.Errorf("Expected role user, got %s", req.Contents[0].Role)
		}
		if req.Contents[0].Parts[0].Text != "test prompt" {
			t.Errorf("Expected text 'test prompt', got %s", req.Contents[0].Parts[0].Text)
		}

		// Send response
		resp := generateContentResponse{
			Candidates: []candidate{
				{
					Content: content{
						Role: "model",
						Parts: []part{
							{Text: `{"decision": true, "confidence": 0.9, "reasoning": ["test"]}`},
						},
					},
					FinishReason: "STOP",
					Index:        0,
				},
			},
			UsageMetadata: usageMetadata{
				PromptTokenCount:     10,
				CandidatesTokenCount: 5,
				TotalTokenCount:      15,
			},
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	// Create provider with test server URL
	provider := New(Config{
		APIKey:  "test-key",
		Model:   "gemini-1.5-flash",
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
	if response.Usage.Total != 15 {
		t.Errorf("Expected 15 total tokens, got %d", response.Usage.Total)
	}
}

func TestGeminiIntegration(t *testing.T) {
	apiKey := os.Getenv("GEMINI_API_KEY")
	if apiKey == "" {
		t.Skip("GEMINI_API_KEY not set, skipping integration test")
	}

	ctx := context.Background()
	provider := New(Config{
		APIKey: apiKey,
		Model:  "gemini-1.5-flash",
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
				"error": {
					"code": 429,
					"message": "Rate limit exceeded",
					"status": "RESOURCE_EXHAUSTED"
				}
			}`,
			expectedError: "rate limit exceeded",
		},
		{
			name:       "API key error",
			statusCode: http.StatusBadRequest,
			responseBody: `{
				"error": {
					"code": 400,
					"message": "API key not valid",
					"status": "INVALID_ARGUMENT"
				}
			}`,
			expectedError: "gemini error (400): API key not valid",
		},
		{
			name:          "Generic error",
			statusCode:    http.StatusInternalServerError,
			responseBody:  `not json`,
			expectedError: "gemini error: status 500",
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
		Model:  "gemini-1.5-pro",
	})

	name := provider.Name()
	if name != "gemini" {
		t.Errorf("Expected 'gemini', got '%s'", name)
	}
}

func TestProviderDefaults(t *testing.T) {
	provider := New(Config{
		APIKey: "test-key",
	})

	if provider.model != "gemini-1.5-flash" {
		t.Errorf("Expected default model gemini-1.5-flash, got %s", provider.model)
	}
	if provider.baseURL != "https://generativelanguage.googleapis.com/v1beta" {
		t.Errorf("Expected default baseURL, got %s", provider.baseURL)
	}
}

func TestRoleConversion(t *testing.T) {
	ctx := context.Background()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req generateContentRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			t.Fatalf("Failed to decode request: %v", err)
		}

		// Verify role conversion
		if len(req.Contents) != 2 {
			t.Fatalf("Expected 2 contents, got %d", len(req.Contents))
		}
		if req.Contents[0].Role != "user" {
			t.Errorf("Expected first role 'user', got '%s'", req.Contents[0].Role)
		}
		if req.Contents[1].Role != "model" {
			t.Errorf("Expected second role 'model' (converted from assistant), got '%s'", req.Contents[1].Role)
		}

		resp := generateContentResponse{
			Candidates: []candidate{
				{
					Content: content{
						Role:  "model",
						Parts: []part{{Text: `{"result": "ok"}`}},
					},
					FinishReason: "STOP",
				},
			},
			UsageMetadata: usageMetadata{
				PromptTokenCount:     20,
				CandidatesTokenCount: 5,
				TotalTokenCount:      25,
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

	messages := []zyn.Message{
		{Role: zyn.RoleUser, Content: "hello"},
		{Role: zyn.RoleAssistant, Content: "hi there"},
	}

	_, err := provider.Call(ctx, messages, 0.5)
	if err != nil {
		t.Fatalf("Call failed: %v", err)
	}
}

func TestSystemInstruction(t *testing.T) {
	ctx := context.Background()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req generateContentRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			t.Fatalf("Failed to decode request: %v", err)
		}

		// Verify system instruction is extracted
		if req.SystemInstruction == nil {
			t.Fatal("Expected systemInstruction to be set")
		}
		if len(req.SystemInstruction.Parts) != 1 {
			t.Fatalf("Expected 1 part in systemInstruction, got %d", len(req.SystemInstruction.Parts))
		}
		expectedSystem := "You are a helpful assistant.\n\nAlways respond in JSON."
		if req.SystemInstruction.Parts[0].Text != expectedSystem {
			t.Errorf("Expected system instruction %q, got %q", expectedSystem, req.SystemInstruction.Parts[0].Text)
		}

		// Verify only user/model messages in contents array
		if len(req.Contents) != 2 {
			t.Fatalf("Expected 2 contents (user + model), got %d", len(req.Contents))
		}
		if req.Contents[0].Role != "user" {
			t.Errorf("Expected first role 'user', got %q", req.Contents[0].Role)
		}
		if req.Contents[1].Role != "model" {
			t.Errorf("Expected second role 'model', got %q", req.Contents[1].Role)
		}

		resp := generateContentResponse{
			Candidates: []candidate{
				{
					Content: content{
						Role:  "model",
						Parts: []part{{Text: `{"result": "ok"}`}},
					},
					FinishReason: "STOP",
				},
			},
			UsageMetadata: usageMetadata{
				PromptTokenCount:     30,
				CandidatesTokenCount: 5,
				TotalTokenCount:      35,
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

func TestNoSystemInstruction(t *testing.T) {
	ctx := context.Background()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req generateContentRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			t.Fatalf("Failed to decode request: %v", err)
		}

		// Verify no systemInstruction when no system messages
		if req.SystemInstruction != nil {
			t.Errorf("Expected nil systemInstruction, got %+v", req.SystemInstruction)
		}

		resp := generateContentResponse{
			Candidates: []candidate{
				{
					Content: content{
						Role:  "model",
						Parts: []part{{Text: `{"result": "ok"}`}},
					},
					FinishReason: "STOP",
				},
			},
			UsageMetadata: usageMetadata{
				PromptTokenCount:     10,
				CandidatesTokenCount: 5,
				TotalTokenCount:      15,
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

	messages := []zyn.Message{
		{Role: zyn.RoleUser, Content: "Hello"},
	}

	_, err := provider.Call(ctx, messages, 0.5)
	if err != nil {
		t.Fatalf("Call failed: %v", err)
	}
}
