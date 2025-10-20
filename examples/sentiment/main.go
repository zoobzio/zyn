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

	// Create sentiment synapse
	synapse := zyn.Sentiment("general sentiment", provider, zyn.WithBackoff(3, 100*time.Millisecond), zyn.WithDebug())

	// Execute sentiment analysis
	response, err := synapse.FireWithDetails(ctx, "I absolutely love this product! It exceeded all my expectations.")
	if err != nil {
		log.Fatalf("Sentiment analysis failed: %v", err)
	}

	fmt.Printf("Overall Sentiment: %s\n", response.Overall)
	fmt.Printf("Confidence: %.2f\n", response.Confidence)
	fmt.Printf("Reasoning: %v\n", response.Reasoning)
}
