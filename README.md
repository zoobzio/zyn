# zyn

[![CI Status](https://github.com/zoobzio/zyn/workflows/CI/badge.svg)](https://github.com/zoobzio/zyn/actions/workflows/ci.yml)
[![codecov](https://codecov.io/gh/zoobzio/zyn/graph/badge.svg?branch=main)](https://codecov.io/gh/zoobzio/zyn)
[![Go Report Card](https://goreportcard.com/badge/github.com/zoobzio/zyn)](https://goreportcard.com/report/github.com/zoobzio/zyn)
[![CodeQL](https://github.com/zoobzio/zyn/workflows/CodeQL/badge.svg)](https://github.com/zoobzio/zyn/security/code-scanning)
[![Go Reference](https://pkg.go.dev/badge/github.com/zoobzio/zyn.svg)](https://pkg.go.dev/github.com/zoobzio/zyn)
[![License](https://img.shields.io/github/license/zoobzio/zyn)](LICENSE)
[![Go Version](https://img.shields.io/github/go-mod/go-version/zoobzio/zyn)](go.mod)
[![Release](https://img.shields.io/github/v/release/zoobzio/zyn)](https://github.com/zoobzio/zyn/releases)

Type-safe LLM orchestration for Go.

Define synapses with typed outputs, fire them with sessions, and get structured responses with built-in reliability.

## Composable Thinking Synapses

Synapses wrap LLM interactions with compile-time type safety.

```go
type Contact struct {
    Name  string `json:"name"`
    Email string `json:"email"`
    Phone string `json:"phone"`
}

func (c Contact) Validate() error {
    if c.Email == "" {
        return errors.New("email required")
    }
    return nil
}

// Define a synapse — typed extraction from unstructured text
extractor, _ := zyn.Extract[Contact]("contact information", provider)

// Fire it — get structured data back
session := zyn.NewSession()
contact, _ := extractor.Fire(ctx, session, "Reach John at john@acme.com or 555-1234")
// contact.Name  → "John"
// contact.Email → "john@acme.com"
// contact.Phone → "555-1234"
```

Sessions carry conversation context. Synapses stay focused on their task.

```go
// Chain synapses — each sees the full conversation history
classifier, _ := zyn.Classification("urgency", []string{"low", "medium", "high"}, provider)
urgency, _ := classifier.Fire(ctx, session, contact.Name + "'s request")

responder, _ := zyn.Transform("write customer response", provider)
response, _ := responder.Fire(ctx, session, fmt.Sprintf("Urgency: %s", urgency))
```

Type-safe at the edges. Conversational in between.

## Install

```bash
go get github.com/zoobzio/zyn
```

Requires Go 1.24+.

## Quick Start

```go
package main

import (
    "context"
    "fmt"
    "os"
    "time"

    "github.com/zoobzio/zyn"
    "github.com/zoobzio/zyn/openai"
)

func main() {
    ctx := context.Background()

    // Create provider
    provider := openai.New(openai.Config{
        APIKey: os.Getenv("OPENAI_API_KEY"),
    })

    // Create synapse with reliability
    classifier, _ := zyn.Classification(
        "email category",
        []string{"spam", "urgent", "newsletter", "personal"},
        provider,
        zyn.WithRetry(3),
        zyn.WithTimeout(10*time.Second),
    )

    // Fire with session context
    session := zyn.NewSession()
    category, _ := classifier.Fire(ctx, session, "URGENT: Your account will be suspended!")

    fmt.Println("Category:", category) // "urgent"
}
```

## Capabilities

| Feature              | Description                                                                      | Docs                                              |
| -------------------- | -------------------------------------------------------------------------------- | ------------------------------------------------- |
| 8 Synapse Types      | Binary, Classification, Ranking, Sentiment, Extract, Transform, Analyze, Convert | [Synapses](docs/5.reference/2.synapses/)          |
| Sessions             | Conversation context across synapse calls                                        | [Sessions](docs/3.guides/3.sessions.md)           |
| Structured Prompts   | Type-driven prompt generation prevents divergence                                | [Concepts](docs/2.learn/2.concepts.md)            |
| Reliability Patterns | Retry, timeout, circuit breaker, rate limiting                                   | [Reliability](docs/3.guides/4.reliability.md)     |
| Observability        | Typed signals via capitan for all LLM operations                                 | [Observability](docs/3.guides/5.observability.md) |
| Testing Utilities    | Mock provider for deterministic tests                                            | [Testing](docs/3.guides/6.testing.md)             |

## Why zyn?

- **Type-safe** — Generics enforce output types at compile time
- **Structured** — LLM responses parse directly into your structs
- **Conversational** — Sessions maintain context across synapse calls
- **Reliable** — [pipz](https://github.com/zoobzio/pipz) patterns built in
- **Observable** — [capitan](https://github.com/zoobzio/capitan) signals for every LLM call
- **Testable** — Mock provider for deterministic unit tests

## Composable LLM Patterns

Zyn enables a pattern: **define synapses, compose with sessions, observe with signals**.

Your synapses define typed LLM interactions. Sessions chain them into workflows with shared context. Reliability patterns wrap the whole thing. Capitan signals make it observable.

```go
// Define synapses for each step
extractor, _ := zyn.Extract[Customer]("customer details", provider)
classifier, _ := zyn.Classification("urgency", []string{"low", "high"}, provider)
responder, _ := zyn.Transform("write response", provider)

// Compose via session — each step sees previous context
session := zyn.NewSession()
customer, _ := extractor.Fire(ctx, session, ticket)
urgency, _ := classifier.Fire(ctx, session, customer.Issue)
response, _ := responder.Fire(ctx, session, urgency)

// Observe via capitan — no instrumentation needed
capitan.Hook(zyn.RequestCompleted, logRequest)
capitan.Hook(zyn.ProviderCallCompleted, trackTokens)
```

Three synapses, one session, full observability.

## Documentation

- [Overview](docs/1.overview.md) — Design philosophy

### Learn

- [Quickstart](docs/2.learn/1.quickstart.md) — Build your first synapse
- [Core Concepts](docs/2.learn/2.concepts.md) — Synapses, sessions, providers
- [Architecture](docs/2.learn/3.architecture.md) — How zyn works under the hood

### Guides

- [Installation](docs/3.guides/1.installation.md) — Installing and configuring
- [Providers](docs/3.guides/2.providers.md) — LLM provider configuration
- [Sessions](docs/3.guides/3.sessions.md) — Managing conversation context
- [Reliability](docs/3.guides/4.reliability.md) — Retry, timeout, circuit breaker
- [Observability](docs/3.guides/5.observability.md) — Monitoring with capitan
- [Testing](docs/3.guides/6.testing.md) — Testing strategies
- [Best Practices](docs/3.guides/7.best-practices.md) — Production guidelines

### Cookbook

- [Classification Workflows](docs/4.cookbook/1.classification-workflows.md) — Real-world classification
- [Extraction Pipelines](docs/4.cookbook/2.extraction-pipelines.md) — Structured data extraction
- [Multi-Turn Conversations](docs/4.cookbook/3.multi-turn-conversations.md) — Complex workflows
- [Error Handling](docs/4.cookbook/4.error-handling.md) — Robust error management

### Reference

- [Cheatsheet](docs/5.reference/1.cheatsheet.md) — Quick reference
- [Synapses](docs/5.reference/2.synapses/) — All synapse types
- [Options](docs/5.reference/3.options.md) — Configuration options
- [Session](docs/5.reference/4.session.md) — Session API

## Contributing

See [CONTRIBUTING.md](CONTRIBUTING.md) for guidelines. Run `make help` for available commands.

## License

MIT License — see [LICENSE](LICENSE) for details.
