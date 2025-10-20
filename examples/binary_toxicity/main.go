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

	// Create binary decision synapse
	synapse := zyn.Binary("does this contain toxic or harmful content?", provider, zyn.WithBackoff(3, 100*time.Millisecond), zyn.WithDebug())

	// Check for toxicity
	response, err := synapse.FireWithInput(ctx, zyn.BinaryInput{
		Subject:     "I completely disagree with your point, but I appreciate your perspective.",
		Context:     "Content moderation for community platform",
		Temperature: 0.3, // Lower temperature for consistent moderation
	})
	if err != nil {
		log.Fatalf("Binary decision failed: %v", err)
	}

	fmt.Printf("Is Toxic: %v\n", response.Decision)
	fmt.Printf("Confidence: %.2f\n", response.Confidence)
	fmt.Printf("Reasoning: %v\n", response.Reasoning)
}
