package zyn

import (
	"context"
	"fmt"

	"github.com/zoobzio/pipz"
)

// BinaryInput contains rich input structure for binary decisions.
type BinaryInput struct {
	Subject     string   // The main item being evaluated
	Context     string   // Background information or situation
	Criteria    []string // Specific criteria for evaluation
	Examples    []string // Examples of positive/negative cases
	Constraints []string // Limitations or requirements
	Temperature float32  // LLM temperature setting for this specific request
}

// BinaryResponse contains the response from a binary synapse.
type BinaryResponse struct {
	Decision   bool     `json:"decision"`   // Binary yes/no result
	Confidence float64  `json:"confidence"` // 0.0 to 1.0 confidence score
	Reasoning  []string `json:"reasoning"`  // Explanation of decision
}

// Validate checks if the response is valid.
func (r BinaryResponse) Validate() error {
	if r.Confidence < 0 || r.Confidence > 1 {
		return fmt.Errorf("confidence must be 0-1, got %f", r.Confidence)
	}
	if len(r.Reasoning) == 0 {
		return fmt.Errorf("reasoning required but empty")
	}
	return nil
}

// BinarySynapse represents a binary (yes/no) decision synapse.
type BinarySynapse struct {
	question string
	schema   string // Pre-computed JSON schema
	defaults BinaryInput
	service  *Service[BinaryResponse]
}

// NewBinary creates a new binary synapse bound to a provider.
// The synapse is immediately usable and can be enhanced with options.
// Returns an error if the JSON schema cannot be generated.
func NewBinary(question string, provider Provider, opts ...Option) (*BinarySynapse, error) {
	// Generate schema once at construction
	schema, err := generateJSONSchema[BinaryResponse]()
	if err != nil {
		return nil, fmt.Errorf("binary synapse: %w", err)
	}

	// Apply options to build pipeline
	pipeline := NewTerminal(provider)
	for _, opt := range opts {
		pipeline = opt(pipeline)
	}

	// Create service with final pipeline and default temperature
	svc := NewService[BinaryResponse](pipeline, "binary", provider, DefaultTemperatureDeterministic)

	return &BinarySynapse{
		question: question,
		schema:   schema,
		service:  svc,
	}, nil
}

// GetPipeline returns the internal pipeline for composition.
// Implements ServiceProvider interface.
func (b *BinarySynapse) GetPipeline() pipz.Chainable[*SynapseRequest] {
	return b.service.GetPipeline()
}

// WithDefaults creates a new Binary with default input values.
// These are merged with user input at execution time.
func (b *BinarySynapse) WithDefaults(defaults BinaryInput) *BinarySynapse {
	b.defaults = defaults
	return b
}

// Fire executes the synapse against a simple string input.
// Returns only the boolean decision.
func (b *BinarySynapse) Fire(ctx context.Context, session *Session, input string) (bool, error) {
	response, err := b.FireWithDetails(ctx, session, input)
	if err != nil {
		return false, err
	}
	return response.Decision, nil
}

// FireWithDetails executes the synapse and returns the full response.
func (b *BinarySynapse) FireWithDetails(ctx context.Context, session *Session, input string) (BinaryResponse, error) {
	binInput := BinaryInput{Subject: input}
	return b.FireWithInput(ctx, session, binInput)
}

// FireWithInput executes the synapse with rich input structure.
func (b *BinarySynapse) FireWithInput(ctx context.Context, session *Session, input BinaryInput) (BinaryResponse, error) {
	// Merge defaults with user input
	merged := b.mergeInputs(input)

	// Build prompt
	prompt := b.buildPrompt(merged)

	// Execute through service with session (service handles temperature fallback)
	return b.service.Execute(ctx, session, prompt, merged.Temperature)
}

// mergeInputs combines defaults with user input.
func (b *BinarySynapse) mergeInputs(input BinaryInput) BinaryInput {
	merged := b.defaults

	if input.Subject != "" {
		merged.Subject = input.Subject
	}
	if input.Context != "" {
		merged.Context = input.Context
	}
	if len(input.Criteria) > 0 {
		merged.Criteria = append(merged.Criteria, input.Criteria...)
	}
	if len(input.Examples) > 0 {
		merged.Examples = append(merged.Examples, input.Examples...)
	}
	if len(input.Constraints) > 0 {
		merged.Constraints = append(merged.Constraints, input.Constraints...)
	}
	if input.Temperature != 0 && input.Temperature != TemperatureUnset {
		merged.Temperature = input.Temperature
	}

	return merged
}

// buildPrompt constructs the prompt from the merged input.
func (b *BinarySynapse) buildPrompt(input BinaryInput) *Prompt {
	prompt := &Prompt{
		Task:    fmt.Sprintf("Determine if %s", b.question),
		Input:   input.Subject,
		Context: input.Context,
		Schema:  b.schema,
	}

	// Build constraints
	prompt.Constraints = []string{
		"decision: true or false only",
		"confidence: 0.0 to 1.0",
		"reasoning: ordered steps explaining decision",
	}

	// Add criteria as constraints if provided
	for _, c := range input.Criteria {
		prompt.Constraints = append(prompt.Constraints, "evaluate: "+c)
	}

	// Add input constraints if provided
	prompt.Constraints = append(prompt.Constraints, input.Constraints...)

	// Add examples if provided
	if len(input.Examples) > 0 {
		prompt.Examples = map[string][]string{
			"examples": input.Examples,
		}
	}

	return prompt
}

// Binary creates a new binary synapse bound to a provider.
// The synapse is immediately usable and can be enhanced with options.
// Returns an error if the JSON schema cannot be generated.
//
// Example:
//
//	synapse, err := Binary("Is this valid?", provider,
//	    WithRetry(3),
//	    WithTimeout(10*time.Second),
//	)
//	result, err := synapse.Fire(ctx, "test@example.com")
func Binary(question string, provider Provider, opts ...Option) (*BinarySynapse, error) {
	return NewBinary(question, provider, opts...)
}
