.PHONY: help build build-all test test-unit test-integration test-property test-all lint fmt clean \
        docker-build docker-run docker-test coverage deps validate-config run install

# Default target
.DEFAULT_GOAL := help

# Variables
BINARY_NAME=keyline
VERSION?=$(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
BUILD_TIME=$(shell date -u '+%Y-%m-%d_%H:%M:%S')
LDFLAGS=-ldflags "-w -s -X main.Version=$(VERSION) -X main.BuildTime=$(BUILD_TIME)"
DOCKER_IMAGE=keyline
DOCKER_TAG?=latest

# Help target
help: ## Show this help message
	@echo "Keyline - Authentication Proxy for Elasticsearch"
	@echo ""
	@echo "Usage: make [target]"
	@echo ""
	@echo "Targets:"
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "  %-20s %s\n", $$1, $$2}'

# Build targets
build: ## Build the binary
	@echo "Building $(BINARY_NAME)..."
	@mkdir -p bin
	go build $(LDFLAGS) -o bin/$(BINARY_NAME) ./cmd/keyline
	@echo "Binary built: bin/$(BINARY_NAME)"

build-all: ## Build for multiple platforms
	@echo "Building for multiple platforms..."
	@mkdir -p bin
	GOOS=linux GOARCH=amd64 go build $(LDFLAGS) -o bin/$(BINARY_NAME)-linux-amd64 ./cmd/keyline
	GOOS=linux GOARCH=arm64 go build $(LDFLAGS) -o bin/$(BINARY_NAME)-linux-arm64 ./cmd/keyline
	GOOS=darwin GOARCH=amd64 go build $(LDFLAGS) -o bin/$(BINARY_NAME)-darwin-amd64 ./cmd/keyline
	GOOS=darwin GOARCH=arm64 go build $(LDFLAGS) -o bin/$(BINARY_NAME)-darwin-arm64 ./cmd/keyline
	GOOS=windows GOARCH=amd64 go build $(LDFLAGS) -o bin/$(BINARY_NAME)-windows-amd64.exe ./cmd/keyline
	@echo "Binaries built in bin/"

install: build ## Install the binary to $GOPATH/bin
	@echo "Installing $(BINARY_NAME)..."
	go install $(LDFLAGS) ./cmd/keyline
	@echo "Installed to $(shell go env GOPATH)/bin/$(BINARY_NAME)"

# Test targets
test: test-unit ## Run unit tests (alias for test-unit)

test-unit: ## Run unit tests with race detection
	@echo "Running unit tests..."
	go test -v -race ./...

test-integration: ## Run integration tests
	@echo "Running integration tests..."
	go test -v -tags=integration ./integration/...

test-property: ## Run property-based tests
	@echo "Running property-based tests..."
	go test -v -tags=property ./...

test-all: ## Run all tests (unit, integration, property)
	@echo "Running all tests..."
	go test -v -race ./...
	go test -v -tags=integration ./integration/...
	go test -v -tags=property ./...

# Code quality targets
lint: ## Run linters
	@echo "Running linters..."
	go vet ./...
	@if command -v golangci-lint >/dev/null 2>&1; then \
		golangci-lint run; \
	else \
		echo "golangci-lint not installed, skipping"; \
	fi
	@echo "Checking formatting..."
	@test -z "$$(gofmt -l -s .)" || (echo "Code not formatted, run 'make fmt'" && exit 1)

fmt: ## Format code
	@echo "Formatting code..."
	gofmt -w -s .
	@if command -v goimports >/dev/null 2>&1; then \
		goimports -w .; \
	fi

# Coverage targets
coverage: ## Generate test coverage report
	@echo "Generating coverage report..."
	go test -coverprofile=coverage.out ./...
	go tool cover -html=coverage.out -o coverage.html
	@echo "Coverage report generated: coverage.html"
	@go tool cover -func=coverage.out | grep total | awk '{print "Total coverage: " $$3}'

coverage-integration: ## Generate integration test coverage report
	@echo "Generating integration test coverage report..."
	go test -coverprofile=coverage-integration.out -tags=integration ./integration/...
	go tool cover -html=coverage-integration.out -o coverage-integration.html
	@echo "Integration coverage report generated: coverage-integration.html"

# Docker targets
docker-build: ## Build Docker image
	@echo "Building Docker image..."
	docker build -t $(DOCKER_IMAGE):$(DOCKER_TAG) .
	@echo "Docker image built: $(DOCKER_IMAGE):$(DOCKER_TAG)"

docker-run: ## Run Docker container
	@echo "Running Docker container..."
	docker run -d \
		--name keyline \
		-p 9000:9000 \
		-v $(PWD)/config/config.example.yaml:/etc/keyline/config.yaml \
		$(DOCKER_IMAGE):$(DOCKER_TAG)
	@echo "Container started. Check logs with: docker logs keyline"

docker-stop: ## Stop Docker container
	@echo "Stopping Docker container..."
	docker stop keyline || true
	docker rm keyline || true

docker-test: docker-build ## Build and test Docker image
	@echo "Testing Docker image..."
	docker run --rm $(DOCKER_IMAGE):$(DOCKER_TAG) --version || true
	@echo "Docker image test complete"

# Development targets
run: ## Run the application
	@echo "Running $(BINARY_NAME)..."
	go run ./cmd/keyline --config config/config.example.yaml

run-debug: ## Run with debug logging
	@echo "Running $(BINARY_NAME) with debug logging..."
	LOG_LEVEL=debug go run ./cmd/keyline --config config/config.example.yaml

validate-config: ## Validate configuration file
	@echo "Validating configuration..."
	go run ./cmd/keyline --validate-config --config config/config.example.yaml

# Dependency targets
deps: ## Download and tidy dependencies
	@echo "Downloading dependencies..."
	go mod download
	go mod tidy
	@echo "Dependencies updated"

deps-upgrade: ## Upgrade all dependencies
	@echo "Upgrading dependencies..."
	go get -u ./...
	go mod tidy
	@echo "Dependencies upgraded"

# Clean targets
clean: ## Clean build artifacts
	@echo "Cleaning build artifacts..."
	rm -rf bin/
	rm -f coverage.out coverage.html
	rm -f coverage-integration.out coverage-integration.html
	@echo "Clean complete"

clean-all: clean ## Clean all artifacts including Docker images
	@echo "Cleaning Docker images..."
	docker rmi $(DOCKER_IMAGE):$(DOCKER_TAG) 2>/dev/null || true
	@echo "Clean all complete"

# CI targets
ci: lint test-all ## Run CI checks (lint + all tests)
	@echo "CI checks complete"

ci-coverage: lint test coverage ## Run CI with coverage
	@echo "CI with coverage complete"
