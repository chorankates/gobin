.PHONY: all build test test-coverage clean run lint

# Default target
all: test build

# Build the server
build:
	@echo "Building server..."
	@go build -o bin/server cmd/server/main.go

# Run the server
run:
	@echo "Running server..."
	@go run cmd/server/main.go

# Run tests
test:
	@echo "Running tests..."
	@go test -v ./...

# Run tests with coverage
test-coverage:
	@echo "Running tests with coverage..."
	@go test -v -coverprofile=coverage.out ./...
	@go tool cover -html=coverage.out -o coverage.html
	@echo "Coverage report generated at coverage.html"

# Clean build artifacts
clean:
	@echo "Cleaning..."
	@rm -rf bin/
	@rm -f coverage.out coverage.html

# Install development dependencies
deps:
	@echo "Installing development dependencies..."
	@go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest

# Run linter
lint:
	@echo "Running linter..."
	@golangci-lint run

# Help target
help:
	@echo "Available targets:"
	@echo "  all            - Run tests and build the server"
	@echo "  build          - Build the server"
	@echo "  run            - Run the server"
	@echo "  test           - Run tests"
	@echo "  test-coverage  - Run tests with coverage report"
	@echo "  clean          - Remove build artifacts"
	@echo "  deps           - Install development dependencies"
	@echo "  lint           - Run linter"
	@echo "  help           - Show this help message" 