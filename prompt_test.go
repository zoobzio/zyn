package zyn

import (
	"strings"
	"testing"
)

func TestPrompt_Render(t *testing.T) {
	t.Run("simple", func(t *testing.T) {
		prompt := &Prompt{
			Task:   "test task",
			Input:  "test input",
			Schema: `{"field": "value"}`,
		}

		rendered := prompt.Render()
		if rendered == "" {
			t.Error("Render returned empty string")
		}
		if !strings.Contains(rendered, "test task") {
			t.Error("Rendered prompt missing task")
		}
	})

	t.Run("reliability", func(t *testing.T) {
		prompt := &Prompt{
			Task:        "test task",
			Input:       "test input",
			Schema:      `{"field": "value"}`,
			Context:     "test context",
			Constraints: []string{"constraint1", "constraint2"},
		}

		rendered := prompt.Render()
		if !strings.Contains(rendered, "test context") {
			t.Error("Rendered prompt missing context")
		}
		if !strings.Contains(rendered, "constraint1") {
			t.Error("Rendered prompt missing constraints")
		}
	})

	t.Run("chaining", func(t *testing.T) {
		prompt := &Prompt{
			Task:   "test task",
			Items:  []string{"item1", "item2"},
			Schema: `{"field": "value"}`,
		}

		rendered := prompt.Render()
		if !strings.Contains(rendered, "Items:") {
			t.Error("Rendered prompt should use Items instead of Input")
		}
		if !strings.Contains(rendered, "item1") {
			t.Error("Rendered prompt missing items")
		}
	})
}

func TestPrompt_Validate(t *testing.T) {
	t.Run("simple", func(t *testing.T) {
		prompt := &Prompt{
			Task:   "test task",
			Input:  "test input",
			Schema: `{"field": "value"}`,
		}

		err := prompt.Validate()
		if err != nil {
			t.Errorf("Valid prompt failed validation: %v", err)
		}
	})

	t.Run("reliability", func(t *testing.T) {
		tests := []struct {
			name   string
			prompt *Prompt
			errMsg string
		}{
			{
				name:   "missing task",
				prompt: &Prompt{Input: "test", Schema: "{}"},
				errMsg: "Task",
			},
			{
				name:   "missing input and items",
				prompt: &Prompt{Task: "test", Schema: "{}"},
				errMsg: "Input or Items",
			},
			{
				name:   "missing schema",
				prompt: &Prompt{Task: "test", Input: "test"},
				errMsg: "Schema",
			},
		}

		for _, tt := range tests {
			err := tt.prompt.Validate()
			if err == nil {
				t.Errorf("%s: expected error but got none", tt.name)
			} else if !strings.Contains(err.Error(), tt.errMsg) {
				t.Errorf("%s: expected error containing '%s', got '%s'", tt.name, tt.errMsg, err.Error())
			}
		}
	})

	t.Run("chaining", func(t *testing.T) {
		prompt := &Prompt{
			Task:   "test task",
			Items:  []string{"item1", "item2"},
			Schema: `{"field": "value"}`,
		}

		err := prompt.Validate()
		if err != nil {
			t.Errorf("Valid prompt with Items failed validation: %v", err)
		}
	})
}
