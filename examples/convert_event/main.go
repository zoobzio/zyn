package main

import (
	"time"
	"context"
	"fmt"
	"log"
	"os"

	"github.com/zoobzio/zyn"
	"github.com/zoobzio/zyn/providers/openai"
)

type RawEvent struct {
	Type      string                 `json:"type"`
	Timestamp int64                  `json:"timestamp"`
	Data      map[string]interface{} `json:"data"`
}

type StructuredEvent struct {
	EventName   string   `json:"event_name"`
	OccurredAt  string   `json:"occurred_at"`
	UserID      string   `json:"user_id"`
	Action      string   `json:"action"`
	Details     []string `json:"details"`
	IsImportant bool     `json:"is_important"`
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
	synapse := zyn.Convert[RawEvent, StructuredEvent]("transform raw event to structured format", provider, zyn.WithBackoff(3, 100*time.Millisecond), zyn.WithDebug())

	// Convert event
	response, err := synapse.FireWithInput(ctx, zyn.ConvertInput[RawEvent]{
		Data: RawEvent{
			Type:      "user_action",
			Timestamp: 1704067200,
			Data: map[string]interface{}{
				"uid":    "u123",
				"act":    "purchase",
				"amount": 99.99,
				"items":  3,
			},
		},
		Context: "E-commerce analytics pipeline",
	})
	if err != nil {
		log.Fatalf("Conversion failed: %v", err)
	}

	fmt.Printf("Structured Event:\n")
	fmt.Printf("  EventName: %s\n", response.EventName)
	fmt.Printf("  OccurredAt: %s\n", response.OccurredAt)
	fmt.Printf("  UserID: %s\n", response.UserID)
	fmt.Printf("  Action: %s\n", response.Action)
	fmt.Printf("  Details: %v\n", response.Details)
	fmt.Printf("  IsImportant: %v\n", response.IsImportant)
}
