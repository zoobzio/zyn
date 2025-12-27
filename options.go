package zyn

import (
	"time"

	"github.com/zoobzio/pipz"
)

// Identities for reliability options.
var (
	retryID          = pipz.NewIdentity("zyn:retry", "Retries failed LLM calls")
	backoffID        = pipz.NewIdentity("zyn:backoff", "Retries with exponential backoff")
	timeoutID        = pipz.NewIdentity("zyn:timeout", "Enforces operation timeout")
	circuitBreakerID = pipz.NewIdentity("zyn:circuit-breaker", "Circuit breaker protection")
	rateLimitID      = pipz.NewIdentity("zyn:rate-limit", "Rate limiting")
	errorHandlerID   = pipz.NewIdentity("zyn:error-handler", "Error handling")
	fallbackID       = pipz.NewIdentity("zyn:fallback", "Fallback alternatives")
)

// Option modifies a pipeline for reliability features.
type Option func(pipz.Chainable[*SynapseRequest]) pipz.Chainable[*SynapseRequest]

// WithRetry adds retry logic to the pipeline.
// Failed requests are retried up to maxAttempts times.
func WithRetry(maxAttempts int) Option {
	return func(pipeline pipz.Chainable[*SynapseRequest]) pipz.Chainable[*SynapseRequest] {
		return pipz.NewRetry(retryID, pipeline, maxAttempts)
	}
}

// WithBackoff adds retry logic with exponential backoff to the pipeline.
// Failed requests are retried with increasing delays between attempts.
// The delay starts at baseDelay and doubles after each failure.
func WithBackoff(maxAttempts int, baseDelay time.Duration) Option {
	return func(pipeline pipz.Chainable[*SynapseRequest]) pipz.Chainable[*SynapseRequest] {
		return pipz.NewBackoff(backoffID, pipeline, maxAttempts, baseDelay)
	}
}

// WithTimeout adds timeout protection to the pipeline.
// Operations exceeding this duration will be canceled.
func WithTimeout(duration time.Duration) Option {
	return func(pipeline pipz.Chainable[*SynapseRequest]) pipz.Chainable[*SynapseRequest] {
		return pipz.NewTimeout(timeoutID, pipeline, duration)
	}
}

// WithCircuitBreaker adds circuit breaker protection to the pipeline.
// After 'failures' consecutive failures, the circuit opens for 'recovery' duration.
func WithCircuitBreaker(failures int, recovery time.Duration) Option {
	return func(pipeline pipz.Chainable[*SynapseRequest]) pipz.Chainable[*SynapseRequest] {
		return pipz.NewCircuitBreaker(circuitBreakerID, pipeline, failures, recovery)
	}
}

// WithRateLimit adds rate limiting to the pipeline.
// rps = requests per second, burst = burst capacity.
func WithRateLimit(rps float64, burst int) Option {
	return func(pipeline pipz.Chainable[*SynapseRequest]) pipz.Chainable[*SynapseRequest] {
		return pipz.NewRateLimiter(rateLimitID, rps, burst, pipeline)
	}
}

// WithErrorHandler adds error handling to the pipeline.
// The error handler receives error context and can process/log/alert as needed.
func WithErrorHandler(handler pipz.Chainable[*pipz.Error[*SynapseRequest]]) Option {
	return func(pipeline pipz.Chainable[*SynapseRequest]) pipz.Chainable[*SynapseRequest] {
		return pipz.NewHandle(errorHandlerID, pipeline, handler)
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
		return pipz.NewFallback(fallbackID, pipeline, fallback.GetPipeline())
	}
}
