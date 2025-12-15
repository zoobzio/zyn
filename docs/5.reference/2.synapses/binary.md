---
title: Binary Synapse
description: Yes/no decisions with confidence scores
author: zoobzio
published: 2025-12-14
updated: 2025-12-14
tags:
  - reference
  - synapse
  - binary
---

# Binary Synapse

Yes/no decisions with confidence scores.

## Constructor

```go
func Binary(task string, provider Provider, opts ...Option) (*BinarySynapse, error)
```

**Parameters:**
- `task` - Description of the decision to make
- `provider` - LLM provider
- `opts` - Optional configuration

**Returns:**
- `*BinarySynapse` - The configured synapse
- `error` - Configuration error

## Methods

### Fire

```go
func (s *BinarySynapse) Fire(ctx context.Context, session *Session, input string) (bool, error)
```

Execute the synapse and return the boolean decision.

**Parameters:**
- `ctx` - Context for cancellation/timeout
- `session` - Session for conversation context
- `input` - Text to evaluate

**Returns:**
- `bool` - The decision (true/false)
- `error` - Execution error

### FireWithDetails

```go
func (s *BinarySynapse) FireWithDetails(ctx context.Context, session *Session, input string) (*BinaryResponse, error)
```

Execute and return full response including confidence and reasoning.

**Returns:**
- `*BinaryResponse` - Full response
- `error` - Execution error

## Response Type

```go
type BinaryResponse struct {
    Decision   bool     `json:"decision"`
    Confidence float64  `json:"confidence"`
    Reasoning  []string `json:"reasoning"`
}
```

## Examples

### Basic Usage

```go
validator, _ := zyn.Binary("Is this a valid email address?", provider)
session := zyn.NewSession()

result, err := validator.Fire(ctx, session, "user@example.com")
// result: true
```

### With Details

```go
validator, _ := zyn.Binary("Is this a valid email address?", provider)
session := zyn.NewSession()

response, err := validator.FireWithDetails(ctx, session, "not-an-email")
// response.Decision: false
// response.Confidence: 0.95
// response.Reasoning: ["Missing @ symbol", "No domain present"]
```

### With Options

```go
validator, _ := zyn.Binary("Is this appropriate content?", provider,
    zyn.WithRetry(3),
    zyn.WithBackoff(3, 100*time.Millisecond),
)
```

## Use Cases

- Email validation
- Content moderation
- Feature flag decisions
- Approval workflows
- Data quality checks
