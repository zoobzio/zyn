// Package testing provides utilities for testing zyn synapses.
package testing

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	"github.com/zoobzio/zyn"
)

// Provider name constants for test helpers.
const (
	SequencedProviderName = "sequenced-mock"
	FailingProviderName   = "failing-mock"
)

// ResponseBuilder provides a fluent interface for constructing mock LLM responses.
type ResponseBuilder struct {
	data map[string]any
}

// NewResponseBuilder creates a new ResponseBuilder.
func NewResponseBuilder() *ResponseBuilder {
	return &ResponseBuilder{
		data: make(map[string]any),
	}
}

// WithDecision sets the decision field (for binary synapses).
func (b *ResponseBuilder) WithDecision(decision bool) *ResponseBuilder {
	b.data["decision"] = decision
	return b
}

// WithConfidence sets the confidence field.
func (b *ResponseBuilder) WithConfidence(confidence float64) *ResponseBuilder {
	b.data["confidence"] = confidence
	return b
}

// WithReasoning sets the reasoning field.
func (b *ResponseBuilder) WithReasoning(reasons ...string) *ResponseBuilder {
	b.data["reasoning"] = reasons
	return b
}

// WithPrimary sets the primary field (for classification synapses).
func (b *ResponseBuilder) WithPrimary(primary string) *ResponseBuilder {
	b.data["primary"] = primary
	return b
}

// WithSecondary sets the secondary field (for classification synapses).
func (b *ResponseBuilder) WithSecondary(secondary string) *ResponseBuilder {
	b.data["secondary"] = secondary
	return b
}

// WithRanked sets the ranked field (for ranking synapses).
func (b *ResponseBuilder) WithRanked(items ...string) *ResponseBuilder {
	b.data["ranked"] = items
	return b
}

// WithOutput sets the output field (for transform synapses).
func (b *ResponseBuilder) WithOutput(output string) *ResponseBuilder {
	b.data["output"] = output
	return b
}

// WithChanges sets the changes field (for transform synapses).
func (b *ResponseBuilder) WithChanges(changes ...string) *ResponseBuilder {
	b.data["changes"] = changes
	return b
}

// WithOverall sets the overall field (for sentiment synapses).
func (b *ResponseBuilder) WithOverall(overall string) *ResponseBuilder {
	b.data["overall"] = overall
	return b
}

// WithScores sets the scores field (for sentiment synapses).
func (b *ResponseBuilder) WithScores(positive, negative, neutral float64) *ResponseBuilder {
	b.data["scores"] = map[string]float64{
		"positive": positive,
		"negative": negative,
		"neutral":  neutral,
	}
	return b
}

// WithEmotions sets the emotions field (for sentiment synapses).
func (b *ResponseBuilder) WithEmotions(emotions ...string) *ResponseBuilder {
	b.data["emotions"] = emotions
	return b
}

// WithAspects sets the aspects field (for sentiment synapses).
func (b *ResponseBuilder) WithAspects(aspects map[string]string) *ResponseBuilder {
	b.data["aspects"] = aspects
	return b
}

// WithField sets an arbitrary field.
func (b *ResponseBuilder) WithField(key string, value any) *ResponseBuilder {
	b.data[key] = value
	return b
}

// Build returns the JSON string representation of the response.
func (b *ResponseBuilder) Build() string {
	jsonBytes, err := json.Marshal(b.data)
	if err != nil {
		return "{}"
	}
	return string(jsonBytes)
}

// BuildBytes returns the JSON bytes of the response.
func (b *ResponseBuilder) BuildBytes() []byte {
	jsonBytes, err := json.Marshal(b.data)
	if err != nil {
		return []byte("{}")
	}
	return jsonBytes
}

// SequencedProvider returns responses in sequence.
// After all responses are exhausted, it returns the last response repeatedly.
type SequencedProvider struct {
	responses []string
	index     atomic.Int64
	mu        sync.Mutex
}

