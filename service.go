package zyn

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/zoobzio/pipz"
)

// Service provides type-safe LLM interactions for a specific response type T.
// It wraps a pipz pipeline and handles JSON parsing of responses.
type Service[T any] struct {
	pipeline pipz.Chainable[*SynapseRequest]
}

// NewService creates a new Service with the given pipeline.
func NewService[T any](pipeline pipz.Chainable[*SynapseRequest]) *Service[T] {
	return &Service[T]{
		pipeline: pipeline,
	}
}

// GetPipeline returns the internal pipeline for composition.
// This is used by WithFallback to combine pipelines.
func (s *Service[T]) GetPipeline() pipz.Chainable[*SynapseRequest] {
	return s.pipeline
}

// Execute processes a prompt through the pipeline and returns a typed response.
// It creates a SynapseRequest, runs it through the pipeline, and parses the result.
func (s *Service[T]) Execute(ctx context.Context, prompt *Prompt, temperature float32) (T, error) {
	var result T

	// Validate prompt
	if err := prompt.Validate(); err != nil {
		return result, fmt.Errorf("invalid prompt: %w", err)
	}

	// Create request
	request := &SynapseRequest{
		Prompt:      prompt,
		Temperature: temperature,
	}

	// Process through pipeline
	processed, err := s.pipeline.Process(ctx, request)
	if err != nil {
		return result, err
	}

	// Parse response to type T
	if processed.Response == "" {
		return result, fmt.Errorf("no response from provider")
	}

	if err := json.Unmarshal([]byte(processed.Response), &result); err != nil {
		return result, fmt.Errorf("failed to parse response: %w", err)
	}

	return result, nil
}
