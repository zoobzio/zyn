# zyn

[![CI Status](https://github.com/zoobzio/zyn/workflows/CI/badge.svg)](https://github.com/zoobzio/zyn/actions/workflows/ci.yml)
[![codecov](https://codecov.io/gh/zoobzio/zyn/graph/badge.svg?branch=main)](https://codecov.io/gh/zoobzio/zyn)
[![Go Report Card](https://goreportcard.com/badge/github.com/zoobzio/zyn)](https://goreportcard.com/report/github.com/zoobzio/zyn)
[![CodeQL](https://github.com/zoobzio/zyn/workflows/CodeQL/badge.svg)](https://github.com/zoobzio/zyn/security/code-scanning)
[![Go Reference](https://pkg.go.dev/badge/github.com/zoobzio/zyn.svg)](https://pkg.go.dev/github.com/zoobzio/zyn)
[![License](https://img.shields.io/github/license/zoobzio/zyn)](LICENSE)
[![Go Version](https://img.shields.io/github/go-mod/go-version/zoobzio/zyn)](go.mod)
[![Release](https://img.shields.io/github/v/release/zoobzio/zyn)](https://github.com/zoobzio/zyn/releases)

**Type-safe LLM orchestration framework with composable reliability patterns**

zyn provides a clean, type-safe way to orchestrate Large Language Model (LLM) interactions with built-in reliability patterns. Instead of fighting with prompt engineering and error handling, focus on your application logic.

## Features

- **8 Synapse Types** covering all LLM interaction patterns
- **Type-Safe Generics** with compile-time guarantees
- **5 Provider Implementations** (OpenAI, Anthropic, Google, Azure, Bedrock)
- **Built-in Reliability** via [pipz](https://github.com/zoobzio/pipz) integration
- **Zero Dependencies** for core library
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
    
    "github.com/zoobzio/zyn"
    "github.com/zoobzio/zyn/providers/openai"
)

func main() {
    // Create provider
    provider := openai.New(openai.Config{
        APIKey: os.Getenv("OPENAI_API_KEY"),
    })
    
    // Create synapse with reliability features
    classifier := zyn.Classification(
        "What type of email is this?",
        []string{"spam", "urgent", "newsletter", "personal"},
        provider,
        zyn.WithRetry(3),
        zyn.WithTimeout(10*time.Second),
    )
    
    // Use it
    ctx := context.Background()
    category, err := classifier.Fire(ctx, "URGENT: Your account will be suspended!")
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
validator := zyn.Binary("Is this a valid email address?", provider)
isValid, err := validator.Fire(ctx, "user@example.com")
// Returns: true
```

### Classification with Examples
```go
classifier := zyn.Classification("Classify sentiment", 
    []string{"positive", "negative", "neutral"}, provider)

input := zyn.ClassificationInput{
    Subject: "I love this product!",
    Examples: map[string][]string{
        "positive": {"Great!", "Amazing!"},
        "negative": {"Terrible", "Awful"},
    },
}

result, err := classifier.FireWithInput(ctx, input)
// Returns: ClassificationResponse{Primary: "positive", Confidence: 0.95, ...}
```

### Structured Data Extraction
```go
type Contact struct {
    Name  string `json:"name"`
    Email string `json:"email"`
    Phone string `json:"phone"`
}

extractor := zyn.Extract[Contact]("contact information", provider)
contact, err := extractor.Fire(ctx, "John Doe at john@example.com or (555) 123-4567")
// Returns: Contact{Name: "John Doe", Email: "john@example.com", Phone: "(555) 123-4567"}
```

### Text Transformation
```go
summarizer := zyn.Transform("summarize into key points", provider)
summary, err := summarizer.Fire(ctx, longArticle)

translator := zyn.Transform("translate to Spanish", provider)
spanish, err := translator.Fire(ctx, "Hello, how are you?")
```

### Data Analysis
```go
type ServerMetrics struct {
    CPU    float64 `json:"cpu_usage"`
    Memory float64 `json:"memory_usage"`
    Errors int     `json:"error_count"`
}

analyzer := zyn.Analyze[ServerMetrics]("performance analysis", provider)
analysis, err := analyzer.Fire(ctx, metrics)
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

converter := zyn.Convert[UserV1, UserV2]("migrate to v2 schema", provider)
v2User, err := converter.Fire(ctx, v1User)
```

## Providers

zyn supports multiple LLM providers:

### OpenAI
```go
provider := openai.New(openai.Config{
    APIKey: os.Getenv("OPENAI_API_KEY"),
    Model:  "gpt-4",
})
```

### Anthropic
```go
provider := anthropic.New(anthropic.Config{
    APIKey: os.Getenv("ANTHROPIC_API_KEY"),
    Model:  "claude-3-opus-20240229",
})
```

### Google Gemini
```go
provider := google.New(google.Config{
    APIKey: os.Getenv("GOOGLE_API_KEY"),
    Model:  "gemini-pro",
})
```

### Azure OpenAI
```go
provider := azure.New(azure.Config{
    Endpoint:   "https://your-resource.openai.azure.com",
    APIKey:     os.Getenv("AZURE_API_KEY"),
    Deployment: "gpt-4-deployment",
})
```

### AWS Bedrock
```go
provider := bedrock.New(bedrock.Config{
    Region:    "us-east-1",
    AccessKey: os.Getenv("AWS_ACCESS_KEY"),
    SecretKey: os.Getenv("AWS_SECRET_KEY"),
    Model:     "anthropic.claude-v2",
})
```

## Reliability Features

zyn integrates with [pipz](https://github.com/zoobzio/pipz) for composable reliability patterns:

```go
synapse := zyn.Binary("question", provider,
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

synapse := zyn.Binary("question", provider,
    zyn.WithErrorHandler(errorLogger),
)
```

## Testing

zyn includes a mock provider for testing:

```go
func TestMyFunction(t *testing.T) {
    mockProvider := zyn.NewMockProviderWithResponse(`{
        "decision": true,
        "confidence": 0.95,
        "reasoning": ["Valid email format", "Contains @ symbol"]
    }`)
    
    validator := zyn.Binary("Is this valid?", mockProvider)
    
    result, err := validator.Fire(context.Background(), "test@example.com")
    assert.NoError(t, err)
    assert.True(t, result)
}
```

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