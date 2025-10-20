package zyn

import (
	"context"
	"fmt"
	"time"

	"github.com/zoobzio/pipz"
)

// Option modifies a pipeline for reliability features.
type Option func(pipz.Chainable[*SynapseRequest]) pipz.Chainable[*SynapseRequest]

// WithRetry adds retry logic to the pipeline.
// Failed requests are retried up to maxAttempts times.
func WithRetry(maxAttempts int) Option {
	return func(pipeline pipz.Chainable[*SynapseRequest]) pipz.Chainable[*SynapseRequest] {
		return pipz.NewRetry("retry", pipeline, maxAttempts)
	}
}

// WithBackoff adds retry logic with exponential backoff to the pipeline.
// Failed requests are retried with increasing delays between attempts.
// The delay starts at baseDelay and doubles after each failure.
func WithBackoff(maxAttempts int, baseDelay time.Duration) Option {
	return func(pipeline pipz.Chainable[*SynapseRequest]) pipz.Chainable[*SynapseRequest] {
		return pipz.NewBackoff("backoff", pipeline, maxAttempts, baseDelay)
	}
}

// WithTimeout adds timeout protection to the pipeline.
// Operations exceeding this duration will be canceled.
func WithTimeout(duration time.Duration) Option {
	return func(pipeline pipz.Chainable[*SynapseRequest]) pipz.Chainable[*SynapseRequest] {
		return pipz.NewTimeout("timeout", pipeline, duration)
	}
}

// WithCircuitBreaker adds circuit breaker protection to the pipeline.
// After 'failures' consecutive failures, the circuit opens for 'recovery' duration.
func WithCircuitBreaker(failures int, recovery time.Duration) Option {
	return func(pipeline pipz.Chainable[*SynapseRequest]) pipz.Chainable[*SynapseRequest] {
		return pipz.NewCircuitBreaker("circuit-breaker", pipeline, failures, recovery)
	}
}

// WithRateLimit adds rate limiting to the pipeline.
// rps = requests per second, burst = burst capacity.
func WithRateLimit(rps float64, burst int) Option {
	return func(pipeline pipz.Chainable[*SynapseRequest]) pipz.Chainable[*SynapseRequest] {
		rateLimiter := pipz.NewRateLimiter[*SynapseRequest]("rate-limit", rps, burst)
		return pipz.NewSequence("rate-limited", rateLimiter, pipeline)
	}
}

// WithErrorHandler adds error handling to the pipeline.
// The error handler receives error context and can process/log/alert as needed.
func WithErrorHandler(handler pipz.Chainable[*pipz.Error[*SynapseRequest]]) Option {
	return func(pipeline pipz.Chainable[*SynapseRequest]) pipz.Chainable[*SynapseRequest] {
		return pipz.NewHandle("error-handler", pipeline, handler)
	}
}

// ServiceProvider is implemented by types that can provide a pipeline for composition.
type ServiceProvider interface {
	GetPipeline() pipz.Chainable[*SynapseRequest]
}

// WithFallback adds a fallback service for resilience.
// If the primary fails, the fallback will be tried.
func WithFallback(fallback ServiceProvider) Option {
	return func(pipeline pipz.Chainable[*SynapseRequest]) pipz.Chainable[*SynapseRequest] {
		return pipz.NewFallback("with-fallback", pipeline, fallback.GetPipeline())
	}
}

// WithDebug adds debug logging that prints the prompt and raw response.
// Useful for troubleshooting and understanding what the LLM sees/returns.
func WithDebug() Option {
	return func(pipeline pipz.Chainable[*SynapseRequest]) pipz.Chainable[*SynapseRequest] {
		debugger := pipz.Apply("debug", func(ctx context.Context, req *SynapseRequest) (*SynapseRequest, error) {
			// Print prompt before calling LLM
			fmt.Println("\n=== DEBUG: Prompt ===")
			fmt.Println(req.Prompt.Render())
			fmt.Println("=====================")

			// Call the actual pipeline
			processed, err := pipeline.Process(ctx, req)
			if err != nil {
				fmt.Printf("\n=== DEBUG: Error ===\n%v\n==================\n\n", err)
				return processed, err
			}

			// Print raw response after LLM call
			fmt.Println("\n=== DEBUG: Raw Response ===")
			fmt.Println(processed.Response)
			fmt.Println("===========================")

			return processed, nil
		})
		return debugger
	}
}
