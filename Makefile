.PHONY: test bench lint coverage clean all help test-providers test-integration test-all ci

# Default target
all: test lint

# Display help
help:
	@echo "zyn Development Commands"
	@echo "============================"
	@echo ""
	@echo "Testing & Quality:"
	@echo "  make test            - Run unit tests with race detector"
	@echo "  make test-providers  - Run provider tests"
	@echo "  make test-integration- Run integration tests"
	@echo "  make test-all        - Run all test suites"
	@echo "  make bench           - Run benchmarks"
	@echo "  make lint            - Run linters"
	@echo "  make lint-fix        - Run linters with auto-fix"
	@echo "  make coverage        - Generate coverage report (HTML)"
	@echo "  make check           - Run tests and lint (quick check)"
	@echo "  make ci              - Full CI simulation"
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

# Run integration tests
test-integration:
	@echo "Running integration tests..."
	@go test -v -race -tags=integration -timeout=10m ./...

# Run all test suites
test-all: test test-providers test-integration
	@echo "All test suites completed!"

# Run benchmarks
bench:
	@echo "Running benchmarks..."
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

# CI simulation - what CI runs
ci: clean lint test test-providers coverage
	@echo "Full CI simulation complete!"