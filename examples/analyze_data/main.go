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

type SalesData struct {
	Quarter string  `json:"quarter"`
	Revenue float64 `json:"revenue"`
	Growth  float64 `json:"growth_percent"`
	Region  string  `json:"region"`
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

	// Create analyze synapse
	synapse := zyn.Analyze[SalesData]("business performance and trends", provider, zyn.WithBackoff(3, 100*time.Millisecond), zyn.WithDebug())

	// Analyze sales data
	response, err := synapse.FireWithInputDetails(ctx, zyn.AnalyzeInput[SalesData]{
		Data: SalesData{
			Quarter: "Q4 2024",
			Revenue: 1250000,
			Growth:  -5.2,
			Region:  "North America",
		},
		Context: "Year-over-year comparison",
	})
	if err != nil {
		log.Fatalf("Analysis failed: %v", err)
	}

	fmt.Printf("Analysis:\n%s\n\n", response.Analysis)
	fmt.Printf("Confidence: %.2f\n", response.Confidence)
	fmt.Printf("Findings: %v\n", response.Findings)
	fmt.Printf("Reasoning: %v\n", response.Reasoning)
}
