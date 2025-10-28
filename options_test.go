package zyn

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/zoobzio/pipz"
)

func TestWithRetry(t *testing.T) {
	t.Run("simple", func(t *testing.T) {
		pipeline := pipz.Apply("test", func(_ context.Context, req *SynapseRequest) (*SynapseRequest, error) {
			return req, nil
		})

		option := WithRetry(3)
		wrapped := option(pipeline)

		if wrapped == nil {
			t.Error("WithRetry returned nil pipeline")
		}
	})

	t.Run("reliability", func(t *testing.T) {
		attempts := 0
		pipeline := pipz.Apply("test", func(_ context.Context, req *SynapseRequest) (*SynapseRequest, error) {
			attempts++
			if attempts < 2 {
				return req, errors.New("temporary error")
			}
			req.Response = "success"
			return req, nil
		})

		option := WithRetry(3)
		wrapped := option(pipeline)

		ctx := context.Background()
		req := &SynapseRequest{Prompt: &Prompt{Task: "test", Input: "test", Schema: "{}"}}
		result, err := wrapped.Process(ctx, req)
		if err != nil {
			t.Errorf("Expected success after retry, got: %v", err)
		}
		if result.Response != "success" {
			t.Error("Expected response after retry")
		}
		if attempts < 2 {
			t.Error("Retry should have been attempted")
		}
	})

	t.Run("chaining", func(t *testing.T) {
		pipeline := pipz.Apply("test", func(_ context.Context, req *SynapseRequest) (*SynapseRequest, error) {
			return req, nil
		})

		option1 := WithRetry(3)
		option2 := WithTimeout(1 * time.Second)

		wrapped := option2(option1(pipeline))

		if wrapped == nil {
			t.Error("Chained options returned nil")
		}
	})
}

func TestWithBackoff(t *testing.T) {
	t.Run("simple", func(t *testing.T) {
		pipeline := pipz.Apply("test", func(_ context.Context, req *SynapseRequest) (*SynapseRequest, error) {
			return req, nil
		})

		option := WithBackoff(3, 10*time.Millisecond)
		wrapped := option(pipeline)

		if wrapped == nil {
			t.Error("WithBackoff returned nil pipeline")
		}
	})

	t.Run("reliability", func(t *testing.T) {
		attempts := 0
		pipeline := pipz.Apply("test", func(_ context.Context, req *SynapseRequest) (*SynapseRequest, error) {
			attempts++
			if attempts < 2 {
				return req, errors.New("temporary error")
			}
			return req, nil
		})

		option := WithBackoff(3, 10*time.Millisecond)
		wrapped := option(pipeline)

		ctx := context.Background()
		req := &SynapseRequest{}
		_, err := wrapped.Process(ctx, req)
		if err != nil {
			t.Errorf("Expected success after backoff, got: %v", err)
		}
		if attempts < 2 {
			t.Error("Backoff should have retried")
		}
	})

	t.Run("chaining", func(t *testing.T) {
		pipeline := pipz.Apply("test", func(_ context.Context, req *SynapseRequest) (*SynapseRequest, error) {
			return req, nil
		})

		option := WithBackoff(3, 10*time.Millisecond)
		wrapped := option(pipeline)

		ctx := context.Background()
		req := &SynapseRequest{}
		_, err := wrapped.Process(ctx, req)
		if err != nil {
			t.Errorf("Backoff pipeline failed: %v", err)
		}
	})
}

func TestWithTimeout(t *testing.T) {
	t.Run("simple", func(t *testing.T) {
		pipeline := pipz.Apply("test", func(_ context.Context, req *SynapseRequest) (*SynapseRequest, error) {
			return req, nil
		})

		option := WithTimeout(1 * time.Second)
		wrapped := option(pipeline)

		if wrapped == nil {
			t.Error("WithTimeout returned nil pipeline")
		}
	})

	t.Run("reliability", func(t *testing.T) {
		pipeline := pipz.Apply("slow", func(_ context.Context, req *SynapseRequest) (*SynapseRequest, error) {
			time.Sleep(50 * time.Millisecond)
			return req, nil
		})

		option := WithTimeout(10 * time.Millisecond)
		wrapped := option(pipeline)

		ctx := context.Background()
		req := &SynapseRequest{}
		_, err := wrapped.Process(ctx, req)
		if err == nil {
			t.Error("Expected timeout error")
		}
	})

	t.Run("chaining", func(t *testing.T) {
		pipeline := pipz.Apply("fast", func(_ context.Context, req *SynapseRequest) (*SynapseRequest, error) {
			return req, nil
		})

		option := WithTimeout(100 * time.Millisecond)
		wrapped := option(pipeline)

		ctx := context.Background()
		req := &SynapseRequest{}
		_, err := wrapped.Process(ctx, req)
		if err != nil {
			t.Errorf("Fast pipeline should complete within timeout: %v", err)
		}
	})
}

