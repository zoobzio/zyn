---
title: Ranking Synapse
description: Order items by specified criteria
author: zoobzio
published: 2025-12-14
updated: 2025-12-14
tags:
  - reference
  - synapse
  - ranking
---

# Ranking Synapse

Order items by specified criteria.

## Constructor

```go
func Ranking(criteria string, provider Provider, opts ...Option) (*RankingSynapse, error)
```

**Parameters:**
- `criteria` - How to rank items (e.g., "most relevant to least relevant")
- `provider` - LLM provider
- `opts` - Optional configuration

**Returns:**
- `*RankingSynapse` - The configured synapse
- `error` - Configuration error

## Methods

### Fire

```go
func (s *RankingSynapse) Fire(ctx context.Context, session *Session, items []string) ([]string, error)
```

Execute and return ranked items.

**Parameters:**
- `items` - Slice of items to rank

**Returns:**
- `[]string` - Items in ranked order
- `error` - Execution error

### FireWithDetails

```go
func (s *RankingSynapse) FireWithDetails(ctx context.Context, session *Session, items []string) (*RankingResponse, error)
```

Execute and return full response.

## Response Type

```go
type RankingResponse struct {
    Ranked     []string `json:"ranked"`
    Confidence float64  `json:"confidence"`
    Reasoning  []string `json:"reasoning"`
}
```

## Examples

### Basic Usage

```go
ranker, _ := zyn.Ranking("most healthy to least healthy", provider)
session := zyn.NewSession()

items := []string{"apple", "candy bar", "salad", "soda"}
result, err := ranker.Fire(ctx, session, items)
// result: ["salad", "apple", "candy bar", "soda"]
```

### Relevance Ranking

```go
ranker, _ := zyn.Ranking("most relevant to the query", provider)

// Add context first
session := zyn.NewSession()
session.Append(zyn.RoleUser, "I'm looking for a family car")

items := []string{"sports car", "minivan", "SUV", "motorcycle"}
result, err := ranker.Fire(ctx, session, items)
// result: ["minivan", "SUV", "sports car", "motorcycle"]
```

### With Details

```go
response, err := ranker.FireWithDetails(ctx, session, items)
// response.Ranked: ["salad", "apple", "candy bar", "soda"]
// response.Confidence: 0.92
// response.Reasoning: [
//     "Salad is most nutritious with vegetables and fiber",
//     "Apple is a natural fruit with vitamins",
//     "Candy bar has high sugar but some energy",
//     "Soda has no nutritional value and high sugar"
// ]
```

## Use Cases

- Search result ordering
- Task prioritization
- Recommendation ranking
- Document relevance
- Option comparison
