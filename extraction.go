package zyn

import (
	"context"
	"fmt"

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
type ExtractionSynapse[T any] struct {
	what     string
	schema   string // Pre-computed JSON schema
	defaults ExtractionInput
	service  *Service[T]
}

// NewExtraction creates a new extraction synapse bound to a provider.
// The type parameter T defines the structure to extract.
func NewExtraction[T any](what string, provider Provider, opts ...Option) *ExtractionSynapse[T] {
	// Generate schema once at construction
	schema := generateJSONSchema[T]()

	// Create terminal processor that calls the provider
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
	svc := NewService[T](pipeline)

	return &ExtractionSynapse[T]{
		what:    what,
		schema:  schema,
		service: svc,
	}
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
func (e *ExtractionSynapse[T]) Fire(ctx context.Context, text string) (T, error) {
	input := ExtractionInput{Text: text}
	return e.FireWithInput(ctx, input)
}

// FireWithInput executes the extraction with rich input structure.
func (e *ExtractionSynapse[T]) FireWithInput(ctx context.Context, input ExtractionInput) (T, error) {
	// Merge defaults with user input
	merged := e.mergeInputs(input)

	// Build prompt
	prompt := e.buildPrompt(merged)

	// Determine temperature
	temperature := merged.Temperature
	if temperature == 0 && e.defaults.Temperature != 0 {
		temperature = e.defaults.Temperature
	}
	if temperature == 0 {
		temperature = 0.1 // Low temperature for consistent extraction
	}

	// Execute through service - it handles JSON unmarshaling to T
	return e.service.Execute(ctx, prompt, temperature)
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
	if input.Temperature != 0 {
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
		for _, line := range splitLines(input.Examples) {
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

// splitLines splits a string by newlines.
func splitLines(s string) []string {
	var lines []string
	current := ""
	for i := 0; i < len(s); i++ {
		if s[i] == '\n' {
			lines = append(lines, current)
			current = ""
		} else {
			current += string(s[i])
		}
	}
	if current != "" {
		lines = append(lines, current)
	}
	return lines
}

// Extract creates a new extraction synapse bound to a provider.
// The type parameter T defines the structure to extract.
//
// Example:
//
//	type Contact struct {
//	    Name  string `json:"name"`
//	    Email string `json:"email"`
//	}
//
//	extractor := Extract[Contact]("contact information", provider)
//	contact, err := extractor.Fire(ctx, "John Doe at john@example.com")
func Extract[T any](what string, provider Provider, opts ...Option) *ExtractionSynapse[T] {
	return NewExtraction[T](what, provider, opts...)
}
