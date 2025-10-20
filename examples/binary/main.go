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

	// Create binary synapse
	synapse := zyn.Binary("Is this a question?", provider, zyn.WithBackoff(3, 100*time.Millisecond), zyn.WithDebug())

	// Execute binary decision
	response, err := synapse.FireWithDetails(ctx, "What time is it?")
	if err != nil {
		log.Fatalf("Binary decision failed: %v", err)
	}

	fmt.Printf("Decision: %v\n", response.Decision)
	fmt.Printf("Confidence: %.2f\n", response.Confidence)
	fmt.Printf("Reasoning: %v\n", response.Reasoning)
}
