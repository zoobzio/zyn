package zyn

import "github.com/zoobzio/capitan"

// Signals for hook events.
var (
	RequestStarted        = capitan.NewSignal("llm.request.started", "LLM synapse request initiated with task, input, and configuration")
	RequestCompleted      = capitan.NewSignal("llm.request.completed", "LLM synapse request succeeded with parsed output and raw response")
	RequestFailed         = capitan.NewSignal("llm.request.failed", "LLM synapse request failed with error details and context")
	ProviderCallStarted   = capitan.NewSignal("llm.provider.call.started", "LLM provider HTTP call initiated with model and parameters")
	ProviderCallCompleted = capitan.NewSignal("llm.provider.call.completed", "LLM provider HTTP call succeeded with token usage and timing metrics")
	ProviderCallFailed    = capitan.NewSignal("llm.provider.call.failed", "LLM provider HTTP call failed with status code and API error details")
	ResponseParseFailed   = capitan.NewSignal("llm.response.failed", "LLM response parsing failed with validation or JSON decode error")
)

// Keys for hook event fields.
var (
	// Request identification.
	RequestIDKey   = capitan.NewStringKey("llm.request.id")
	SynapseTypeKey = capitan.NewStringKey("llm.synapse.type")
	PromptTaskKey  = capitan.NewStringKey("llm.prompt.task")
	TemperatureKey = capitan.NewFloat64Key("llm.temperature")

	// Input/Output data.
	InputKey  = capitan.NewStringKey("llm.input")
	OutputKey = capitan.NewStringKey("llm.output")

	// Response data.
	ResponseKey = capitan.NewStringKey("llm.response")

	// Error information.
	ErrorKey     = capitan.NewStringKey("llm.error")
	ErrorTypeKey = capitan.NewStringKey("llm.error.type")

	// Provider information.
	ProviderKey = capitan.NewStringKey("llm.provider")
	ModelKey    = capitan.NewStringKey("llm.model")

	// Provider metrics.
	PromptTokensKey     = capitan.NewIntKey("llm.tokens.prompt")
	CompletionTokensKey = capitan.NewIntKey("llm.tokens.completion")
	TotalTokensKey      = capitan.NewIntKey("llm.tokens.total")
	DurationMsKey       = capitan.NewIntKey("llm.duration.ms")

	// HTTP/API metadata.
	HTTPStatusCodeKey = capitan.NewIntKey("llm.http.status.code")
	APIErrorTypeKey   = capitan.NewStringKey("llm.api.error.type")
	APIErrorCodeKey   = capitan.NewStringKey("llm.api.error.code")

	// Response metadata.
	ResponseIDKey           = capitan.NewStringKey("llm.response.id")
	ResponseFinishReasonKey = capitan.NewStringKey("llm.response.finish.reason")
	ResponseCreatedKey      = capitan.NewIntKey("llm.response.created")
)
