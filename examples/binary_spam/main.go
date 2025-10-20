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
	synapse := zyn.Binary("is this spam or promotional content?", provider, zyn.WithBackoff(3, 100*time.Millisecond), zyn.WithDebug())

	// Check for spam
	response, err := synapse.FireWithInput(ctx, zyn.BinaryInput{
		Subject: "Congratulations! You've won $1,000,000! Click here to claim your prize now!!!",
		Context: "Email filtering system",
	})
	if err != nil {
		log.Fatalf("Binary decision failed: %v", err)
	}

	fmt.Printf("Is Spam: %v\n", response.Decision)
	fmt.Printf("Confidence: %.2f\n", response.Confidence)
	fmt.Printf("Reasoning: %v\n", response.Reasoning)
}
