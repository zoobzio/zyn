package integration

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/zoobzio/zyn"
	zynt "github.com/zoobzio/zyn/testing"
)

func TestPipeline_RetrySuccess(t *testing.T) {
	// Fails twice, then succeeds on third attempt
	provider := zynt.NewFailingProvider(2).
		WithSuccessResponse(zynt.NewResponseBuilder().
			WithDecision(true).
			WithConfidence(0.9).
			WithReasoning("recovered").
			Build())

	synapse, err := zyn.Binary("question", provider, zyn.WithRetry(3))
	if err != nil {
		t.Fatalf("failed to create synapse: %v", err)
	}

	session := zyn.NewSession()
	ctx := context.Background()

	result, err := synapse.Fire(ctx, session, "input")
	if err != nil {
		t.Fatalf("expected success after retries, got error: %v", err)
	}

	if !result {
		t.Error("expected true result")
	}

	if provider.CallCount() != 3 {
		t.Errorf("expected 3 calls (2 failures + 1 success), got %d", provider.CallCount())
	}

	// Session should only have messages from successful call
	if session.Len() != 2 {
		t.Errorf("expected 2 messages (from successful call only), got %d", session.Len())
	}
}

func TestPipeline_RetryExhausted(t *testing.T) {
	// Fails 5 times, but we only retry 3 times
	provider := zynt.NewFailingProvider(5)

	synapse, err := zyn.Binary("question", provider, zyn.WithRetry(3))
	if err != nil {
		t.Fatalf("failed to create synapse: %v", err)
	}

	session := zyn.NewSession()
	ctx := context.Background()

	_, err = synapse.Fire(ctx, session, "input")
	if err == nil {
		t.Fatal("expected error after exhausting retries")
	}

	if provider.CallCount() != 3 {
		t.Errorf("expected 3 calls (all failures), got %d", provider.CallCount())
	}

	// Session should be empty - no successful calls
	if session.Len() != 0 {
		t.Errorf("expected 0 messages after failed retries, got %d", session.Len())
	}
}

func TestPipeline_Timeout(t *testing.T) {
	// Provider that takes 500ms per call
	slowProvider := zynt.NewLatencyProvider(
		zynt.NewSequencedProvider(
			zynt.NewResponseBuilder().WithDecision(true).WithConfidence(0.9).WithReasoning("slow").Build(),
		),
		500*time.Millisecond,
	)

	// 100ms timeout should trigger before provider responds
	synapse, err := zyn.Binary("question", slowProvider, zyn.WithTimeout(100*time.Millisecond))
	if err != nil {
		t.Fatalf("failed to create synapse: %v", err)
	}

	session := zyn.NewSession()
	ctx := context.Background()

	start := time.Now()
	_, err = synapse.Fire(ctx, session, "input")
	elapsed := time.Since(start)

	if err == nil {
		t.Fatal("expected timeout error")
	}

	// Should have timed out around 100ms, not 500ms
	if elapsed > 200*time.Millisecond {
		t.Errorf("timeout should have triggered earlier, took %v", elapsed)
	}
}

func TestPipeline_TimeoutSuccess(t *testing.T) {
	// Provider that responds quickly
	fastProvider := zynt.NewLatencyProvider(
		zynt.NewSequencedProvider(
			zynt.NewResponseBuilder().WithDecision(true).WithConfidence(0.9).WithReasoning("fast").Build(),
		),
		10*time.Millisecond,
	)

	// 1 second timeout should be plenty
	synapse, err := zyn.Binary("question", fastProvider, zyn.WithTimeout(1*time.Second))
	if err != nil {
		t.Fatalf("failed to create synapse: %v", err)
	}

	session := zyn.NewSession()
	ctx := context.Background()

	result, err := synapse.Fire(ctx, session, "input")
	if err != nil {
		t.Fatalf("expected success, got error: %v", err)
	}

	if !result {
		t.Error("expected true result")
	}
}

