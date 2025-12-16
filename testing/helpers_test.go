package testing

import (
	"context"
	"encoding/json"
	"sync"
	"testing"
	"time"

	"github.com/zoobzio/zyn"
)

func TestResponseBuilder_BinaryResponse(t *testing.T) {
	response := NewResponseBuilder().
		WithDecision(true).
		WithConfidence(0.95).
		WithReasoning("reason1", "reason2").
		Build()

	var data map[string]any
	if err := json.Unmarshal([]byte(response), &data); err != nil {
		t.Fatalf("failed to unmarshal response: %v", err)
	}

	if data["decision"] != true {
		t.Errorf("expected decision=true, got %v", data["decision"])
	}
	if data["confidence"] != 0.95 {
		t.Errorf("expected confidence=0.95, got %v", data["confidence"])
	}
	reasoning, ok := data["reasoning"].([]any)
	if !ok || len(reasoning) != 2 {
		t.Errorf("expected 2 reasoning items, got %v", data["reasoning"])
	}
}

func TestResponseBuilder_ClassificationResponse(t *testing.T) {
	response := NewResponseBuilder().
		WithPrimary("spam").
		WithSecondary("promotional").
		WithConfidence(0.87).
		WithReasoning("contains sales language").
		Build()

	var data map[string]any
	if err := json.Unmarshal([]byte(response), &data); err != nil {
		t.Fatalf("failed to unmarshal response: %v", err)
	}

	if data["primary"] != "spam" {
		t.Errorf("expected primary=spam, got %v", data["primary"])
	}
	if data["secondary"] != "promotional" {
		t.Errorf("expected secondary=promotional, got %v", data["secondary"])
	}
}

func TestResponseBuilder_RankingResponse(t *testing.T) {
	response := NewResponseBuilder().
		WithRanked("first", "second", "third").
		WithConfidence(0.9).
		WithReasoning("ordered by relevance").
		Build()

	var data map[string]any
	if err := json.Unmarshal([]byte(response), &data); err != nil {
		t.Fatalf("failed to unmarshal response: %v", err)
	}

	ranked, ok := data["ranked"].([]any)
	if !ok || len(ranked) != 3 {
		t.Errorf("expected 3 ranked items, got %v", data["ranked"])
	}
	if ranked[0] != "first" {
		t.Errorf("expected first item=first, got %v", ranked[0])
	}
}

func TestResponseBuilder_SentimentResponse(t *testing.T) {
	response := NewResponseBuilder().
		WithOverall("positive").
		WithConfidence(0.85).
		WithScores(0.7, 0.1, 0.2).
		WithEmotions("joy", "satisfaction").
		WithAspects(map[string]string{"service": "positive", "price": "neutral"}).
		WithReasoning("generally positive tone").
		Build()

	var data map[string]any
	if err := json.Unmarshal([]byte(response), &data); err != nil {
		t.Fatalf("failed to unmarshal response: %v", err)
	}

	if data["overall"] != "positive" {
		t.Errorf("expected overall=positive, got %v", data["overall"])
	}

	scores, ok := data["scores"].(map[string]any)
	if !ok {
		t.Fatalf("expected scores map, got %T", data["scores"])
	}
	if scores["positive"] != 0.7 {
		t.Errorf("expected positive score=0.7, got %v", scores["positive"])
	}
}

func TestResponseBuilder_TransformResponse(t *testing.T) {
	response := NewResponseBuilder().
		WithOutput("transformed text").
		WithConfidence(0.92).
		WithChanges("capitalized", "trimmed").
		WithReasoning("applied formatting").
		Build()

	var data map[string]any
	if err := json.Unmarshal([]byte(response), &data); err != nil {
		t.Fatalf("failed to unmarshal response: %v", err)
	}

	if data["output"] != "transformed text" {
		t.Errorf("expected output='transformed text', got %v", data["output"])
	}
}

func TestResponseBuilder_CustomField(t *testing.T) {
	response := NewResponseBuilder().
		WithField("custom_key", "custom_value").
		WithField("nested", map[string]string{"inner": "value"}).
		Build()

	var data map[string]any
	if err := json.Unmarshal([]byte(response), &data); err != nil {
		t.Fatalf("failed to unmarshal response: %v", err)
	}

	if data["custom_key"] != "custom_value" {
		t.Errorf("expected custom_key=custom_value, got %v", data["custom_key"])
	}
}

