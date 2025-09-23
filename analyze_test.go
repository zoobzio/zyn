package zyn

import (
	"context"
	"strings"
	"testing"
	"time"
)

// Test structs for analysis
type EmailData struct {
	From    string   `json:"from"`
	To      []string `json:"to"`
	Subject string   `json:"subject"`
	Body    string   `json:"body"`
	Links   []string `json:"links"`
}

type CodeMetrics struct {
	LinesOfCode  int      `json:"lines_of_code"`
	Complexity   int      `json:"complexity"`
	Coverage     float64  `json:"coverage"`
	Dependencies []string `json:"dependencies"`
}

type ServerConfig struct {
	Port          int               `json:"port"`
	MaxConns      int               `json:"max_connections"`
	Timeout       string            `json:"timeout"`
	AllowedHosts  []string          `json:"allowed_hosts"`
	Features      map[string]bool   `json:"features"`
}

func TestAnalyzeEmail(t *testing.T) {
	provider := NewMockProviderWithResponse(`{
		"analysis": "This email appears to be a phishing attempt. The sender domain does not match the claimed organization, and the links redirect through suspicious tracking URLs. The urgent language and request for credentials are classic phishing indicators.",
		"confidence": 0.92,
		"findings": [
			"Sender domain mismatch",
			"Suspicious link redirects",
			"Urgent action language",
			"Credential request"
		],
		"reasoning": [
			"Analyzed sender authentication",
			"Checked link destinations",
			"Evaluated message content patterns",
			"Compared against phishing indicators"
		]
	}`)

	analyzer := Analyze[EmailData]("check for phishing indicators", provider, WithTimeout(5*time.Second))

	email := EmailData{
		From:    "security@amaz0n-verify.com",
		To:      []string{"user@example.com"},
		Subject: "Urgent: Verify Your Account",
		Body:    "Your account will be suspended. Click here immediately to verify.",
		Links:   []string{"http://bit.ly/verify-account"},
	}

	ctx := context.Background()
	
	// Test simple Fire
	analysis, err := analyzer.Fire(ctx, email)
	if err != nil {
		t.Fatalf("Fire failed: %v", err)
	}
	if !strings.Contains(analysis, "phishing") {
		t.Errorf("Analysis should identify phishing")
	}

	// Test FireWithDetails
	details, err := analyzer.FireWithDetails(ctx, email)
	if err != nil {
		t.Fatalf("FireWithDetails failed: %v", err)
	}
	if details.Confidence != 0.92 {
		t.Errorf("Expected confidence 0.92, got %f", details.Confidence)
	}
	if len(details.Findings) != 4 {
		t.Errorf("Expected 4 findings, got %d", len(details.Findings))
	}
	if !strings.Contains(details.Findings[0], "domain mismatch") {
		t.Errorf("Expected domain mismatch finding")
	}
}

func TestAnalyzeCodeMetrics(t *testing.T) {
	provider := NewMockProviderWithResponse(`{
		"analysis": "The codebase shows moderate complexity with adequate test coverage. The high number of dependencies suggests potential maintenance challenges. Consider refactoring complex functions and reducing external dependencies.",
		"confidence": 0.85,
		"findings": [
			"Complexity exceeds recommended threshold",
			"Test coverage is acceptable at 78%",
			"High dependency count"
		],
		"reasoning": [
			"Evaluated complexity metrics",
			"Analyzed coverage percentage",
			"Assessed dependency risks"
		]
	}`)

	analyzer := Analyze[CodeMetrics]("code quality assessment", provider)

	metrics := CodeMetrics{
		LinesOfCode:  5000,
		Complexity:   42,
		Coverage:     0.78,
		Dependencies: []string{"react", "redux", "axios", "lodash", "moment", "uuid"},
	}

	ctx := context.Background()
	result, err := analyzer.Fire(ctx, metrics)
	if err != nil {
		t.Fatalf("Fire failed: %v", err)
	}
	if !strings.Contains(result, "complexity") {
		t.Errorf("Analysis should mention complexity")
	}
	if !strings.Contains(result, "coverage") {
		t.Errorf("Analysis should mention coverage")
	}
}

func TestAnalyzeServerConfig(t *testing.T) {
	provider := NewMockProviderWithResponse(`{
		"analysis": "Configuration has security concerns: debug mode enabled in production, overly permissive host whitelist, and very high connection limit without rate limiting.",
		"confidence": 0.88,
		"findings": [
			"Debug mode enabled",
			"Wildcard in allowed hosts",
			"No rate limiting with high connection limit"
		],
		"reasoning": [
			"Checked production settings",
			"Evaluated security configurations",
			"Analyzed resource limits"
		]
	}`)

	analyzer := Analyze[ServerConfig]("security configuration review", provider)

	config := ServerConfig{
		Port:         8080,
		MaxConns:     10000,
		Timeout:      "30s",
		AllowedHosts: []string{"*"},
		Features: map[string]bool{
			"debug":        true,
			"rate_limit":   false,
			"auth_enabled": true,
		},
	}

	ctx := context.Background()
	details, err := analyzer.FireWithDetails(ctx, config)
	if err != nil {
		t.Fatalf("FireWithDetails failed: %v", err)
	}
	if !strings.Contains(details.Analysis, "security concerns") {
		t.Errorf("Analysis should identify security concerns")
	}
	if len(details.Findings) != 3 {
		t.Errorf("Expected 3 findings, got %d", len(details.Findings))
	}
}