// NewSequencedProvider creates a provider that returns responses in order.
func NewSequencedProvider(responses ...string) *SequencedProvider {
	if len(responses) == 0 {
		responses = []string{`{"error": "no responses configured"}`}
	}
	return &SequencedProvider{
		responses: responses,
	}
}

// Call returns the next response in sequence.
func (p *SequencedProvider) Call(_ context.Context, _ []zyn.Message, _ float32) (*zyn.ProviderResponse, error) {
	idx := p.index.Add(1) - 1
	p.mu.Lock()
	defer p.mu.Unlock()

	// Clamp to last response if exhausted
	if int(idx) >= len(p.responses) {
		idx = int64(len(p.responses) - 1)
	}

	return &zyn.ProviderResponse{
		Content: p.responses[idx],
		Usage: zyn.TokenUsage{
			Prompt:     100,
			Completion: 50,
			Total:      150,
		},
	}, nil
}

// Name returns the provider identifier.
func (*SequencedProvider) Name() string {
	return SequencedProviderName
}

// CallCount returns the number of calls made.
func (p *SequencedProvider) CallCount() int {
	return int(p.index.Load())
}

// Reset resets the call counter.
func (p *SequencedProvider) Reset() {
	p.index.Store(0)
}

// FailingProvider fails a specified number of times before succeeding.
type FailingProvider struct {
	failCount    int
	currentCount atomic.Int64
	successResp  string
	failError    string
}

// NewFailingProvider creates a provider that fails failCount times then succeeds.
func NewFailingProvider(failCount int) *FailingProvider {
	return &FailingProvider{
		failCount:   failCount,
		successResp: `{"decision": true, "confidence": 0.9, "reasoning": ["recovered"]}`,
		failError:   "simulated provider failure",
	}
}

// WithSuccessResponse sets the response returned after failures are exhausted.
func (p *FailingProvider) WithSuccessResponse(response string) *FailingProvider {
	p.successResp = response
	return p
}

// WithFailError sets the error message for failures.
func (p *FailingProvider) WithFailError(errMsg string) *FailingProvider {
	p.failError = errMsg
	return p
}

// Call fails until failCount is reached, then succeeds.
func (p *FailingProvider) Call(_ context.Context, _ []zyn.Message, _ float32) (*zyn.ProviderResponse, error) {
	count := p.currentCount.Add(1)
	if int(count) <= p.failCount {
		return nil, fmt.Errorf("%s (attempt %d/%d)", p.failError, count, p.failCount)
	}

	return &zyn.ProviderResponse{
		Content: p.successResp,
		Usage: zyn.TokenUsage{
			Prompt:     100,
			Completion: 50,
			Total:      150,
		},
	}, nil
}

// Name returns the provider identifier.
func (*FailingProvider) Name() string {
	return FailingProviderName
}

// CallCount returns the number of calls made.
func (p *FailingProvider) CallCount() int {
	return int(p.currentCount.Load())
}

// Reset resets the call counter.
func (p *FailingProvider) Reset() {
	p.currentCount.Store(0)
}

// RecordedCall represents a single call to a provider.
type RecordedCall struct {
	Messages    []zyn.Message
	Temperature float32
}

// CallRecorder wraps a provider and records all calls made to it.
type CallRecorder struct {
	provider zyn.Provider
	calls    []RecordedCall
	mu       sync.Mutex
}

// NewCallRecorder wraps a provider with call recording.
func NewCallRecorder(provider zyn.Provider) *CallRecorder {
	return &CallRecorder{
		provider: provider,
		calls:    make([]RecordedCall, 0),
	}
}

// Call delegates to the wrapped provider and records the call.
func (r *CallRecorder) Call(ctx context.Context, messages []zyn.Message, temperature float32) (*zyn.ProviderResponse, error) {
	// Record the call (copy messages to avoid aliasing)
	msgCopy := make([]zyn.Message, len(messages))
	copy(msgCopy, messages)

	r.mu.Lock()
	r.calls = append(r.calls, RecordedCall{
		Messages:    msgCopy,
		Temperature: temperature,
	})
	r.mu.Unlock()

	return r.provider.Call(ctx, messages, temperature)
}

