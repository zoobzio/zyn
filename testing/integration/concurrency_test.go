package integration

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/zoobzio/zyn"
	zynt "github.com/zoobzio/zyn/testing"
)

func TestConcurrency_MultipleGoroutinesFiring(t *testing.T) {
	// Single synapse, multiple goroutines firing concurrently
	provider := zyn.NewMockProvider()
	synapse, err := zyn.Binary("Is this valid?", provider)
	if err != nil {
		t.Fatalf("failed to create synapse: %v", err)
	}

	ctx := context.Background()
	var wg sync.WaitGroup
	var successCount atomic.Int64
	var errorCount atomic.Int64

	goroutines := 50
	callsPerGoroutine := 10

	for i := 0; i < goroutines; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			// Each goroutine gets its own session
			session := zyn.NewSession()
			for j := 0; j < callsPerGoroutine; j++ {
				_, err := synapse.Fire(ctx, session, "test@example.com")
				if err != nil {
					errorCount.Add(1)
				} else {
					successCount.Add(1)
				}
			}
		}()
	}

	wg.Wait()

	expectedCalls := int64(goroutines * callsPerGoroutine)
	if successCount.Load() != expectedCalls {
		t.Errorf("expected %d successful calls, got %d (errors: %d)",
			expectedCalls, successCount.Load(), errorCount.Load())
	}
}

func TestConcurrency_SharedSessionAccess(t *testing.T) {
	// Multiple goroutines sharing a single session
	provider := zyn.NewMockProvider()
	synapse, err := zyn.Binary("Is this valid?", provider)
	if err != nil {
		t.Fatalf("failed to create synapse: %v", err)
	}

	ctx := context.Background()
	session := zyn.NewSession()
	var wg sync.WaitGroup
	var successCount atomic.Int64

	goroutines := 20
	callsPerGoroutine := 5

	for i := 0; i < goroutines; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < callsPerGoroutine; j++ {
				_, err := synapse.Fire(ctx, session, "input")
				if err == nil {
					successCount.Add(1)
				}
			}
		}()
	}

	wg.Wait()

	expectedCalls := int64(goroutines * callsPerGoroutine)
	if successCount.Load() != expectedCalls {
		t.Errorf("expected %d successful calls, got %d", expectedCalls, successCount.Load())
	}

	// Session should have 2 messages per successful call
	expectedMessages := int(expectedCalls) * 2
	if session.Len() != expectedMessages {
		t.Errorf("expected %d messages, got %d", expectedMessages, session.Len())
	}
}

func TestConcurrency_MultipleSynapsesSharedProvider(t *testing.T) {
	// Multiple synapse types sharing the same provider
	provider := zyn.NewMockProvider()

	binary, _ := zyn.Binary("Is valid?", provider)
	classify, _ := zyn.Classification("Type?", []string{"a", "b", "c"}, provider)
	sentiment, _ := zyn.Sentiment("Sentiment?", provider)

	ctx := context.Background()
	var wg sync.WaitGroup
	var totalCalls atomic.Int64

	// Launch goroutines for each synapse type
	for i := 0; i < 10; i++ {
		wg.Add(3)

		go func() {
			defer wg.Done()
			session := zyn.NewSession()
			for j := 0; j < 10; j++ {
				_, _ = binary.Fire(ctx, session, "input")
				totalCalls.Add(1)
			}
		}()

		go func() {
			defer wg.Done()
			session := zyn.NewSession()
			for j := 0; j < 10; j++ {
				_, _ = classify.Fire(ctx, session, "input")
				totalCalls.Add(1)
			}
		}()

		go func() {
			defer wg.Done()
			session := zyn.NewSession()
			for j := 0; j < 10; j++ {
				_, _ = sentiment.Fire(ctx, session, "input")
				totalCalls.Add(1)
			}
		}()
	}

	wg.Wait()

	expectedCalls := int64(10 * 3 * 10) // 10 iterations * 3 synapse types * 10 calls each
	if totalCalls.Load() != expectedCalls {
		t.Errorf("expected %d total calls, got %d", expectedCalls, totalCalls.Load())
	}
}

func TestConcurrency_SessionManipulationDuringFire(t *testing.T) {
	// One goroutine fires, another manipulates session concurrently
	provider := zyn.NewMockProvider()
	synapse, _ := zyn.Binary("question", provider)

	ctx := context.Background()
	session := zyn.NewSession()

	var wg sync.WaitGroup
	var fireCount atomic.Int64
	var manipCount atomic.Int64

	// Fire goroutine
	wg.Add(1)
	go func() {
		defer wg.Done()
		for i := 0; i < 100; i++ {
			_, _ = synapse.Fire(ctx, session, "input")
			fireCount.Add(1)
		}
	}()

	// Manipulation goroutine - reads and occasionally clears
	wg.Add(1)
	go func() {
		defer wg.Done()
		for i := 0; i < 100; i++ {
			_ = session.Messages()
			_ = session.Len()
			manipCount.Add(1)
			if i%20 == 0 {
				session.Clear()
			}
			time.Sleep(time.Microsecond)
		}
	}()

	wg.Wait()

	// Just verify no panics or races occurred
	if fireCount.Load() != 100 {
		t.Errorf("expected 100 fires, got %d", fireCount.Load())
	}
	if manipCount.Load() != 100 {
		t.Errorf("expected 100 manipulations, got %d", manipCount.Load())
	}
}

