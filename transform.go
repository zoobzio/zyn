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

// Validate checks if the response is valid.
func (r TransformResponse) Validate() error {
	if r.Output == "" {
		return fmt.Errorf("output required but empty")
	}
	if r.Confidence < 0 || r.Confidence > 1 {
		return fmt.Errorf("confidence must be 0-1, got %f", r.Confidence)
	}
	return nil
}

// TransformSynapse transforms text according to specified instructions.
type TransformSynapse struct {
	instruction string // What transformation to perform
	schema      string // Pre-computed JSON schema
	defaults    TransformInput
	service     *Service[TransformResponse]
}

// Transform creates a new text transformation synapse.
// Returns an error if the JSON schema cannot be generated.
func Transform(instruction string, provider Provider, opts ...Option) (*TransformSynapse, error) {
	// Generate schema once at construction
	schema, err := generateJSONSchema[TransformResponse]()
	if err != nil {
		return nil, fmt.Errorf("transform synapse: %w", err)
	}

	// Apply options to build pipeline
	var pipeline pipz.Chainable[*SynapseRequest] = NewTerminal(provider)
	for _, opt := range opts {
		pipeline = opt(pipeline)
	}

	// Create service with final pipeline and default temperature
	svc := NewService[TransformResponse](pipeline, "transform", provider, DefaultTemperatureCreative)

	return &TransformSynapse{
		instruction: instruction,
		schema:      schema,
		service:     svc,
	}, nil
}

// GetPipeline returns the underlying pipeline.
func (t *TransformSynapse) GetPipeline() pipz.Chainable[*SynapseRequest] {
	return t.service.GetPipeline()
}

// Fire performs the transformation with a simple string input.
func (t *TransformSynapse) Fire(ctx context.Context, session *Session, text string) (string, error) {
	input := TransformInput{Text: text}
	return t.FireWithInput(ctx, session, input)
}

// FireWithDetails performs the transformation and returns detailed response.
func (t *TransformSynapse) FireWithDetails(ctx context.Context, session *Session, text string) (*TransformResponse, error) {
	input := TransformInput{Text: text}
	return t.FireWithInputDetails(ctx, session, input)
}

// FireWithInput performs the transformation with rich input.
func (t *TransformSynapse) FireWithInput(ctx context.Context, session *Session, input TransformInput) (string, error) {
	response, err := t.FireWithInputDetails(ctx, session, input)
	if err != nil {
		return "", err
	}
	return response.Output, nil
}

// FireWithInputDetails performs the transformation and returns full details.
func (t *TransformSynapse) FireWithInputDetails(ctx context.Context, session *Session, input TransformInput) (*TransformResponse, error) {
	// Merge defaults with user input
	merged := t.mergeInputs(input)

	// Build prompt
	prompt := t.buildPrompt(merged)

	// Execute through service with session (service handles temperature fallback)
	response, err := t.service.Execute(ctx, session, prompt, merged.Temperature)
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
	if input.Temperature != 0 && input.Temperature != TemperatureUnset {
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

	prompt.Schema = t.schema

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
