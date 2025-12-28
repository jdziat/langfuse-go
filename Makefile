# Makefile for langfuse-go

# Go parameters
GOCMD=go
GOBUILD=$(GOCMD) build
GOTEST=$(GOCMD) test
GOVET=$(GOCMD) vet
GOFMT=$(GOCMD) fmt
GOMOD=$(GOCMD) mod
GOGET=$(GOCMD) get
GOGENERATE=$(GOCMD) generate
GOCOVER=$(GOCMD) tool cover

# Build parameters
BINARY_NAME=langfuse-hooks
CMD_PATH=./cmd/langfuse-hooks
BUILD_DIR=bin

# Coverage
COVERAGE_FILE=coverage.out
COVERAGE_HTML=coverage.html

# Linting
GOLANGCI_LINT_VERSION=v1.62.2

.PHONY: all build test test-race test-cover test-integration bench clean fmt vet lint lint-fix tidy deps generate help install-tools check

# Default target
all: check build

## Build targets
build: ## Build the CLI binary
	@mkdir -p $(BUILD_DIR)
	$(GOBUILD) -o $(BUILD_DIR)/$(BINARY_NAME) $(CMD_PATH)

build-all: ## Build for multiple platforms
	@mkdir -p $(BUILD_DIR)
	GOOS=linux GOARCH=amd64 $(GOBUILD) -o $(BUILD_DIR)/$(BINARY_NAME)-linux-amd64 $(CMD_PATH)
	GOOS=linux GOARCH=arm64 $(GOBUILD) -o $(BUILD_DIR)/$(BINARY_NAME)-linux-arm64 $(CMD_PATH)
	GOOS=darwin GOARCH=amd64 $(GOBUILD) -o $(BUILD_DIR)/$(BINARY_NAME)-darwin-amd64 $(CMD_PATH)
	GOOS=darwin GOARCH=arm64 $(GOBUILD) -o $(BUILD_DIR)/$(BINARY_NAME)-darwin-arm64 $(CMD_PATH)
	GOOS=windows GOARCH=amd64 $(GOBUILD) -o $(BUILD_DIR)/$(BINARY_NAME)-windows-amd64.exe $(CMD_PATH)

## Test targets
test: ## Run unit tests
	$(GOTEST) -v ./...

test-short: ## Run short tests only
	$(GOTEST) -v -short ./...

test-race: ## Run tests with race detector
	$(GOTEST) -v -race ./...

test-cover: ## Run tests with coverage
	$(GOTEST) -v -coverprofile=$(COVERAGE_FILE) -covermode=atomic ./...
	$(GOCOVER) -func=$(COVERAGE_FILE)

test-cover-html: test-cover ## Generate HTML coverage report
	$(GOCOVER) -html=$(COVERAGE_FILE) -o $(COVERAGE_HTML)
	@echo "Coverage report generated: $(COVERAGE_HTML)"

test-integration: ## Run integration tests
	$(GOTEST) -v -tags=integration ./...

bench: ## Run benchmarks
	$(GOTEST) -bench=. -benchmem ./...

bench-cpu: ## Run benchmarks with CPU profiling
	$(GOTEST) -bench=. -benchmem -cpuprofile=cpu.prof ./...

bench-mem: ## Run benchmarks with memory profiling
	$(GOTEST) -bench=. -benchmem -memprofile=mem.prof ./...

## Code quality targets
fmt: ## Format code
	$(GOFMT) ./...

vet: ## Run go vet
	$(GOVET) ./...

lint: ## Run golangci-lint
	@which golangci-lint > /dev/null || (echo "golangci-lint not installed. Run 'make install-tools'" && exit 1)
	golangci-lint run ./...

lint-fix: ## Run golangci-lint with auto-fix
	@which golangci-lint > /dev/null || (echo "golangci-lint not installed. Run 'make install-tools'" && exit 1)
	golangci-lint run --fix ./...

staticcheck: ## Run staticcheck
	@which staticcheck > /dev/null || (echo "staticcheck not installed. Run 'make install-tools'" && exit 1)
	staticcheck ./...

## Dependency targets
tidy: ## Tidy go modules
	$(GOMOD) tidy

deps: ## Download dependencies
	$(GOMOD) download

deps-update: ## Update all dependencies
	$(GOGET) -u ./...
	$(GOMOD) tidy

deps-verify: ## Verify dependencies
	$(GOMOD) verify

## Development targets
generate: ## Run go generate
	$(GOGENERATE) ./...

clean: ## Clean build artifacts
	rm -rf $(BUILD_DIR)
	rm -f $(COVERAGE_FILE) $(COVERAGE_HTML)
	rm -f cpu.prof mem.prof
	$(GOCMD) clean -cache -testcache

## Tool installation
install-tools: ## Install development tools
	@echo "Installing golangci-lint..."
	go install github.com/golangci/golangci-lint/cmd/golangci-lint@$(GOLANGCI_LINT_VERSION)
	@echo "Installing staticcheck..."
	go install honnef.co/go/tools/cmd/staticcheck@latest
	@echo "Installing goimports..."
	go install golang.org/x/tools/cmd/goimports@latest
	@echo "Tools installed successfully"

## CI/CD targets
check: fmt vet ## Run all checks (format, vet)
	@echo "All checks passed"

ci: deps check lint test-race test-cover ## Run full CI pipeline
	@echo "CI pipeline completed"

## Documentation
doc: ## Start godoc server
	@echo "Starting godoc server at http://localhost:6060"
	godoc -http=:6060

## Help
help: ## Show this help
	@echo "Available targets:"
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "  \033[36m%-18s\033[0m %s\n", $$1, $$2}'