func TestConcurrency_RetryUnderLoad(t *testing.T) {
	// Multiple goroutines with retrying synapses
	var callCount atomic.Int64
	provider := zyn.NewMockProviderWithCallback(func(_ string, _ float32) (string, error) {
		count := callCount.Add(1)
		// Fail every 3rd call
		if count%3 == 0 {
			return "", fmt.Errorf("simulated failure")
		}
		return zynt.NewResponseBuilder().WithDecision(true).WithConfidence(0.9).WithReasoning("ok").Build(), nil
	})

	synapse, _ := zyn.Binary("question", provider, zyn.WithRetry(3))

	ctx := context.Background()
	var wg sync.WaitGroup
	var successCount atomic.Int64

	goroutines := 20
	for i := 0; i < goroutines; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			session := zyn.NewSession()
			_, err := synapse.Fire(ctx, session, "input")
			if err == nil {
				successCount.Add(1)
			}
		}()
	}

	wg.Wait()

	// Most should succeed due to retries
	if successCount.Load() < int64(goroutines/2) {
		t.Errorf("expected at least %d successes, got %d", goroutines/2, successCount.Load())
	}
}

func TestConcurrency_TimeoutUnderLoad(t *testing.T) {
	// Concurrent calls with tight timeouts
	var callCount atomic.Int64
	provider := zyn.NewMockProviderWithCallback(func(_ string, _ float32) (string, error) {
		callCount.Add(1)
		// Randomly delay some calls
		if callCount.Load()%5 == 0 {
			time.Sleep(200 * time.Millisecond)
		}
		return zynt.NewResponseBuilder().WithDecision(true).WithConfidence(0.9).WithReasoning("ok").Build(), nil
	})

	synapse, _ := zyn.Binary("question", provider, zyn.WithTimeout(100*time.Millisecond))

	ctx := context.Background()
	var wg sync.WaitGroup
	var successCount atomic.Int64
	var timeoutCount atomic.Int64

	goroutines := 30
	for i := 0; i < goroutines; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			session := zyn.NewSession()
			_, err := synapse.Fire(ctx, session, "input")
			if err == nil {
				successCount.Add(1)
			} else {
				timeoutCount.Add(1)
			}
		}()
	}

	wg.Wait()

	// Should have mix of successes and timeouts
	total := successCount.Load() + timeoutCount.Load()
	if total != int64(goroutines) {
		t.Errorf("expected %d total results, got %d", goroutines, total)
	}

	t.Logf("Results: %d successes, %d timeouts", successCount.Load(), timeoutCount.Load())
}

func TestConcurrency_ProviderCallRecording(t *testing.T) {
	// Verify CallRecorder is thread-safe under concurrent load
	inner := zyn.NewMockProvider()
	recorder := zynt.NewCallRecorder(inner)

	synapse, _ := zyn.Binary("question", recorder)

	ctx := context.Background()
	var wg sync.WaitGroup

	goroutines := 50
	callsPerGoroutine := 20

	for i := 0; i < goroutines; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			session := zyn.NewSession()
			for j := 0; j < callsPerGoroutine; j++ {
				_, _ = synapse.Fire(ctx, session, "input")
			}
		}()
	}

	wg.Wait()

	expectedCalls := goroutines * callsPerGoroutine
	if recorder.CallCount() != expectedCalls {
		t.Errorf("expected %d recorded calls, got %d", expectedCalls, recorder.CallCount())
	}

	calls := recorder.Calls()
	if len(calls) != expectedCalls {
		t.Errorf("expected %d calls in slice, got %d", expectedCalls, len(calls))
	}
}

func TestConcurrency_UsageAccumulation(t *testing.T) {
	// Verify UsageAccumulator is thread-safe
	provider := zyn.NewMockProvider()
	synapse, _ := zyn.Binary("question", provider)

	ctx := context.Background()
	acc := zynt.NewUsageAccumulator()
	var wg sync.WaitGroup

	goroutines := 100
	for i := 0; i < goroutines; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			session := zyn.NewSession()
			_, _ = synapse.Fire(ctx, session, "input")
			acc.Add(session)
		}()
	}

	wg.Wait()

	if acc.CallCount() != goroutines {
		t.Errorf("expected %d accumulated calls, got %d", goroutines, acc.CallCount())
	}

	// Mock provider returns 150 total tokens per call
	expectedTokens := goroutines * 150
	if acc.TotalTokens() != expectedTokens {
		t.Errorf("expected %d total tokens, got %d", expectedTokens, acc.TotalTokens())
	}
}
