package zyn

// Default temperature constants for different synapse types.
// Temperature controls the randomness/creativity of LLM responses.
// Lower values (0.1) produce more deterministic outputs.
// Higher values (0.3) allow more creative/varied responses.
const (
	// DefaultTemperatureDeterministic is used for tasks requiring consistent,
	// precise outputs with minimal variation (binary decisions, extraction, conversion).
	DefaultTemperatureDeterministic = 0.1

	// DefaultTemperatureAnalytical is used for tasks requiring consistent analysis
	// with some flexibility (sentiment analysis, ranking, data analysis).
	DefaultTemperatureAnalytical = 0.2

	// DefaultTemperatureCreative is used for tasks benefiting from more varied
	// outputs (classification, text transformation).
	DefaultTemperatureCreative = 0.3
)
