package zyn

import (
	"context"
	"crypto/rand"
	"encoding/json"
	"fmt"

	"github.com/zoobzio/capitan"
	"github.com/zoobzio/pipz"
)

// Validator defines the interface for response validation.
// All response types must implement this to ensure LLM outputs are valid.
type Validator interface {
	Validate() error
}

// Service provides type-safe LLM interactions for a specific response type T.
// It wraps a pipz pipeline and handles JSON parsing of responses.
// T must implement Validator to ensure response validation.
type Service[T Validator] struct {
	pipeline     pipz.Chainable[*SynapseRequest]
	synapseType  string
	providerName string
}

// NewService creates a new Service with the given pipeline, synapse type, and provider.
func NewService[T Validator](pipeline pipz.Chainable[*SynapseRequest], synapseType string, provider Provider) *Service[T] {
	return &Service[T]{
		pipeline:     pipeline,
		synapseType:  synapseType,
		providerName: provider.Name(),
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

	// Generate unique request ID
	requestID := rand.Text()

	// Create request
	request := &SynapseRequest{
		Prompt:       prompt,
		Temperature:  temperature,
		RequestID:    requestID,
		SynapseType:  s.synapseType,
		ProviderName: s.providerName,
	}

	// Emit request.started hook
	capitan.Emit(ctx, RequestStarted,
		RequestIDKey.Field(requestID),
		SynapseTypeKey.Field(s.synapseType),
		ProviderKey.Field(s.providerName),
		PromptTaskKey.Field(prompt.Task),
		InputKey.Field(prompt.Input),
		TemperatureKey.Field(float64(temperature)),
	)

	// Process through pipeline
	processed, err := s.pipeline.Process(ctx, request)
	if err != nil {
		// Emit request.failed hook
		capitan.Emit(ctx, RequestFailed,
			RequestIDKey.Field(requestID),
			SynapseTypeKey.Field(s.synapseType),
			ProviderKey.Field(s.providerName),
			PromptTaskKey.Field(prompt.Task),
			ErrorKey.Field(err.Error()),
		)
		return result, err
	}

	// Parse response to type T
	if processed.Response == "" {
		return result, fmt.Errorf("no response from provider")
	}

	if parseErr := json.Unmarshal([]byte(processed.Response), &result); parseErr != nil {
		// Emit response.failed hook
		capitan.Emit(ctx, ResponseParseFailed,
			RequestIDKey.Field(requestID),
			SynapseTypeKey.Field(s.synapseType),
			ProviderKey.Field(s.providerName),
			PromptTaskKey.Field(prompt.Task),
			ResponseKey.Field(processed.Response),
			ErrorKey.Field(parseErr.Error()),
			ErrorTypeKey.Field("parse_error"),
		)
		return result, fmt.Errorf("failed to parse response: %w", parseErr)
	}

	// Validate response (T is constrained to Validator)
	if validationErr := result.Validate(); validationErr != nil {
		// Emit response.failed hook
		capitan.Emit(ctx, ResponseParseFailed,
			RequestIDKey.Field(requestID),
			SynapseTypeKey.Field(s.synapseType),
			ProviderKey.Field(s.providerName),
			PromptTaskKey.Field(prompt.Task),
			ResponseKey.Field(processed.Response),
			ErrorKey.Field(validationErr.Error()),
			ErrorTypeKey.Field("validation_error"),
		)
		return result, fmt.Errorf("invalid response: %w", validationErr)
	}

	// Marshal result to JSON for output field
	outputJSON, marshalErr := json.Marshal(result)
	if marshalErr != nil {
		// This should never fail since we already unmarshaled successfully
		outputJSON = []byte("{}")
	}

	// Emit request.completed hook
	capitan.Emit(ctx, RequestCompleted,
		RequestIDKey.Field(requestID),
		SynapseTypeKey.Field(s.synapseType),
		ProviderKey.Field(s.providerName),
		PromptTaskKey.Field(prompt.Task),
		InputKey.Field(prompt.Input),
		OutputKey.Field(string(outputJSON)),
		ResponseKey.Field(processed.Response),
	)

	return result, nil
}
