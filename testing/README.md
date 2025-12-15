# Testing Infrastructure

This directory contains integration tests, benchmarks, and shared testing utilities for zyn.

## Directory Structure

```
testing/
├── README.md                 # This file
├── helpers.go                # Shared test utilities
├── helpers_test.go           # Tests for helpers
├── integration/
│   ├── README.md             # Integration test documentation
│   ├── session_test.go       # Multi-turn conversation tests
│   ├── pipeline_test.go      # Reliability pattern tests
│   └── provider_test.go      # Real provider tests (requires API key)
└── benchmarks/
    ├── README.md             # Benchmark documentation
    └── synapse_test.go       # Synapse performance benchmarks
```

## Test Categories

### Unit Tests (root package)

Located alongside source files in the package root. These use mock providers for fast, deterministic testing.

```bash
go test -v -race ./...
```

### Integration Tests

Multi-component tests validating real-world scenarios:

```bash
go test -v -race ./testing/integration/...
```

### Provider Tests (requires API key)

Tests against real LLM providers. Skipped without `OPENAI_API_KEY`:

```bash
OPENAI_API_KEY=sk-... go test -v ./testing/integration/... -run TestProvider
```

### Benchmarks

Performance measurements for critical paths:

```bash
go test -v -bench=. -benchmem ./testing/benchmarks/...
```

## Testing Utilities

### ResponseBuilder

Fluent builder for constructing mock LLM responses:

```go
response := testing.NewResponseBuilder().
    WithDecision(true).
    WithConfidence(0.95).
    WithReasoning("Valid format", "Contains @ symbol").
    Build()
```

### SequencedProvider

Mock provider that returns responses in sequence:

```go
provider := testing.NewSequencedProvider(
    `{"decision": true, "confidence": 0.9, "reasoning": ["first"]}`,
    `{"decision": false, "confidence": 0.8, "reasoning": ["second"]}`,
)
// First call returns first response, second call returns second, etc.
```

### FailingProvider

Mock provider for testing error handling and retries:

```go
provider := testing.NewFailingProvider(2) // Fails first 2 calls, then succeeds
```

### CallRecorder

Records all calls to a provider for assertion:

```go
recorder := testing.NewCallRecorder(mockProvider)
// ... use recorder as provider ...
calls := recorder.Calls()
assert.Equal(t, 3, len(calls))
assert.Contains(t, calls[0].Messages[0].Content, "expected prompt")
```

## Testing Strategy

### Mock-First Approach

Unit and most integration tests use mock providers for:
- Speed (no network latency)
- Determinism (reproducible results)
- Cost (no API charges)
- Isolation (no external dependencies)

### Real Provider Tests

A subset of integration tests exercise real providers to validate:
- API contract compatibility
- Response parsing with real LLM output
- Token usage tracking accuracy

These tests are skipped without the appropriate API key environment variable.

### Recorded Responses (Future)

For complex scenarios, we may add HTTP record/replay capability to capture real API responses and replay them deterministically in tests.

## Writing Tests

### Integration Test Pattern

```go
func TestScenario_Description(t *testing.T) {
    // Setup
    provider := testing.NewSequencedProvider(responses...)
    synapse, err := zyn.Binary("question", provider)
    require.NoError(t, err)

    session := zyn.NewSession()

    // Execute
    result, err := synapse.Fire(context.Background(), session, "input")

    // Assert
    require.NoError(t, err)
    assert.True(t, result)
    assert.Equal(t, 2, session.Len()) // user + assistant messages
}
```

### Benchmark Pattern

```go
func BenchmarkSynapse_Fire(b *testing.B) {
    provider := zyn.NewMockProvider()
    synapse, _ := zyn.Binary("question", provider)
    session := zyn.NewSession()
    ctx := context.Background()

    b.ResetTimer()
    b.ReportAllocs()

    for i := 0; i < b.N; i++ {
        session.Clear()
        _, _ = synapse.Fire(ctx, session, "input")
    }
}
```

## Running All Tests

```bash
# Unit tests only
make test

# With integration tests
go test -v -race ./... ./testing/...

# With benchmarks
go test -v -race ./... && go test -bench=. -benchmem ./testing/benchmarks/...

# Full suite with real provider (requires API key)
OPENAI_API_KEY=sk-... go test -v -race ./... ./testing/...
```
