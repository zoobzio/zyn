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

	// Create sentiment synapse
	synapse, err := zyn.Sentiment("product review sentiment", provider, zyn.WithBackoff(3, 100*time.Millisecond))
	if err != nil {
		panic(err)
	}

	// Analyze product review
	response, err := synapse.FireWithInput(ctx, zyn.NewSession(), zyn.SentimentInput{
		Text:    "The product quality is excellent but the delivery was extremely slow.",
		Context: "Customer product review",
		Aspects: []string{"product quality", "delivery speed"},
	})
	if err != nil {
		log.Fatalf("Sentiment analysis failed: %v", err)
	}

	fmt.Printf("Overall Sentiment: %s\n", response.Overall)
	fmt.Printf("Confidence: %.2f\n", response.Confidence)
	fmt.Printf("Aspect Sentiments:\n")
	for aspect, sentiment := range response.Aspects {
		fmt.Printf("  %s: %s\n", aspect, sentiment)
	}
	fmt.Printf("Reasoning: %v\n", response.Reasoning)
}
