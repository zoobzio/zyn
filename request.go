package zyn

// SynapseRequest flows through the pipz pipeline.
// It contains the prompt, parameters, and response data.
type SynapseRequest struct {
	// Input fields
	Prompt      *Prompt // The structured prompt to send to LLM
	Temperature float32 // Temperature parameter for response generation

	// Metadata fields
	RequestID    string // Unique identifier for this request
	SynapseType  string // Type of synapse (binary, extraction, etc.)
	ProviderName string // Name of the provider being used

	// Output fields (populated by pipeline)
	Response string // Raw text response from provider
	Error    error  // Any error that occurred during processing
}
