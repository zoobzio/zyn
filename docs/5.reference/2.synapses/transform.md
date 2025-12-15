---
title: Transform Synapse
description: Transform text to text
author: zoobzio
published: 2025-12-14
updated: 2025-12-14
tags:
  - reference
  - synapse
  - transform
---

# Transform Synapse

Transform text to text (translation, summarization, rewriting, etc.).

## Constructor

```go
func Transform(task string, provider Provider, opts ...Option) (*TransformSynapse, error)
```

**Parameters:**
- `task` - Description of the transformation
- `provider` - LLM provider
- `opts` - Optional configuration

**Returns:**
- `*TransformSynapse` - The configured synapse
- `error` - Configuration error

## Methods

### Fire

```go
func (s *TransformSynapse) Fire(ctx context.Context, session *Session, input string) (string, error)
```

Execute and return transformed text.

**Returns:**
- `string` - Transformed text
- `error` - Execution error

### FireWithDetails

```go
func (s *TransformSynapse) FireWithDetails(ctx context.Context, session *Session, input string) (*TransformResponse, error)
```

Execute and return full response.

## Response Type

```go
type TransformResponse struct {
    Output     string   `json:"output"`
    Confidence float64  `json:"confidence"`
    Changes    []string `json:"changes,omitempty"`
    Reasoning  []string `json:"reasoning"`
}
```

## Examples

### Translation

```go
translator, _ := zyn.Transform("Translate to Spanish", provider)
session := zyn.NewSession()

result, err := translator.Fire(ctx, session, "Hello, how are you?")
// result: "Hola, ¿cómo estás?"
```

### Summarization

```go
summarizer, _ := zyn.Transform("Summarize into 3 bullet points", provider)
session := zyn.NewSession()

result, err := summarizer.Fire(ctx, session, longArticle)
// result: "• Point 1\n• Point 2\n• Point 3"
```

### Rewriting

```go
rewriter, _ := zyn.Transform("Rewrite in formal business English", provider)
session := zyn.NewSession()

result, err := rewriter.Fire(ctx, session, "hey can u check this out asap?")
// result: "Could you please review this at your earliest convenience?"
```

### With Details

```go
response, err := transformer.FireWithDetails(ctx, session, input)
// response.Output: "transformed text"
// response.Confidence: 0.92
// response.Changes: ["Corrected grammar", "Improved clarity"]
// response.Reasoning: ["Applied formal tone", "Fixed punctuation"]
```

## Use Cases

- Translation
- Summarization
- Tone adjustment
- Grammar correction
- Format conversion (prose to bullets, etc.)
- Content expansion
