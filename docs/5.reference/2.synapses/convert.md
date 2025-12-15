---
title: Convert Synapse
description: Convert between struct types
author: zoobzio
published: 2025-12-14
updated: 2025-12-14
tags:
  - reference
  - synapse
  - convert
---

# Convert Synapse

Convert between struct types with LLM intelligence.

## Constructor

```go
func Convert[TIn any, TOut Validator](task string, provider Provider, opts ...Option) (*ConvertSynapse[TIn, TOut], error)
```

**Type Parameters:**
- `TIn` - Input struct type
- `TOut` - Output struct type (must implement `Validator`)

**Parameters:**
- `task` - Description of the conversion
- `provider` - LLM provider
- `opts` - Optional configuration

**Returns:**
- `*ConvertSynapse[TIn, TOut]` - The configured synapse
- `error` - Configuration error

## Methods

### Fire

```go
func (s *ConvertSynapse[TIn, TOut]) Fire(ctx context.Context, session *Session, input TIn) (*TOut, error)
```

Execute and return converted struct.

**Returns:**
- `*TOut` - Converted and validated struct
- `error` - Execution or validation error

### FireWithDetails

```go
func (s *ConvertSynapse[TIn, TOut]) FireWithDetails(ctx context.Context, session *Session, input TIn) (*ConvertResponse[TOut], error)
```

Execute and return full response.

## Response Type

```go
type ConvertResponse[T any] struct {
    Data       T        `json:"data"`
    Confidence float64  `json:"confidence"`
    Reasoning  []string `json:"reasoning"`
}
```

## Examples

### Schema Migration

```go
type UserV1 struct {
    Name      string `json:"name"`
    Email     string `json:"email"`
    BirthDate string `json:"birth_date"` // "1990-05-15"
}

type UserV2 struct {
    FullName string `json:"full_name"`
    Contact  struct {
        Email string `json:"email"`
    } `json:"contact"`
    Age int `json:"age"`
}

func (u UserV2) Validate() error {
    if u.FullName == "" {
        return fmt.Errorf("full name required")
    }
    return nil
}

converter, _ := zyn.Convert[UserV1, UserV2]("migrate to v2 schema", provider)
session := zyn.NewSession()

v1User := UserV1{
    Name:      "John Doe",
    Email:     "john@example.com",
    BirthDate: "1990-05-15",
}

v2User, err := converter.Fire(ctx, session, v1User)
// v2User: &UserV2{
//     FullName: "John Doe",
//     Contact:  {Email: "john@example.com"},
//     Age:      34,  // Calculated from BirthDate
// }
```

### Format Conversion

```go
type CSVRow struct {
    Col1 string `json:"col1"`
    Col2 string `json:"col2"`
    Col3 string `json:"col3"`
}

type StructuredData struct {
    Name     string  `json:"name"`
    Quantity int     `json:"quantity"`
    Price    float64 `json:"price"`
}

func (s StructuredData) Validate() error {
    if s.Name == "" {
        return fmt.Errorf("name required")
    }
    return nil
}

converter, _ := zyn.Convert[CSVRow, StructuredData]("parse CSV columns", provider)

row := CSVRow{Col1: "Widget", Col2: "100", Col3: "$19.99"}
data, err := converter.Fire(ctx, session, row)
// data: &StructuredData{Name: "Widget", Quantity: 100, Price: 19.99}
```

### API Response Normalization

```go
type ExternalAPIResponse struct {
    StatusCode int               `json:"status_code"`
    Payload    map[string]any    `json:"payload"`
    Metadata   map[string]string `json:"metadata"`
}

type NormalizedResponse struct {
    Success bool   `json:"success"`
    Data    string `json:"data"`
    Error   string `json:"error,omitempty"`
}

func (n NormalizedResponse) Validate() error { return nil }

converter, _ := zyn.Convert[ExternalAPIResponse, NormalizedResponse](
    "normalize API response to standard format",
    provider,
)
```

## Use Cases

- Schema migrations
- API response normalization
- Data format conversion
- Legacy system integration
- Data enrichment/transformation