func TestAnalyzeWithFocus(t *testing.T) {
	provider := NewMockProviderWithResponse(`{
		"analysis": "Performance analysis: The configuration shows potential bottlenecks with low connection limit (100) and short timeout (5s) that may cause issues under load.",
		"confidence": 0.82,
		"findings": [
			"Connection limit may be too restrictive",
			"Timeout too short for slow clients"
		],
		"reasoning": [
			"Focused on performance implications",
			"Evaluated throughput limitations"
		]
	}`)

	analyzer := Analyze[ServerConfig]("configuration review", provider)

	config := ServerConfig{
		Port:         3000,
		MaxConns:     100,
		Timeout:      "5s",
		AllowedHosts: []string{"localhost", "api.example.com"},
		Features: map[string]bool{
			"caching": false,
		},
	}

	input := AnalyzeInput[ServerConfig]{
		Data:  config,
		Focus: "performance implications",
	}

	ctx := context.Background()
	details, err := analyzer.FireWithInputDetails(ctx, input)
	if err != nil {
		t.Fatalf("FireWithInputDetails failed: %v", err)
	}
	if !strings.Contains(details.Analysis, "Performance") {
		t.Errorf("Analysis should focus on performance")
	}
}

func TestAnalyzeWithContext(t *testing.T) {
	provider := NewMockProviderWithResponse(`{
		"analysis": "For a development environment, the configuration is appropriate. Debug mode and wildcard hosts are acceptable for local development but must be changed before production deployment.",
		"confidence": 0.90,
		"findings": [
			"Settings appropriate for development",
			"Must update before production"
		],
		"reasoning": [
			"Evaluated in development context",
			"Different standards than production"
		]
	}`)

	analyzer := Analyze[ServerConfig]("configuration review", provider)

	config := ServerConfig{
		Port:         8080,
		MaxConns:     10000,
		Timeout:      "30s",
		AllowedHosts: []string{"*"},
		Features: map[string]bool{
			"debug": true,
		},
	}

	input := AnalyzeInput[ServerConfig]{
		Data:    config,
		Context: "This is for a local development environment",
	}

	ctx := context.Background()
	result, err := analyzer.FireWithInput(ctx, input)
	if err != nil {
		t.Fatalf("FireWithInput failed: %v", err)
	}
	if !strings.Contains(result, "development") {
		t.Errorf("Analysis should acknowledge development context")
	}
}

func TestAnalyzePromptStructure(t *testing.T) {
	var capturedPrompt string
	provider := NewMockProviderWithCallback(func(prompt string, temp float32) (string, error) {
		capturedPrompt = prompt
		return `{"analysis": "test", "confidence": 1.0, "findings": [], "reasoning": ["test"]}`, nil
	})

	type SimpleData struct {
		Value int `json:"value"`
	}

	analyzer := Analyze[SimpleData]("data validation", provider)

	ctx := context.Background()
	_, err := analyzer.Fire(ctx, SimpleData{Value: 42})
	if err != nil {
		t.Fatalf("Fire failed: %v", err)
	}

	// Check prompt structure
	if !strings.Contains(capturedPrompt, "Task: Analyze: data validation") {
		t.Error("Prompt missing task description")
	}
	if !strings.Contains(capturedPrompt, "Input:") {
		t.Error("Prompt missing input section")
	}
	if !strings.Contains(capturedPrompt, `"value": 42`) {
		t.Error("Prompt missing JSON data")
	}
	if !strings.Contains(capturedPrompt, "Return JSON:") {
		t.Error("Prompt missing JSON structure")
	}
	if !strings.Contains(capturedPrompt, `"analysis"`) {
		t.Error("Schema missing analysis field")
	}
	if !strings.Contains(capturedPrompt, `"findings"`) {
		t.Error("Schema missing findings field")
	}
}

func TestAnalyzeSliceInput(t *testing.T) {
	provider := NewMockProviderWithResponse(`{
		"analysis": "The log entries show a pattern of failed authentication attempts from the same IP, suggesting a potential brute force attack.",
		"confidence": 0.87,
		"findings": [
			"Multiple failed auth attempts",
			"Same source IP",
			"Rapid succession timing"
		],
		"reasoning": [
			"Pattern analysis of log entries",
			"Timing correlation"
		]
	}`)

	type LogEntry struct {
		Timestamp string `json:"timestamp"`
		Level     string `json:"level"`
		Message   string `json:"message"`
		IP        string `json:"ip"`
	}

	analyzer := Analyze[[]LogEntry]("security event analysis", provider)

	logs := []LogEntry{
		{Timestamp: "2024-01-01T10:00:00Z", Level: "ERROR", Message: "Auth failed", IP: "192.168.1.100"},
		{Timestamp: "2024-01-01T10:00:05Z", Level: "ERROR", Message: "Auth failed", IP: "192.168.1.100"},
		{Timestamp: "2024-01-01T10:00:10Z", Level: "ERROR", Message: "Auth failed", IP: "192.168.1.100"},
	}

	ctx := context.Background()
	analysis, err := analyzer.Fire(ctx, logs)
	if err != nil {
		t.Fatalf("Fire failed: %v", err)
	}
	if !strings.Contains(analysis, "brute force") || !strings.Contains(analysis, "pattern") {
		t.Errorf("Analysis should identify the attack pattern")
	}
}