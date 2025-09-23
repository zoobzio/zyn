package zyn

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/zoobzio/pipz"
)

// TestServiceJSON tests JSON parsing functionality.
func TestServiceJSON(t *testing.T) {
	// Create a simple pipeline that returns JSON
	pipeline := pipz.Apply("test", func(ctx context.Context, req *SynapseRequest) (*SynapseRequest, error) {
		req.Response = `{"decision": true, "confidence": 0.9, "reasoning": ["test"]}`
		return req, nil
	})

	service := NewService[BinaryResponse](pipeline)

	ctx := context.Background()
	prompt := &Prompt{
		Task:  "test task",
		Input: "test input",
		Schema: `{"decision": true/false}`,
	}
	response, err := service.Execute(ctx, prompt, 0.5)
	if err != nil {
		t.Fatalf("Execute failed: %v", err)
	}

	if !response.Decision {
		t.Error("Expected decision to be true")
	}
	if response.Confidence != 0.9 {
		t.Errorf("Expected confidence 0.9, got %f", response.Confidence)
	}
	if len(response.Reasoning) != 1 || response.Reasoning[0] != "test" {
		t.Errorf("Expected reasoning ['test'], got %v", response.Reasoning)
	}
}

// TestServiceInvalidJSON tests handling of invalid JSON.
func TestServiceInvalidJSON(t *testing.T) {
	// Create a pipeline that returns invalid JSON
	pipeline := pipz.Apply("test", func(ctx context.Context, req *SynapseRequest) (*SynapseRequest, error) {
		req.Response = "not json"
		return req, nil
	})

	service := NewService[BinaryResponse](pipeline)

	ctx := context.Background()
	prompt := &Prompt{
		Task:  "test task",
		Input: "test input",
		Schema: `{"decision": true/false}`,
	}
	_, err := service.Execute(ctx, prompt, 0.5)
	if err == nil {
		t.Error("Expected error for invalid JSON")
	}
	// Just check that we got an error containing "invalid"
	if !strings.Contains(err.Error(), "invalid") && !strings.Contains(err.Error(), "cannot unmarshal") {
		t.Errorf("Expected JSON parse error, got: %v", err)
	}
}

// TestServicePipelineError tests error propagation from pipeline.
func TestServicePipelineError(t *testing.T) {
	// Create a pipeline that returns an error
	expectedErr := errors.New("pipeline error")
	pipeline := pipz.Apply("test", func(ctx context.Context, req *SynapseRequest) (*SynapseRequest, error) {
		return req, expectedErr
	})

	service := NewService[BinaryResponse](pipeline)

	ctx := context.Background()
	prompt := &Prompt{
		Task:  "test task",
		Input: "test input",
		Schema: `{"decision": true/false}`,
	}
	_, err := service.Execute(ctx, prompt, 0.5)
	if err == nil {
		t.Error("Expected error, got nil")
	}
	// pipz wraps errors, so check the message contains our error
	if !strings.Contains(err.Error(), "pipeline error") {
		t.Errorf("Expected error containing 'pipeline error', got %v", err)
	}
}

// TestServiceGetPipeline tests pipeline retrieval.
func TestServiceGetPipeline(t *testing.T) {
	pipeline := pipz.Apply("test", func(ctx context.Context, req *SynapseRequest) (*SynapseRequest, error) {
		return req, nil
	})

	service := NewService[BinaryResponse](pipeline)

	retrieved := service.GetPipeline()
	if retrieved == nil {
		t.Error("GetPipeline returned nil")
	}
	// We can't directly compare pipelines, but we can verify it's not nil
}

// TestServiceWithDifferentTypes tests Service with different response types.
func TestServiceWithDifferentTypes(t *testing.T) {
	// Test with ClassificationResponse
	classificationPipeline := pipz.Apply("test", func(ctx context.Context, req *SynapseRequest) (*SynapseRequest, error) {
		req.Response = `{"primary": "category1", "secondary": "category2", "confidence": 0.8, "reasoning": ["test"]}`
		return req, nil
	})

	classificationService := NewService[ClassificationResponse](classificationPipeline)
	ctx := context.Background()
	
	classPrompt := &Prompt{
		Task:  "classify",
		Input: "test",
		Schema: `{}`,
	}
	classResponse, err := classificationService.Execute(ctx, classPrompt, 0.5)
	if err != nil {
		t.Fatalf("Execute failed for ClassificationResponse: %v", err)
	}
	if classResponse.Primary != "category1" {
		t.Errorf("Expected primary 'category1', got %s", classResponse.Primary)
	}

	// Test with RankingResponse
	rankingPipeline := pipz.Apply("test", func(ctx context.Context, req *SynapseRequest) (*SynapseRequest, error) {
		req.Response = `{"ranked": ["item1", "item2"], "confidence": 0.7, "reasoning": ["test"]}`
		return req, nil
	})

	rankingService := NewService[RankingResponse](rankingPipeline)
	
	rankPrompt := &Prompt{
		Task:  "rank",
		Input: "test",
		Schema: `{}`,
	}
	rankResponse, err := rankingService.Execute(ctx, rankPrompt, 0.5)
	if err != nil {
		t.Fatalf("Execute failed for RankingResponse: %v", err)
	}
	if len(rankResponse.Ranked) != 2 || rankResponse.Ranked[0] != "item1" {
		t.Errorf("Expected ranked ['item1', 'item2'], got %v", rankResponse.Ranked)
	}
}