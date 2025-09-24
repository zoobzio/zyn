package zyn

import (
	"fmt"
	"strings"
)

// Prompt represents a structured LLM prompt with consistent formatting.
// It enforces a canonical structure across all synapse types.
type Prompt struct {
	Task        string              // Required: what the LLM should do
	Input       string              // Required: the main content to process
	Context     string              // Optional: additional context
	Categories  []string            // For classification synapses
	Items       []string            // For ranking synapses
	Aspects     []string            // For sentiment analysis
	Examples    map[string][]string // Category->examples for classification
	Schema      string              // Required: JSON schema for response
	Constraints []string            // Required: rules and constraints
}

// Render converts the structured prompt to a string for the LLM.
// It enforces consistent ordering and formatting across all synapses.
func (p *Prompt) Render() string {
	var sections []string

	// Task is always first
	if p.Task != "" {
		sections = append(sections, "Task: "+p.Task)
	}

	// Input is always second
	if p.Input != "" {
		sections = append(sections, "Input: "+p.Input)
	}

	// Optional context
	if p.Context != "" {
		sections = append(sections, "Context: "+p.Context)
	}

	// Categories (for classification)
	if len(p.Categories) > 0 {
		cat := "Categories:\n"
		for i, c := range p.Categories {
			cat += fmt.Sprintf("  %d. %s\n", i+1, c)
		}
		sections = append(sections, strings.TrimSpace(cat))
	}

	// Items (for ranking)
	if len(p.Items) > 0 {
		items := "Items:\n"
		for i, item := range p.Items {
			items += fmt.Sprintf("  %d. %s\n", i+1, item)
		}
		sections = append(sections, strings.TrimSpace(items))
	}

	// Aspects (for sentiment)
	if len(p.Aspects) > 0 {
		aspects := "Aspects:\n"
		for i, aspect := range p.Aspects {
			aspects += fmt.Sprintf("  %d. %s\n", i+1, aspect)
		}
		sections = append(sections, strings.TrimSpace(aspects))
	}

	// Examples (if provided)
	if len(p.Examples) > 0 {
		examples := "Examples:\n"
		for category, exs := range p.Examples {
			if len(exs) > 0 {
				examples += fmt.Sprintf("  %s:\n", category)
				for _, ex := range exs {
					examples += fmt.Sprintf("    - %s\n", ex)
				}
			}
		}
		sections = append(sections, strings.TrimSpace(examples))
	}

	// Schema - always required
	if p.Schema != "" {
		sections = append(sections, "Return JSON:\n"+p.Schema)
	}

	// Constraints - always last
	if len(p.Constraints) > 0 {
		con := "Constraints:\n"
		for _, c := range p.Constraints {
			con += "- " + c + "\n"
		}
		sections = append(sections, strings.TrimSpace(con))
	}

	return strings.Join(sections, "\n\n")
}

// Validate checks if the prompt has required fields.
func (p *Prompt) Validate() error {
	if p.Task == "" {
		return fmt.Errorf("prompt missing required Task field")
	}
	if p.Input == "" && len(p.Items) == 0 {
		return fmt.Errorf("prompt missing required Input or Items field")
	}
	if p.Schema == "" {
		return fmt.Errorf("prompt missing required Schema field")
	}
	return nil
}