func TestSequencedProvider_ReturnsInOrder(t *testing.T) {
	provider := NewSequencedProvider(
		`{"index": 0}`,
		`{"index": 1}`,
		`{"index": 2}`,
	)

	ctx := context.Background()
	for i := 0; i < 3; i++ {
		resp, err := provider.Call(ctx, nil, 0)
		if err != nil {
			t.Fatalf("call %d failed: %v", i, err)
		}

		var data map[string]any
		if err := json.Unmarshal([]byte(resp.Content), &data); err != nil {
			t.Fatalf("failed to unmarshal response %d: %v", i, err)
		}

		if int(data["index"].(float64)) != i {
			t.Errorf("call %d: expected index=%d, got %v", i, i, data["index"])
		}
	}

	if provider.CallCount() != 3 {
		t.Errorf("expected 3 calls, got %d", provider.CallCount())
	}
}

func TestSequencedProvider_RepeatsLastResponse(t *testing.T) {
	provider := NewSequencedProvider(
		`{"value": "first"}`,
		`{"value": "last"}`,
	)

	ctx := context.Background()

	// Exhaust responses
	_, _ = provider.Call(ctx, nil, 0)
	_, _ = provider.Call(ctx, nil, 0)

	// Additional calls should return last response
	for i := 0; i < 3; i++ {
		resp, err := provider.Call(ctx, nil, 0)
		if err != nil {
			t.Fatalf("call failed: %v", err)
		}

		var data map[string]any
		_ = json.Unmarshal([]byte(resp.Content), &data)

		if data["value"] != "last" {
			t.Errorf("expected value=last, got %v", data["value"])
		}
	}
}

func TestSequencedProvider_Reset(t *testing.T) {
	provider := NewSequencedProvider(
		`{"value": "first"}`,
		`{"value": "second"}`,
	)

	ctx := context.Background()
	_, _ = provider.Call(ctx, nil, 0)
	_, _ = provider.Call(ctx, nil, 0)

	provider.Reset()

	if provider.CallCount() != 0 {
		t.Errorf("expected call count 0 after reset, got %d", provider.CallCount())
	}

	resp, _ := provider.Call(ctx, nil, 0)
	var data map[string]any
	_ = json.Unmarshal([]byte(resp.Content), &data)

	if data["value"] != "first" {
		t.Errorf("expected value=first after reset, got %v", data["value"])
	}
}

func TestFailingProvider_FailsThenSucceeds(t *testing.T) {
	provider := NewFailingProvider(2).
		WithSuccessResponse(`{"success": true}`).
		WithFailError("test error")

	ctx := context.Background()

	// First two calls should fail
	for i := 0; i < 2; i++ {
		_, err := provider.Call(ctx, nil, 0)
		if err == nil {
			t.Errorf("call %d: expected error, got nil", i)
		}
	}

	// Third call should succeed
	resp, err := provider.Call(ctx, nil, 0)
	if err != nil {
		t.Fatalf("call 3: expected success, got error: %v", err)
	}

	var data map[string]any
	_ = json.Unmarshal([]byte(resp.Content), &data)

	if data["success"] != true {
		t.Errorf("expected success=true, got %v", data["success"])
	}

	if provider.CallCount() != 3 {
		t.Errorf("expected 3 calls, got %d", provider.CallCount())
	}
}

func TestFailingProvider_Reset(t *testing.T) {
	provider := NewFailingProvider(1)

	ctx := context.Background()
	_, _ = provider.Call(ctx, nil, 0) // fail
	_, _ = provider.Call(ctx, nil, 0) // succeed

	provider.Reset()

	if provider.CallCount() != 0 {
		t.Errorf("expected call count 0 after reset, got %d", provider.CallCount())
	}

	// Should fail again after reset
	_, err := provider.Call(ctx, nil, 0)
	if err == nil {
		t.Error("expected error after reset, got nil")
	}
}

func TestCallRecorder_RecordsCalls(t *testing.T) {
	inner := NewSequencedProvider(`{"ok": true}`)
	recorder := NewCallRecorder(inner)

	ctx := context.Background()
	messages := []zyn.Message{
		{Role: zyn.RoleUser, Content: "hello"},
		{Role: zyn.RoleAssistant, Content: "hi"},
	}

	_, err := recorder.Call(ctx, messages, 0.5)
	if err != nil {
		t.Fatalf("call failed: %v", err)
	}

	calls := recorder.Calls()
	if len(calls) != 1 {
		t.Fatalf("expected 1 call, got %d", len(calls))
	}

	if len(calls[0].Messages) != 2 {
		t.Errorf("expected 2 messages, got %d", len(calls[0].Messages))
	}
	if calls[0].Temperature != 0.5 {
		t.Errorf("expected temperature=0.5, got %v", calls[0].Temperature)
	}
	if calls[0].Messages[0].Content != "hello" {
		t.Errorf("expected first message='hello', got %v", calls[0].Messages[0].Content)
	}
}

