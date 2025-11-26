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
	synapse, err := zyn.Classification("content type", []string{"tutorial", "reference", "opinion", "news", "documentation"}, provider, zyn.WithBackoff(3, 100*time.Millisecond))
	if err != nil {
		panic(err)
	}

	// Classify blog post
	response, err := synapse.FireWithInput(ctx, zyn.NewSession(), zyn.ClassificationInput{
		Subject: "In this guide, we'll walk through setting up a Go web server from scratch. " +
			"You'll learn how to handle HTTP requests, set up routing, and deploy to production.",
		Context: "Blog post categorization",
		Examples: map[string][]string{
			"tutorial": {"Learn the basics of Go syntax"},
			"news":     {"Go 1.21 released today"},
		},
	})
	if err != nil {
		log.Fatalf("Classification failed: %v", err)
	}

	fmt.Printf("Primary Category: %s\n", response.Primary)
	fmt.Printf("Secondary Category: %s\n", response.Secondary)
	fmt.Printf("Confidence: %.2f\n", response.Confidence)
	fmt.Printf("Reasoning: %v\n", response.Reasoning)
}
