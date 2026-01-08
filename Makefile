.PHONY: test bench bench-all lint coverage clean all help test-providers test-integration test-benchmarks test-reliability test-all ci check lint-fix install-tools install-hooks examples example-list

.DEFAULT_GOAL := help

# Default target
all: test lint

# Display help
help:
	@echo "zyn Development Commands"
	@echo "========================"
	@echo ""
	@echo "Testing & Quality:"
	@echo "  make test            - Run unit tests with race detector"
	@echo "  make test-providers  - Run provider tests"
	@echo "  make test-integration- Run integration tests with race detector"
	@echo "  make test-benchmarks - Run performance benchmarks"
	@echo "  make test-reliability- Run reliability/resilience tests"
	@echo "  make test-all        - Run all test suites (unit + integration + reliability)"
	@echo "  make bench           - Run core library benchmarks"
	@echo "  make bench-all       - Run all benchmarks"
	@echo "  make lint            - Run linters"
	@echo "  make lint-fix        - Run linters with auto-fix"
	@echo "  make coverage        - Generate coverage report (HTML)"
	@echo "  make check           - Run tests and lint (quick check)"
	@echo "  make ci              - Full CI simulation (all tests + quality checks)"
	@echo ""
	@echo "Examples:"
	@echo "  make examples        - Run all examples (requires .env with OPENAI_API_KEY)"
	@echo "  make example-list    - List all available examples"
	@echo "  make example EX=<name> - Run specific example (e.g., make example EX=sentiment)"
	@echo ""
	@echo "Other:"
	@echo "  make install-tools   - Install required development tools"
	@echo "  make clean           - Clean generated files"
	@echo "  make all             - Run tests and lint (default)"

# Run tests with race detector
test:
	@echo "Running core tests..."
	@go test -v -race ./...

# Run provider tests
test-providers:
	@echo "Running provider tests..."
	@go test -v -race ./openai/... ./anthropic/... ./gemini/...

# Run integration tests - component interaction verification
test-integration:
	@echo "Running integration tests..."
	@go test -v -race -tags=integration -timeout=10m ./...

# Run benchmark tests - performance regression detection
test-benchmarks:
	@echo "Running performance benchmarks..."
	@go test -v -bench=. -benchmem -benchtime=100ms -timeout=15m ./...

# Run reliability tests - resilience pattern verification
test-reliability:
	@echo "Running reliability tests..."
	@go test -v -race -timeout=10m -run TestResilience ./...
	@go test -v -race -timeout=5m -run TestPanic ./...
	@go test -v -race -timeout=10m -run TestConcurrent ./...

# Comprehensive test suite - all tests with race detection
test-all: test test-integration test-reliability
	@echo "All test suites completed!"

# Run core benchmarks
bench:
	@echo "Running core benchmarks..."
	@go test -bench=. -benchmem -benchtime=100ms -timeout=15m .

# Run all benchmarks
bench-all:
	@echo "Running all benchmarks..."
	@go test -bench=. -benchmem -benchtime=100ms -timeout=15m ./...

# Run linters
lint:
	@echo "Running linters..."
	@golangci-lint run --config=.golangci.yml --timeout=5m

# Run linters with auto-fix
lint-fix:
	@echo "Running linters with auto-fix..."
	@golangci-lint run --config=.golangci.yml --fix

# Generate coverage report
coverage:
	@echo "Generating coverage report..."
	@go test -coverprofile=coverage.out ./...
	@go tool cover -html=coverage.out -o coverage.html
	@go tool cover -func=coverage.out | tail -1
	@echo "Coverage report generated: coverage.html"

# Clean generated files
clean:
	@echo "Cleaning..."
	@rm -f coverage.out coverage.html
	@rm -f zyn
	@find . -name "*.test" -delete
	@find . -name "*.prof" -delete
	@find . -name "*.out" -delete

# Install development tools
install-tools:
	@echo "Installing development tools..."
	@go install github.com/golangci/golangci-lint/v2/cmd/golangci-lint@v2.7.2

# Install git pre-commit hook
install-hooks:
	@echo "Installing git hooks..."
	@mkdir -p .git/hooks
	@echo '#!/bin/sh' > .git/hooks/pre-commit
	@echo 'make check' >> .git/hooks/pre-commit
	@chmod +x .git/hooks/pre-commit
	@echo "Pre-commit hook installed"

# Quick check - run tests and lint
check: test lint
	@echo "All checks passed!"

# CI simulation - what CI runs locally
ci: clean lint test test-integration test-benchmarks test-reliability coverage
	@echo "Full CI simulation complete!"

# List all available examples
example-list:
	@echo "Available examples:"
	@echo ""
	@echo "Sentiment Analysis:"
	@echo "  sentiment          - Basic sentiment analysis"
	@echo "  sentiment_review   - Product review sentiment with aspects"
	@echo "  sentiment_social   - Social media sentiment analysis"
	@echo ""
	@echo "Binary Decisions:"
	@echo "  binary             - Simple yes/no question"
	@echo "  binary_spam        - Spam detection"
	@echo "  binary_toxicity    - Toxicity detection"
	@echo ""
	@echo "Classification:"
	@echo "  classification     - Text categorization"
	@echo "  classification_email   - Email priority classification"
	@echo "  classification_content - Content type classification"
	@echo ""
	@echo "Extraction:"
	@echo "  extraction         - Extract technologies from text"
	@echo "  extraction_entities    - Extract names and organizations"
	@echo "  extraction_dates   - Extract dates and deadlines"
	@echo ""
	@echo "Ranking:"
	@echo "  ranking_popularity     - Rank by popularity"
	@echo "  ranking_priority   - Rank by urgency"
	@echo "  ranking_performance    - Rank by performance"
	@echo ""
	@echo "Transformation:"
	@echo "  transform_summarize    - Summarize text"
	@echo "  transform_formalize    - Convert to formal language"
	@echo "  transform_translate    - Translate jargon to plain English"
	@echo ""
	@echo "Analysis:"
	@echo "  analyze_code       - Analyze code for bugs"
	@echo "  analyze_data       - Analyze business data"
	@echo "  analyze_config     - Analyze system configuration"
	@echo ""
	@echo "Conversion:"
	@echo "  convert_user       - Convert legacy to modern schema"
	@echo "  convert_event      - Convert raw to structured event"
	@echo "  convert_metric     - Normalize metrics"
	@echo ""
	@echo "Usage: make example EX=<name>"

# Run specific example
example:
	@if [ -z "$(EX)" ]; then \
		echo "Error: Please specify an example name"; \
		echo "Usage: make example EX=<name>"; \
		echo "Run 'make example-list' to see available examples"; \
		exit 1; \
	fi
	@if [ ! -f .env ]; then \
		echo "Warning: .env file not found. Copy .env.example to .env and add your API keys."; \
		echo ""; \
	fi
	@set -a && [ -f .env ] && . ./.env && set +a; go run examples/$(EX)/main.go

# Run all examples
examples:
	@if [ ! -f .env ]; then \
		echo "Error: .env file not found"; \
		echo "Please copy .env.example to .env and add your OPENAI_API_KEY"; \
		exit 1; \
	fi
	@echo "Running all examples..."
	@set -a && . ./.env && set +a && \
	for example in examples/*/main.go; do \
		echo ""; \
		echo "=== Running $$example ==="; \
		go run $$example || true; \
	done
	@echo ""
	@echo "All examples completed!"