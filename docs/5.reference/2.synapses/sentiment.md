---
title: Sentiment Synapse
description: Analyze emotional tone with detailed breakdown
author: zoobzio
published: 2025-12-14
updated: 2025-12-14
tags:
  - reference
  - synapse
  - sentiment
---

# Sentiment Synapse

Analyze emotional tone with detailed breakdown.

## Constructor

```go
func Sentiment(task string, provider Provider, opts ...Option) (*SentimentSynapse, error)
```

**Parameters:**
- `task` - Description of sentiment analysis focus
- `provider` - LLM provider
- `opts` - Optional configuration

**Returns:**
- `*SentimentSynapse` - The configured synapse
- `error` - Configuration error

## Methods

### Fire

```go
func (s *SentimentSynapse) Fire(ctx context.Context, session *Session, input string) (string, error)
```

Execute and return overall sentiment.

**Returns:**
- `string` - Overall sentiment ("positive", "negative", "neutral", "mixed")
- `error` - Execution error

### FireWithDetails

```go
func (s *SentimentSynapse) FireWithDetails(ctx context.Context, session *Session, input string) (*SentimentResponse, error)
```

Execute and return full response with scores and aspects.

## Response Type

```go
type SentimentResponse struct {
    Overall    string             `json:"overall"`
    Confidence float64            `json:"confidence"`
    Scores     SentimentScores    `json:"scores"`
    Emotions   []string           `json:"emotions,omitempty"`
    Aspects    map[string]string  `json:"aspects,omitempty"`
    Reasoning  []string           `json:"reasoning"`
}

type SentimentScores struct {
    Positive float64 `json:"positive"`
    Negative float64 `json:"negative"`
    Neutral  float64 `json:"neutral"`
}
```

## Examples

### Basic Usage

```go
analyzer, _ := zyn.Sentiment("Analyze customer feedback", provider)
session := zyn.NewSession()

result, err := analyzer.Fire(ctx, session, "I love this product! Best purchase ever!")
// result: "positive"
```

### With Details

```go
response, err := analyzer.FireWithDetails(ctx, session, "The product is great but shipping was slow")
// response.Overall: "mixed"
// response.Confidence: 0.85
// response.Scores: {Positive: 0.6, Negative: 0.3, Neutral: 0.1}
// response.Emotions: ["satisfaction", "frustration"]
// response.Aspects: {"product": "positive", "shipping": "negative"}
// response.Reasoning: ["Positive about product quality", "Negative about delivery time"]
```

### Aspect-Based Analysis

```go
analyzer, _ := zyn.Sentiment("Analyze sentiment by aspect: product, service, price", provider)

response, err := analyzer.FireWithDetails(ctx, session,
    "Great product, terrible customer service, but the price was fair")
// response.Aspects: {
//     "product": "positive",
//     "service": "negative",
//     "price": "neutral"
// }
```

## Use Cases

- Customer feedback analysis
- Social media monitoring
- Product review analysis
- Brand sentiment tracking
- Support ticket prioritization
