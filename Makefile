.PHONY: build test lint run clean docker-build docker-run property-test integration-test coverage

# Build the binary
build:
	go build -o bin/keyline ./cmd/keyline

# Run tests
test:
	go test -v -race ./...

# Run linter
lint:
	go vet ./...
	gofmt -l -s .

# Run the application
run:
	go run ./cmd/keyline

# Clean build artifacts
clean:
	rm -rf bin/

# Build Docker image
docker-build:
	docker build -t keyline:latest .

# Run Docker container
docker-run:
	docker run -p 9000:9000 -v $(PWD)/config:/config keyline:latest

# Run property-based tests
property-test:
	go test -v -tags=property ./...

# Run integration tests
integration-test:
	go test -v -tags=integration ./integration/...

# Generate test coverage report
coverage:
	go test -coverprofile=coverage.out ./...
	go tool cover -html=coverage.out -o coverage.html
	@echo "Coverage report generated: coverage.html"

# Format code
fmt:
	gofmt -w -s .
	goimports -w .

# Install dependencies
deps:
	go mod download
	go mod tidy

# Validate configuration
validate-config:
	go run ./cmd/keyline --validate-config
