package zyn

import (
	"context"
	"strings"
	"testing"
)

func TestNewMockProvider(t *testing.T) {
	t.Run("simple", func(t *testing.T) {
		provider := NewMockProvider()

		if provider == nil {
			t.Fatal("NewMockProvider returned nil")
		}
	})

	t.Run("reliability", func(t *testing.T) {
		provider := NewMockProvider()

		ctx := context.Background()
		response, err := provider.Call(ctx, "test prompt", 0.5)
		if err != nil {
			t.Errorf("Call failed: %v", err)
		}
		if response == "" {
			t.Error("Expected non-empty response")
		}
	})

	t.Run("chaining", func(t *testing.T) {
		provider := NewMockProvider()

		name := provider.Name()
		if name == "" {
			t.Error("Provider name should not be empty")
		}
	})
}

func TestNewMockProviderWithName(t *testing.T) {
	t.Run("simple", func(t *testing.T) {
		provider := NewMockProviderWithName("test-provider")

		if provider == nil {
			t.Fatal("NewMockProviderWithName returned nil")
		}
		if provider.Name() != "test-provider" {
			t.Errorf("Expected name 'test-provider', got '%s'", provider.Name())
		}
	})

	t.Run("reliability", func(t *testing.T) {
		provider := NewMockProviderWithName("reliable-provider")

		ctx := context.Background()
		response, err := provider.Call(ctx, "test", 0.5)
		if err != nil {
			t.Errorf("Call failed: %v", err)
		}
		if response == "" {
			t.Error("Expected response from named provider")
		}
	})

	t.Run("chaining", func(t *testing.T) {
		provider := NewMockProviderWithName("provider1")
		provider2 := NewMockProviderWithName("provider2")

		if provider.Name() == provider2.Name() {
			t.Error("Different providers should have different names")
		}
	})
}

func TestMockProvider_Call(t *testing.T) {
	t.Run("simple", func(t *testing.T) {
		provider := NewMockProvider()

		ctx := context.Background()
		response, err := provider.Call(ctx, "test prompt", 0.5)
		if err != nil {
			t.Fatalf("Call failed: %v", err)
		}
		if response == "" {
			t.Error("Expected non-empty response")
		}
	})

	t.Run("reliability", func(t *testing.T) {
		provider := NewMockProviderWithName("test")

		ctx := context.Background()
		response1, err := provider.Call(ctx, "prompt1", 0.5)
		if err != nil {
			t.Errorf("First call failed: %v", err)
		}

		response2, err := provider.Call(ctx, "prompt2", 0.5)
		if err != nil {
			t.Errorf("Second call failed: %v", err)
		}

		if response1 == "" || response2 == "" {
			t.Error("Expected responses from both calls")
		}
	})

	t.Run("chaining", func(t *testing.T) {
		provider := NewMockProvider()

		ctx := context.Background()
		response, err := provider.Call(ctx, "test", 0.5)
		if err != nil {
			t.Fatalf("Call failed: %v", err)
		}

		// Response should be parseable as various types
		if response == "" {
			t.Error("Expected valid response for chaining")
		}
	})
}

func TestMockProvider_Name(t *testing.T) {
	t.Run("simple", func(t *testing.T) {
		provider := NewMockProvider()

		name := provider.Name()
		if name == "" {
			t.Error("Name returned empty string")
		}
	})

	t.Run("reliability", func(t *testing.T) {
		provider := NewMockProviderWithName("custom-name")

		name := provider.Name()
		if name != "custom-name" {
			t.Errorf("Expected 'custom-name', got '%s'", name)
		}
	})

	t.Run("chaining", func(t *testing.T) {
		provider := NewMockProviderWithName("test")

		name1 := provider.Name()
		name2 := provider.Name()
		if name1 != name2 {
			t.Error("Name should be consistent")
		}
	})
}

