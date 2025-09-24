.PHONY: test bench bench-all lint coverage clean all help test-providers test-integration test-benchmarks test-reliability test-all ci check lint-fix install-tools

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
	@go test -v -race ./providers/...

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
	@go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest

# Quick check - run tests and lint
check: test lint
	@echo "All checks passed!"

# CI simulation - what CI runs locally
ci: clean lint test test-integration test-benchmarks test-reliability coverage
	@echo "Full CI simulation complete!"