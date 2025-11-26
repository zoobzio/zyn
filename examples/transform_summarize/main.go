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
	synapse, err := zyn.Transform("summarize to one concise paragraph", provider, zyn.WithBackoff(3, 100*time.Millisecond))
	if err != nil {
		panic(err)
	}

	// Summarize text
	response, err := synapse.FireWithInputDetails(ctx, zyn.NewSession(), zyn.TransformInput{
		Text: `The Go programming language was designed at Google in 2007 by Robert Griesemer,
Rob Pike, and Ken Thompson. It was created to address software engineering challenges at
Google, particularly in the context of networked servers and large codebases. The language
was officially announced in 2009 and reached version 1.0 in 2012. Go combines the efficiency
of statically typed compiled languages with the ease of programming of dynamic languages.
It features garbage collection, native concurrency support through goroutines, and a simple,
clean syntax that emphasizes readability.`,
		MaxLength: 150,
	})
	if err != nil {
		log.Fatalf("Transformation failed: %v", err)
	}

	fmt.Printf("Summary:\n%s\n\n", response.Output)
	fmt.Printf("Confidence: %.2f\n", response.Confidence)
	fmt.Printf("Changes: %v\n", response.Changes)
	fmt.Printf("Reasoning: %v\n", response.Reasoning)
}
