package zyn

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/zoobzio/pipz"
)

// AnalyzeInput contains rich input structure for analysis.
type AnalyzeInput[T any] struct {
	Data        T       // The structured data to analyze
	Context     string  // Optional context for analysis
	Focus       string  // Optional specific aspect to focus on
	Temperature float32 // Temperature for analysis
}

// AnalyzeResponse contains the analysis with metadata.
type AnalyzeResponse struct {
	Analysis   string   `json:"analysis"`   // The main analysis text
	Confidence float64  `json:"confidence"` // Confidence in analysis
	Findings   []string `json:"findings"`   // Key findings or issues
	Reasoning  []string `json:"reasoning"`  // Explanation of analysis approach
}

// Validate checks if the response is valid.
func (r AnalyzeResponse) Validate() error {
	if r.Analysis == "" {
		return fmt.Errorf("analysis required but empty")
	}
	if r.Confidence < 0 || r.Confidence > 1 {
		return fmt.Errorf("confidence must be 0-1, got %f", r.Confidence)
	}
	return nil
}

// AnalyzeSynapse analyzes structured data and produces text analysis.
type AnalyzeSynapse[T any] struct {
	what     string // What kind of analysis to perform
	schema   string // Pre-computed JSON schema
	defaults AnalyzeInput[T]
	service  *Service[AnalyzeResponse]
}

// Analyze creates a new analysis synapse for structured input.
// Returns an error if the JSON schema cannot be generated.
func Analyze[T any](what string, provider Provider, opts ...Option) (*AnalyzeSynapse[T], error) {
	// Generate schema once at construction
	schema, err := generateJSONSchema[AnalyzeResponse]()
	if err != nil {
		return nil, fmt.Errorf("analyze synapse: %w", err)
	}

	// Apply options to build pipeline
	var pipeline pipz.Chainable[*SynapseRequest] = NewTerminal(provider)
	for _, opt := range opts {
		pipeline = opt(pipeline)
	}

	// Create service with final pipeline and default temperature
	svc := NewService[AnalyzeResponse](pipeline, "analyze", provider, DefaultTemperatureAnalytical)

	return &AnalyzeSynapse[T]{
		what:    what,
		schema:  schema,
		service: svc,
	}, nil
}

// GetPipeline returns the underlying pipeline.
func (a *AnalyzeSynapse[T]) GetPipeline() pipz.Chainable[*SynapseRequest] {
	return a.service.GetPipeline()
}

// Fire performs the analysis with structured input.
func (a *AnalyzeSynapse[T]) Fire(ctx context.Context, session *Session, data T) (string, error) {
	input := AnalyzeInput[T]{Data: data}
	return a.FireWithInput(ctx, session, input)
}

// FireWithDetails performs the analysis and returns detailed response.
func (a *AnalyzeSynapse[T]) FireWithDetails(ctx context.Context, session *Session, data T) (*AnalyzeResponse, error) {
	input := AnalyzeInput[T]{Data: data}
	return a.FireWithInputDetails(ctx, session, input)
}

// FireWithInput performs the analysis with rich input.
func (a *AnalyzeSynapse[T]) FireWithInput(ctx context.Context, session *Session, input AnalyzeInput[T]) (string, error) {
	response, err := a.FireWithInputDetails(ctx, session, input)
	if err != nil {
		return "", err
	}
	return response.Analysis, nil
}

// FireWithInputDetails performs the analysis and returns full details.
func (a *AnalyzeSynapse[T]) FireWithInputDetails(ctx context.Context, session *Session, input AnalyzeInput[T]) (*AnalyzeResponse, error) {
	// Merge defaults with user input
	merged := a.mergeInputs(input)

	// Build prompt
	prompt := a.buildPrompt(merged)

	// Execute through service with session (service handles temperature fallback)
	response, err := a.service.Execute(ctx, session, prompt, merged.Temperature)
	if err != nil {
		return nil, fmt.Errorf("analysis failed: %w", err)
	}

	return &response, nil
}

// mergeInputs combines defaults with user input.
func (a *AnalyzeSynapse[T]) mergeInputs(input AnalyzeInput[T]) AnalyzeInput[T] {
	merged := a.defaults

	// Data is always taken from input
	merged.Data = input.Data

	if input.Context != "" {
		merged.Context = input.Context
	}
	if input.Focus != "" {
		merged.Focus = input.Focus
	}
	if input.Temperature != 0 && input.Temperature != TemperatureUnset {
		merged.Temperature = input.Temperature
	}

	return merged
}

// buildPrompt constructs the prompt from the merged input.
func (a *AnalyzeSynapse[T]) buildPrompt(input AnalyzeInput[T]) *Prompt {
	// Convert the structured data to JSON string for the prompt
	dataJSON, err := json.MarshalIndent(input.Data, "", "  ")
	if err != nil {
		// Fallback to simple string representation
		dataJSON = []byte(fmt.Sprintf("%+v", input.Data))
	}

	prompt := &Prompt{
		Task:    fmt.Sprintf("Analyze: %s", a.what),
		Input:   string(dataJSON),
		Context: input.Context,
		Schema:  a.schema,
	}

	// Build constraints
	constraints := []string{
		"analysis: comprehensive text analysis of the input data",
		"confidence: 0.0 to 1.0",
		"findings: list of key findings or issues discovered",
		"reasoning: explanation of analysis methodology",
	}

	if input.Focus != "" {
		constraints = append(constraints, fmt.Sprintf("focus: %s", input.Focus))
	}

	prompt.Constraints = constraints

	return prompt
}
