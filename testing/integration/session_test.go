package integration

import (
	"context"
	"testing"

	"github.com/zoobzio/zyn"
	zynt "github.com/zoobzio/zyn/testing"
)

func TestSession_MessageAccumulation(t *testing.T) {
	provider := zynt.NewSequencedProvider(
		zynt.NewResponseBuilder().WithDecision(true).WithConfidence(0.9).WithReasoning("valid").Build(),
		zynt.NewResponseBuilder().WithDecision(false).WithConfidence(0.8).WithReasoning("invalid").Build(),
	)

	synapse, err := zyn.Binary("Is this valid?", provider)
	if err != nil {
		t.Fatalf("failed to create synapse: %v", err)
	}

	session := zyn.NewSession()
	ctx := context.Background()

	// First call
	_, err = synapse.Fire(ctx, session, "input1")
	if err != nil {
		t.Fatalf("first call failed: %v", err)
	}

	if session.Len() != 2 {
		t.Errorf("expected 2 messages after first call, got %d", session.Len())
	}

	// Second call
	_, err = synapse.Fire(ctx, session, "input2")
	if err != nil {
		t.Fatalf("second call failed: %v", err)
	}

	if session.Len() != 4 {
		t.Errorf("expected 4 messages after second call, got %d", session.Len())
	}

	// Verify message roles alternate
	messages := session.Messages()
	for i, msg := range messages {
		expectedRole := zyn.RoleUser
		if i%2 == 1 {
			expectedRole = zyn.RoleAssistant
		}
		if msg.Role != expectedRole {
			t.Errorf("message %d: expected role=%s, got %s", i, expectedRole, msg.Role)
		}
	}
}

func TestSession_MultiSynapseWorkflow(t *testing.T) {
	// Simulate a workflow: classify -> extract -> transform
	classifyResp := zynt.NewResponseBuilder().
		WithPrimary("support").
		WithSecondary("").
		WithConfidence(0.9).
		WithReasoning("customer inquiry").
		Build()

	extractResp := `{"name": "John", "issue": "order not delivered"}`

	transformResp := zynt.NewResponseBuilder().
		WithOutput("Dear John, we apologize for the delay...").
		WithConfidence(0.95).
		WithChanges("personalized greeting", "added apology").
		WithReasoning("customer service template").
		Build()

	provider := zynt.NewSequencedProvider(classifyResp, extractResp, transformResp)

	session := zyn.NewSession()
	ctx := context.Background()

	// Step 1: Classify
	classifier, _ := zyn.Classification("type", []string{"support", "sales", "spam"}, provider)
	category, err := classifier.Fire(ctx, session, "My order hasn't arrived")
	if err != nil {
		t.Fatalf("classification failed: %v", err)
	}
	if category != "support" {
		t.Errorf("expected category=support, got %s", category)
	}

	// Step 2: Extract (using mock that returns valid JSON)
	// Note: In a real test we'd use Extract[T], but that requires a Validator type
	// For this integration test, we just verify session accumulation

	// Step 3: Transform
	transformer, _ := zyn.Transform("write response", provider)
	// Skip one call to consume the extract response
	_, _ = provider.Call(ctx, nil, 0)

	response, err := transformer.Fire(ctx, session, "Generate customer response")
	if err != nil {
		t.Fatalf("transform failed: %v", err)
	}
	if response == "" {
		t.Error("expected non-empty response")
	}

	// Session should have accumulated messages from both calls
	if session.Len() < 4 {
		t.Errorf("expected at least 4 messages, got %d", session.Len())
	}
}

func TestSession_Prune(t *testing.T) {
	provider := zynt.NewSequencedProvider(
		zynt.NewResponseBuilder().WithDecision(true).WithConfidence(0.9).WithReasoning("r1").Build(),
		zynt.NewResponseBuilder().WithDecision(true).WithConfidence(0.9).WithReasoning("r2").Build(),
		zynt.NewResponseBuilder().WithDecision(true).WithConfidence(0.9).WithReasoning("r3").Build(),
	)

	synapse, _ := zyn.Binary("question", provider)
	session := zyn.NewSession()
	ctx := context.Background()

	// Make 3 calls = 6 messages
	for i := 0; i < 3; i++ {
		_, _ = synapse.Fire(ctx, session, "input")
	}

	if session.Len() != 6 {
		t.Fatalf("expected 6 messages, got %d", session.Len())
	}

	// Prune last 2 pairs (4 messages)
	if err := session.Prune(2); err != nil {
		t.Fatalf("prune failed: %v", err)
	}

	if session.Len() != 2 {
		t.Errorf("expected 2 messages after prune, got %d", session.Len())
	}
}

