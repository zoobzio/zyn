package zyn

import (
	"context"
	"strings"
	"testing"
	"time"
)

// Test structs for conversion.
type UserV1 struct {
	Name     string `json:"name"`
	Email    string `json:"email"`
	Age      int    `json:"age"`
	Location string `json:"location"`
}

type UserV2 struct {
	FullName string `json:"full_name"`
	Contact  struct {
		Email string `json:"email"`
	} `json:"contact"`
	Demographics struct {
		Age      int    `json:"age"`
		Location string `json:"location"`
	} `json:"demographics"`
}

type APIProduct struct {
	ID          string  `json:"product_id"`
	Name        string  `json:"product_name"`
	Price       float64 `json:"price_usd"`
	InStock     bool    `json:"available"`
	Description string  `json:"desc"`
}

type DBProduct struct {
	ID           int    `json:"id"`
	Name         string `json:"name"`
	PriceCents   int    `json:"price_cents"`
	Availability string `json:"availability"`
	Description  string `json:"description"`
}

type ExternalEvent struct {
	EventType string            `json:"type"`
	Timestamp string            `json:"timestamp"`
	Data      map[string]string `json:"data"`
}

type InternalEvent struct {
	Type       string    `json:"event_type"`
	OccurredAt time.Time `json:"occurred_at"`
	UserID     string    `json:"user_id"`
	Action     string    `json:"action"`
	Metadata   string    `json:"metadata"`
}

func TestConvertUserSchema(t *testing.T) {
	provider := NewMockProviderWithResponse(`{
		"full_name": "John Doe",
		"contact": {
			"email": "john@example.com"
		},
		"demographics": {
			"age": 30,
			"location": "New York"
		}
	}`)

	converter := Convert[UserV1, UserV2]("migrate to v2 schema", provider, WithTimeout(5*time.Second))

	v1User := UserV1{
		Name:     "John Doe",
		Email:    "john@example.com",
		Age:      30,
		Location: "New York",
	}

	ctx := context.Background()
	v2User, err := converter.Fire(ctx, v1User)
	if err != nil {
		t.Fatalf("Fire failed: %v", err)
	}

	if v2User.FullName != "John Doe" {
		t.Errorf("Expected full_name 'John Doe', got '%s'", v2User.FullName)
	}
	if v2User.Contact.Email != "john@example.com" {
		t.Errorf("Expected email 'john@example.com', got '%s'", v2User.Contact.Email)
	}
	if v2User.Demographics.Age != 30 {
		t.Errorf("Expected age 30, got %d", v2User.Demographics.Age)
	}
	if v2User.Demographics.Location != "New York" {
		t.Errorf("Expected location 'New York', got '%s'", v2User.Demographics.Location)
	}
}

func TestConvertAPIToDatabase(t *testing.T) {
	provider := NewMockProviderWithResponse(`{
		"id": 12345,
		"name": "Premium Widget",
		"price_cents": 299900,
		"availability": "in_stock",
		"description": "High-quality widget for professionals"
	}`)

	mapper := Convert[APIProduct, DBProduct]("map API response to database model", provider)

	apiProduct := APIProduct{
		ID:          "prod_12345",
		Name:        "Premium Widget",
		Price:       2999.00,
		InStock:     true,
		Description: "High-quality widget for professionals",
	}

	ctx := context.Background()
	dbProduct, err := mapper.Fire(ctx, apiProduct)
	if err != nil {
		t.Fatalf("Fire failed: %v", err)
	}

	if dbProduct.ID != 12345 {
		t.Errorf("Expected ID 12345, got %d", dbProduct.ID)
	}
	if dbProduct.PriceCents != 299900 {
		t.Errorf("Expected price_cents 299900, got %d", dbProduct.PriceCents)
	}
	if dbProduct.Availability != "in_stock" {
		t.Errorf("Expected availability 'in_stock', got '%s'", dbProduct.Availability)
	}
}

func TestConvertWithRules(t *testing.T) {
	provider := NewMockProviderWithResponse(`{
		"id": 999,
		"name": "Special Offer Item",
		"price_cents": 1000,
		"availability": "limited",
		"description": "Flash sale: Limited time offer"
	}`)

	mapper := Convert[APIProduct, DBProduct]("apply pricing rules", provider)

	apiProduct := APIProduct{
		ID:          "flash_999",
		Name:        "Special Offer Item",
		Price:       19.99,
		InStock:     true,
		Description: "Limited time offer",
	}

	input := ConvertInput[APIProduct]{
		Data:  apiProduct,
		Rules: "Apply 50% discount for flash sale items, mark as 'limited' availability",
	}

	ctx := context.Background()
	dbProduct, err := mapper.FireWithInput(ctx, input)
	if err != nil {
		t.Fatalf("FireWithInput failed: %v", err)
	}

	// Should apply discount rule
	if dbProduct.PriceCents != 1000 {
		t.Errorf("Expected discounted price_cents ~1000, got %d", dbProduct.PriceCents)
	}
	if dbProduct.Availability != "limited" {
		t.Errorf("Expected availability 'limited', got '%s'", dbProduct.Availability)
	}
}

