[![CI Status](https://github.com/zoobzio/zyn/workflows/CI/badge.svg)](https://github.com/zoobzio/zyn/actions/workflows/ci.yml)
[![codecov](https://codecov.io/gh/zoobzio/zyn/graph/badge.svg?branch=main)](https://codecov.io/gh/zoobzio/zyn)
[![Go Report Card](https://goreportcard.com/badge/github.com/zoobzio/zyn)](https://goreportcard.com/report/github.com/zoobzio/zyn)
[![CodeQL](https://github.com/zoobzio/zyn/workflows/CodeQL/badge.svg)](https://github.com/zoobzio/zyn/security/code-scanning)
[![Go Reference](https://pkg.go.dev/badge/github.com/zoobzio/zyn.svg)](https://pkg.go.dev/github.com/zoobzio/zyn)
[![License](https://img.shields.io/github/license/zoobzio/zyn)](LICENSE)
[![Go Version](https://img.shields.io/github/go-mod/go-version/zoobzio/zyn)](go.mod)
[![Release](https://img.shields.io/github/v/release/zoobzio/zyn)](https://github.com/zoobzio/zyn/releases)

# zyn

**Type-safe LLM orchestration framework with composable reliability patterns**

zyn provides a clean, type-safe way to orchestrate Large Language Model (LLM) interactions with built-in reliability patterns. Instead of fighting with prompt engineering and error handling, focus on your application logic.

## Features

- **8 Synapse Types** covering all LLM interaction patterns
- **Type-Safe Generics** with compile-time guarantees and automatic validation
- **OpenAI Provider** with extensible provider interface
- **Built-in Reliability** via [pipz](https://github.com/zoobzio/pipz) integration
- **Production Observability** via [capitan](https://github.com/zoobzio/capitan) hooks
- **Structured Prompts** preventing prompt divergence

## Quick Start

```bash
go get github.com/zoobzio/zyn
```

```go
package main

import (
    "context"
    "fmt"
    "os"
    "time"

    "github.com/zoobzio/zyn"
    "github.com/zoobzio/zyn/providers/openai"
)

func main() {
    // Create provider
    provider := openai.New(openai.Config{
        APIKey: os.Getenv("OPENAI_API_KEY"),
    })

    // Create synapse with reliability features
    classifier, err := zyn.Classification(
        "What type of email is this?",
        []string{"spam", "urgent", "newsletter", "personal"},
        provider,
        zyn.WithRetry(3),
        zyn.WithTimeout(10*time.Second),
    )
    if err != nil {
        panic(err)
    }

    // Create a session for conversation context
    session := zyn.NewSession()

    // Use it
    ctx := context.Background()
    category, err := classifier.Fire(ctx, session, "URGENT: Your account will be suspended!")
    if err != nil {
        panic(err)
    }

    fmt.Println("Category:", category) // "urgent"
}
```

## Synapse Types

zyn provides 8 synapse types covering all LLM interaction patterns:

### Decision & Analysis

- **Binary** - Yes/no decisions with confidence scores
- **Classification** - Categorize into predefined classes
- **Ranking** - Order items by specified criteria
- **Sentiment** - Analyze emotional tone with aspects

### Data Transformation

- **Extraction** - Extract structured data from text (string → struct[T])
- **Transform** - Transform text (string → string)
- **Analyze** - Analyze structured data (struct[T] → string)
- **Convert** - Convert between types (struct[TInput] → struct[TOutput])

## Examples

### Binary Decision

```go
session := zyn.NewSession()
validator, err := zyn.Binary("Is this a valid email address?", provider)
if err != nil {
    panic(err)
}
isValid, err := validator.Fire(ctx, session, "user@example.com")
// Returns: true
```

### Classification with Examples

```go
classifier, err := zyn.Classification("Classify sentiment",
    []string{"positive", "negative", "neutral"}, provider)
if err != nil {
    panic(err)
}

input := zyn.ClassificationInput{
    Subject: "I love this product!",
    Examples: map[string][]string{
        "positive": {"Great!", "Amazing!"},
        "negative": {"Terrible", "Awful"},
    },
}

session := zyn.NewSession()
result, err := classifier.FireWithInput(ctx, session, input)
// Returns: ClassificationResponse{Primary: "positive", Confidence: 0.95, ...}
```

### Structured Data Extraction

```go
type Contact struct {
    Name  string `json:"name"`
    Email string `json:"email"`
    Phone string `json:"phone"`
}

// Validate is REQUIRED - all response types must implement the Validator interface
// This ensures LLM outputs are validated before being returned to your application
func (c Contact) Validate() error {
    if c.Email == "" {
        return fmt.Errorf("email required")
    }
    return nil
}

session := zyn.NewSession()
extractor, err := zyn.Extract[Contact]("contact information", provider)
if err != nil {
    panic(err)
}
contact, err := extractor.Fire(ctx, session, "John Doe at john@example.com or (555) 123-4567")
// Returns: Contact{Name: "John Doe", Email: "john@example.com", Phone: "(555) 123-4567"}
```

**Important:** All custom types used with synapses must implement the `Validator` interface:

```go
type Validator interface {
    Validate() error
}
```

This ensures LLM responses are validated before being returned. The framework automatically calls `Validate()` after parsing the JSON response and returns an error if validation fails.

### Text Transformation

```go
session := zyn.NewSession()
summarizer, _ := zyn.Transform("summarize into key points", provider)
summary, err := summarizer.Fire(ctx, session, longArticle)

translator, _ := zyn.Transform("translate to Spanish", provider)
spanish, err := translator.Fire(ctx, session, "Hello, how are you?")
```

### Data Analysis

```go
type ServerMetrics struct {
    CPU    float64 `json:"cpu_usage"`
    Memory float64 `json:"memory_usage"`
    Errors int     `json:"error_count"`
}

session := zyn.NewSession()
analyzer, _ := zyn.Analyze[ServerMetrics]("performance analysis", provider)
analysis, err := analyzer.Fire(ctx, session, metrics)
// Returns: "CPU usage is high at 85%. Consider scaling..."
```

### Type Conversion

```go
type UserV1 struct {
    Name  string `json:"name"`
    Email string `json:"email"`
}

type UserV2 struct {
    FullName string `json:"full_name"`
    Contact  struct {
        Email string `json:"email"`
    } `json:"contact"`
}

// Validate is REQUIRED for the output type
func (u UserV2) Validate() error {
    if u.Contact.Email == "" {
        return fmt.Errorf("contact email required")
    }
    return nil
}

session := zyn.NewSession()
converter, _ := zyn.Convert[UserV1, UserV2]("migrate to v2 schema", provider)
v2User, err := converter.Fire(ctx, session, v1User)
```

## Sessions

Sessions enable conversation context across multiple synapse calls, allowing synapses to access previous interactions. This is essential for building complex LLM-powered workflows where later steps depend on earlier context.

### Basic Usage

```go
// Create a session - lightweight wrapper around message history
session := zyn.NewSession()

// All Fire() methods require a session
classifier, _ := zyn.Classification("sentiment", []string{"positive", "negative"}, provider)
sentiment, err := classifier.Fire(ctx, session, "I love this!")
// sentiment = "positive"

// Session now contains the conversation history
// Next call in same session will have access to previous context
followup, _ := zyn.Binary("Was the previous message enthusiastic?", provider)
enthusiastic, err := followup.Fire(ctx, session, "Check the sentiment")
// enthusiastic = true (LLM sees previous "I love this!" message)
```

### Multi-Synapse Workflows

Sessions are designed for chaining synapses in complex workflows where context matters:

```go
session := zyn.NewSession()

// Extract customer info
type Customer struct {
    Name  string `json:"name"`
    Issue string `json:"issue"`
}
func (c Customer) Validate() error {
    if c.Name == "" { return fmt.Errorf("name required") }
    return nil
}

extractor, _ := zyn.Extract[Customer]("customer details", provider)
customer, _ := extractor.Fire(ctx, session, "Hi, I'm John and my order hasn't arrived")

// Classify urgency (with context from extraction)
urgency, _ := zyn.Classification("urgency level", []string{"low", "medium", "high"}, provider)
level, _ := urgency.Fire(ctx, session, customer.Issue)

// Generate response (with full context)
responder, _ := zyn.Transform("write customer service response", provider)
response, _ := responder.Fire(ctx, session, fmt.Sprintf("Customer: %s, Urgency: %s", customer.Name, level))
```

### Session Management

```go
// Check message history
messages := session.Messages() // Returns []Message
count := session.Len()         // Number of messages

// Clear all messages
session.Clear()

// Remove last N message pairs (useful for context window management)
err := session.Prune(2) // Removes last 2 user/assistant message pairs
```

### Token Tracking

Sessions track token usage from provider responses:

```go
session := zyn.NewSession()
synapse.Fire(ctx, session, "input")

// Get usage from last successful call
if usage := session.LastUsage(); usage != nil {
    fmt.Printf("Tokens: prompt=%d completion=%d total=%d\n",
        usage.Prompt, usage.Completion, usage.Total)
}
```

### Message Manipulation

Sessions expose primitives for custom context management strategies:

```go
// Access individual messages
msg, err := session.At(0)           // Get message at index
err := session.Remove(2)            // Remove message at index
err := session.Replace(1, newMsg)   // Replace message at index
err := session.Insert(0, systemMsg) // Insert message at index

// Bulk operations
session.Truncate(2, 2)              // Keep first 2 and last 2 messages
session.SetMessages(msgs)           // Replace entire history
```

These primitives enable building custom context strategies externally:
- Sliding window (keep last N messages)
- Summarization (replace old messages with summary)
- Selective pruning (remove low-value exchanges)

### Heterogeneous Conversations

Sessions support mixing different synapse types - each synapse stays focused on its task while accessing shared context:

```go
session := zyn.NewSession()

// Different synapse types, same session
binary, _ := zyn.Binary("question?", provider)
classifier, _ := zyn.Classification("category", []string{"a", "b"}, provider)
extractor, _ := zyn.Extract[Data]("extract", provider)

binary.Fire(ctx, session, "input1")
classifier.Fire(ctx, session, "input2")      // Sees binary's context
extractor.Fire(ctx, session, "input3")       // Sees both previous contexts
```

### Prompt Caching Benefits

Sessions leverage provider-side prompt caching for efficiency:

- Full message history is sent with each call
- Providers cache the history server-side (5min-1hr TTL)
- Subsequent calls reuse cached context, reducing costs by 60-90%
- Cached tokens don't count against rate limits (Claude)

No special configuration needed - caching is automatic when using sessions.

### Error Handling

Sessions use transactional updates - messages are only appended after successful provider responses:

```go
session := zyn.NewSession()

synapse, _ := zyn.Binary("question", provider, zyn.WithRetry(3))
result, err := synapse.Fire(ctx, session, "input")

// If Fire() fails (even after retries), session is unchanged
// If Fire() succeeds, session contains both user message and assistant response
```

This prevents retry attempts from corrupting the session with duplicate messages.

## Provider

zyn uses OpenAI as its LLM provider:

```go
provider := openai.New(openai.Config{
    APIKey: os.Getenv("OPENAI_API_KEY"),
    Model:  "gpt-4",
})
```

## Reliability Features

zyn integrates with [pipz](https://github.com/zoobzio/pipz) for composable reliability patterns:

```go
synapse, _ := zyn.Binary("question", provider,
    // Retry with exponential backoff
    zyn.WithRetry(3),

    // Circuit breaker protection
    zyn.WithCircuitBreaker(5, 30*time.Second),

    // Rate limiting
    zyn.WithRateLimit(10, 100),

    // Timeout protection
    zyn.WithTimeout(30*time.Second),

    // Fallback to another provider
    zyn.WithFallback(backupSynapse),

    // Custom error handling
    zyn.WithErrorHandler(errorPipeline),
)
```

## Error Handling

Handle errors with custom pipelines:

```go
errorLogger := pipz.Apply("log-errors", func(ctx context.Context, e *pipz.Error[*zyn.SynapseRequest]) (*pipz.Error[*zyn.SynapseRequest], error) {
    log.Printf("Synapse failed: %v", e.Err)

    // Track metrics
    if strings.Contains(e.Err.Error(), "rate limit") {
        metrics.Increment("rate_limit_errors")
    }

    return e, nil
})

synapse, _ := zyn.Binary("question", provider,
    zyn.WithErrorHandler(errorLogger),
)
```

## Observability

zyn emits [capitan](https://github.com/zoobzio/capitan) hooks for observability into LLM requests. All hooks include request correlation IDs and rich metadata.

### Available Signals

**llm.request.started** - Emitted before LLM call
- `llm.request.id` - Unique request identifier
- `llm.synapse.type` - Type of synapse (binary, extraction, etc.)
- `llm.provider` - Provider name (openai, etc.)
- `llm.prompt.task` - Task description
- `llm.input` - Input text
- `llm.temperature` - Temperature setting

**llm.request.completed** - Emitted after successful execution
- `llm.request.id` - Unique request identifier
- `llm.synapse.type` - Type of synapse
- `llm.provider` - Provider name
- `llm.prompt.task` - Task description
- `llm.input` - Input text
- `llm.output` - Parsed result as JSON
- `llm.response` - Raw LLM response

**llm.request.failed** - Emitted on pipeline failure
- `llm.request.id` - Unique request identifier
- `llm.synapse.type` - Type of synapse
- `llm.provider` - Provider name
- `llm.prompt.task` - Task description
- `llm.error` - Error message

**llm.response.failed** - Emitted on parse/validation error
- `llm.request.id` - Unique request identifier
- `llm.synapse.type` - Type of synapse
- `llm.provider` - Provider name
- `llm.prompt.task` - Task description
- `llm.response` - Raw response that failed to parse
- `llm.error` - Error message
- `llm.error.type` - Error type ("parse_error", "validation_error")

**llm.provider.call.started** - Emitted before provider API call
- `llm.provider` - Provider name
- `llm.model` - Model to be used

**llm.provider.call.completed** - Emitted after successful provider call
- `llm.provider` - Provider name
- `llm.model` - Model used (actual model, may differ from requested)
- `llm.tokens.prompt` - Prompt token count
- `llm.tokens.completion` - Completion token count
- `llm.tokens.total` - Total token count
- `llm.duration.ms` - Provider call duration in milliseconds
- `llm.http.status.code` - HTTP status code (e.g., 200)
- `llm.response.id` - Provider's response ID for debugging
- `llm.response.created` - Server timestamp
- `llm.response.finish.reason` - Completion reason ("stop", "length", "content_filter", etc.)

**llm.provider.call.failed** - Emitted on provider API failure
- `llm.provider` - Provider name
- `llm.model` - Model that was requested
- `llm.http.status.code` - HTTP status code (e.g., 429, 500)
- `llm.duration.ms` - Time until failure
- `llm.error` - Error message
- `llm.api.error.type` - API error type (e.g., "rate_limit_error")
- `llm.api.error.code` - API error code (if provided)

### Usage Example

```go
import (
    "context"
    "log"
    "github.com/zoobzio/capitan"
    "github.com/zoobzio/zyn"
)

// Track token usage and costs
capitan.Hook(zyn.ProviderCallCompleted, func(ctx context.Context, e *capitan.Event) {
    model, _ := zyn.ModelKey.From(e)
    tokens, _ := zyn.TotalTokensKey.From(e)
    duration, _ := zyn.DurationMsKey.From(e)

    log.Printf("LLM Call: model=%s tokens=%d duration=%dms", model, tokens, duration)
})

// Log all request failures
capitan.Hook(zyn.RequestFailed, func(ctx context.Context, e *capitan.Event) {
    requestID, _ := zyn.RequestIDKey.From(e)
    synapseType, _ := zyn.SynapseTypeKey.From(e)
    err, _ := zyn.ErrorKey.From(e)

    log.Printf("Request failed: id=%s type=%s error=%s", requestID, synapseType, err)
})

// Observe all events for debugging
observer := capitan.Observe(func(ctx context.Context, e *capitan.Event) {
    log.Printf("Event: %s", e.Signal())
})
defer observer.Close()
```

All hooks fire asynchronously and include request correlation via `llm.request.id` for tracing complete request lifecycles.

## Testing

zyn includes a mock provider for testing:

```go
func TestMyFunction(t *testing.T) {
    mockProvider := zyn.NewMockProviderWithResponse(`{
        "decision": true,
        "confidence": 0.95,
        "reasoning": ["Valid email format", "Contains @ symbol"]
    }`)

    validator, err := zyn.Binary("Is this valid?", mockProvider)
    assert.NoError(t, err)

    session := zyn.NewSession()
    result, err := validator.Fire(context.Background(), session, "test@example.com")
    assert.NoError(t, err)
    assert.True(t, result)
}
```

## Documentation

Full documentation is available in the [docs/](./docs/) directory.

### Learn

- [Quickstart](./docs/2.learn/1.quickstart.md) - Build your first synapse in 10 minutes
- [Core Concepts](./docs/2.learn/2.concepts.md) - Understand synapses, sessions, and providers
- [Architecture](./docs/2.learn/3.architecture.md) - How zyn works under the hood

### Guides

- [Installation](./docs/3.guides/1.installation.md) - Installing and configuring zyn
- [Providers](./docs/3.guides/2.providers.md) - Configuring LLM providers
- [Sessions](./docs/3.guides/3.sessions.md) - Managing conversation context
- [Reliability](./docs/3.guides/4.reliability.md) - Retry, timeout, circuit breaker
- [Observability](./docs/3.guides/5.observability.md) - Monitoring with hooks
- [Testing](./docs/3.guides/6.testing.md) - Testing strategies
- [Best Practices](./docs/3.guides/7.best-practices.md) - Production guidelines

### Cookbook

- [Classification Workflows](./docs/4.cookbook/1.classification-workflows.md) - Real-world classification patterns
- [Extraction Pipelines](./docs/4.cookbook/2.extraction-pipelines.md) - Extract structured data
- [Multi-Turn Conversations](./docs/4.cookbook/3.multi-turn-conversations.md) - Complex workflows
- [Error Handling](./docs/4.cookbook/4.error-handling.md) - Robust error management

### Reference

- [Cheatsheet](./docs/5.reference/1.cheatsheet.md) - Quick reference for zyn API
- [Synapses](./docs/5.reference/2.synapses/) - All synapse types
- [Options](./docs/5.reference/3.options.md) - Configuration options
- [Session](./docs/5.reference/4.session.md) - Session API

## Development

```bash
# Clone the repository
git clone https://github.com/zoobzio/zyn.git
cd zyn

# Install tools
make install-tools

# Run tests
make test

# Run linters
make lint

# Generate coverage
make coverage

# Run all checks
make ci
```

## Contributing

See [CONTRIBUTING.md](CONTRIBUTING.md) for guidelines.

## Security

See [SECURITY.md](SECURITY.md) for security considerations and reporting vulnerabilities.

## License

MIT License. See [LICENSE](LICENSE) for details.

## Related Projects

- [pipz](https://github.com/zoobzio/pipz) - Composable pipeline framework for reliability patterns
