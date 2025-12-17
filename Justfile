# justfile for bored - A TUI for Azure DevOps Boards

# Default recipe to list available commands
default:
    @just --list

# Build the application
build:
    go build -o bin/bored .

# Run the application
run:
    go run .

# Run tests
test:
    go test ./...

# Run tests with verbose output
test-v:
    go test -v ./...

# Format code
fmt:
    go fmt ./...

# Tidy dependencies
tidy:
    go mod tidy

# Vet code for potential issues
vet:
    go vet ./...

# Run all checks (fmt, vet, test)
check: fmt vet test

# Clean build artifacts
clean:
    rm -rf bin/

# Build and run
dev: build
    ./bin/bored

# Install the application to GOPATH/bin
install:
    go install .
