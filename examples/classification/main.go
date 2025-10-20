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

	// Create classification synapse
	synapse := zyn.Classification("what category", []string{"technology", "sports", "politics", "entertainment"}, provider, zyn.WithBackoff(3, 100*time.Millisecond), zyn.WithDebug())

	// Execute classification
	response, err := synapse.FireWithDetails(ctx, "The new iPhone 15 was announced today with groundbreaking features.")
	if err != nil {
		log.Fatalf("Classification failed: %v", err)
	}

	fmt.Printf("Primary: %s\n", response.Primary)
	fmt.Printf("Secondary: %s\n", response.Secondary)
	fmt.Printf("Confidence: %.2f\n", response.Confidence)
	fmt.Printf("Reasoning: %v\n", response.Reasoning)
}
