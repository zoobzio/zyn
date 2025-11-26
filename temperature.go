package zyn

// Default temperature constants for different synapse types.
// Temperature controls the randomness/creativity of LLM responses.
// Lower values (0.1) produce more deterministic outputs.
// Higher values (0.3) allow more creative/varied responses.
const (
	// TemperatureUnset indicates that no temperature has been explicitly set.
	// When this value is encountered, the synapse will use its default temperature.
	// Note: A zero-value float32 (0.0) is also treated as unset for ergonomic struct initialization.
	TemperatureUnset float32 = -1

	// TemperatureZero provides an explicitly near-zero temperature for maximum determinism.
	// Use this instead of 0.0 since zero is treated as "unset".
	TemperatureZero float32 = 0.0001

	// DefaultTemperatureDeterministic is used for tasks requiring consistent,
	// precise outputs with minimal variation (binary decisions, extraction, conversion).
	DefaultTemperatureDeterministic float32 = 0.1

	// DefaultTemperatureAnalytical is used for tasks requiring consistent analysis
	// with some flexibility (sentiment analysis, ranking, data analysis).
	DefaultTemperatureAnalytical float32 = 0.2

	// DefaultTemperatureCreative is used for tasks benefiting from more varied
	// outputs (classification, text transformation).
	DefaultTemperatureCreative float32 = 0.3
)
