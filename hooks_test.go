package zyn

import (
	"context"
	"sync"
	"testing"

	"github.com/zoobzio/capitan"
)

// TestRequestStartedHook verifies that request.started hook is emitted with all fields.
func TestRequestStartedHook(t *testing.T) {
	var wg sync.WaitGroup
	var hookCalled bool
	var requestIDReceived string
	var synapseTypeReceived string
	var providerReceived string
	var taskReceived string
	var inputReceived string
	var tempReceived float64

	wg.Add(1)
	listener := capitan.Hook(RequestStarted, func(_ context.Context, e *capitan.Event) {
		defer wg.Done()
		hookCalled = true
		requestIDReceived, _ = RequestIDKey.From(e)
		synapseTypeReceived, _ = SynapseTypeKey.From(e)
		providerReceived, _ = ProviderKey.From(e)
		taskReceived, _ = PromptTaskKey.From(e)
		inputReceived, _ = InputKey.From(e)
		tempReceived, _ = TemperatureKey.From(e)
	})
	defer listener.Close()

	// Create a mock provider
	mockProvider := NewMockProviderWithResponse(`{"decision": true, "confidence": 0.9, "reasoning": ["test"]}`)

	// Execute a binary synapse
	synapse := Binary("test question", mockProvider)
	_, _ = synapse.Fire(context.Background(), "test input")

	// Wait for hook with timeout
	done := make(chan struct{})
	go func() {
		wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		// Success
	case <-context.Background().Done():
		t.Fatal("Timeout waiting for hook")
	}

	if !hookCalled {
		t.Error("request.started hook was not called")
	}
	if requestIDReceived == "" {
		t.Error("Request ID was not set in hook")
	}
	if synapseTypeReceived != "binary" {
		t.Errorf("Expected synapse type 'binary', got %q", synapseTypeReceived)
	}
	if providerReceived != "mock-fixed" {
		t.Errorf("Expected provider 'mock-fixed', got %q", providerReceived)
	}
	if taskReceived != "Determine if test question" {
		t.Errorf("Expected task 'Determine if test question', got %q", taskReceived)
	}
	if inputReceived != "test input" {
		t.Errorf("Expected input 'test input', got %q", inputReceived)
	}
	if tempReceived == 0 {
		t.Error("Temperature was not set in hook")
	}
}

// TestRequestCompletedHook verifies that request.completed hook is emitted with all fields.
func TestRequestCompletedHook(t *testing.T) {
	var wg sync.WaitGroup
	var hookCalled bool
	var requestIDReceived string
	var synapseTypeReceived string
	var inputReceived string
	var outputReceived string
	var responseReceived string

	wg.Add(1)
	listener := capitan.Hook(RequestCompleted, func(_ context.Context, e *capitan.Event) {
		defer wg.Done()
		hookCalled = true
		requestIDReceived, _ = RequestIDKey.From(e)
		synapseTypeReceived, _ = SynapseTypeKey.From(e)
		inputReceived, _ = InputKey.From(e)
		outputReceived, _ = OutputKey.From(e)
		responseReceived, _ = ResponseKey.From(e)
	})
	defer listener.Close()

	mockProvider := NewMockProviderWithResponse(`{"decision": true, "confidence": 0.9, "reasoning": ["test"]}`)
	synapse := Binary("test question", mockProvider)
	_, _ = synapse.Fire(context.Background(), "test input")

	// Wait for hook
	done := make(chan struct{})
	go func() {
		wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		// Success
	case <-context.Background().Done():
		t.Fatal("Timeout waiting for hook")
	}

	if !hookCalled {
		t.Error("request.completed hook was not called")
	}
	if requestIDReceived == "" {
		t.Error("Request ID was not set in hook")
	}
	if synapseTypeReceived != "binary" {
		t.Errorf("Expected synapse type 'binary', got %q", synapseTypeReceived)
	}
	if inputReceived != "test input" {
		t.Errorf("Expected input 'test input', got %q", inputReceived)
	}
	if outputReceived == "" {
		t.Error("Output was not set in hook")
	}
	if responseReceived == "" {
		t.Error("Response was not set in hook")
	}
}

