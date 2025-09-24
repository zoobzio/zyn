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

// RankingSynapse represents a ranking/sorting synapse.
type RankingSynapse struct {
	criteria string
	defaults RankingInput
	service  *Service[RankingResponse]
}

// NewRanking creates a new ranking synapse bound to a provider.
func NewRanking(criteria string, provider Provider, opts ...Option) *RankingSynapse {
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
	svc := NewService[RankingResponse](pipeline)

	return &RankingSynapse{
		criteria: criteria,
		service:  svc,
	}
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
func (r *RankingSynapse) Fire(ctx context.Context, items []string) ([]string, error) {
	response, err := r.FireWithDetails(ctx, items)
	if err != nil {
		return nil, err
	}
	return response.Ranked, nil
}

// FireWithDetails executes the ranking and returns the full response.
func (r *RankingSynapse) FireWithDetails(ctx context.Context, items []string) (RankingResponse, error) {
	rankInput := RankingInput{Items: items}
	return r.FireWithInput(ctx, rankInput)
}

// FireWithInput executes the ranking with rich input structure.
func (r *RankingSynapse) FireWithInput(ctx context.Context, input RankingInput) (RankingResponse, error) {
	// Merge defaults with user input
	merged := r.mergeInputs(input)

	// Build prompt
	prompt := r.buildPrompt(merged)

	// Determine temperature
	temperature := merged.Temperature
	if temperature == 0 && r.defaults.Temperature != 0 {
		temperature = r.defaults.Temperature
	}
	if temperature == 0 {
		temperature = 0.2 // Low temperature for consistent ranking
	}

	// Execute through service
	response, err := r.service.Execute(ctx, prompt, temperature)
	if err != nil {
		return response, err
	}

	// Validate response
	if err := r.validateResponse(response, merged); err != nil {
		return response, fmt.Errorf("ranking validation failed: %w", err)
	}

	return response, nil
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
	if input.Temperature != 0 {
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
	}

	// Add examples if provided
	if len(input.Examples) > 0 {
		prompt.Examples = map[string][]string{
			"rankings": input.Examples,
		}
	}

	// Build schema
	if input.TopN > 0 {
		prompt.Schema = fmt.Sprintf(`{
  "ranked": [%d items],
  "confidence": 0.0-1.0,
  "reasoning": ["step 1", "step 2", "step 3"]
}`, input.TopN)
	} else {
		prompt.Schema = `{
  "ranked": ["item1", "item2", "..."],
  "confidence": 0.0-1.0,
  "reasoning": ["step 1", "step 2", "step 3"]
}`
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

// validateResponse ensures the ranking is valid.
func (*RankingSynapse) validateResponse(response RankingResponse, input RankingInput) error {
	// Empty response is always invalid
	if len(response.Ranked) == 0 {
		return fmt.Errorf("empty ranking returned")
	}

	// For TopN, we expect fewer items
	if input.TopN > 0 {
		// Just check that returned items are subset of original
		itemSet := make(map[string]bool)
		for _, item := range input.Items {
			itemSet[item] = true
		}
		for _, ranked := range response.Ranked {
			if !itemSet[ranked] {
				return fmt.Errorf("ranked item '%s' not in original list", ranked)
			}
		}
		return nil
	}

	// Full ranking - check completeness
	if len(response.Ranked) != len(input.Items) {
		return fmt.Errorf("expected %d items, got %d", len(input.Items), len(response.Ranked))
	}

	// Check all items present and no duplicates
	seen := make(map[string]bool)
	for _, item := range response.Ranked {
		if seen[item] {
			return fmt.Errorf("duplicate item in ranking: %s", item)
		}
		seen[item] = true
	}

	// Check all original items are present
	for _, original := range input.Items {
		if !seen[original] {
			return fmt.Errorf("missing item in ranking: %s", original)
		}
	}

	return nil
}

// Ranking creates a new ranking synapse bound to a provider.
// The synapse orders items based on the specified criteria.
//
// Example:
//
//	synapse := Ranking("urgency and impact",
//	    provider,
//	    WithTimeout(10*time.Second),
//	)
//	ordered, err := synapse.Fire(ctx, []string{"Fix typo", "Security patch", "Add feature"})
func Ranking(criteria string, provider Provider, opts ...Option) *RankingSynapse {
	return NewRanking(criteria, provider, opts...)
}
