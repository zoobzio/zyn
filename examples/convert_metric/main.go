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

type RawMetric struct {
	Name  string  `json:"name"`
	Value float64 `json:"value"`
	Unit  string  `json:"unit"`
	Tags  string  `json:"tags"` // Comma-separated
}

type ProcessedMetric struct {
	MetricName    string            `json:"metric_name"`
	NormalizedVal float64           `json:"normalized_value"`
	StandardUnit  string            `json:"standard_unit"`
	Labels        map[string]string `json:"labels"`
	Category      string            `json:"category"`
}

func (p ProcessedMetric) Validate() error {
	if p.MetricName == "" {
		return fmt.Errorf("metric_name required")
	}
	if p.StandardUnit == "" {
		return fmt.Errorf("standard_unit required")
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

	// Create convert synapse
	synapse, err := zyn.Convert[RawMetric, ProcessedMetric]("normalize metrics to standard format", provider, zyn.WithBackoff(3, 100*time.Millisecond))
	if err != nil {
		panic(err)
	}

	// Convert metric
	response, err := synapse.FireWithInput(ctx, zyn.NewSession(), zyn.ConvertInput[RawMetric]{
		Data: RawMetric{
			Name:  "http_req_duration",
			Value: 250,
			Unit:  "ms",
			Tags:  "env=prod,region=us-east",
		},
		Rules: "Parse tags into labels map, categorize by metric type, normalize to seconds",
	})
	if err != nil {
		log.Fatalf("Conversion failed: %v", err)
	}

	fmt.Printf("Processed Metric:\n")
	fmt.Printf("  MetricName: %s\n", response.MetricName)
	fmt.Printf("  NormalizedVal: %.4f\n", response.NormalizedVal)
	fmt.Printf("  StandardUnit: %s\n", response.StandardUnit)
	fmt.Printf("  Labels: %v\n", response.Labels)
	fmt.Printf("  Category: %s\n", response.Category)
}
