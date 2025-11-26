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
	synapse, err := zyn.Ranking("by popularity and market demand", provider, zyn.WithBackoff(3, 100*time.Millisecond))
	if err != nil {
		panic(err)
	}

	// Rank programming languages
	response, err := synapse.FireWithInput(ctx, zyn.NewSession(), zyn.RankingInput{
		Items:   []string{"Rust", "Python", "Go", "JavaScript", "TypeScript", "Java"},
		Context: "Consider current industry trends and job market demand",
	})
	if err != nil {
		log.Fatalf("Ranking failed: %v", err)
	}

	fmt.Printf("Ranked by popularity:\n")
	for i, item := range response.Ranked {
		fmt.Printf("%d. %s\n", i+1, item)
	}
	fmt.Printf("\nConfidence: %.2f\n", response.Confidence)
	fmt.Printf("Reasoning: %v\n", response.Reasoning)
}
