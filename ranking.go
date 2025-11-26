package zyn

import (
	"context"
	"fmt"

	"github.com/zoobzio/pipz"
)

// RankingInput contains rich input structure for ranking.
type RankingInput struct {
	Items       []string // The items to rank
	Context     string   // Additional context for ranking
	Examples    []string // Example rankings to guide
	TopN        int      // If set, only return top N items
	Temperature float32  // LLM temperature setting
}

// RankingResponse contains the response from a ranking synapse.
type RankingResponse struct {
	Ranked     []string `json:"ranked"`     // Items in ranked order
	Confidence float64  `json:"confidence"` // Overall confidence
	Reasoning  []string `json:"reasoning"`  // Explanation of ranking
}

// Validate checks if the response is valid.
func (r RankingResponse) Validate() error {
	if len(r.Ranked) == 0 {
		return fmt.Errorf("ranked list required but empty")
	}
	if r.Confidence < 0 || r.Confidence > 1 {
		return fmt.Errorf("confidence must be 0-1, got %f", r.Confidence)
	}
	if len(r.Reasoning) == 0 {
		return fmt.Errorf("reasoning required but empty")
	}
	return nil
}

// RankingSynapse represents a ranking/sorting synapse.
type RankingSynapse struct {
	criteria string
	schema   string // Pre-computed JSON schema
	defaults RankingInput
	service  *Service[RankingResponse]
}

// NewRanking creates a new ranking synapse bound to a provider.
// Returns an error if the JSON schema cannot be generated.
func NewRanking(criteria string, provider Provider, opts ...Option) (*RankingSynapse, error) {
	// Generate schema once at construction
	schema, err := generateJSONSchema[RankingResponse]()
	if err != nil {
		return nil, fmt.Errorf("ranking synapse: %w", err)
	}

	// Apply options to build pipeline
	var pipeline pipz.Chainable[*SynapseRequest] = NewTerminal(provider)
	for _, opt := range opts {
		pipeline = opt(pipeline)
	}

	// Create service with final pipeline and default temperature
	svc := NewService[RankingResponse](pipeline, "ranking", provider, DefaultTemperatureAnalytical)

	return &RankingSynapse{
		criteria: criteria,
		schema:   schema,
		service:  svc,
	}, nil
}

// GetPipeline returns the internal pipeline for composition.
func (r *RankingSynapse) GetPipeline() pipz.Chainable[*SynapseRequest] {
	return r.service.GetPipeline()
}

// WithDefaults creates a new Ranking with default input values.
func (r *RankingSynapse) WithDefaults(defaults RankingInput) *RankingSynapse {
	r.defaults = defaults
	return r
}

// Fire executes the ranking against a list of items.
// Returns the items in ranked order.
func (r *RankingSynapse) Fire(ctx context.Context, session *Session, items []string) ([]string, error) {
	response, err := r.FireWithDetails(ctx, session, items)
	if err != nil {
		return nil, err
	}
	return response.Ranked, nil
}

// FireWithDetails executes the ranking and returns the full response.
func (r *RankingSynapse) FireWithDetails(ctx context.Context, session *Session, items []string) (RankingResponse, error) {
	rankInput := RankingInput{Items: items}
	return r.FireWithInput(ctx, session, rankInput)
}

// FireWithInput executes the ranking with rich input structure.
func (r *RankingSynapse) FireWithInput(ctx context.Context, session *Session, input RankingInput) (RankingResponse, error) {
	// Merge defaults with user input
	merged := r.mergeInputs(input)

	// Build prompt
	prompt := r.buildPrompt(merged)

	// Execute through service with session (service handles temperature fallback)
	return r.service.Execute(ctx, session, prompt, merged.Temperature)
}

// mergeInputs combines defaults with user input.
func (r *RankingSynapse) mergeInputs(input RankingInput) RankingInput {
	merged := r.defaults

	if len(input.Items) > 0 {
		merged.Items = input.Items
	}
	if input.Context != "" {
		merged.Context = input.Context
	}
	if len(input.Examples) > 0 {
		merged.Examples = append(merged.Examples, input.Examples...)
	}
	if input.TopN > 0 {
		merged.TopN = input.TopN
	}
	if input.Temperature != 0 && input.Temperature != TemperatureUnset {
		merged.Temperature = input.Temperature
	}

	return merged
}

// buildPrompt constructs the prompt from the merged input.
func (r *RankingSynapse) buildPrompt(input RankingInput) *Prompt {
	prompt := &Prompt{
		Task:    fmt.Sprintf("Rank by %s", r.criteria),
		Items:   input.Items,
		Context: input.Context,
		Schema:  r.schema,
	}

	// Add examples if provided
	if len(input.Examples) > 0 {
		prompt.Examples = map[string][]string{
			"rankings": input.Examples,
		}
	}

	// Build constraints
	if input.TopN > 0 {
		prompt.Constraints = []string{
			fmt.Sprintf("ranked: select top %d items only", input.TopN),
			"ranked: ordered highest to lowest",
			"ranked: preserve exact item text",
			"confidence: 0.0 to 1.0",
		}
	} else {
		prompt.Constraints = []string{
			"ranked: all items, ordered highest to lowest",
			"ranked: include every item exactly once",
			"ranked: preserve exact item text",
			"confidence: 0.0 to 1.0",
		}
	}

	return prompt
}

// Ranking creates a new ranking synapse bound to a provider.
// The synapse orders items based on the specified criteria.
// Returns an error if the JSON schema cannot be generated.
//
// Example:
//
//	synapse, err := Ranking("urgency and impact",
//	    provider,
//	    WithTimeout(10*time.Second),
//	)
//	ordered, err := synapse.Fire(ctx, []string{"Fix typo", "Security patch", "Add feature"})
func Ranking(criteria string, provider Provider, opts ...Option) (*RankingSynapse, error) {
	return NewRanking(criteria, provider, opts...)
}
