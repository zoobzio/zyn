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

	// Create classification synapse
	synapse, err := zyn.Classification("priority level", []string{"urgent", "normal", "low-priority", "spam"}, provider, zyn.WithBackoff(3, 100*time.Millisecond))
	if err != nil {
		panic(err)
	}

	// Classify email
	response, err := synapse.FireWithInput(ctx, zyn.NewSession(), zyn.ClassificationInput{
		Subject: "URGENT: Server down in production. Customers cannot access the application.",
		Context: "Support ticket triage",
	})
	if err != nil {
		log.Fatalf("Classification failed: %v", err)
	}

	fmt.Printf("Primary Category: %s\n", response.Primary)
	fmt.Printf("Secondary Category: %s\n", response.Secondary)
	fmt.Printf("Confidence: %.2f\n", response.Confidence)
	fmt.Printf("Reasoning: %v\n", response.Reasoning)
}
