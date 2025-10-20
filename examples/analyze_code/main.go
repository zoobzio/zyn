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

type CodeSnippet struct {
	Language string `json:"language"`
	Code     string `json:"code"`
	Purpose  string `json:"purpose"`
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
	synapse := zyn.Analyze[CodeSnippet]("code for potential bugs and improvements", provider, zyn.WithBackoff(3, 100*time.Millisecond), zyn.WithDebug())

	// Analyze code
	response, err := synapse.FireWithInputDetails(ctx, zyn.AnalyzeInput[CodeSnippet]{
		Data: CodeSnippet{
			Language: "Go",
			Code: `func divide(a, b int) int {
	return a / b
}`,
			Purpose: "Divide two integers",
		},
		Focus: "safety and edge cases",
	})
	if err != nil {
		log.Fatalf("Analysis failed: %v", err)
	}

	fmt.Printf("Analysis:\n%s\n\n", response.Analysis)
	fmt.Printf("Confidence: %.2f\n", response.Confidence)
	fmt.Printf("Findings: %v\n", response.Findings)
	fmt.Printf("Reasoning: %v\n", response.Reasoning)
}
