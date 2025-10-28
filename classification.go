package zyn

import (
	"context"
	"fmt"

	"github.com/zoobzio/pipz"
)

// ClassificationInput contains rich input structure for classification.
type ClassificationInput struct {
	Subject     string              // The main item being classified
	Context     string              // Background information
	Examples    map[string][]string // Examples per category
	Temperature float32             // LLM temperature setting
}

// ClassificationResponse contains the response from a classification synapse.
type ClassificationResponse struct {
	Primary    string   `json:"primary"`    // Best matching category
	Secondary  string   `json:"secondary"`  // Optional second choice
	Confidence float64  `json:"confidence"` // Confidence in primary choice
	Reasoning  []string `json:"reasoning"`  // Explanation of classification
}

// Validate checks if the response is valid.
func (r ClassificationResponse) Validate() error {
	if r.Primary == "" {
		return fmt.Errorf("primary category required but empty")
	}
	if r.Confidence < 0 || r.Confidence > 1 {
		return fmt.Errorf("confidence must be 0-1, got %f", r.Confidence)
	}
	if len(r.Reasoning) == 0 {
		return fmt.Errorf("reasoning required but empty")
	}
	return nil
}

// ClassificationSynapse represents a multi-class classification synapse.
type ClassificationSynapse struct {
	question   string
	categories []string
	defaults   ClassificationInput
	service    *Service[ClassificationResponse]
}

// NewClassification creates a new classification synapse bound to a provider.
func NewClassification(question string, categories []string, provider Provider, opts ...Option) *ClassificationSynapse {
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
	svc := NewService[ClassificationResponse](pipeline, "classification", provider)

	return &ClassificationSynapse{
		question:   question,
		categories: categories,
		service:    svc,
	}
}

// GetPipeline returns the internal pipeline for composition.
func (c *ClassificationSynapse) GetPipeline() pipz.Chainable[*SynapseRequest] {
	return c.service.GetPipeline()
}

// WithDefaults creates a new Classification with default input values.
func (c *ClassificationSynapse) WithDefaults(defaults ClassificationInput) *ClassificationSynapse {
	c.defaults = defaults
	return c
}

// Fire executes the synapse against a simple string input.
// Returns only the primary category.
func (c *ClassificationSynapse) Fire(ctx context.Context, input string) (string, error) {
	response, err := c.FireWithDetails(ctx, input)
	if err != nil {
		return "", err
	}
	return response.Primary, nil
}

// FireWithDetails executes the synapse and returns the full response.
func (c *ClassificationSynapse) FireWithDetails(ctx context.Context, input string) (ClassificationResponse, error) {
	classInput := ClassificationInput{Subject: input}
	return c.FireWithInput(ctx, classInput)
}

// FireWithInput executes the synapse with rich input structure.
func (c *ClassificationSynapse) FireWithInput(ctx context.Context, input ClassificationInput) (ClassificationResponse, error) {
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
		temperature = 0.3 // Slightly higher than binary for more nuanced classification
	}

	// Execute through service (validation happens in Service.Execute)
	return c.service.Execute(ctx, prompt, temperature)
}

// mergeInputs combines defaults with user input.
func (c *ClassificationSynapse) mergeInputs(input ClassificationInput) ClassificationInput {
	merged := c.defaults

	if input.Subject != "" {
		merged.Subject = input.Subject
	}
	if input.Context != "" {
		merged.Context = input.Context
	}
	if len(input.Examples) > 0 {
		if merged.Examples == nil {
			merged.Examples = make(map[string][]string)
		}
		for cat, exs := range input.Examples {
			merged.Examples[cat] = append(merged.Examples[cat], exs...)
		}
	}
	if input.Temperature != 0 {
		merged.Temperature = input.Temperature
	}

	return merged
}

// buildPrompt constructs the prompt from the merged input.
func (c *ClassificationSynapse) buildPrompt(input ClassificationInput) *Prompt {
	prompt := &Prompt{
		Task:       c.question,
		Input:      input.Subject,
		Context:    input.Context,
		Categories: c.categories,
		Examples:   input.Examples,
	}

	// Build schema using sentinel
	prompt.Schema = generateJSONSchema[ClassificationResponse]()

	// Build constraints
	prompt.Constraints = []string{
		"primary: required, from categories list",
		"secondary: optional, from categories list or empty string",
		"confidence: 0.0 to 1.0",
		"reasoning: ordered steps explaining classification",
	}

	return prompt
}

// Classification creates a new classification synapse bound to a provider.
// The synapse categorizes inputs into one of the provided categories.
//
// Example:
//
//	synapse := Classification("What type of error?",
//	    []string{"network", "database", "auth", "validation"},
//	    provider,
//	    WithTimeout(10*time.Second),
//	)
//	category, err := synapse.Fire(ctx, "Connection refused on port 5432")
func Classification(question string, categories []string, provider Provider, opts ...Option) *ClassificationSynapse {
	return NewClassification(question, categories, provider, opts...)
}
