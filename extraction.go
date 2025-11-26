package zyn

import (
	"context"
	"fmt"
	"strings"

	"github.com/zoobzio/pipz"
)

// ExtractionInput contains rich input structure for extraction.
type ExtractionInput struct {
	Text        string  // The text to extract from
	Context     string  // Additional context
	Examples    string  // Example extractions
	Temperature float32 // LLM temperature setting
}

// ExtractionSynapse represents a generic extraction synapse.
// It extracts structured data of type T from unstructured text.
// T must implement Validator to ensure extracted data is valid.
type ExtractionSynapse[T Validator] struct {
	what     string
	schema   string // Pre-computed JSON schema
	defaults ExtractionInput
	service  *Service[T]
}

// NewExtraction creates a new extraction synapse bound to a provider.
// The type parameter T defines the structure to extract and must implement Validator.
// Returns an error if the JSON schema cannot be generated.
func NewExtraction[T Validator](what string, provider Provider, opts ...Option) (*ExtractionSynapse[T], error) {
	// Generate schema once at construction
	schema, err := generateJSONSchema[T]()
	if err != nil {
		return nil, fmt.Errorf("extraction synapse: %w", err)
	}

	// Apply options to build pipeline
	var pipeline pipz.Chainable[*SynapseRequest] = NewTerminal(provider)
	for _, opt := range opts {
		pipeline = opt(pipeline)
	}

	// Create service with final pipeline and default temperature
	svc := NewService[T](pipeline, "extraction", provider, DefaultTemperatureDeterministic)

	return &ExtractionSynapse[T]{
		what:    what,
		schema:  schema,
		service: svc,
	}, nil
}

// GetPipeline returns the internal pipeline for composition.
func (e *ExtractionSynapse[T]) GetPipeline() pipz.Chainable[*SynapseRequest] {
	return e.service.GetPipeline()
}

// WithDefaults creates a new Extraction with default input values.
func (e *ExtractionSynapse[T]) WithDefaults(defaults ExtractionInput) *ExtractionSynapse[T] {
	e.defaults = defaults
	return e
}

// Fire executes the extraction against text.
func (e *ExtractionSynapse[T]) Fire(ctx context.Context, session *Session, text string) (T, error) {
	input := ExtractionInput{Text: text}
	return e.FireWithInput(ctx, session, input)
}

// FireWithInput executes the extraction with rich input structure.
func (e *ExtractionSynapse[T]) FireWithInput(ctx context.Context, session *Session, input ExtractionInput) (T, error) {
	// Merge defaults with user input
	merged := e.mergeInputs(input)

	// Build prompt
	prompt := e.buildPrompt(merged)

	// Execute through service with session (service handles temperature fallback)
	return e.service.Execute(ctx, session, prompt, merged.Temperature)
}

// mergeInputs combines defaults with user input.
func (e *ExtractionSynapse[T]) mergeInputs(input ExtractionInput) ExtractionInput {
	merged := e.defaults

	if input.Text != "" {
		merged.Text = input.Text
	}
	if input.Context != "" {
		merged.Context = input.Context
	}
	if input.Examples != "" {
		merged.Examples = input.Examples
	}
	if input.Temperature != 0 && input.Temperature != TemperatureUnset {
		merged.Temperature = input.Temperature
	}

	return merged
}

// buildPrompt constructs the prompt from the merged input.
func (e *ExtractionSynapse[T]) buildPrompt(input ExtractionInput) *Prompt {
	prompt := &Prompt{
		Task:    fmt.Sprintf("Extract %s", e.what),
		Input:   input.Text,
		Context: input.Context,
		Schema:  e.schema,
	}

	// Add examples if provided
	if input.Examples != "" {
		// Split examples by newline
		lines := []string{}
		for _, line := range strings.Split(input.Examples, "\n") {
			if line != "" {
				lines = append(lines, line)
			}
		}
		if len(lines) > 0 {
			prompt.Examples = map[string][]string{
				"examples": lines,
			}
		}
	}

	// Build constraints
	prompt.Constraints = []string{
		fmt.Sprintf("extract only %s", e.what),
		"use null for missing values",
		"match exact JSON structure",
	}

	return prompt
}

// Extract creates a new extraction synapse bound to a provider.
// The type parameter T defines the structure to extract and must implement Validator.
// Returns an error if the JSON schema cannot be generated.
//
// Example:
//
//	type Contact struct {
//	    Name  string `json:"name"`
//	    Email string `json:"email"`
//	}
//
//	func (c Contact) Validate() error {
//	    if c.Email == "" {
//	        return fmt.Errorf("email required")
//	    }
//	    return nil
//	}
//
//	extractor, err := Extract[Contact]("contact information", provider)
//	contact, err := extractor.Fire(ctx, "John Doe at john@example.com")
func Extract[T Validator](what string, provider Provider, opts ...Option) (*ExtractionSynapse[T], error) {
	return NewExtraction[T](what, provider, opts...)
}
