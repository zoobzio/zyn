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

	// Create ranking synapse
	synapse := zyn.Ranking("by urgency and business impact", provider, zyn.WithBackoff(3, 100*time.Millisecond), zyn.WithDebug())

	// Rank tasks
	response, err := synapse.FireWithInput(ctx, zyn.RankingInput{
		Items: []string{
			"Fix critical security vulnerability",
			"Add new feature request",
			"Update documentation",
			"Refactor legacy code",
			"Fix UI bug",
		},
		Context: "Engineering team sprint planning",
		TopN:    3,
	})
	if err != nil {
		log.Fatalf("Ranking failed: %v", err)
	}

	fmt.Printf("Top priority tasks:\n")
	for i, item := range response.Ranked {
		fmt.Printf("%d. %s\n", i+1, item)
	}
	fmt.Printf("\nConfidence: %.2f\n", response.Confidence)
	fmt.Printf("Reasoning: %v\n", response.Reasoning)
}
