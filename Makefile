# Makefile for Bon HTTP Router Library

.PHONY: help test lint clean bench coverage fmt vet mod-tidy mod-download test-verbose dev-setup ci check perf profile-mem profile-cpu bench-compare qt qb watch

# Default target
.DEFAULT_GOAL := help

# Colors for output
CYAN := \033[36m
RESET := \033[0m

# Help - Show available commands
help: ## Show this help message
	@echo "Available commands:"
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "$(CYAN)%-20s$(RESET) %s\n", $$1, $$2}'

# ==================== Basic Commands ====================

# Run all tests
test: ## Run all tests
	@echo "Running tests..."
	go test -v ./...

# Run linter
lint: ## Run golangci-lint
	@echo "Running lint..."
	@which golangci-lint > /dev/null || (echo "golangci-lint not found. Install with: go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest"; exit 1)
	golangci-lint run

# Clean generated files
clean: ## Clean test artifacts and generated files
	@echo "Cleaning up..."
	go clean
	rm -f coverage.out coverage.html *.prof

# ==================== Code Quality ====================

# Format code
fmt: ## Format code using go fmt and gofmt
	@echo "Formatting code..."
	go fmt ./...
	gofmt -s -w .

# Run go vet
vet: ## Run go vet
	@echo "Running go vet..."
	go vet ./...

# Run all checks
check: fmt vet lint test ## Run all code quality checks (fmt, vet, lint, test)

# ==================== Testing ====================

# Run tests with race detector and coverage
test-verbose: ## Run tests with race detector and coverage report
	@echo "Running tests with coverage..."
	go test -v -race -coverprofile=coverage.out ./...
	go tool cover -html=coverage.out -o coverage.html
	@echo "Coverage report generated: coverage.html"

# Generate coverage report
coverage: ## Generate test coverage report
	@echo "Generating coverage report..."
	go test -coverprofile=coverage.out ./...
	go tool cover -html=coverage.out -o coverage.html
	@echo "Coverage report generated: coverage.html"

# ==================== Benchmarks ====================

# Run all benchmarks
bench: ## Run all benchmarks
	@echo "Running benchmarks..."
	go test -bench=. -benchmem ./...

# Run benchmarks and save to bench.md
bench-save: ## Run benchmarks and save results to bench.md
	@echo "Running benchmarks and saving to bench.md..."
	@echo "# Benchmark Results" > bench.md
	@echo "" >> bench.md
	@echo "Generated on: $$(date '+%Y-%m-%d %H:%M:%S')" >> bench.md
	@echo "" >> bench.md
	@echo "## System Information" >> bench.md
	@echo "\`\`\`" >> bench.md
	@echo "OS: $$(go env GOOS)" >> bench.md
	@echo "Arch: $$(go env GOARCH)" >> bench.md
	@echo "Go Version: $$(go version | awk '{print $$3}')" >> bench.md
	@echo "CPU: $$(sysctl -n machdep.cpu.brand_string 2>/dev/null || grep -m 1 'model name' /proc/cpuinfo 2>/dev/null | cut -d: -f2 | xargs || echo 'Unknown')" >> bench.md
	@echo "\`\`\`" >> bench.md
	@echo "" >> bench.md
	@echo "## Benchmark Results" >> bench.md
	@echo "\`\`\`" >> bench.md
	@go test -bench=. -benchmem ./... >> bench.md 2>&1
	@echo "\`\`\`" >> bench.md
	@echo "Benchmark results saved to bench.md"

# Run specific benchmarks for comparison
bench-compare: ## Run router benchmarks for comparison
	@echo "Running router comparison benchmarks..."
	go test -bench=BenchmarkMux -benchmem ./...

# Run performance tests
perf: ## Run performance benchmarks (5s duration)
	@echo "Running performance tests..."
	go test -run=XXX -bench=. -benchtime=5s -benchmem ./...

# ==================== Profiling ====================

# Generate memory profile
profile-mem: ## Generate memory profile
	@echo "Generating memory profile..."
	go test -bench=BenchmarkMuxStaticRoute -benchmem -memprofile=mem.prof
	@echo "View profile with: go tool pprof mem.prof"

# Generate CPU profile
profile-cpu: ## Generate CPU profile
	@echo "Generating CPU profile..."
	go test -bench=BenchmarkMuxStaticRoute -cpuprofile=cpu.prof
	@echo "View profile with: go tool pprof cpu.prof"

# ==================== Dependencies ====================

# Tidy go modules
mod-tidy: ## Run go mod tidy
	@echo "Tidying modules..."
	go mod tidy

# Download dependencies
mod-download: ## Download go modules
	@echo "Downloading dependencies..."
	go mod download

# ==================== Development ====================

# Setup development environment
dev-setup: ## Setup development environment (install tools)
	@echo "Setting up development environment..."
	go mod download
	@echo "Installing development tools..."
	go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
	@echo "Development environment ready!"

# ==================== CI/CD ====================

# Run CI pipeline
ci: lint test ## Run CI pipeline (lint, test)

# ==================== Quick Commands ====================

# Quick test without verbose output
qt: ## Quick test (no verbose)
	@go test ./...

# Quick benchmark
qb: ## Quick benchmark (1s duration)
	@go test -bench=. -benchtime=1s ./...

# Watch for changes and run tests (requires entr)
watch: ## Watch files and run tests on change (requires entr)
	@which entr > /dev/null || (echo "entr not found. Install with your package manager."; exit 1)
	@find . -name "*.go" | entr -c make test