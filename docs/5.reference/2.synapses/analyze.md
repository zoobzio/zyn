---
title: Analyze Synapse
description: Analyze structured data and produce text analysis
author: zoobzio
published: 2025-12-14
updated: 2025-12-14
tags:
  - reference
  - synapse
  - analyze
---

# Analyze Synapse

Analyze structured data and produce text analysis.

## Constructor

```go
func Analyze[T any](task string, provider Provider, opts ...Option) (*AnalyzeSynapse[T], error)
```

**Type Parameter:**
- `T` - Struct type to analyze (no Validator required for input)

**Parameters:**
- `task` - Description of the analysis
- `provider` - LLM provider
- `opts` - Optional configuration

**Returns:**
- `*AnalyzeSynapse[T]` - The configured synapse
- `error` - Configuration error

## Methods

### Fire

```go
func (s *AnalyzeSynapse[T]) Fire(ctx context.Context, session *Session, input T) (string, error)
```

Execute and return analysis text.

**Returns:**
- `string` - Analysis text
- `error` - Execution error

### FireWithDetails

```go
func (s *AnalyzeSynapse[T]) FireWithDetails(ctx context.Context, session *Session, input T) (*AnalyzeResponse, error)
```

Execute and return full response.

## Response Type

```go
type AnalyzeResponse struct {
    Analysis   string   `json:"analysis"`
    Confidence float64  `json:"confidence"`
    Reasoning  []string `json:"reasoning"`
}
```

## Examples

### Basic Usage

```go
type ServerMetrics struct {
    CPU      float64 `json:"cpu_usage"`
    Memory   float64 `json:"memory_usage"`
    DiskIO   float64 `json:"disk_io"`
    Requests int     `json:"requests_per_second"`
}

analyzer, _ := zyn.Analyze[ServerMetrics]("system performance analysis", provider)
session := zyn.NewSession()

metrics := ServerMetrics{
    CPU:      85.5,
    Memory:   72.3,
    DiskIO:   45.0,
    Requests: 1200,
}

analysis, err := analyzer.Fire(ctx, session, metrics)
// analysis: "CPU usage is high at 85.5%, which may indicate a processing bottleneck.
//            Memory usage at 72.3% is within acceptable range but should be monitored.
//            Consider scaling horizontally to handle the 1200 req/s load."
```

### Financial Analysis

```go
type QuarterlyReport struct {
    Revenue    float64 `json:"revenue"`
    Expenses   float64 `json:"expenses"`
    Growth     float64 `json:"growth_percent"`
    NewClients int     `json:"new_clients"`
}

analyzer, _ := zyn.Analyze[QuarterlyReport]("business performance summary", provider)

report := QuarterlyReport{
    Revenue:    1500000,
    Expenses:   1200000,
    Growth:     12.5,
    NewClients: 45,
}

analysis, err := analyzer.Fire(ctx, session, report)
```

### With Context

```go
// Add historical context
session := zyn.NewSession()
session.Append(zyn.RoleUser, "Previous quarter had 8% growth and 30 new clients")

// Analysis will consider historical context
analysis, err := analyzer.Fire(ctx, session, currentReport)
// analysis references improvement from previous quarter
```

## Use Cases

- System monitoring summaries
- Financial report analysis
- Performance reviews
- Audit summaries
- Data interpretation
