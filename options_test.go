package zyn

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/zoobzio/pipz"
)

// TestWithTimeout tests the timeout option.
func TestWithTimeout(t *testing.T) {
	// Create a slow pipeline
	slowPipeline := pipz.Apply("slow", func(_ context.Context, req *SynapseRequest) (*SynapseRequest, error) {
		time.Sleep(100 * time.Millisecond)
		req.Response = "slow response"
		return req, nil
	})

	// Apply timeout that's shorter than the operation
	withTimeout := WithTimeout(10 * time.Millisecond)
	pipeline := withTimeout(slowPipeline)

	ctx := context.Background()
	prompt := &Prompt{Task: "test", Input: "test", Schema: `{}`}
	req := &SynapseRequest{Prompt: prompt}
	_, err := pipeline.Process(ctx, req)

	if err == nil {
		t.Error("Expected timeout error")
	}
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Errorf("Expected DeadlineExceeded error, got %v", err)
	}
}

// TestWithRetry tests the retry option.
func TestWithRetry(t *testing.T) {
	// Track number of attempts
	attempts := 0

	// Create a pipeline that fails first 2 times
	failingPipeline := pipz.Apply("failing", func(_ context.Context, req *SynapseRequest) (*SynapseRequest, error) {
		attempts++
		if attempts < 3 {
			return req, errors.New("temporary error")
		}
		req.Response = "success after retries"
		return req, nil
	})

	// Apply retry with 3 max attempts
	withRetry := WithRetry(3)
	pipeline := withRetry(failingPipeline)

	ctx := context.Background()
	prompt := &Prompt{Task: "test", Input: "test", Schema: `{}`}
	req := &SynapseRequest{Prompt: prompt}
	result, err := pipeline.Process(ctx, req)

	if err != nil {
		t.Errorf("Expected success after retries, got error: %v", err)
	}
	if result.Response != "success after retries" {
		t.Errorf("Expected 'success after retries', got %s", result.Response)
	}
	if attempts != 3 {
		t.Errorf("Expected 3 attempts, got %d", attempts)
	}
}

// TestWithBackoff tests the backoff option.
func TestWithBackoff(t *testing.T) {
	// Track attempts and timing
	attempts := 0
	var timestamps []time.Time

	// Create a pipeline that fails first 2 times
	failingPipeline := pipz.Apply("failing", func(_ context.Context, req *SynapseRequest) (*SynapseRequest, error) {
		attempts++
		timestamps = append(timestamps, time.Now())
		if attempts < 3 {
			return req, errors.New("temporary error")
		}
		req.Response = "success after backoff"
		return req, nil
	})

	// Apply backoff with 10ms base delay
	withBackoff := WithBackoff(3, 10*time.Millisecond)
	pipeline := withBackoff(failingPipeline)

	// Execute
	ctx := context.Background()
	req := &SynapseRequest{Temperature: 0.7}
	result, err := pipeline.Process(ctx, req)

	// Verify success after retries
	if err != nil {
		t.Fatalf("Process failed: %v", err)
	}
	if result.Response != "success after backoff" {
		t.Errorf("Expected 'success after backoff', got %s", result.Response)
	}
	if attempts != 3 {
		t.Errorf("Expected 3 attempts, got %d", attempts)
	}

	// Verify delays are increasing (exponential backoff)
	if len(timestamps) >= 3 {
		delay1 := timestamps[1].Sub(timestamps[0])
		delay2 := timestamps[2].Sub(timestamps[1])

		// Second delay should be roughly double the first (allowing for some variance)
		ratio := float64(delay2) / float64(delay1)
		if ratio < 1.5 || ratio > 2.5 {
			t.Errorf("Expected exponential backoff, got delays %v and %v (ratio: %f)",
				delay1, delay2, ratio)
		}
	}
}

// TestOptionFallback tests the fallback option.
func TestOptionFallback(t *testing.T) {
	// Create a failing primary provider
	failingProvider := NewMockProviderWithCallback(func(_ string, _ float32) (string, error) {
		return "", errors.New("primary failed")
	})

	// Create a successful fallback provider
	fallbackProvider := NewMockProviderWithCallback(func(_ string, _ float32) (string, error) {
		return `{"decision": true, "confidence": 0.5, "reasoning": ["fallback"]}`, nil
	})

	// Create fallback synapse (ServiceProvider)
	fallbackSynapse := Binary("test", fallbackProvider)

	// Create primary pipeline that will fail
	primaryPipeline := pipz.Apply("primary", func(ctx context.Context, req *SynapseRequest) (*SynapseRequest, error) {
		promptStr := req.Prompt.Render()
		response, err := failingProvider.Call(ctx, promptStr, req.Temperature)
		if err != nil {
			return req, err
		}
		req.Response = response
		return req, nil
	})

	// Apply fallback using the synapse as ServiceProvider
	withFallback := WithFallback(fallbackSynapse)
	pipeline := withFallback(primaryPipeline)

	ctx := context.Background()
	prompt := &Prompt{Task: "test", Input: "test", Schema: `{}`}
	req := &SynapseRequest{Prompt: prompt, Temperature: 0.5}
	result, err := pipeline.Process(ctx, req)

	if err != nil {
		t.Errorf("Expected fallback to succeed, got error: %v", err)
	}
	// The response will be from the fallback synapse
	if !strings.Contains(result.Response, "fallback") {
		t.Errorf("Expected fallback response, got %s", result.Response)
	}
}

// TestOptionComposition tests multiple options together.
func TestOptionComposition(t *testing.T) {
	attempts := 0

	// Create a pipeline that fails once then succeeds
	pipeline := pipz.Apply("test", func(_ context.Context, req *SynapseRequest) (*SynapseRequest, error) {
		attempts++
		if attempts == 1 {
			return req, errors.New("first attempt fails")
		}
		req.Response = "success"
		return req, nil
	})

	// Apply multiple options
	withOptions := pipz.Chainable[*SynapseRequest](pipeline)
	withOptions = WithTimeout(1 * time.Second)(withOptions)
	withOptions = WithRetry(2)(withOptions)

	ctx := context.Background()
	prompt := &Prompt{Task: "test", Input: "test", Schema: `{}`}
	req := &SynapseRequest{Prompt: prompt}
	result, err := withOptions.Process(ctx, req)

	if err != nil {
		t.Errorf("Expected success, got error: %v", err)
	}
	if result.Response != "success" {
		t.Errorf("Expected 'success', got %s", result.Response)
	}
	if attempts != 2 {
		t.Errorf("Expected 2 attempts, got %d", attempts)
	}
}
