# Makefile for Reverse Proxy

# Build targets
.PHONY: all build test run clean lint docker

# Default target
all: build

# Build the application
build:
	@echo "Building reverse-proxy..."
	@go build -o bin/reverse-proxy ./cmd/app/
	@echo "Build complete: bin/reverse-proxy"

# Run tests
test:
	@echo "Running tests..."
	@go test ./... -v -cover

# Run the application
run:
	@echo "Starting reverse-proxy..."
	@./bin/reverse-proxy --config ./sample/config/

# Clean build artifacts
clean:
	@echo "Cleaning..."
	@rm -rf bin/
	@echo "Clean complete"

# Run linter
lint:
	@echo "Running linter..."
	@go vet ./...
	@golangci-lint run

# Build Docker image
docker:
	@echo "Building Docker image..."
	@docker build -t reverse-proxy .
	@echo "Docker image built: reverse-proxy"

# Generate documentation
docs:
	@echo "Generating documentation..."
	@go doc -all > docs/generated.txt
	@echo "Documentation generated in docs/generated.txt"

# Format code
fmt:
	@echo "Formatting code..."
	@gofmt -w .
	@echo "Code formatting complete"

# Install dependencies
deps:
	@echo "Installing dependencies..."
	@go mod tidy
	@echo "Dependencies installed"

# Run with sample configuration
sample:
	@echo "Running with sample configuration..."
	@./bin/reverse-proxy --config ./sample/config/

# Build and run in one step
dev:
	@make build
	@make run

# Help
help:
	@echo "Available targets:"
	@echo "  all       - Build the application (default)"
	@echo "  build     - Build the application"
	@echo "  test      - Run tests"
	@echo "  run       - Run the application"
	@echo "  clean     - Clean build artifacts"
	@echo "  lint      - Run linter"
	@echo "  docker    - Build Docker image"
	@echo "  docs      - Generate documentation"
	@echo "  fmt       - Format code"
	@echo "  deps      - Install dependencies"
	@echo "  sample    - Run with sample configuration"
	@echo "  dev       - Build and run"
	@echo "  help      - Show this help message"