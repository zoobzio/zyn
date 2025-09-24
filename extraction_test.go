package zyn

import (
	"context"
	"reflect"
	"strings"
	"testing"
	"time"
)

// Test structs for extraction.
type ContactInfo struct {
	Name  string `json:"name"`
	Email string `json:"email"`
	Phone string `json:"phone"`
}

type Product struct {
	Name        string  `json:"name"`
	Price       float64 `json:"price"`
	InStock     bool    `json:"in_stock"`
	Description string  `json:"description"`
}

type Meeting struct {
	Title     string   `json:"title"`
	Date      string   `json:"date"`
	Time      string   `json:"time"`
	Attendees []string `json:"attendees"`
	Location  string   `json:"location"`
}

func TestExtractionBasic(t *testing.T) {
	provider := NewMockProviderWithResponse(`{
		"name": "John Doe",
		"email": "john@example.com",
		"phone": "(555) 123-4567"
	}`)

	extractor := Extract[ContactInfo]("contact information", provider, WithTimeout(5*time.Second))

	ctx := context.Background()
	contact, err := extractor.Fire(ctx, "John Doe can be reached at john@example.com or by phone at (555) 123-4567")

	if err != nil {
		t.Fatalf("Fire failed: %v", err)
	}
	if contact.Name != "John Doe" {
		t.Errorf("Expected name 'John Doe', got '%s'", contact.Name)
	}
	if contact.Email != "john@example.com" {
		t.Errorf("Expected email 'john@example.com', got '%s'", contact.Email)
	}
	if contact.Phone != "(555) 123-4567" {
		t.Errorf("Expected phone '(555) 123-4567', got '%s'", contact.Phone)
	}
}

func TestExtractionWithSlice(t *testing.T) {
	provider := NewMockProviderWithResponse(`{
		"title": "Team Standup",
		"date": "2024-01-15",
		"time": "10:00 AM",
		"attendees": ["Alice", "Bob", "Charlie"],
		"location": "Conference Room A"
	}`)

	extractor := Extract[Meeting]("meeting details", provider)

	ctx := context.Background()
	meeting, err := extractor.Fire(ctx, "Team Standup on January 15, 2024 at 10:00 AM in Conference Room A with Alice, Bob, and Charlie")

	if err != nil {
		t.Fatalf("Fire failed: %v", err)
	}
	if meeting.Title != "Team Standup" {
		t.Errorf("Expected title 'Team Standup', got '%s'", meeting.Title)
	}
	if len(meeting.Attendees) != 3 {
		t.Errorf("Expected 3 attendees, got %d", len(meeting.Attendees))
	}
	if meeting.Attendees[0] != "Alice" {
		t.Errorf("Expected first attendee 'Alice', got '%s'", meeting.Attendees[0])
	}
}

func TestExtractionWithNumbers(t *testing.T) {
	provider := NewMockProviderWithResponse(`{
		"name": "Laptop Pro X",
		"price": 1299.99,
		"in_stock": true,
		"description": "High-performance laptop with 16GB RAM"
	}`)

	extractor := Extract[Product]("product information", provider)

	ctx := context.Background()
	product, err := extractor.Fire(ctx, "The Laptop Pro X costs $1299.99 and is currently in stock. It's a high-performance laptop with 16GB RAM.")

	if err != nil {
		t.Fatalf("Fire failed: %v", err)
	}
	if product.Price != 1299.99 {
		t.Errorf("Expected price 1299.99, got %f", product.Price)
	}
	if !product.InStock {
		t.Error("Expected in_stock to be true")
	}
}

func TestExtractionSliceOfStrings(t *testing.T) {
	provider := NewMockProviderWithResponse(`["email1@example.com", "email2@test.org", "email3@company.net"]`)

	extractor := Extract[[]string]("email addresses", provider)

	ctx := context.Background()
	emails, err := extractor.Fire(ctx, "Contact us at email1@example.com, email2@test.org, or email3@company.net")

	if err != nil {
		t.Fatalf("Fire failed: %v", err)
	}
	if len(emails) != 3 {
		t.Errorf("Expected 3 emails, got %d", len(emails))
	}
	if emails[0] != "email1@example.com" {
		t.Errorf("Expected first email 'email1@example.com', got '%s'", emails[0])
	}
}

func TestExtractionPromptStructure(t *testing.T) {
	var capturedPrompt string
	provider := NewMockProviderWithCallback(func(prompt string, _ float32) (string, error) {
		capturedPrompt = prompt
		return `{"name": "test", "email": "test@example.com", "phone": "123"}`, nil
	})

	extractor := Extract[ContactInfo]("contacts", provider)

	ctx := context.Background()
	_, err := extractor.Fire(ctx, "test input")
	if err != nil {
		t.Fatalf("Fire failed: %v", err)
	}

	// Check prompt structure
	if !strings.Contains(capturedPrompt, "Task: Extract contacts") {
		t.Error("Prompt missing task description")
	}
	if !strings.Contains(capturedPrompt, "Input: test input") {
		t.Error("Prompt missing input")
	}
	if !strings.Contains(capturedPrompt, "Return JSON:") {
		t.Error("Prompt missing JSON structure section")
	}
	// Check for schema fields
	if !strings.Contains(capturedPrompt, `"name"`) {
		t.Error("Schema missing name field")
	}
	if !strings.Contains(capturedPrompt, `"email"`) {
		t.Error("Schema missing email field")
	}
	if !strings.Contains(capturedPrompt, `"phone"`) {
		t.Error("Schema missing phone field")
	}
}

func TestExtractionWithContext(t *testing.T) {
	provider := NewMockProviderWithResponse(`{
		"name": "Support Team",
		"email": "support@company.com",
		"phone": "(800) 555-1234"
	}`)

	extractor := Extract[ContactInfo]("technical support contact", provider)

	input := ExtractionInput{
		Text:    "For technical issues, reach out to our Support Team at support@company.com or call (800) 555-1234",
		Context: "Looking for technical support information only, not sales contacts",
	}

	ctx := context.Background()
	contact, err := extractor.FireWithInput(ctx, input)

	if err != nil {
		t.Fatalf("FireWithInput failed: %v", err)
	}
	if contact.Name != "Support Team" {
		t.Errorf("Expected name 'Support Team', got '%s'", contact.Name)
	}
}

// Test schema generation for various types.
func TestSchemaGeneration(t *testing.T) {
	tests := []struct {
		name     string
		typ      reflect.Type
		contains []string
	}{
		{
			name:     "Simple struct",
			typ:      reflect.TypeOf(ContactInfo{}),
			contains: []string{`"name"`, `"email"`, `"phone"`},
		},
		{
			name:     "Struct with slice",
			typ:      reflect.TypeOf(Meeting{}),
			contains: []string{`"title"`, `"attendees"`, `[`},
		},
		{
			name:     "Slice of strings",
			typ:      reflect.TypeOf([]string{}),
			contains: []string{`[`, `"string"`},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			schema := generateJSONSchema(tt.typ)
			for _, expected := range tt.contains {
				if !strings.Contains(schema, expected) {
					t.Errorf("Schema missing expected content '%s'\nGot: %s", expected, schema)
				}
			}
		})
	}
}