func TestCallRecorder_LastCall(t *testing.T) {
	inner := NewSequencedProvider(`{"ok": true}`)
	recorder := NewCallRecorder(inner)

	// No calls yet
	if recorder.LastCall() != nil {
		t.Error("expected nil LastCall before any calls")
	}

	ctx := context.Background()
	_, _ = recorder.Call(ctx, []zyn.Message{{Role: zyn.RoleUser, Content: "first"}}, 0.1)
	_, _ = recorder.Call(ctx, []zyn.Message{{Role: zyn.RoleUser, Content: "second"}}, 0.2)

	last := recorder.LastCall()
	if last == nil {
		t.Fatal("expected non-nil LastCall")
	}
	if last.Messages[0].Content != "second" {
		t.Errorf("expected last message='second', got %v", last.Messages[0].Content)
	}
	if last.Temperature != 0.2 {
		t.Errorf("expected temperature=0.2, got %v", last.Temperature)
	}
}

func TestCallRecorder_Reset(t *testing.T) {
	inner := NewSequencedProvider(`{"ok": true}`)
	recorder := NewCallRecorder(inner)

	ctx := context.Background()
	_, _ = recorder.Call(ctx, nil, 0)
	_, _ = recorder.Call(ctx, nil, 0)

	recorder.Reset()

	if recorder.CallCount() != 0 {
		t.Errorf("expected 0 calls after reset, got %d", recorder.CallCount())
	}
	if len(recorder.Calls()) != 0 {
		t.Errorf("expected empty calls after reset")
	}
}

func TestCallRecorder_ConcurrentSafety(t *testing.T) {
	inner := NewSequencedProvider(`{"ok": true}`)
	recorder := NewCallRecorder(inner)

	ctx := context.Background()
	var wg sync.WaitGroup

	// Concurrent calls
	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_, _ = recorder.Call(ctx, []zyn.Message{{Role: zyn.RoleUser, Content: "test"}}, 0)
		}()
	}

	wg.Wait()

	if recorder.CallCount() != 100 {
		t.Errorf("expected 100 calls, got %d", recorder.CallCount())
	}
}

func TestLatencyProvider_AddsLatency(t *testing.T) {
	inner := NewSequencedProvider(`{"ok": true}`)
	provider := NewLatencyProvider(inner, 50*time.Millisecond)

	ctx := context.Background()
	start := time.Now()
	_, err := provider.Call(ctx, nil, 0)
	elapsed := time.Since(start)

	if err != nil {
		t.Fatalf("call failed: %v", err)
	}
	if elapsed < 50*time.Millisecond {
		t.Errorf("expected at least 50ms latency, got %v", elapsed)
	}
}

func TestLatencyProvider_RespectsContextCancellation(t *testing.T) {
	inner := NewSequencedProvider(`{"ok": true}`)
	provider := NewLatencyProvider(inner, 1*time.Second)

	ctx, cancel := context.WithCancel(context.Background())

	// Cancel after 50ms
	go func() {
		time.Sleep(50 * time.Millisecond)
		cancel()
	}()

	start := time.Now()
	_, err := provider.Call(ctx, nil, 0)
	elapsed := time.Since(start)

	if err == nil {
		t.Fatal("expected context cancellation error")
	}

	// Should have been canceled around 50ms, not 1 second
	if elapsed > 200*time.Millisecond {
		t.Errorf("context cancellation should have been faster, took %v", elapsed)
	}
}

func TestUsageAccumulator_AccumulatesUsage(t *testing.T) {
	acc := NewUsageAccumulator()

	acc.AddUsage(&zyn.TokenUsage{Prompt: 100, Completion: 50, Total: 150})
	acc.AddUsage(&zyn.TokenUsage{Prompt: 200, Completion: 100, Total: 300})

	if acc.PromptTokens() != 300 {
		t.Errorf("expected prompt tokens=300, got %d", acc.PromptTokens())
	}
	if acc.CompletionTokens() != 150 {
		t.Errorf("expected completion tokens=150, got %d", acc.CompletionTokens())
	}
	if acc.TotalTokens() != 450 {
		t.Errorf("expected total tokens=450, got %d", acc.TotalTokens())
	}
	if acc.CallCount() != 2 {
		t.Errorf("expected call count=2, got %d", acc.CallCount())
	}
}

