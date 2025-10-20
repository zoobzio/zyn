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
	synapse := zyn.Transform("convert to formal business language", provider, zyn.WithBackoff(3, 100*time.Millisecond), zyn.WithDebug())

	// Transform casual email to formal
	response, err := synapse.FireWithInputDetails(ctx, zyn.TransformInput{
		Text:    "Hey! Just wanted to let you know the project is coming along great. We should be done soon.",
		Context: "Email to executive stakeholder",
		Style:   "Professional and formal",
	})
	if err != nil {
		log.Fatalf("Transformation failed: %v", err)
	}

	fmt.Printf("Formal version:\n%s\n\n", response.Output)
	fmt.Printf("Confidence: %.2f\n", response.Confidence)
	fmt.Printf("Changes: %v\n", response.Changes)
	fmt.Printf("Reasoning: %v\n", response.Reasoning)
}