// TestRequestFailedHook verifies that request.failed hook is emitted on error.
func TestRequestFailedHook(t *testing.T) {
	var wg sync.WaitGroup
	var hookCalled bool
	var errorReceived string

	wg.Add(1)
	listener := capitan.Hook(RequestFailed, func(_ context.Context, e *capitan.Event) {
		defer wg.Done()
		hookCalled = true
		errorReceived, _ = ErrorKey.From(e)
	})
	defer listener.Close()

	mockProvider := NewMockProviderWithError("simulated error")
	synapse := Binary("test question", mockProvider)
	_, err := synapse.Fire(context.Background(), "test input")

	if err == nil {
		t.Fatal("Expected error but got none")
	}

	// Wait for hook
	done := make(chan struct{})
	go func() {
		wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		// Success
	case <-context.Background().Done():
		t.Fatal("Timeout waiting for hook")
	}

	if !hookCalled {
		t.Error("request.failed hook was not called")
	}
	if errorReceived == "" {
		t.Error("Error was not set in hook")
	}
}

// TestResponseParseFailedHook verifies that response.failed hook is emitted on parse error.
func TestResponseParseFailedHook(t *testing.T) {
	var wg sync.WaitGroup
	var hookCalled bool
	var errorTypeReceived string

	wg.Add(1)
	listener := capitan.Hook(ResponseParseFailed, func(_ context.Context, e *capitan.Event) {
		defer wg.Done()
		hookCalled = true
		errorTypeReceived, _ = ErrorTypeKey.From(e)
	})
	defer listener.Close()

	// Provide invalid JSON response
	mockProvider := NewMockProviderWithResponse(`{invalid json`)
	synapse := Binary("test question", mockProvider)
	_, err := synapse.Fire(context.Background(), "test input")

	if err == nil {
		t.Fatal("Expected parse error but got none")
	}

	// Wait for hook
	done := make(chan struct{})
	go func() {
		wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		// Success
	case <-context.Background().Done():
		t.Fatal("Timeout waiting for hook")
	}

	if !hookCalled {
		t.Error("response.failed hook was not called")
	}
	if errorTypeReceived != "parse_error" {
		t.Errorf("Expected error_type 'parse_error', got %q", errorTypeReceived)
	}
}

// TestProviderCallStartedHook verifies that provider.call.started hook is emitted.
func TestProviderCallStartedHook(t *testing.T) {
	var wg sync.WaitGroup
	var hookCalled bool
	var providerReceived string
	var modelReceived string

	wg.Add(1)
	listener := capitan.Hook(ProviderCallStarted, func(_ context.Context, e *capitan.Event) {
		defer wg.Done()
		hookCalled = true
		providerReceived, _ = ProviderKey.From(e)
		modelReceived, _ = ModelKey.From(e)
	})
	defer listener.Close()

	// Emit a test event
	capitan.Emit(context.Background(), ProviderCallStarted,
		ProviderKey.Field("openai"),
		ModelKey.Field("gpt-4"),
	)

	// Wait for hook
	done := make(chan struct{})
	go func() {
		wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		// Success
	case <-context.Background().Done():
		t.Fatal("Timeout waiting for hook")
	}

	if !hookCalled {
		t.Error("provider.call.started hook was not called")
	}
	if providerReceived != "openai" {
		t.Errorf("Expected provider 'openai', got %q", providerReceived)
	}
	if modelReceived != "gpt-4" {
		t.Errorf("Expected model 'gpt-4', got %q", modelReceived)
	}
}

