package zyn

import (
	"context"
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
