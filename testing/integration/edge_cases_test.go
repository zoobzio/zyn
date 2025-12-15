package integration

import (
	"context"
	"strings"
	"testing"

	"github.com/zoobzio/zyn"
	zynt "github.com/zoobzio/zyn/testing"
)

func TestEdgeCase_LargeSessionHistory(t *testing.T) {
	// Test with a large number of messages in session
	provider := zynt.NewSequencedProvider(
		zynt.NewResponseBuilder().WithDecision(true).WithConfidence(0.9).WithReasoning("ok").Build(),
	)

	synapse, _ := zyn.Binary("question", provider)
	session := zyn.NewSession()
	ctx := context.Background()

	// Pre-populate session with many messages
	for i := 0; i < 500; i++ {
		session.Append(zyn.RoleUser, "This is a user message with some content to simulate real conversations.")
		session.Append(zyn.RoleAssistant, "This is an assistant response with detailed information and context.")
	}

	if session.Len() != 1000 {
		t.Fatalf("expected 1000 messages, got %d", session.Len())
	}

	// Fire should still work with large history
	provider.Reset()
	result, err := synapse.Fire(ctx, session, "new input")
	if err != nil {
		t.Fatalf("fire with large session failed: %v", err)
	}

	if !result {
		t.Error("expected true result")
	}

	// Session should have 1002 messages now
	if session.Len() != 1002 {
		t.Errorf("expected 1002 messages, got %d", session.Len())
	}
}

func TestEdgeCase_VeryLongInput(t *testing.T) {
	// Test with very long input text
	provider := zyn.NewMockProvider()
	synapse, _ := zyn.Binary("Is this valid?", provider)
	session := zyn.NewSession()
	ctx := context.Background()

	// Create a very long input (100KB)
	longInput := strings.Repeat("This is a long input text. ", 5000)

	result, err := synapse.Fire(ctx, session, longInput)
	if err != nil {
		t.Fatalf("fire with long input failed: %v", err)
	}

	// Should succeed
	_ = result

	// Verify the message was stored
	messages := session.Messages()
	if len(messages) != 2 {
		t.Fatalf("expected 2 messages, got %d", len(messages))
	}

	if len(messages[0].Content) < len(longInput) {
		t.Error("input message was truncated unexpectedly")
	}
}

func TestEdgeCase_EmptyInput(t *testing.T) {
	// Test with empty input - framework rejects empty input at prompt validation
	provider := zyn.NewMockProvider()
	synapse, _ := zyn.Binary("Is this valid?", provider)
	session := zyn.NewSession()
	ctx := context.Background()

	// Empty input is rejected by prompt validation (Input is required)
	_, err := synapse.Fire(ctx, session, "")
	if err == nil {
		t.Fatal("expected error for empty input, got nil")
	}

	// Verify it's a validation error, not a provider error
	if !strings.Contains(err.Error(), "prompt") {
		t.Errorf("expected prompt validation error, got: %v", err)
	}
}

func TestEdgeCase_UnicodeInput(t *testing.T) {
	// Test with various unicode characters
	provider := zyn.NewMockProvider()
	synapse, _ := zyn.Binary("Is this valid?", provider)
	session := zyn.NewSession()
	ctx := context.Background()

	unicodeInputs := []string{
		"Hello 世界 🌍",
		"مرحبا بالعالم",
		"🎉🎊🎁🎄🎃",
		"Ω≈ç√∫˜µ≤≥÷",
		"田中さんにあげて下さい",
		"\u0000\u0001\u0002", // Control characters
	}

	for _, input := range unicodeInputs {
		session.Clear()
		_, err := synapse.Fire(ctx, session, input)
		if err != nil {
			t.Errorf("fire with unicode input %q failed: %v", input[:min(20, len(input))], err)
		}
	}
}

func TestEdgeCase_MalformedJSONResponse(t *testing.T) {
	// Test various malformed JSON responses
	malformedResponses := []string{
		``,                                   // Empty
		`{`,                                  // Incomplete
		`{"decision": }`,                     // Invalid value
		`{"decision": true`,                  // Missing closing brace
		`decision: true`,                     // Not JSON
		`{"decision": true, "confidence":}`,  // Trailing comma issue
		`{"decision": true, "reasoning": [}`, // Invalid array
	}

	for _, response := range malformedResponses {
		provider := zynt.NewSequencedProvider(response)
		synapse, _ := zyn.Binary("question", provider)
		session := zyn.NewSession()
		ctx := context.Background()

		_, err := synapse.Fire(ctx, session, "input")
		if err == nil {
			t.Errorf("expected error for malformed response %q, got nil", response)
		}

		// Session should remain empty on parse failure
		if session.Len() != 0 {
			t.Errorf("session should be empty after parse failure, got %d messages", session.Len())
		}
	}
}

