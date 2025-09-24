package zyn

import (
	"context"
	"fmt"

	"github.com/zoobzio/pipz"
)

// TransformInput contains rich input structure for transformation.
type TransformInput struct {
	Text        string            // The text to transform
	Context     string            // Optional context
	Style       string            // Optional style guidance
	Examples    map[string]string // Optional input->output examples
	MaxLength   int               // Optional maximum length
	Temperature float32           // Temperature for creativity
}

// TransformResponse contains the transformed output with metadata.
type TransformResponse struct {
	Output     string   `json:"output"`     // The transformed text
	Confidence float64  `json:"confidence"` // Confidence in transformation
	Changes    []string `json:"changes"`    // Key changes made
	Reasoning  []string `json:"reasoning"`  // Explanation of approach
}

// TransformSynapse transforms text according to specified instructions.
type TransformSynapse struct {
	instruction string // What transformation to perform
	defaults    TransformInput
	service     *Service[TransformResponse]
}

// Transform creates a new text transformation synapse.
func Transform(instruction string, provider Provider, opts ...Option) *TransformSynapse {
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
	svc := NewService[TransformResponse](pipeline)

	return &TransformSynapse{
		instruction: instruction,
		defaults: TransformInput{
			Temperature: 0.3, // Lower temperature for consistent transformations
		},
		service: svc,
	}
}

// GetPipeline returns the underlying pipeline.
func (t *TransformSynapse) GetPipeline() pipz.Chainable[*SynapseRequest] {
	return t.service.GetPipeline()
}

// Fire performs the transformation with a simple string input.
func (t *TransformSynapse) Fire(ctx context.Context, text string) (string, error) {
	input := TransformInput{Text: text}
	return t.FireWithInput(ctx, input)
}

// FireWithDetails performs the transformation and returns detailed response.
func (t *TransformSynapse) FireWithDetails(ctx context.Context, text string) (*TransformResponse, error) {
	input := TransformInput{Text: text}
	return t.FireWithInputDetails(ctx, input)
}

// FireWithInput performs the transformation with rich input.
func (t *TransformSynapse) FireWithInput(ctx context.Context, input TransformInput) (string, error) {
	response, err := t.FireWithInputDetails(ctx, input)
	if err != nil {
		return "", err
	}
	return response.Output, nil
}

// FireWithInputDetails performs the transformation and returns full details.
func (t *TransformSynapse) FireWithInputDetails(ctx context.Context, input TransformInput) (*TransformResponse, error) {
	// Merge defaults with user input
	merged := t.mergeInputs(input)

	// Build prompt
	prompt := t.buildPrompt(merged)

	// Determine temperature
	temperature := merged.Temperature
	if temperature == 0 && t.defaults.Temperature != 0 {
		temperature = t.defaults.Temperature
	}
	if temperature == 0 {
		temperature = 0.3
	}

	// Execute through service
	response, err := t.service.Execute(ctx, prompt, temperature)
	if err != nil {
		return nil, fmt.Errorf("transform failed: %w", err)
	}

	return &response, nil
}

// mergeInputs combines defaults with user input.
func (t *TransformSynapse) mergeInputs(input TransformInput) TransformInput {
	merged := t.defaults

	if input.Text != "" {
		merged.Text = input.Text
	}
	if input.Context != "" {
		merged.Context = input.Context
	}
	if input.Style != "" {
		merged.Style = input.Style
	}
	if len(input.Examples) > 0 {
		if merged.Examples == nil {
			merged.Examples = make(map[string]string)
		}
		for k, v := range input.Examples {
			merged.Examples[k] = v
		}
	}
	if input.MaxLength > 0 {
		merged.MaxLength = input.MaxLength
	}
	if input.Temperature != 0 {
		merged.Temperature = input.Temperature
	}

	return merged
}

// buildPrompt constructs the prompt from the merged input.
func (t *TransformSynapse) buildPrompt(input TransformInput) *Prompt {
	prompt := &Prompt{
		Task:    fmt.Sprintf("Transform: %s", t.instruction),
		Input:   input.Text,
		Context: input.Context,
	}

	// Add examples if provided
	if len(input.Examples) > 0 {
		examples := make(map[string][]string)
		for inputEx, outputEx := range input.Examples {
			examples["Input"] = append(examples["Input"], inputEx)
			examples["Output"] = append(examples["Output"], outputEx)
		}
		prompt.Examples = examples
	}

	// Build schema
	prompt.Schema = `{
  "output": "transformed text",
  "confidence": 0.0-1.0,
  "changes": ["change 1", "change 2"],
  "reasoning": ["step 1", "step 2"]
}`

	// Build constraints
	constraints := []string{
		"output: the transformed text",
		"confidence: 0.0 to 1.0",
		"changes: list of key transformations made",
		"reasoning: explanation of transformation approach",
	}

	if input.Style != "" {
		constraints = append(constraints, fmt.Sprintf("style: %s", input.Style))
	}

	if input.MaxLength > 0 {
		constraints = append(constraints, fmt.Sprintf("maximum length: %d characters", input.MaxLength))
	}

	prompt.Constraints = constraints

	return prompt
}