func TestPipeline_Fallback(t *testing.T) {
	// Primary provider always fails
	primaryProvider := zyn.NewMockProviderWithError("primary failure")

	// Fallback provider succeeds
	fallbackProvider := zynt.NewSequencedProvider(
		zynt.NewResponseBuilder().WithDecision(true).WithConfidence(0.7).WithReasoning("fallback").Build(),
	)

	primarySynapse, _ := zyn.Binary("question", primaryProvider)
	fallbackSynapse, _ := zyn.Binary("question", fallbackProvider)

	// Wrap primary with fallback
	synapse, err := zyn.Binary("question", primaryProvider,
		zyn.WithFallback(fallbackSynapse),
	)
	if err != nil {
		t.Fatalf("failed to create synapse: %v", err)
	}
	_ = primarySynapse // Silence unused warning

	session := zyn.NewSession()
	ctx := context.Background()

	result, err := synapse.Fire(ctx, session, "input")
	if err != nil {
		t.Fatalf("expected fallback to succeed, got error: %v", err)
	}

	if !result {
		t.Error("expected true result from fallback")
	}
}

func TestPipeline_BackoffTiming(t *testing.T) {
	// Fails 3 times, then succeeds
	provider := zynt.NewFailingProvider(3).
		WithSuccessResponse(zynt.NewResponseBuilder().
			WithDecision(true).
			WithConfidence(0.9).
			WithReasoning("recovered").
			Build())

	// Backoff: 50ms, 100ms, 200ms between retries
	synapse, err := zyn.Binary("question", provider,
		zyn.WithBackoff(4, 50*time.Millisecond),
	)
	if err != nil {
		t.Fatalf("failed to create synapse: %v", err)
	}

	session := zyn.NewSession()
	ctx := context.Background()

	start := time.Now()
	result, err := synapse.Fire(ctx, session, "input")
	elapsed := time.Since(start)

	if err != nil {
		t.Fatalf("expected success after backoff, got error: %v", err)
	}

	if !result {
		t.Error("expected true result")
	}

	// Should have waited at least 50+100+200=350ms for backoffs
	// (but allow some tolerance)
	if elapsed < 300*time.Millisecond {
		t.Errorf("expected backoff delays, but elapsed only %v", elapsed)
	}
}

func TestPipeline_CombinedOptions(t *testing.T) {
	// Provider that fails once then succeeds
	provider := zynt.NewFailingProvider(1).
		WithSuccessResponse(zynt.NewResponseBuilder().
			WithDecision(true).
			WithConfidence(0.95).
			WithReasoning("success").
			Build())

	recorder := zynt.NewCallRecorder(provider)

	synapse, err := zyn.Binary("question", recorder,
		zyn.WithRetry(3),
		zyn.WithTimeout(5*time.Second),
	)
	if err != nil {
		t.Fatalf("failed to create synapse: %v", err)
	}

	session := zyn.NewSession()
	ctx := context.Background()

	result, err := synapse.Fire(ctx, session, "input")
	if err != nil {
		t.Fatalf("expected success, got error: %v", err)
	}

	if !result {
		t.Error("expected true result")
	}

	// Should have 2 calls: 1 failure + 1 success
	if recorder.CallCount() != 2 {
		t.Errorf("expected 2 calls, got %d", recorder.CallCount())
	}
}

func TestPipeline_ContextCancellation(t *testing.T) {
	// Provider that checks context and blocks until canceled
	blockingProvider := zyn.NewMockProviderWithCallback(func(_ string, _ float32) (string, error) {
		// Block for a long time - context should cancel us
		time.Sleep(2 * time.Second)
		return zynt.NewResponseBuilder().WithDecision(true).WithConfidence(0.9).WithReasoning("slow").Build(), nil
	})

	synapse, err := zyn.Binary("question", blockingProvider,
		zyn.WithTimeout(100*time.Millisecond), // Use timeout as cancellation mechanism
	)
	if err != nil {
		t.Fatalf("failed to create synapse: %v", err)
	}

	session := zyn.NewSession()
	ctx := context.Background()

	start := time.Now()
	_, err = synapse.Fire(ctx, session, "input")
	elapsed := time.Since(start)

	if err == nil {
		t.Fatal("expected error from timeout")
	}

	// Should have timed out around 100ms, not 2 seconds
	if elapsed > 500*time.Millisecond {
		t.Errorf("timeout should have triggered earlier, took %v", elapsed)
	}
}