func TestWithCircuitBreaker(t *testing.T) {
	t.Run("simple", func(t *testing.T) {
		pipeline := pipz.Apply("test", func(_ context.Context, req *SynapseRequest) (*SynapseRequest, error) {
			return req, nil
		})

		option := WithCircuitBreaker(5, 1*time.Second)
		wrapped := option(pipeline)

		if wrapped == nil {
			t.Error("WithCircuitBreaker returned nil pipeline")
		}
	})

	t.Run("reliability", func(t *testing.T) {
		pipeline := pipz.Apply("test", func(_ context.Context, req *SynapseRequest) (*SynapseRequest, error) {
			return req, errors.New("persistent failure")
		})

		option := WithCircuitBreaker(2, 100*time.Millisecond)
		wrapped := option(pipeline)

		ctx := context.Background()
		req := &SynapseRequest{}

		// First two failures should be attempted
		for i := 0; i < 2; i++ {
			_, err := wrapped.Process(ctx, req)
			if err == nil {
				t.Error("Expected error from failing pipeline")
			}
		}

		// Third attempt should be rejected by circuit breaker
		_, err := wrapped.Process(ctx, req)
		if err == nil {
			t.Error("Expected circuit breaker to reject request")
		}
	})

	t.Run("chaining", func(t *testing.T) {
		pipeline := pipz.Apply("test", func(_ context.Context, req *SynapseRequest) (*SynapseRequest, error) {
			return req, nil
		})

		option := WithCircuitBreaker(5, 1*time.Second)
		wrapped := option(pipeline)

		ctx := context.Background()
		req := &SynapseRequest{}
		_, err := wrapped.Process(ctx, req)
		if err != nil {
			t.Errorf("Successful pipeline should work with circuit breaker: %v", err)
		}
	})
}

func TestWithRateLimit(t *testing.T) {
	t.Run("simple", func(t *testing.T) {
		pipeline := pipz.Apply("test", func(_ context.Context, req *SynapseRequest) (*SynapseRequest, error) {
			return req, nil
		})

		option := WithRateLimit(10, 20)
		wrapped := option(pipeline)

		if wrapped == nil {
			t.Error("WithRateLimit returned nil pipeline")
		}
	})

	t.Run("reliability", func(t *testing.T) {
		pipeline := pipz.Apply("test", func(_ context.Context, req *SynapseRequest) (*SynapseRequest, error) {
			return req, nil
		})

		option := WithRateLimit(100, 10)
		wrapped := option(pipeline)

		ctx := context.Background()
		req := &SynapseRequest{}

		// Make several requests - should be rate limited
		for i := 0; i < 5; i++ {
			_, err := wrapped.Process(ctx, req)
			if err != nil {
				t.Errorf("Request %d failed: %v", i, err)
			}
		}
	})

	t.Run("chaining", func(t *testing.T) {
		pipeline := pipz.Apply("test", func(_ context.Context, req *SynapseRequest) (*SynapseRequest, error) {
			return req, nil
		})

		option := WithRateLimit(100, 10)
		wrapped := option(pipeline)

		ctx := context.Background()
		req := &SynapseRequest{}
		_, err := wrapped.Process(ctx, req)
		if err != nil {
			t.Errorf("Rate limited pipeline failed: %v", err)
		}
	})
}

func TestWithErrorHandler(t *testing.T) {
	t.Run("simple", func(t *testing.T) {
		pipeline := pipz.Apply("test", func(_ context.Context, req *SynapseRequest) (*SynapseRequest, error) {
			return req, nil
		})

		handler := pipz.Apply("handler", func(_ context.Context, e *pipz.Error[*SynapseRequest]) (*pipz.Error[*SynapseRequest], error) {
			return e, nil
		})

		option := WithErrorHandler(handler)
		wrapped := option(pipeline)

		if wrapped == nil {
			t.Error("WithErrorHandler returned nil pipeline")
		}
	})

	t.Run("reliability", func(t *testing.T) {
		pipeline := pipz.Apply("failing", func(_ context.Context, req *SynapseRequest) (*SynapseRequest, error) {
			return req, errors.New("test error")
		})

		handled := false
		handler := pipz.Apply("handler", func(_ context.Context, e *pipz.Error[*SynapseRequest]) (*pipz.Error[*SynapseRequest], error) {
			handled = true
			return e, nil
		})

		option := WithErrorHandler(handler)
		wrapped := option(pipeline)

		ctx := context.Background()
		req := &SynapseRequest{}
		_, _ = wrapped.Process(ctx, req)

		if !handled {
			t.Error("Error handler was not called")
		}
	})

	t.Run("chaining", func(t *testing.T) {
		pipeline := pipz.Apply("test", func(_ context.Context, req *SynapseRequest) (*SynapseRequest, error) {
			return req, nil
		})

		handler := pipz.Apply("handler", func(_ context.Context, e *pipz.Error[*SynapseRequest]) (*pipz.Error[*SynapseRequest], error) {
			return e, nil
		})

		option := WithErrorHandler(handler)
		wrapped := option(pipeline)

		ctx := context.Background()
		req := &SynapseRequest{}
		_, err := wrapped.Process(ctx, req)
		if err != nil {
			t.Errorf("Pipeline with error handler failed: %v", err)
		}
	})
}

