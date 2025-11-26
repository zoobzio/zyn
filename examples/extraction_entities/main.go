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

type EntityExtraction struct {
	Extracted  []string `json:"extracted"`
	Confidence float64  `json:"confidence"`
	Reasoning  []string `json:"reasoning"`
}

func (e EntityExtraction) Validate() error {
	if len(e.Extracted) == 0 {
		return fmt.Errorf("no entities extracted")
	}
	return nil
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
	synapse, err := zyn.Extract[EntityExtraction]("person names and organizations", provider, zyn.WithBackoff(3, 100*time.Millisecond))
	if err != nil {
		panic(err)
	}

	// Extract entities
	response, err := synapse.FireWithInput(ctx, zyn.NewSession(), zyn.ExtractionInput{
		Text: "Apple CEO Tim Cook announced a partnership with Microsoft and Google. " +
			"Satya Nadella and Sundar Pichai will join the board.",
		Context: "News article processing",
	})
	if err != nil {
		log.Fatalf("Extraction failed: %v", err)
	}

	fmt.Printf("Extracted Entities:\n")
	for _, entity := range response.Extracted {
		fmt.Printf("  - %s\n", entity)
	}
	fmt.Printf("\nConfidence: %.2f\n", response.Confidence)
	fmt.Printf("Reasoning: %v\n", response.Reasoning)
}
