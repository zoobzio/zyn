package zyn

import (
	"context"
	"fmt"
	"strings"

	"github.com/zoobzio/pipz"
)

// Sentiment constants.
const (
	sentimentPositive = "positive"
	sentimentNegative = "negative"
	sentimentNeutral  = "neutral"
	sentimentMixed    = "mixed"
)

// SentimentInput contains rich input structure for sentiment analysis.
type SentimentInput struct {
	Text        string   // The text to analyze
	Context     string   // Additional context about the text
	Aspects     []string // Specific aspects to analyze (e.g., "product quality", "customer service")
	Temperature float32  // LLM temperature setting
}

// SentimentResponse contains the sentiment analysis results.
type SentimentResponse struct {
	Overall    string            `json:"overall"`    // Primary sentiment: positive, negative, neutral, mixed
	Confidence float64           `json:"confidence"` // Confidence in overall sentiment
	Scores     SentimentScores   `json:"scores"`     // Detailed sentiment scores
	Aspects    map[string]string `json:"aspects"`    // Sentiment per aspect if requested
	Emotions   []string          `json:"emotions"`   // Detected emotions (joy, anger, fear, etc.)
	Reasoning  []string          `json:"reasoning"`  // Explanation of analysis
}

// Validate checks if the response is valid.
func (r SentimentResponse) Validate() error {
	if r.Overall == "" {
		return fmt.Errorf("overall sentiment required but empty")
	}
	if r.Confidence < 0 || r.Confidence > 1 {
		return fmt.Errorf("confidence must be 0-1, got %f", r.Confidence)
	}
	if len(r.Reasoning) == 0 {
		return fmt.Errorf("reasoning required but empty")
	}
	// Validate scores
	if err := r.Scores.Validate(); err != nil {
		return fmt.Errorf("invalid scores: %w", err)
	}
	return nil
}

// SentimentScores contains detailed sentiment breakdowns.
type SentimentScores struct {
	Positive float64 `json:"positive"` // 0.0-1.0 positive sentiment strength
	Negative float64 `json:"negative"` // 0.0-1.0 negative sentiment strength
	Neutral  float64 `json:"neutral"`  // 0.0-1.0 neutral sentiment strength
}

// Validate checks if sentiment scores are valid.
func (s SentimentScores) Validate() error {
	if s.Positive < 0 || s.Positive > 1 {
		return fmt.Errorf("positive score must be 0-1, got %f", s.Positive)
	}
	if s.Negative < 0 || s.Negative > 1 {
		return fmt.Errorf("negative score must be 0-1, got %f", s.Negative)
	}
	if s.Neutral < 0 || s.Neutral > 1 {
		return fmt.Errorf("neutral score must be 0-1, got %f", s.Neutral)
	}
	// Allow some tolerance for floating point arithmetic
	sum := s.Positive + s.Negative + s.Neutral
	if sum < 0.95 || sum > 1.05 {
		return fmt.Errorf("sentiment scores must sum to ~1.0, got %f", sum)
	}
	return nil
}

// SentimentSynapse represents a sentiment analysis synapse.
type SentimentSynapse struct {
	analysisType string // What kind of sentiment to analyze
	schema       string // Pre-computed JSON schema
	defaults     SentimentInput
	service      *Service[SentimentResponse]
}

// NewSentiment creates a new sentiment analysis synapse bound to a provider.
// Returns an error if the JSON schema cannot be generated.
func NewSentiment(analysisType string, provider Provider, opts ...Option) (*SentimentSynapse, error) {
	// Generate schema once at construction
	schema, err := generateJSONSchema[SentimentResponse]()
	if err != nil {
		return nil, fmt.Errorf("sentiment synapse: %w", err)
	}

	// Apply options to build pipeline
	pipeline := NewTerminal(provider)
	for _, opt := range opts {
		pipeline = opt(pipeline)
	}

	// Create service with final pipeline and default temperature
	svc := NewService[SentimentResponse](pipeline, "sentiment", provider, DefaultTemperatureAnalytical)

	return &SentimentSynapse{
		analysisType: analysisType,
		schema:       schema,
		service:      svc,
	}, nil
}