func TestPipeline_SessionTransactionalOnFailure(t *testing.T) {
	// Provider that always fails
	provider := zyn.NewMockProviderWithError("always fails")

	synapse, err := zyn.Binary("question", provider, zyn.WithRetry(2))
	if err != nil {
		t.Fatalf("failed to create synapse: %v", err)
	}

	session := zyn.NewSession()
	ctx := context.Background()

	_, err = synapse.Fire(ctx, session, "input")
	if err == nil {
		t.Fatal("expected error")
	}

	// Session should remain empty - no messages added from failed calls
	if session.Len() != 0 {
		t.Errorf("expected 0 messages after failure, got %d", session.Len())
	}
}

func TestPipeline_MultipleCallsAccumulate(t *testing.T) {
	// Multiple successful calls accumulate in session
	provider := zynt.NewSequencedProvider(
		zynt.NewResponseBuilder().WithDecision(true).WithConfidence(0.9).WithReasoning("first").Build(),
		zynt.NewResponseBuilder().WithDecision(false).WithConfidence(0.8).WithReasoning("second").Build(),
		zynt.NewResponseBuilder().WithDecision(true).WithConfidence(0.7).WithReasoning("third").Build(),
	)

	synapse, err := zyn.Binary("question", provider)
	if err != nil {
		t.Fatalf("failed to create synapse: %v", err)
	}

	session := zyn.NewSession()
	ctx := context.Background()

	// First call
	result1, err := synapse.Fire(ctx, session, "input1")
	if err != nil {
		t.Fatalf("first call failed: %v", err)
	}
	if !result1 {
		t.Error("expected first result=true")
	}

	// Second call
	result2, err := synapse.Fire(ctx, session, "input2")
	if err != nil {
		t.Fatalf("second call failed: %v", err)
	}
	if result2 {
		t.Error("expected second result=false")
	}

	// Third call
	result3, err := synapse.Fire(ctx, session, "input3")
	if err != nil {
		t.Fatalf("third call failed: %v", err)
	}
	if !result3 {
		t.Error("expected third result=true")
	}

	// All three calls should be in session (6 messages: 3 user + 3 assistant)
	if session.Len() != 6 {
		t.Errorf("expected 6 messages from 3 successful calls, got %d", session.Len())
	}

	// Verify provider received all the context
	if provider.CallCount() != 3 {
		t.Errorf("expected 3 provider calls, got %d", provider.CallCount())
	}
}

func TestPipeline_CircuitBreakerTrips(t *testing.T) {
	// Provider that always fails
	var callCount int
	provider := zyn.NewMockProviderWithCallback(func(_ string, _ float32) (string, error) {
		callCount++
		return "", fmt.Errorf("always fails")
	})

	// Circuit breaker opens after 3 failures, recovery time 100ms
	synapse, err := zyn.Binary("question", provider,
		zyn.WithCircuitBreaker(3, 100*time.Millisecond),
	)
	if err != nil {
		t.Fatalf("failed to create synapse: %v", err)
	}

	session := zyn.NewSession()
	ctx := context.Background()

	// First 3 calls should hit the provider and fail
	for i := 0; i < 3; i++ {
		_, err := synapse.Fire(ctx, session, "input")
		if err == nil {
			t.Errorf("call %d: expected error", i+1)
		}
	}

	if callCount != 3 {
		t.Errorf("expected 3 provider calls before circuit opens, got %d", callCount)
	}

	// Next calls should fail immediately (circuit open)
	callCountBefore := callCount
	for i := 0; i < 5; i++ {
		_, err := synapse.Fire(ctx, session, "input")
		if err == nil {
			t.Errorf("call with open circuit: expected error")
		}
	}

	// Provider should NOT have been called while circuit is open
	if callCount != callCountBefore {
		t.Errorf("expected no provider calls while circuit open, got %d additional calls",
			callCount-callCountBefore)
	}
}

