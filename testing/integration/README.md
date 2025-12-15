# Integration Tests

This directory contains integration tests for zyn that validate multi-component interactions and real-world scenarios.

## Test Files

- **session_test.go** - Multi-turn conversation tests
- **pipeline_test.go** - Reliability pattern tests (retry, circuit breaker, timeout)
- **provider_test.go** - Real provider tests (requires API key)

## Running Tests

```bash
# All integration tests (mock providers)
go test -v -race ./testing/integration/...

# With real provider (requires API key)
OPENAI_API_KEY=sk-... go test -v ./testing/integration/... -run TestProvider
```

## Test Categories

### Session Tests

Validate session behavior across multiple synapse calls:
- Message history accumulation
- Multi-synapse workflows
- Session manipulation (prune, truncate, etc.)
- Token usage tracking

### Pipeline Tests

Validate reliability patterns:
- Retry with transient failures
- Circuit breaker tripping and recovery
- Timeout handling
- Fallback behavior
- Error handler invocation

### Provider Tests

Validate real LLM provider integration:
- API contract compatibility
- Response parsing accuracy
- Token usage reporting
- Error handling for API failures

These tests are skipped without `OPENAI_API_KEY` environment variable.

## Writing Integration Tests

```go
func TestScenario_Description(t *testing.T) {
    // Use testing helpers for mock providers
    provider := testing.NewSequencedProvider(
        testing.NewResponseBuilder().WithDecision(true).WithConfidence(0.9).Build(),
        testing.NewResponseBuilder().WithDecision(false).WithConfidence(0.8).Build(),
    )

    // Create synapse with options
    synapse, err := zyn.Binary("question", provider, zyn.WithRetry(3))
    require.NoError(t, err)

    // Use fresh session per test
    session := zyn.NewSession()

    // Execute and assert
    result, err := synapse.Fire(context.Background(), session, "input")
    require.NoError(t, err)
    assert.True(t, result)
}
```