func TestWithFallback(t *testing.T) {
	t.Run("simple", func(t *testing.T) {
		primaryPipeline := pipz.Apply("primary", func(_ context.Context, req *SynapseRequest) (*SynapseRequest, error) {
			return req, nil
		})

		fallbackSynapse := Binary("fallback question", NewMockProvider())

		option := WithFallback(fallbackSynapse)
		wrapped := option(primaryPipeline)

		if wrapped == nil {
			t.Error("WithFallback returned nil pipeline")
		}
	})

	t.Run("reliability", func(t *testing.T) {
		primaryPipeline := pipz.Apply("primary", func(_ context.Context, req *SynapseRequest) (*SynapseRequest, error) {
			return req, errors.New("primary failed")
		})

		provider := NewMockProviderWithResponse(`{"decision": false, "confidence": 0.5, "reasoning": ["fallback"]}`)
		fallbackSynapse := Binary("fallback", provider)

		option := WithFallback(fallbackSynapse)
		wrapped := option(primaryPipeline)

		ctx := context.Background()
		prompt := &Prompt{Task: "test", Input: "test", Schema: "{}"}
		req := &SynapseRequest{Prompt: prompt, Temperature: 0.5}
		result, err := wrapped.Process(ctx, req)
		if err != nil {
			t.Errorf("Fallback should have succeeded: %v", err)
		}
		if result.Response == "" {
			t.Error("Expected response from fallback")
		}
	})

	t.Run("chaining", func(t *testing.T) {
		primaryPipeline := pipz.Apply("primary", func(_ context.Context, req *SynapseRequest) (*SynapseRequest, error) {
			req.Response = "primary response"
			return req, nil
		})

		fallbackSynapse := Binary("fallback", NewMockProvider())

		option := WithFallback(fallbackSynapse)
		wrapped := option(primaryPipeline)

		ctx := context.Background()
		prompt := &Prompt{Task: "test", Input: "test", Schema: "{}"}
		req := &SynapseRequest{Prompt: prompt}
		result, err := wrapped.Process(ctx, req)
		if err != nil {
			t.Errorf("Primary should succeed without fallback: %v", err)
		}
		if result.Response != "primary response" {
			t.Error("Expected primary response when primary succeeds")
		}
	})
}

func TestWithDebug(t *testing.T) {
	t.Run("simple", func(t *testing.T) {
		pipeline := pipz.Apply("test", func(_ context.Context, req *SynapseRequest) (*SynapseRequest, error) {
			return req, nil
		})

		option := WithDebug()
		wrapped := option(pipeline)

		if wrapped == nil {
			t.Error("WithDebug returned nil pipeline")
		}
	})

	t.Run("reliability", func(t *testing.T) {
		pipeline := pipz.Apply("test", func(_ context.Context, req *SynapseRequest) (*SynapseRequest, error) {
			req.Response = "test response"
			return req, nil
		})

		option := WithDebug()
		wrapped := option(pipeline)

		ctx := context.Background()
		prompt := &Prompt{Task: "test task", Input: "test input", Schema: "{}"}
		req := &SynapseRequest{Prompt: prompt}
		result, err := wrapped.Process(ctx, req)
		if err != nil {
			t.Errorf("Debug wrapper should not cause errors: %v", err)
		}
		if result.Response != "test response" {
			t.Error("Debug should pass through response")
		}
	})

	t.Run("chaining", func(t *testing.T) {
		pipeline := pipz.Apply("test", func(_ context.Context, req *SynapseRequest) (*SynapseRequest, error) {
			return req, nil
		})

		option := WithDebug()
		wrapped := option(pipeline)

		ctx := context.Background()
		req := &SynapseRequest{Prompt: &Prompt{Task: "test", Input: "test", Schema: "{}"}}
		_, err := wrapped.Process(ctx, req)
		if err != nil {
			t.Errorf("Debug pipeline failed: %v", err)
		}
	})
}
