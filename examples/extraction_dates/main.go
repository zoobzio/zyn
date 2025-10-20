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

type DateExtraction struct {
	Extracted  []string `json:"extracted"`
	Confidence float64  `json:"confidence"`
	Reasoning  []string `json:"reasoning"`
}

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

	// Create extraction synapse
	synapse := zyn.Extract[DateExtraction]("all dates and deadlines", provider, zyn.WithBackoff(3, 100*time.Millisecond), zyn.WithDebug())

	// Extract dates
	response, err := synapse.FireWithInput(ctx, zyn.ExtractionInput{
		Text: "The project kickoff is scheduled for January 15th. " +
			"First milestone due by end of February. Final delivery on March 30, 2025.",
		Examples: "Q4 2024, next Tuesday, 2025-01-15",
	})
	if err != nil {
		log.Fatalf("Extraction failed: %v", err)
	}

	fmt.Printf("Extracted Dates:\n")
	for _, date := range response.Extracted {
		fmt.Printf("  - %s\n", date)
	}
	fmt.Printf("\nConfidence: %.2f\n", response.Confidence)
	fmt.Printf("Reasoning: %v\n", response.Reasoning)
}
