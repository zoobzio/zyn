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

	// Create ranking synapse
	synapse := zyn.Ranking("by performance and efficiency", provider, zyn.WithBackoff(3, 100*time.Millisecond), zyn.WithDebug())

	// Rank databases
	response, err := synapse.FireWithInput(ctx, zyn.RankingInput{
		Items:   []string{"PostgreSQL", "MySQL", "MongoDB", "Redis", "Cassandra"},
		Context: "For high-throughput read-heavy workload with occasional writes",
		Examples: []string{
			"Redis typically ranks highest for pure read performance",
			"PostgreSQL excels at complex queries",
		},
	})
	if err != nil {
		log.Fatalf("Ranking failed: %v", err)
	}

	fmt.Printf("Databases ranked by performance:\n")
	for i, item := range response.Ranked {
		fmt.Printf("%d. %s\n", i+1, item)
	}
	fmt.Printf("\nConfidence: %.2f\n", response.Confidence)
	fmt.Printf("Reasoning: %v\n", response.Reasoning)
}
