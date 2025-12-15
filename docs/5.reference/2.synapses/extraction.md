---
title: Extraction Synapse
description: Extract structured data from text
author: zoobzio
published: 2025-12-14
updated: 2025-12-14
tags:
  - reference
  - synapse
  - extraction
---

# Extraction Synapse

Extract structured data from unstructured text.

## Constructor

```go
func Extract[T Validator](task string, provider Provider, opts ...Option) (*ExtractionSynapse[T], error)
```

**Type Parameter:**
- `T` - Struct type to extract (must implement `Validator`)

**Parameters:**
- `task` - Description of what to extract
- `provider` - LLM provider
- `opts` - Optional configuration

**Returns:**
- `*ExtractionSynapse[T]` - The configured synapse
- `error` - Configuration error

## Methods

### Fire

```go
func (s *ExtractionSynapse[T]) Fire(ctx context.Context, session *Session, text string) (T, error)
```

Execute and return extracted struct.

**Returns:**
- `T` - Extracted and validated struct
- `error` - Execution or validation error

### FireWithInput

```go
func (s *ExtractionSynapse[T]) FireWithInput(ctx context.Context, session *Session, input ExtractionInput) (T, error)
```

Execute with rich input structure.

### WithDefaults

```go
func (s *ExtractionSynapse[T]) WithDefaults(defaults ExtractionInput) *ExtractionSynapse[T]
```

Set default input values that are merged with user input at execution time.

## Input Type

```go
type ExtractionInput struct {
    Text        string  // The text to extract from
    Context     string  // Additional context
    Examples    string  // Example extractions (newline-separated)
    Temperature float32 // LLM temperature setting
}
```

## Examples

### Basic Usage

```go
type Contact struct {
    Name  string `json:"name"`
    Email string `json:"email"`
    Phone string `json:"phone,omitempty"`
}

func (c Contact) Validate() error {
    if c.Name == "" {
        return fmt.Errorf("name required")
    }
    if c.Email == "" {
        return fmt.Errorf("email required")
    }
    return nil
}

extractor, _ := zyn.Extract[Contact]("contact information", provider)
session := zyn.NewSession()

contact, err := extractor.Fire(ctx, session, "John Doe at john@example.com, call (555) 123-4567")
// contact: Contact{Name: "John Doe", Email: "john@example.com", Phone: "(555) 123-4567"}
```

### With Nested Structs

```go
type Address struct {
    Street  string `json:"street"`
    City    string `json:"city"`
    Country string `json:"country"`
}

type Company struct {
    Name    string  `json:"name"`
    Address Address `json:"address"`
}

func (c Company) Validate() error {
    if c.Name == "" {
        return fmt.Errorf("company name required")
    }
    return nil
}

extractor, _ := zyn.Extract[Company]("company information", provider)
```

### With JSON Tags

Use `json` tags to control field names and optionality:

```go
type Order struct {
    ID          string  `json:"order_id"`           // Maps to "order_id" in JSON
    Amount      float64 `json:"amount"`
    Description string  `json:"description,omitempty"` // Optional
}
```

### With Description Tags

Use `description` tags for LLM guidance:

```go
type Product struct {
    Name  string  `json:"name" description:"Product name without brand"`
    Price float64 `json:"price" description:"Price in USD"`
    SKU   string  `json:"sku" description:"Stock keeping unit code"`
}
```

## Validation

The `Validate()` method is called automatically after parsing:

```go
func (o Order) Validate() error {
    if o.ID == "" {
        return fmt.Errorf("order ID required")
    }
    if o.Amount <= 0 {
        return fmt.Errorf("amount must be positive")
    }
    if o.Amount > 1000000 {
        return fmt.Errorf("amount exceeds maximum")
    }
    return nil
}
```

If validation fails, `Fire()` returns an error and the session is not updated.

## Use Cases

- Contact extraction
- Invoice parsing
- Resume/CV parsing
- Product information extraction
- Event details extraction
