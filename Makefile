.PHONY: help test test-race test-coverage test-integration lint fmt vet build clean install

# Variables
GO_VERSION := 1.21
BINARY_NAME := ipe
CMD_PATH := ./cmd
COVERAGE_FILE := coverage.out

help: ## Show this help message
	@echo 'Usage: make [target]'
	@echo ''
	@echo 'Available targets:'
	@awk 'BEGIN {FS = ":.*?## "} /^[a-zA-Z_-]+:.*?## / {printf "  %-20s %s\n", $$1, $$2}' $(MAKEFILE_LIST)

test: ## Run all tests
	go test -v -timeout 5m ./...

test-race: ## Run tests with race detector
	go test -v -race -timeout 10m ./...

test-coverage: ## Run tests with coverage
	go test -v -race -coverprofile=$(COVERAGE_FILE) -covermode=atomic -timeout 10m ./...
	go tool cover -html=$(COVERAGE_FILE) -o coverage.html
	@echo "Coverage report generated: coverage.html"

test-unit: ## Run unit tests only (no Redis required)
	go test -v -timeout 5m \
		./api/... \
		./app/... \
		./channel/... \
		./connection/... \
		./events/... \
		./storage/... \
		./utils/... \
		./websockets/... \
		./concurrency/...

test-integration: ## Run integration tests (requires Redis)
	@if ! redis-cli ping > /dev/null 2>&1; then \
		echo "Error: Redis is not running. Please start Redis first."; \
		exit 1; \
	fi
	REDIS_HOST=localhost REDIS_PORT=6379 REDIS_DB=0 go test -v -race -timeout 10m ./redis/... ./integration/...

test-all: test-unit test-integration ## Run all tests (unit + integration)

lint: ## Run linters
	@if ! command -v golangci-lint > /dev/null; then \
		echo "Installing golangci-lint..."; \
		go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest; \
	fi
	golangci-lint run ./...

fmt: ## Format code
	go fmt ./...
	@echo "Code formatted"

fmt-check: ## Check if code is formatted
	@if [ "$(shell gofmt -l . | wc -l)" -gt 0 ]; then \
		echo "Code is not formatted. Run 'make fmt'"; \
		gofmt -d .; \
		exit 1; \
	fi
	@echo "Code is properly formatted"

vet: ## Run go vet
	go vet ./...

build: ## Build the binary
	go build -v -o $(BINARY_NAME) $(CMD_PATH)

build-race: ## Build with race detector enabled
	go build -race -v -o $(BINARY_NAME) $(CMD_PATH)

install: ## Install the binary
	go install $(CMD_PATH)

clean: ## Clean build artifacts
	go clean
	rm -f $(BINARY_NAME) $(BINARY_NAME).exe
	rm -f $(COVERAGE_FILE) coverage.html
	rm -f *.out

deps: ## Download dependencies
	go mod download
	go mod verify

deps-update: ## Update dependencies
	go get -u ./...
	go mod tidy

deps-check: ## Check for dependency updates
	go list -u -m all

ci: fmt-check vet lint test-race ## Run all CI checks locally

.DEFAULT_GOAL := help