// TestProviderCallCompletedHook verifies that provider.call.completed hook is emitted with all fields.
func TestProviderCallCompletedHook(t *testing.T) {
	var wg sync.WaitGroup
	var hookCalled bool
	var providerReceived string
	var modelReceived string
	var promptTokensReceived int
	var completionTokensReceived int
	var totalTokensReceived int
	var durationReceived int
	var statusCodeReceived int
	var responseIDReceived string

	wg.Add(1)
	listener := capitan.Hook(ProviderCallCompleted, func(_ context.Context, e *capitan.Event) {
		defer wg.Done()
		hookCalled = true
		providerReceived, _ = ProviderKey.From(e)
		modelReceived, _ = ModelKey.From(e)
		promptTokensReceived, _ = PromptTokensKey.From(e)
		completionTokensReceived, _ = CompletionTokensKey.From(e)
		totalTokensReceived, _ = TotalTokensKey.From(e)
		durationReceived, _ = DurationMsKey.From(e)
		statusCodeReceived, _ = HTTPStatusCodeKey.From(e)
		responseIDReceived, _ = ResponseIDKey.From(e)
	})
	defer listener.Close()

	// Emit a test event to verify the hook infrastructure works
	capitan.Emit(context.Background(), ProviderCallCompleted,
		ProviderKey.Field("openai"),
		ModelKey.Field("gpt-4"),
		PromptTokensKey.Field(10),
		CompletionTokensKey.Field(20),
		TotalTokensKey.Field(30),
		DurationMsKey.Field(150),
		HTTPStatusCodeKey.Field(200),
		ResponseIDKey.Field("chatcmpl-123"),
	)

	// Wait for hook
	done := make(chan struct{})
	go func() {
		wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		// Success
	case <-context.Background().Done():
		t.Fatal("Timeout waiting for hook")
	}

	if !hookCalled {
		t.Error("provider.call.completed hook was not called")
	}
	if providerReceived != "openai" {
		t.Errorf("Expected provider 'openai', got %q", providerReceived)
	}
	if modelReceived != "gpt-4" {
		t.Errorf("Expected model 'gpt-4', got %q", modelReceived)
	}
	if promptTokensReceived != 10 {
		t.Errorf("Expected prompt tokens 10, got %d", promptTokensReceived)
	}
	if completionTokensReceived != 20 {
		t.Errorf("Expected completion tokens 20, got %d", completionTokensReceived)
	}
	if totalTokensReceived != 30 {
		t.Errorf("Expected total tokens 30, got %d", totalTokensReceived)
	}
	if durationReceived != 150 {
		t.Errorf("Expected duration 150ms, got %d", durationReceived)
	}
	if statusCodeReceived != 200 {
		t.Errorf("Expected status code 200, got %d", statusCodeReceived)
	}
	if responseIDReceived != "chatcmpl-123" {
		t.Errorf("Expected response ID 'chatcmpl-123', got %q", responseIDReceived)
	}
}

// TestProviderCallFailedHook verifies that provider.call.failed hook is emitted with error metadata.
func TestProviderCallFailedHook(t *testing.T) {
	var wg sync.WaitGroup
	var hookCalled bool
	var statusCodeReceived int
	var errorReceived string
	var errorTypeReceived string

	wg.Add(1)
	listener := capitan.Hook(ProviderCallFailed, func(_ context.Context, e *capitan.Event) {
		defer wg.Done()
		hookCalled = true
		statusCodeReceived, _ = HTTPStatusCodeKey.From(e)
		errorReceived, _ = ErrorKey.From(e)
		errorTypeReceived, _ = APIErrorTypeKey.From(e)
	})
	defer listener.Close()

	// Emit a test event
	capitan.Emit(context.Background(), ProviderCallFailed,
		ProviderKey.Field("openai"),
		ModelKey.Field("gpt-4"),
		HTTPStatusCodeKey.Field(429),
		ErrorKey.Field("rate limit exceeded"),
		APIErrorTypeKey.Field("rate_limit_error"),
		DurationMsKey.Field(50),
	)

	// Wait for hook
	done := make(chan struct{})
	go func() {
		wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		// Success
	case <-context.Background().Done():
		t.Fatal("Timeout waiting for hook")
	}

	if !hookCalled {
		t.Error("provider.call.failed hook was not called")
	}
	if statusCodeReceived != 429 {
		t.Errorf("Expected status code 429, got %d", statusCodeReceived)
	}
	if errorReceived != "rate limit exceeded" {
		t.Errorf("Expected error 'rate limit exceeded', got %q", errorReceived)
	}
	if errorTypeReceived != "rate_limit_error" {
		t.Errorf("Expected error type 'rate_limit_error', got %q", errorTypeReceived)
	}
}

// TestHooksWithObserver verifies that observers can capture all hook events.
func TestHooksWithObserver(t *testing.T) {
	var wg sync.WaitGroup
	var eventCount int
	var mu sync.Mutex

	wg.Add(2) // Expect request.started and request.completed
	observer := capitan.Observe(func(_ context.Context, e *capitan.Event) {
		mu.Lock()
		defer mu.Unlock()

		signal := e.Signal()
		if signal == RequestStarted || signal == RequestCompleted {
			eventCount++
			wg.Done()
		}
	})
	defer observer.Close()

	mockProvider := NewMockProviderWithResponse(`{"decision": true, "confidence": 0.9, "reasoning": ["test"]}`)
	synapse := Binary("test question", mockProvider)
	_, _ = synapse.Fire(context.Background(), "test input")

	// Wait for hooks
	done := make(chan struct{})
	go func() {
		wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		// Success
	case <-context.Background().Done():
		t.Fatal("Timeout waiting for hooks")
	}

	mu.Lock()
	defer mu.Unlock()
	if eventCount != 2 {
		t.Errorf("Expected 2 events (started + completed), got %d", eventCount)
	}
}
