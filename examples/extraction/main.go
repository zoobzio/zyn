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

type TechExtraction struct {
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
	synapse := zyn.Extract[TechExtraction]("all mentioned technologies", provider, zyn.WithBackoff(3, 100*time.Millisecond), zyn.WithDebug())

	// Execute extraction
	response, err := synapse.Fire(ctx, "We use Python, Go, and Rust for backend development, with React for the frontend.")
	if err != nil {
		log.Fatalf("Extraction failed: %v", err)
	}

	fmt.Printf("Extracted: %v\n", response.Extracted)
	fmt.Printf("Confidence: %.2f\n", response.Confidence)
	fmt.Printf("Reasoning: %v\n", response.Reasoning)
}