func TestConvertEventFormat(t *testing.T) {
	// Mock a time for consistent testing
	provider := NewMockProviderWithResponse(`{
		"event_type": "user_login",
		"occurred_at": "2024-01-15T10:30:00Z",
		"user_id": "12345",
		"action": "login",
		"metadata": "{\"ip\":\"192.168.1.1\",\"browser\":\"Chrome\"}"
	}`)

	normalizer := Convert[ExternalEvent, InternalEvent]("normalize event format", provider)

	externalEvent := ExternalEvent{
		EventType: "user.login",
		Timestamp: "2024-01-15T10:30:00Z",
		Data: map[string]string{
			"user_id": "12345",
			"ip":      "192.168.1.1",
			"browser": "Chrome",
		},
	}

	ctx := context.Background()
	internalEvent, err := normalizer.Fire(ctx, externalEvent)
	if err != nil {
		t.Fatalf("Fire failed: %v", err)
	}

	if internalEvent.Type != "user_login" {
		t.Errorf("Expected type 'user_login', got '%s'", internalEvent.Type)
	}
	if internalEvent.UserID != "12345" {
		t.Errorf("Expected user_id '12345', got '%s'", internalEvent.UserID)
	}
	if internalEvent.Action != "login" {
		t.Errorf("Expected action 'login', got '%s'", internalEvent.Action)
	}
	if !strings.Contains(internalEvent.Metadata, "192.168.1.1") {
		t.Errorf("Expected metadata to contain IP address")
	}
}

func TestConvertWithContext(t *testing.T) {
	provider := NewMockProviderWithResponse(`{
		"full_name": "Dr. Jane Smith",
		"contact": {
			"email": "dr.smith@hospital.org"
		},
		"demographics": {
			"age": 45,
			"location": "Medical District, Chicago"
		}
	}`)

	converter := Convert[UserV1, UserV2]("enhance with context", provider)

	v1User := UserV1{
		Name:     "Jane Smith",
		Email:    "dr.smith@hospital.org",
		Age:      45,
		Location: "Chicago",
	}

	input := ConvertInput[UserV1]{
		Data:    v1User,
		Context: "This is a medical professional, add appropriate title and district info",
	}

	ctx := context.Background()
	v2User, err := converter.FireWithInput(ctx, input)
	if err != nil {
		t.Fatalf("FireWithInput failed: %v", err)
	}

	if !strings.Contains(v2User.FullName, "Dr.") {
		t.Errorf("Expected title in full name, got '%s'", v2User.FullName)
	}
	if !strings.Contains(v2User.Demographics.Location, "Medical") {
		t.Errorf("Expected medical district in location, got '%s'", v2User.Demographics.Location)
	}
}

func TestConvertPromptStructure(t *testing.T) {
	var capturedPrompt string
	provider := NewMockProviderWithCallback(func(prompt string, _ float32) (string, error) {
		capturedPrompt = prompt
		return `{"full_name": "test", "contact": {"email": "test@test.com"}, "demographics": {"age": 0, "location": "test"}}`, nil
	})

	converter := Convert[UserV1, UserV2]("test conversion", provider)

	ctx := context.Background()
	_, err := converter.Fire(ctx, UserV1{Name: "test"})
	if err != nil {
		t.Fatalf("Fire failed: %v", err)
	}

	// Check prompt structure
	if !strings.Contains(capturedPrompt, "Task: Convert: test conversion") {
		t.Error("Prompt missing task description")
	}
	if !strings.Contains(capturedPrompt, "Input:") {
		t.Error("Prompt missing input section")
	}
	if !strings.Contains(capturedPrompt, `"name": "test"`) {
		t.Error("Prompt missing input JSON")
	}
	if !strings.Contains(capturedPrompt, "Response JSON Schema:") {
		t.Error("Prompt missing JSON schema")
	}
	// Check for output schema structure
	if !strings.Contains(capturedPrompt, "full_name") {
		t.Error("Schema missing full_name field")
	}
	if !strings.Contains(capturedPrompt, "demographics") {
		t.Error("Schema missing demographics field")
	}
	if !strings.Contains(capturedPrompt, "Constraints:") {
		t.Error("Prompt missing constraints")
	}
}
