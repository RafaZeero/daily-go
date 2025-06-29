.PHONY: build run clean test help

# Default target
help:
	@echo "Available commands:"
	@echo "  build    - Build the application"
	@echo "  run      - Run the application"
	@echo "  clean    - Clean build artifacts"
	@echo "  test     - Run tests"
	@echo "  deps     - Install dependencies"

# Build the application
build:
	go build -o daily-go .

# Run the application
run:
	go run .

# Clean build artifacts
clean:
	rm -f daily-go

# Run tests
test:
	go test ./...

# Install dependencies
deps:
	go mod tidy
	go mod download 