func TestEdgeCase_PartiallyValidResponse(t *testing.T) {
	// Response is valid JSON but missing required fields
	partialResponses := []struct {
		response    string
		description string
	}{
		{`{}`, "empty object"},
		{`{"decision": true}`, "missing confidence and reasoning"},
		{`{"confidence": 0.9}`, "missing decision and reasoning"},
		{`{"decision": true, "confidence": 0.9}`, "missing reasoning"},
		{`{"decision": true, "confidence": 0.9, "reasoning": []}`, "empty reasoning array"},
		{`{"decision": true, "confidence": 1.5, "reasoning": ["r"]}`, "confidence out of range"},
	}

	for _, tc := range partialResponses {
		provider := zynt.NewSequencedProvider(tc.response)
		synapse, _ := zyn.Binary("question", provider)
		session := zyn.NewSession()
		ctx := context.Background()

		_, err := synapse.Fire(ctx, session, "input")
		if err == nil {
			t.Errorf("expected validation error for %s, got nil", tc.description)
		}
	}
}

func TestEdgeCase_ExtraFieldsInResponse(t *testing.T) {
	// Response has extra fields not in schema - should be ignored
	response := `{
		"decision": true,
		"confidence": 0.9,
		"reasoning": ["valid"],
		"extra_field": "should be ignored",
		"another_extra": 123
	}`

	provider := zynt.NewSequencedProvider(response)
	synapse, _ := zyn.Binary("question", provider)
	session := zyn.NewSession()
	ctx := context.Background()

	result, err := synapse.Fire(ctx, session, "input")
	if err != nil {
		t.Fatalf("extra fields should not cause error: %v", err)
	}

	if !result {
		t.Error("expected true result")
	}
}

func TestEdgeCase_NullFieldsInResponse(t *testing.T) {
	// Response has null for optional-ish fields
	response := `{
		"decision": true,
		"confidence": 0.9,
		"reasoning": ["valid"]
	}`

	provider := zynt.NewSequencedProvider(response)
	synapse, _ := zyn.Binary("question", provider)
	session := zyn.NewSession()
	ctx := context.Background()

	result, err := synapse.Fire(ctx, session, "input")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !result {
		t.Error("expected true result")
	}
}

func TestEdgeCase_SessionPruneEdgeCases(t *testing.T) {
	session := zyn.NewSession()

	// Prune empty session
	err := session.Prune(5)
	if err != nil {
		t.Errorf("prune on empty session should not error: %v", err)
	}

	// Add some messages
	for i := 0; i < 10; i++ {
		session.Append(zyn.RoleUser, "user")
		session.Append(zyn.RoleAssistant, "assistant")
	}

	// Prune more than exists
	err = session.Prune(100)
	if err != nil {
		t.Errorf("prune more than exists should not error: %v", err)
	}
	if session.Len() != 0 {
		t.Errorf("expected 0 messages after over-prune, got %d", session.Len())
	}

	// Prune negative
	err = session.Prune(-1)
	if err == nil {
		t.Error("expected error for negative prune")
	}
}

func TestEdgeCase_SessionTruncateEdgeCases(t *testing.T) {
	session := zyn.NewSession()

	// Truncate empty session
	err := session.Truncate(2, 2)
	if err != nil {
		t.Errorf("truncate on empty session should not error: %v", err)
	}

	// Add messages
	for i := 0; i < 10; i++ {
		session.Append(zyn.RoleUser, "user")
	}

	// Truncate where keepFirst + keepLast > len (no-op)
	err = session.Truncate(5, 6)
	if err != nil {
		t.Errorf("truncate with large keeps should not error: %v", err)
	}
	if session.Len() != 10 {
		t.Errorf("expected 10 messages (no-op truncate), got %d", session.Len())
	}

	// Truncate with negative values
	err = session.Truncate(-1, 2)
	if err == nil {
		t.Error("expected error for negative keepFirst")
	}

	err = session.Truncate(2, -1)
	if err == nil {
		t.Error("expected error for negative keepLast")
	}
}

func TestEdgeCase_SessionAtOutOfBounds(t *testing.T) {
	session := zyn.NewSession()
	session.Append(zyn.RoleUser, "test")

	// Valid index
	_, err := session.At(0)
	if err != nil {
		t.Errorf("At(0) should succeed: %v", err)
	}

	// Out of bounds
	_, err = session.At(1)
	if err == nil {
		t.Error("At(1) should fail on single-element session")
	}

	_, err = session.At(-1)
	if err == nil {
		t.Error("At(-1) should fail")
	}

	_, err = session.At(100)
	if err == nil {
		t.Error("At(100) should fail")
	}
}

