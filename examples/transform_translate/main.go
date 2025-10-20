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

	// Create transformation synapse
	synapse := zyn.Transform("translate technical jargon to plain English", provider, zyn.WithBackoff(3, 100*time.Millisecond), zyn.WithDebug())

	// Simplify technical text
	response, err := synapse.FireWithInputDetails(ctx, zyn.TransformInput{
		Text: "The distributed consensus algorithm leverages Raft protocol to ensure consistency across replicas.",
		Examples: map[string]string{
			"The API uses RESTful endpoints": "The system uses web addresses to send and receive data",
			"Implements OAuth 2.0 flow":      "Uses a secure login system",
		},
	})
	if err != nil {
		log.Fatalf("Transformation failed: %v", err)
	}

	fmt.Printf("Plain English:\n%s\n\n", response.Output)
	fmt.Printf("Confidence: %.2f\n", response.Confidence)
	fmt.Printf("Changes: %v\n", response.Changes)
	fmt.Printf("Reasoning: %v\n", response.Reasoning)
}
