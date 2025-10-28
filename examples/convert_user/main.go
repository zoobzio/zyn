package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/zoobzio/zyn"
	"github.com/zoobzio/zyn/providers/openai"
)

type LegacyUser struct {
	ID       int    `json:"id"`
	Name     string `json:"name"`
	Email    string `json:"email"`
	IsActive int    `json:"is_active"` // 1 or 0
}

type ModernUser struct {
	UserID   string `json:"user_id"`
	FullName string `json:"full_name"`
	Contact  string `json:"contact"`
	Active   bool   `json:"active"`
}

func (m ModernUser) Validate() error {
	if m.UserID == "" {
		return fmt.Errorf("user_id required")
	}
	if m.Contact == "" {
		return fmt.Errorf("contact required")
	}
	return nil
}

func main() {
	apiKey := os.Getenv("OPENAI_API_KEY")
	if apiKey == "" {
		log.Fatal("OPENAI_API_KEY environment variable not set")
	}

	ctx := context.Background()

	// Create OpenAI provider
	provider := openai.New(openai.Config{
		APIKey: apiKey,
		Model:  "gpt-3.5-turbo",
	})

	// Create convert synapse
	synapse := zyn.Convert[LegacyUser, ModernUser]("migrate legacy user to modern schema", provider, zyn.WithBackoff(3, 100*time.Millisecond), zyn.WithDebug())

	// Convert user data
	response, err := synapse.FireWithInput(ctx, zyn.ConvertInput[LegacyUser]{
		Data: LegacyUser{
			ID:       12345,
			Name:     "John Doe",
			Email:    "john.doe@example.com",
			IsActive: 1,
		},
		Rules: "Convert ID to string with 'USR-' prefix, is_active: 1=true, 0=false",
	})
	if err != nil {
		log.Fatalf("Conversion failed: %v", err)
	}

	fmt.Printf("Converted User:\n")
	fmt.Printf("  UserID: %s\n", response.UserID)
	fmt.Printf("  FullName: %s\n", response.FullName)
	fmt.Printf("  Contact: %s\n", response.Contact)
	fmt.Printf("  Active: %v\n", response.Active)
}