// GetPipeline returns the internal pipeline for composition.
func (s *SentimentSynapse) GetPipeline() pipz.Chainable[*SynapseRequest] {
	return s.service.GetPipeline()
}

// WithDefaults creates a new Sentiment with default input values.
func (s *SentimentSynapse) WithDefaults(defaults SentimentInput) *SentimentSynapse {
	s.defaults = defaults
	return s
}

// Fire executes sentiment analysis on text.
// Returns the overall sentiment classification.
func (s *SentimentSynapse) Fire(ctx context.Context, session *Session, text string) (string, error) {
	response, err := s.FireWithDetails(ctx, session, text)
	if err != nil {
		return "", err
	}
	return response.Overall, nil
}

// FireWithDetails executes sentiment analysis and returns full details.
func (s *SentimentSynapse) FireWithDetails(ctx context.Context, session *Session, text string) (SentimentResponse, error) {
	input := SentimentInput{Text: text}
	return s.FireWithInput(ctx, session, input)
}

// FireWithInput executes sentiment analysis with rich input structure.
func (s *SentimentSynapse) FireWithInput(ctx context.Context, session *Session, input SentimentInput) (SentimentResponse, error) {
	// Merge defaults with user input
	merged := s.mergeInputs(input)

	// Build prompt
	prompt := s.buildPrompt(merged)

	// Execute through service with session (service handles temperature fallback)
	response, err := s.service.Execute(ctx, session, prompt, merged.Temperature)
	if err != nil {
		return response, err
	}

	// Normalize the overall sentiment to standard values
	response.Overall = normalizeSentiment(response.Overall)

	return response, nil
}

// mergeInputs combines defaults with user input.
func (s *SentimentSynapse) mergeInputs(input SentimentInput) SentimentInput {
	merged := s.defaults

	if input.Text != "" {
		merged.Text = input.Text
	}
	if input.Context != "" {
		merged.Context = input.Context
	}
	if len(input.Aspects) > 0 {
		merged.Aspects = append(merged.Aspects, input.Aspects...)
	}
	if input.Temperature != 0 && input.Temperature != TemperatureUnset {
		merged.Temperature = input.Temperature
	}

	return merged
}

// buildPrompt constructs the prompt from the merged input.
func (s *SentimentSynapse) buildPrompt(input SentimentInput) *Prompt {
	prompt := &Prompt{
		Task:    fmt.Sprintf("Analyze %s sentiment", s.analysisType),
		Input:   input.Text,
		Context: input.Context,
		Aspects: input.Aspects,
		Schema:  s.schema,
	}

	// Build constraints
	prompt.Constraints = []string{
		"overall: positive, negative, neutral, or mixed only",
		"scores: sum to 1.0",
		"emotions: standard emotion categories",
		"confidence: 0.0 to 1.0",
	}

	if len(input.Aspects) > 0 {
		prompt.Constraints = append(prompt.Constraints, "aspects: analyze each specified aspect")
	}

	return prompt
}

// normalizeSentiment ensures sentiment values are standard.
func normalizeSentiment(sentiment string) string {
	lower := strings.ToLower(strings.TrimSpace(sentiment))
	switch lower {
	case sentimentPositive, "pos":
		return sentimentPositive
	case sentimentNegative, "neg":
		return sentimentNegative
	case sentimentNeutral, "neu":
		return sentimentNeutral
	case sentimentMixed, "mix":
		return sentimentMixed
	default:
		// If unclear, return as-is but log concern
		return lower
	}
}

// Sentiment creates a new sentiment analysis synapse bound to a provider.
// The synapse analyzes emotional tone and sentiment of text.
// Returns an error if the JSON schema cannot be generated.
//
// Example:
//
//	synapse, err := Sentiment("customer feedback", provider)
//	sentiment, err := synapse.Fire(ctx, "This product exceeded my expectations!")
//	// Returns: "positive"
//
//	details, err := synapse.FireWithDetails(ctx, text)
//	// Returns full analysis with scores and emotions
func Sentiment(analysisType string, provider Provider, opts ...Option) (*SentimentSynapse, error) {
	return NewSentiment(analysisType, provider, opts...)
}