func TestSession_Truncate(t *testing.T) {
	provider := zynt.NewSequencedProvider(
		zynt.NewResponseBuilder().WithDecision(true).WithConfidence(0.9).WithReasoning("r").Build(),
		zynt.NewResponseBuilder().WithDecision(true).WithConfidence(0.9).WithReasoning("r").Build(),
		zynt.NewResponseBuilder().WithDecision(true).WithConfidence(0.9).WithReasoning("r").Build(),
		zynt.NewResponseBuilder().WithDecision(true).WithConfidence(0.9).WithReasoning("r").Build(),
		zynt.NewResponseBuilder().WithDecision(true).WithConfidence(0.9).WithReasoning("r").Build(),
	)

	synapse, _ := zyn.Binary("question", provider)
	session := zyn.NewSession()
	ctx := context.Background()

	// Make 5 calls = 10 messages
	for i := 0; i < 5; i++ {
		_, _ = synapse.Fire(ctx, session, "input")
	}

	if session.Len() != 10 {
		t.Fatalf("expected 10 messages, got %d", session.Len())
	}

	// Keep first 2 and last 2
	if err := session.Truncate(2, 2); err != nil {
		t.Fatalf("truncate failed: %v", err)
	}

	if session.Len() != 4 {
		t.Errorf("expected 4 messages after truncate, got %d", session.Len())
	}
}

func TestSession_TokenTracking(t *testing.T) {
	provider := zynt.NewSequencedProvider(
		zynt.NewResponseBuilder().WithDecision(true).WithConfidence(0.9).WithReasoning("r").Build(),
	)

	synapse, _ := zyn.Binary("question", provider)
	session := zyn.NewSession()
	ctx := context.Background()

	// Before any calls
	if session.LastUsage() != nil {
		t.Error("expected nil usage before any calls")
	}

	_, _ = synapse.Fire(ctx, session, "input")

	usage := session.LastUsage()
	if usage == nil {
		t.Fatal("expected non-nil usage after call")
	}

	// Mock provider returns 100/50/150
	if usage.Prompt != 100 {
		t.Errorf("expected prompt=100, got %d", usage.Prompt)
	}
	if usage.Completion != 50 {
		t.Errorf("expected completion=50, got %d", usage.Completion)
	}
	if usage.Total != 150 {
		t.Errorf("expected total=150, got %d", usage.Total)
	}
}

func TestSession_Clear(t *testing.T) {
	provider := zynt.NewSequencedProvider(
		zynt.NewResponseBuilder().WithDecision(true).WithConfidence(0.9).WithReasoning("r").Build(),
	)

	synapse, _ := zyn.Binary("question", provider)
	session := zyn.NewSession()
	ctx := context.Background()

	_, _ = synapse.Fire(ctx, session, "input")

	if session.Len() == 0 {
		t.Fatal("expected messages before clear")
	}

	session.Clear()

	if session.Len() != 0 {
		t.Errorf("expected 0 messages after clear, got %d", session.Len())
	}
}

func TestSession_IndependentSessions(t *testing.T) {
	provider := zynt.NewSequencedProvider(
		zynt.NewResponseBuilder().WithDecision(true).WithConfidence(0.9).WithReasoning("r").Build(),
		zynt.NewResponseBuilder().WithDecision(false).WithConfidence(0.8).WithReasoning("r").Build(),
	)

	synapse, _ := zyn.Binary("question", provider)
	ctx := context.Background()

	session1 := zyn.NewSession()
	session2 := zyn.NewSession()

	_, _ = synapse.Fire(ctx, session1, "input1")
	_, _ = synapse.Fire(ctx, session2, "input2")

	// Sessions should be independent
	if session1.ID() == session2.ID() {
		t.Error("sessions should have different IDs")
	}

	if session1.Len() != 2 {
		t.Errorf("session1 expected 2 messages, got %d", session1.Len())
	}
	if session2.Len() != 2 {
		t.Errorf("session2 expected 2 messages, got %d", session2.Len())
	}
}

func TestSession_HeterogeneousSynapses(t *testing.T) {
	// Different synapse types using the same session
	binaryResp := zynt.NewResponseBuilder().WithDecision(true).WithConfidence(0.9).WithReasoning("valid").Build()
	classifyResp := zynt.NewResponseBuilder().WithPrimary("spam").WithSecondary("").WithConfidence(0.85).WithReasoning("promotional").Build()
	sentimentResp := zynt.NewResponseBuilder().
		WithOverall("negative").
		WithConfidence(0.8).
		WithScores(0.1, 0.7, 0.2).
		WithEmotions("frustration").
		WithAspects(map[string]string{}).
		WithReasoning("negative tone").
		Build()

	provider := zynt.NewSequencedProvider(binaryResp, classifyResp, sentimentResp)
	session := zyn.NewSession()
	ctx := context.Background()

	binary, _ := zyn.Binary("Is this spam?", provider)
	_, _ = binary.Fire(ctx, session, "Buy now!")

	classify, _ := zyn.Classification("type", []string{"spam", "ham"}, provider)
	_, _ = classify.Fire(ctx, session, "Check category")

	sentiment, _ := zyn.Sentiment("Analyze tone", provider)
	_, _ = sentiment.Fire(ctx, session, "This is terrible")

	// All three synapse calls should accumulate in session
	if session.Len() != 6 {
		t.Errorf("expected 6 messages from 3 calls, got %d", session.Len())
	}
}
