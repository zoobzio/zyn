// Package tendo provides a zyn provider for local LLM inference via tendo.
package tendo

import (
	"strings"

	"github.com/zoobzio/zyn"
)

// TemplateType identifies the chat template format to use.
type TemplateType int

const (
	// TemplateAuto selects template based on model config.
	TemplateAuto TemplateType = iota
	// TemplateLlama3 uses Llama 3.x format with header tags.
	TemplateLlama3
	// TemplateChatML uses ChatML format (<|im_start|>/<|im_end|>).
	TemplateChatML
	// TemplateRaw concatenates message content without formatting.
	TemplateRaw
)

// Template converts zyn messages to a prompt string for the model.
type Template interface {
	Apply(messages []zyn.Message) string
}

// templateForModel returns the appropriate template for a model config.
func templateForModel(modelConfig string) Template {
	switch strings.ToLower(modelConfig) {
	case "llama3.2-1b", "llama3.2-3b", "tinyllama":
		return Llama3Template{}
	case "smollm-135m", "smollm-360m", "qwen2-0.5b":
		return ChatMLTemplate{}
	default:
		return ChatMLTemplate{}
	}
}

// templateForType returns a template instance for the given type.
func templateForType(t TemplateType) Template {
	switch t {
	case TemplateLlama3:
		return Llama3Template{}
	case TemplateChatML:
		return ChatMLTemplate{}
	case TemplateRaw:
		return RawTemplate{}
	default:
		return ChatMLTemplate{}
	}
}

// Llama3Template formats messages for Llama 3.x models.
type Llama3Template struct{}

// Apply converts messages to Llama 3 format.
func (Llama3Template) Apply(messages []zyn.Message) string {
	var b strings.Builder

	b.WriteString("<|begin_of_text|>")

	for _, msg := range messages {
		b.WriteString("<|start_header_id|>")
		b.WriteString(msg.Role)
		b.WriteString("<|end_header_id|>\n\n")
		b.WriteString(msg.Content)
		b.WriteString("<|eot_id|>")
	}

	// Prompt for assistant response
	b.WriteString("<|start_header_id|>assistant<|end_header_id|>\n\n")

	return b.String()
}

// ChatMLTemplate formats messages for ChatML-compatible models (Qwen2, SmolLM).
type ChatMLTemplate struct{}

// Apply converts messages to ChatML format.
func (ChatMLTemplate) Apply(messages []zyn.Message) string {
	var b strings.Builder

	for _, msg := range messages {
		b.WriteString("<|im_start|>")
		b.WriteString(msg.Role)
		b.WriteString("\n")
		b.WriteString(msg.Content)
		b.WriteString("<|im_end|>\n")
	}

	// Prompt for assistant response
	b.WriteString("<|im_start|>assistant\n")

	return b.String()
}

// RawTemplate concatenates message content without special formatting.
type RawTemplate struct{}

// Apply concatenates all message content with newlines.
func (RawTemplate) Apply(messages []zyn.Message) string {
	var b strings.Builder

	for i, msg := range messages {
		if i > 0 {
			b.WriteString("\n")
		}
		b.WriteString(msg.Content)
	}

	return b.String()
}