func TestUsageAccumulator_AddFromSession(t *testing.T) {
	acc := NewUsageAccumulator()

	// Create a session and set usage
	session := zyn.NewSession()
	session.SetUsage(&zyn.TokenUsage{Prompt: 100, Completion: 50, Total: 150})

	acc.Add(session)

	if acc.TotalTokens() != 150 {
		t.Errorf("expected total tokens=150, got %d", acc.TotalTokens())
	}
}

func TestUsageAccumulator_Reset(t *testing.T) {
	acc := NewUsageAccumulator()
	acc.AddUsage(&zyn.TokenUsage{Prompt: 100, Completion: 50, Total: 150})

	acc.Reset()

	if acc.TotalTokens() != 0 {
		t.Errorf("expected 0 tokens after reset, got %d", acc.TotalTokens())
	}
	if acc.CallCount() != 0 {
		t.Errorf("expected 0 calls after reset, got %d", acc.CallCount())
	}
}

func TestUsageAccumulator_ConcurrentSafety(t *testing.T) {
	acc := NewUsageAccumulator()

	var wg sync.WaitGroup
	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			acc.AddUsage(&zyn.TokenUsage{Prompt: 10, Completion: 5, Total: 15})
		}()
	}

	wg.Wait()

	if acc.CallCount() != 100 {
		t.Errorf("expected 100 calls, got %d", acc.CallCount())
	}
	if acc.TotalTokens() != 1500 {
		t.Errorf("expected 1500 total tokens, got %d", acc.TotalTokens())
	}
}

func TestResponseBuilder_BuildBytes(t *testing.T) {
	t.Run("valid_response", func(t *testing.T) {
		bytes := NewResponseBuilder().
			WithDecision(true).
			WithConfidence(0.9).
			WithReasoning("test").
			BuildBytes()

		if len(bytes) == 0 {
			t.Error("expected non-empty bytes")
		}

		var data map[string]any
		if err := json.Unmarshal(bytes, &data); err != nil {
			t.Fatalf("failed to unmarshal bytes: %v", err)
		}

		if data["decision"] != true {
			t.Errorf("expected decision=true, got %v", data["decision"])
		}
	})

	t.Run("empty_builder", func(t *testing.T) {
		bytes := NewResponseBuilder().BuildBytes()

		if len(bytes) == 0 {
			t.Error("expected non-empty bytes for empty builder")
		}

		// Should be valid JSON (empty object)
		var data map[string]any
		if err := json.Unmarshal(bytes, &data); err != nil {
			t.Fatalf("failed to unmarshal empty builder bytes: %v", err)
		}
	})
}

func TestSequencedProvider_Name(t *testing.T) {
	provider := NewSequencedProvider(`{"ok": true}`)
	name := provider.Name()

	if name != SequencedProviderName {
		t.Errorf("expected name='%s', got '%s'", SequencedProviderName, name)
	}
}

func TestFailingProvider_Name(t *testing.T) {
	provider := NewFailingProvider(1)
	name := provider.Name()

	if name != FailingProviderName {
		t.Errorf("expected name='%s', got '%s'", FailingProviderName, name)
	}
}

func TestCallRecorder_Name(t *testing.T) {
	inner := NewSequencedProvider(`{"ok": true}`)
	recorder := NewCallRecorder(inner)
	name := recorder.Name()

	// Should return the wrapped provider's name
	if name != SequencedProviderName {
		t.Errorf("expected name='%s', got '%s'", SequencedProviderName, name)
	}
}

func TestLatencyProvider_Name(t *testing.T) {
	inner := NewSequencedProvider(`{"ok": true}`)
	provider := NewLatencyProvider(inner, 10*time.Millisecond)
	name := provider.Name()

	// Should return the wrapped provider's name
	if name != SequencedProviderName {
		t.Errorf("expected name='%s', got '%s'", SequencedProviderName, name)
	}
}

func TestSequencedProvider_EmptyResponses(t *testing.T) {
	// Test with no responses - should use default error response
	provider := NewSequencedProvider()

	ctx := context.Background()
	resp, err := provider.Call(ctx, nil, 0)
	if err != nil {
		t.Fatalf("call failed: %v", err)
	}

	var data map[string]any
	if err := json.Unmarshal([]byte(resp.Content), &data); err != nil {
		t.Fatalf("failed to unmarshal response: %v", err)
	}

	if _, ok := data["error"]; !ok {
		t.Error("expected error field in default response")
	}
}