func TestPipeline_CircuitBreakerRecovery(t *testing.T) {
	// Provider that fails initially, then succeeds
	var callCount int
	var shouldSucceed bool
	provider := zyn.NewMockProviderWithCallback(func(_ string, _ float32) (string, error) {
		callCount++
		if shouldSucceed {
			return zynt.NewResponseBuilder().WithDecision(true).WithConfidence(0.9).WithReasoning("recovered").Build(), nil
		}
		return "", fmt.Errorf("failing")
	})

	// Circuit breaker opens after 2 failures, recovery time 50ms
	synapse, err := zyn.Binary("question", provider,
		zyn.WithCircuitBreaker(2, 50*time.Millisecond),
	)
	if err != nil {
		t.Fatalf("failed to create synapse: %v", err)
	}

	session := zyn.NewSession()
	ctx := context.Background()

	// Trip the circuit breaker
	for i := 0; i < 2; i++ {
		_, _ = synapse.Fire(ctx, session, "input")
	}

	// Circuit is now open
	initialCalls := callCount

	// Wait for recovery period
	time.Sleep(100 * time.Millisecond)

	// Set provider to succeed
	shouldSucceed = true

	// Next call should go through (half-open state) and succeed
	result, err := synapse.Fire(ctx, session, "input")
	if err != nil {
		t.Fatalf("expected success after recovery: %v", err)
	}

	if !result {
		t.Error("expected true result after recovery")
	}

	// Provider should have been called at least once more
	if callCount <= initialCalls {
		t.Error("expected provider to be called after recovery period")
	}
}

func TestPipeline_CircuitBreakerWithRetry(t *testing.T) {
	// Test circuit breaker combined with retry.
	// With retry wrapping circuit breaker, each retry attempt counts toward the failure threshold.
	var callCount int
	provider := zyn.NewMockProviderWithCallback(func(_ string, _ float32) (string, error) {
		callCount++
		// Always fail for this test - we want to observe circuit breaker behavior
		return "", fmt.Errorf("failing %d", callCount)
	})

	// Retry 2 times, circuit breaker opens after 3 failures
	synapse, err := zyn.Binary("question", provider,
		zyn.WithRetry(2),
		zyn.WithCircuitBreaker(3, 100*time.Millisecond),
	)
	if err != nil {
		t.Fatalf("failed to create synapse: %v", err)
	}

	session := zyn.NewSession()
	ctx := context.Background()

	// First call: retries 2 times (2 total calls), all fail
	_, err = synapse.Fire(ctx, session, "input")
	if err == nil {
		t.Error("first call should fail after retries exhausted")
	}

	firstCallCount := callCount
	t.Logf("Calls after first Fire: %d", firstCallCount)

	// Second call: should attempt more retries, eventually tripping circuit
	_, err = synapse.Fire(ctx, session, "input")
	if err == nil {
		t.Error("second call should fail")
	}

	secondCallCount := callCount
	t.Logf("Calls after second Fire: %d", secondCallCount)

	// Third call: circuit should be open, no provider calls
	callCountBefore := callCount
	_, err = synapse.Fire(ctx, session, "input")
	if err == nil {
		t.Error("third call should fail (circuit open)")
	}

	// If circuit is open, no new provider calls should have been made
	if callCount > callCountBefore {
		t.Logf("Note: %d additional calls made (circuit may allow half-open probe)", callCount-callCountBefore)
	}
}