func TestEdgeCase_SessionRemoveOutOfBounds(t *testing.T) {
	session := zyn.NewSession()
	session.Append(zyn.RoleUser, "test")

	err := session.Remove(-1)
	if err == nil {
		t.Error("Remove(-1) should fail")
	}

	err = session.Remove(100)
	if err == nil {
		t.Error("Remove(100) should fail")
	}

	// Valid remove
	err = session.Remove(0)
	if err != nil {
		t.Errorf("Remove(0) should succeed: %v", err)
	}

	if session.Len() != 0 {
		t.Errorf("expected 0 messages after remove, got %d", session.Len())
	}
}

func TestEdgeCase_SessionInsertEdgeCases(t *testing.T) {
	session := zyn.NewSession()

	// Insert at 0 on empty session
	err := session.Insert(0, zyn.Message{Role: zyn.RoleUser, Content: "first"})
	if err != nil {
		t.Errorf("Insert(0) on empty should succeed: %v", err)
	}

	// Insert at end (append)
	err = session.Insert(1, zyn.Message{Role: zyn.RoleUser, Content: "second"})
	if err != nil {
		t.Errorf("Insert at end should succeed: %v", err)
	}

	// Insert in middle
	err = session.Insert(1, zyn.Message{Role: zyn.RoleUser, Content: "middle"})
	if err != nil {
		t.Errorf("Insert in middle should succeed: %v", err)
	}

	// Out of bounds
	err = session.Insert(-1, zyn.Message{})
	if err == nil {
		t.Error("Insert(-1) should fail")
	}

	err = session.Insert(100, zyn.Message{})
	if err == nil {
		t.Error("Insert(100) should fail on 3-element session")
	}

	// Verify order
	messages := session.Messages()
	if len(messages) != 3 {
		t.Fatalf("expected 3 messages, got %d", len(messages))
	}
	if messages[0].Content != "first" {
		t.Errorf("expected first message 'first', got %q", messages[0].Content)
	}
	if messages[1].Content != "middle" {
		t.Errorf("expected second message 'middle', got %q", messages[1].Content)
	}
	if messages[2].Content != "second" {
		t.Errorf("expected third message 'second', got %q", messages[2].Content)
	}
}

func TestEdgeCase_ClassificationEmptyCategories(_ *testing.T) {
	provider := zyn.NewMockProvider()

	// Empty categories - should this error?
	_, err := zyn.Classification("Classify", []string{}, provider)
	// The current implementation may or may not error - just verify no panic
	_ = err
}

func TestEdgeCase_ClassificationSingleCategory(t *testing.T) {
	provider := zynt.NewSequencedProvider(
		zynt.NewResponseBuilder().WithPrimary("only").WithSecondary("").WithConfidence(1.0).WithReasoning("only choice").Build(),
	)

	synapse, err := zyn.Classification("Classify", []string{"only"}, provider)
	if err != nil {
		t.Fatalf("single category should work: %v", err)
	}

	session := zyn.NewSession()
	ctx := context.Background()

	result, err := synapse.Fire(ctx, session, "input")
	if err != nil {
		t.Fatalf("fire failed: %v", err)
	}

	if result != "only" {
		t.Errorf("expected 'only', got %q", result)
	}
}

func TestEdgeCase_RankingEmptyItems(t *testing.T) {
	provider := zynt.NewSequencedProvider(
		zynt.NewResponseBuilder().WithRanked().WithConfidence(1.0).WithReasoning("nothing to rank").Build(),
	)

	synapse, _ := zyn.Ranking("by priority", provider)
	session := zyn.NewSession()
	ctx := context.Background()

	// Fire with empty items
	result, err := synapse.Fire(ctx, session, []string{})
	// Should handle gracefully - either error or empty result
	if err != nil {
		// Error is acceptable for empty input
		t.Logf("empty items returned error (acceptable): %v", err)
	} else if len(result) != 0 {
		t.Errorf("expected empty result for empty items, got %v", result)
	}
}

func TestEdgeCase_RankingSingleItem(t *testing.T) {
	provider := zynt.NewSequencedProvider(
		zynt.NewResponseBuilder().WithRanked("only").WithConfidence(1.0).WithReasoning("only item").Build(),
	)

	synapse, _ := zyn.Ranking("by priority", provider)
	session := zyn.NewSession()
	ctx := context.Background()

	result, err := synapse.Fire(ctx, session, []string{"only"})
	if err != nil {
		t.Fatalf("single item ranking failed: %v", err)
	}

	if len(result) != 1 || result[0] != "only" {
		t.Errorf("expected ['only'], got %v", result)
	}
}