// Name returns the wrapped provider's name.
func (r *CallRecorder) Name() string {
	return r.provider.Name()
}

// Calls returns a copy of all recorded calls.
func (r *CallRecorder) Calls() []RecordedCall {
	r.mu.Lock()
	defer r.mu.Unlock()

	calls := make([]RecordedCall, len(r.calls))
	copy(calls, r.calls)
	return calls
}

// CallCount returns the number of calls recorded.
func (r *CallRecorder) CallCount() int {
	r.mu.Lock()
	defer r.mu.Unlock()
	return len(r.calls)
}

// LastCall returns the most recent call, or nil if no calls made.
func (r *CallRecorder) LastCall() *RecordedCall {
	r.mu.Lock()
	defer r.mu.Unlock()

	if len(r.calls) == 0 {
		return nil
	}
	call := r.calls[len(r.calls)-1]
	return &call
}

// Reset clears all recorded calls.
func (r *CallRecorder) Reset() {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.calls = make([]RecordedCall, 0)
}

// LatencyProvider wraps a provider and adds artificial latency.
type LatencyProvider struct {
	provider zyn.Provider
	delay    time.Duration
}

// NewLatencyProvider wraps a provider with artificial delay.
// The delay is applied before each provider call and respects context cancellation.
func NewLatencyProvider(provider zyn.Provider, delay time.Duration) *LatencyProvider {
	return &LatencyProvider{
		provider: provider,
		delay:    delay,
	}
}

// Call adds latency then delegates to the wrapped provider.
// Respects context cancellation during the delay period.
func (p *LatencyProvider) Call(ctx context.Context, messages []zyn.Message, temperature float32) (*zyn.ProviderResponse, error) {
	if p.delay > 0 {
		select {
		case <-time.After(p.delay):
			// Delay completed
		case <-ctx.Done():
			return nil, ctx.Err()
		}
	}
	return p.provider.Call(ctx, messages, temperature)
}

// Name returns the wrapped provider's name.
func (p *LatencyProvider) Name() string {
	return p.provider.Name()
}

// UsageAccumulator tracks total token usage across multiple calls.
type UsageAccumulator struct {
	promptTokens     atomic.Int64
	completionTokens atomic.Int64
	totalTokens      atomic.Int64
	callCount        atomic.Int64
}

// NewUsageAccumulator creates a new usage accumulator.
func NewUsageAccumulator() *UsageAccumulator {
	return &UsageAccumulator{}
}

// Add accumulates usage from a session's last usage.
func (a *UsageAccumulator) Add(session *zyn.Session) {
	if usage := session.LastUsage(); usage != nil {
		a.promptTokens.Add(int64(usage.Prompt))
		a.completionTokens.Add(int64(usage.Completion))
		a.totalTokens.Add(int64(usage.Total))
		a.callCount.Add(1)
	}
}

// AddUsage accumulates usage directly.
func (a *UsageAccumulator) AddUsage(usage *zyn.TokenUsage) {
	if usage != nil {
		a.promptTokens.Add(int64(usage.Prompt))
		a.completionTokens.Add(int64(usage.Completion))
		a.totalTokens.Add(int64(usage.Total))
		a.callCount.Add(1)
	}
}

// PromptTokens returns total prompt tokens.
func (a *UsageAccumulator) PromptTokens() int {
	return int(a.promptTokens.Load())
}

// CompletionTokens returns total completion tokens.
func (a *UsageAccumulator) CompletionTokens() int {
	return int(a.completionTokens.Load())
}

// TotalTokens returns total tokens.
func (a *UsageAccumulator) TotalTokens() int {
	return int(a.totalTokens.Load())
}

// CallCount returns number of calls accumulated.
func (a *UsageAccumulator) CallCount() int {
	return int(a.callCount.Load())
}

// Reset clears all accumulated values.
func (a *UsageAccumulator) Reset() {
	a.promptTokens.Store(0)
	a.completionTokens.Store(0)
	a.totalTokens.Store(0)
	a.callCount.Store(0)
}
