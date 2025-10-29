package zyn

import "github.com/zoobzio/capitan"

// Signals for hook events.
const (
	RequestStarted        = capitan.Signal("llm.request.started")
	RequestCompleted      = capitan.Signal("llm.request.completed")
	RequestFailed         = capitan.Signal("llm.request.failed")
	ProviderCallStarted   = capitan.Signal("llm.provider.call.started")
	ProviderCallCompleted = capitan.Signal("llm.provider.call.completed")
	ProviderCallFailed    = capitan.Signal("llm.provider.call.failed")
	ResponseParseFailed   = capitan.Signal("llm.response.failed")
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
