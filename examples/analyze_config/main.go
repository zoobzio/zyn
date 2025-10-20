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

type SystemConfig struct {
	Database    DatabaseConfig `json:"database"`
	Cache       CacheConfig    `json:"cache"`
	APISettings APIConfig      `json:"api"`
}

type DatabaseConfig struct {
	MaxConnections int    `json:"max_connections"`
	Timeout        string `json:"timeout"`
}

type CacheConfig struct {
	TTL  string `json:"ttl"`
	Size string `json:"size"`
}

type APIConfig struct {
	RateLimit  int  `json:"rate_limit"`
	EnableCORS bool `json:"enable_cors"`
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
	synapse := zyn.Analyze[SystemConfig]("configuration for production readiness", provider, zyn.WithBackoff(3, 100*time.Millisecond), zyn.WithDebug())

	// Analyze configuration
	response, err := synapse.FireWithInputDetails(ctx, zyn.AnalyzeInput[SystemConfig]{
		Data: SystemConfig{
			Database: DatabaseConfig{
				MaxConnections: 10,
				Timeout:        "5s",
			},
			Cache: CacheConfig{
				TTL:  "1h",
				Size: "100MB",
			},
			APISettings: APIConfig{
				RateLimit:  1000,
				EnableCORS: true,
			},
		},
		Context: "High-traffic production environment",
		Focus:   "scalability and security",
	})
	if err != nil {
		log.Fatalf("Analysis failed: %v", err)
	}

	fmt.Printf("Analysis:\n%s\n\n", response.Analysis)
	fmt.Printf("Confidence: %.2f\n", response.Confidence)
	fmt.Printf("Findings: %v\n", response.Findings)
	fmt.Printf("Reasoning: %v\n", response.Reasoning)
}