func TestMockProvider_SetAvailable(t *testing.T) {
	t.Run("simple", func(t *testing.T) {
		provider := NewMockProviderWithName("test")

		provider.SetAvailable(false)

		ctx := context.Background()
		_, err := provider.Call(ctx, "test", 0.5)
		if err == nil {
			t.Error("Expected error when unavailable")
		}
	})

	t.Run("reliability", func(t *testing.T) {
		provider := NewMockProviderWithName("test")

		ctx := context.Background()

		// Initially available
		_, err := provider.Call(ctx, "test", 0.5)
		if err != nil {
			t.Errorf("Provider should be available initially: %v", err)
		}

		// Set unavailable
		provider.SetAvailable(false)
		_, err = provider.Call(ctx, "test", 0.5)
		if err == nil {
			t.Error("Expected error when unavailable")
		}
		if !strings.Contains(err.Error(), "unavailable") {
			t.Errorf("Expected 'unavailable' in error, got: %v", err)
		}

		// Set available again
		provider.SetAvailable(true)
		_, err = provider.Call(ctx, "test", 0.5)
		if err != nil {
			t.Errorf("Provider should be available again: %v", err)
		}
	})

	t.Run("chaining", func(t *testing.T) {
		provider := NewMockProviderWithName("test")
		ctx := context.Background()

		provider.SetAvailable(false)
		_, err := provider.Call(ctx, "test", 0.5)
		if err == nil {
			t.Error("Expected unavailable error")
		}

		provider.SetAvailable(true)
		_, err = provider.Call(ctx, "test", 0.5)
		if err != nil {
			t.Error("Should be available after re-enabling")
		}
	})
}

func TestNewMockProviderWithResponse(t *testing.T) {
	t.Run("simple", func(t *testing.T) {
		provider := NewMockProviderWithResponse(`{"test": "value"}`)

		if provider == nil {
			t.Fatal("NewMockProviderWithResponse returned nil")
		}
	})

	t.Run("reliability", func(t *testing.T) {
		expectedResponse := `{"decision": true, "confidence": 0.9, "reasoning": ["test"]}`
		provider := NewMockProviderWithResponse(expectedResponse)

		ctx := context.Background()
		response, err := provider.Call(ctx, "any prompt", 0.5)
		if err != nil {
			t.Errorf("Call failed: %v", err)
		}
		if response != expectedResponse {
			t.Errorf("Expected fixed response '%s', got '%s'", expectedResponse, response)
		}
	})

	t.Run("chaining", func(t *testing.T) {
		provider := NewMockProviderWithResponse(`{"test": "fixed"}`)

		ctx := context.Background()
		response1, _ := provider.Call(ctx, "prompt1", 0.5)
		response2, _ := provider.Call(ctx, "prompt2", 0.5)

		if response1 != response2 {
			t.Error("Fixed response provider should return same response")
		}
	})
}

func TestNewMockProviderWithCallback(t *testing.T) {
	t.Run("simple", func(t *testing.T) {
		provider := NewMockProviderWithCallback(func(_ string, _ float32) (string, error) {
			return "callback response", nil
		})

		if provider == nil {
			t.Fatal("NewMockProviderWithCallback returned nil")
		}
	})

	t.Run("reliability", func(t *testing.T) {
		callCount := 0
		provider := NewMockProviderWithCallback(func(prompt string, _ float32) (string, error) {
			callCount++
			return "response " + prompt, nil
		})

		ctx := context.Background()
		response, err := provider.Call(ctx, "test", 0.5)
		if err != nil {
			t.Errorf("Call failed: %v", err)
		}
		if response != "response test" {
			t.Errorf("Expected 'response test', got '%s'", response)
		}
		if callCount != 1 {
			t.Errorf("Expected callback to be called once, got %d", callCount)
		}
	})

	t.Run("chaining", func(t *testing.T) {
		provider := NewMockProviderWithCallback(func(prompt string, _ float32) (string, error) {
			if strings.Contains(prompt, "error") {
				return "", nil
			}
			return `{"result": "` + prompt + `"}`, nil
		})

		ctx := context.Background()
		response1, _ := provider.Call(ctx, "prompt1", 0.5)
		response2, _ := provider.Call(ctx, "prompt2", 0.5)

		if response1 == response2 {
			t.Error("Callback should produce different responses for different prompts")
		}
	})
}

func TestNewMockProviderWithError(t *testing.T) {
	t.Run("simple", func(t *testing.T) {
		provider := NewMockProviderWithError("test error")

		if provider == nil {
			t.Fatal("NewMockProviderWithError returned nil")
		}
	})

	t.Run("reliability", func(t *testing.T) {
		expectedError := "simulated failure"
		provider := NewMockProviderWithError(expectedError)

		ctx := context.Background()
		_, err := provider.Call(ctx, "test", 0.5)
		if err == nil {
			t.Error("Expected error but got none")
		}
		if !strings.Contains(err.Error(), expectedError) {
			t.Errorf("Expected error containing '%s', got '%v'", expectedError, err)
		}
	})

	t.Run("chaining", func(t *testing.T) {
		provider := NewMockProviderWithError("persistent error")

		ctx := context.Background()
		_, err1 := provider.Call(ctx, "test1", 0.5)
		_, err2 := provider.Call(ctx, "test2", 0.5)

		if err1 == nil || err2 == nil {
			t.Error("Error provider should always return error")
		}
	})
}
