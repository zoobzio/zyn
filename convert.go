package zyn

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/zoobzio/pipz"
)

// ConvertInput contains rich input structure for conversion.
type ConvertInput[T any] struct {
	Data        T       // The structured data to convert
	Context     string  // Optional context for conversion
	Rules       string  // Optional conversion rules or mappings
	Temperature float32 // Temperature for conversion
}

// ConvertSynapse converts structured data from one type to another.
// TOutput must implement Validator to ensure converted data is valid.
type ConvertSynapse[TInput any, TOutput Validator] struct {
	instruction  string // What conversion to perform
	outputSchema string // Pre-computed JSON schema for output type
	defaults     ConvertInput[TInput]
	service      *Service[TOutput]
}

// Convert creates a new struct-to-struct conversion synapse.
// TOutput must implement Validator to ensure converted data is valid.
func Convert[TInput any, TOutput Validator](instruction string, provider Provider, opts ...Option) *ConvertSynapse[TInput, TOutput] {
	// Pre-compute the output schema once at construction
	outputSchema := generateJSONSchema[TOutput]()

	// Create terminal pipeline stage that calls the provider
	terminal := pipz.Apply("llm-call", func(ctx context.Context, req *SynapseRequest) (*SynapseRequest, error) {
		// Render prompt to string for provider
		promptStr := req.Prompt.Render()
		response, err := provider.Call(ctx, promptStr, req.Temperature)
		if err != nil {
			return req, err
		}
		req.Response = response
		return req, nil
	})

	// Apply options to build pipeline
	var pipeline pipz.Chainable[*SynapseRequest] = terminal
	for _, opt := range opts {
		pipeline = opt(pipeline)
	}

	// Create service with final pipeline
	svc := NewService[TOutput](pipeline, "convert", provider)

	return &ConvertSynapse[TInput, TOutput]{
		instruction:  instruction,
		outputSchema: outputSchema,
		defaults: ConvertInput[TInput]{
			Temperature: 0.1, // Very low temperature for consistent conversion
		},
		service: svc,
	}
}

// GetPipeline returns the underlying pipeline.
func (c *ConvertSynapse[TInput, TOutput]) GetPipeline() pipz.Chainable[*SynapseRequest] {
	return c.service.GetPipeline()
}

// Fire performs the conversion with structured input.
func (c *ConvertSynapse[TInput, TOutput]) Fire(ctx context.Context, data TInput) (TOutput, error) {
	input := ConvertInput[TInput]{Data: data}
	return c.FireWithInput(ctx, input)
}

// FireWithInput performs the conversion with rich input.
func (c *ConvertSynapse[TInput, TOutput]) FireWithInput(ctx context.Context, input ConvertInput[TInput]) (TOutput, error) {
	// Merge defaults with user input
	merged := c.mergeInputs(input)

	// Build prompt
	prompt := c.buildPrompt(merged)

	// Determine temperature
	temperature := merged.Temperature
	if temperature == 0 && c.defaults.Temperature != 0 {
		temperature = c.defaults.Temperature
	}
	if temperature == 0 {
		temperature = 0.1
	}

	// Execute through service
	result, err := c.service.Execute(ctx, prompt, temperature)
	if err != nil {
		var zero TOutput
		return zero, fmt.Errorf("conversion failed: %w", err)
	}

	return result, nil
}

// mergeInputs combines defaults with user input.
func (c *ConvertSynapse[TInput, TOutput]) mergeInputs(input ConvertInput[TInput]) ConvertInput[TInput] {
	merged := c.defaults

	// Data is always taken from input
	merged.Data = input.Data

	if input.Context != "" {
		merged.Context = input.Context
	}
	if input.Rules != "" {
		merged.Rules = input.Rules
	}
	if input.Temperature != 0 {
		merged.Temperature = input.Temperature
	}

	return merged
}

// buildPrompt constructs the prompt from the merged input.
func (c *ConvertSynapse[TInput, TOutput]) buildPrompt(input ConvertInput[TInput]) *Prompt {
	// Convert the input data to JSON string
	inputJSON, err := json.MarshalIndent(input.Data, "", "  ")
	if err != nil {
		// Fallback to simple string representation
		inputJSON = []byte(fmt.Sprintf("%+v", input.Data))
	}

	prompt := &Prompt{
		Task:    fmt.Sprintf("Convert: %s", c.instruction),
		Input:   string(inputJSON),
		Context: input.Context,
	}

	// Use pre-computed output schema
	prompt.Schema = c.outputSchema

	// Build constraints
	constraints := []string{
		"Convert input data to match the exact output schema",
		"Preserve all relevant information during conversion",
		"Apply the specified transformation rules",
		"Ensure output is valid JSON matching the schema",
	}

	if input.Rules != "" {
		constraints = append(constraints, fmt.Sprintf("Conversion rules: %s", input.Rules))
	}

	prompt.Constraints = constraints

	return prompt
}
