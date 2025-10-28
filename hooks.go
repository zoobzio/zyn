package zyn

import "github.com/zoobzio/capitan"

// Signals for hook events.
const (
	RequestStarted        = capitan.Signal("request.started")
	RequestCompleted      = capitan.Signal("request.completed")
	RequestFailed         = capitan.Signal("request.failed")
	ProviderCallCompleted = capitan.Signal("provider.call.completed")
	ResponseParseFailed   = capitan.Signal("response.failed")
)

// Keys for hook event fields.
var (
	// Request identification.
	RequestIDKey   = capitan.NewStringKey("request_id")
	SynapseTypeKey = capitan.NewStringKey("synapse_type")
	PromptTaskKey  = capitan.NewStringKey("prompt_task")
	TemperatureKey = capitan.NewFloat64Key("temperature")

	// Input/Output data.
	InputKey  = capitan.NewStringKey("input")
	OutputKey = capitan.NewStringKey("output")

	// Response data.
	ResponseKey = capitan.NewStringKey("response")

	// Error information.
	ErrorKey     = capitan.NewStringKey("error")
	ErrorTypeKey = capitan.NewStringKey("error_type")

	// Provider information.
	ProviderKey = capitan.NewStringKey("provider")
	ModelKey    = capitan.NewStringKey("model")

	// Provider metrics.
	PromptTokensKey     = capitan.NewIntKey("prompt_tokens")
	CompletionTokensKey = capitan.NewIntKey("completion_tokens")
	TotalTokensKey      = capitan.NewIntKey("total_tokens")
	DurationMsKey       = capitan.NewIntKey("duration_ms")
)
