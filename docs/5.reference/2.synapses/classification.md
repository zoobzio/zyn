---
title: Classification Synapse
description: Categorize text into predefined classes
author: zoobzio
published: 2025-12-14
updated: 2025-12-14
tags:
  - reference
  - synapse
  - classification
---

# Classification Synapse

Categorize text into predefined classes.

## Constructor

```go
func Classification(task string, categories []string, provider Provider, opts ...Option) (*ClassificationSynapse, error)
```

**Parameters:**
- `task` - Description of what to classify
- `categories` - Valid category options
- `provider` - LLM provider
- `opts` - Optional configuration

**Returns:**
- `*ClassificationSynapse` - The configured synapse
- `error` - Configuration error

## Methods

### Fire

```go
func (s *ClassificationSynapse) Fire(ctx context.Context, session *Session, input string) (string, error)
```

Execute and return the primary category.

**Returns:**
- `string` - Primary category (one of the provided categories)
- `error` - Execution error

### FireWithInput

```go
func (s *ClassificationSynapse) FireWithInput(ctx context.Context, session *Session, input ClassificationInput) (string, error)
```

Execute with structured input including examples.

### FireWithDetails

```go
func (s *ClassificationSynapse) FireWithDetails(ctx context.Context, session *Session, input string) (*ClassificationResponse, error)
```

Execute and return full response.

## Types

```go
type ClassificationInput struct {
    Subject  string              // Text to classify
    Examples map[string][]string // Optional examples per category
}

type ClassificationResponse struct {
    Primary    string   `json:"primary"`
    Secondary  string   `json:"secondary,omitempty"`
    Confidence float64  `json:"confidence"`
    Reasoning  []string `json:"reasoning"`
}
```

## Examples

### Basic Usage

```go
classifier, _ := zyn.Classification(
    "What type of email is this?",
    []string{"spam", "urgent", "newsletter", "personal"},
    provider,
)
session := zyn.NewSession()

result, err := classifier.Fire(ctx, session, "URGENT: Your account will be suspended!")
// result: "urgent"
```

### With Examples

```go
input := zyn.ClassificationInput{
    Subject: "Great product, fast shipping!",
    Examples: map[string][]string{
        "positive": {"Love it!", "Excellent quality"},
        "negative": {"Terrible", "Waste of money"},
        "neutral":  {"It's okay", "As expected"},
    },
}

result, err := classifier.FireWithInput(ctx, session, input)
// result: "positive"
```

### With Details

```go
response, err := classifier.FireWithDetails(ctx, session, "Check out this amazing deal!")
// response.Primary: "spam"
// response.Secondary: "promotional"
// response.Confidence: 0.87
// response.Reasoning: ["Contains promotional language", "Urgency tactics"]
```

## Use Cases

- Email routing
- Support ticket triage
- Content categorization
- Sentiment analysis (simple)
- Intent detection
