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

# Run tests with coverage
cover:
    go test -coverprofile=coverage.out ./...
    go tool cover -func=coverage.out

# Run tests with coverage and open HTML report
cover-html:
    go test -coverprofile=coverage.out ./...
    go tool cover -html=coverage.out -o coverage.html
    open coverage.html

# Serve godoc documentation locally
doc:
    @echo "Starting godoc server at http://localhost:6060/pkg/github.com/laupski/bored/"
    go run golang.org/x/tools/cmd/godoc@latest -http=:6060

# Format code
fmt:
    go fmt ./...

# Tidy dependencies
tidy:
    go mod tidy

# Vet code for potential issues
vet:
    go vet ./...

# Run golangci-lint for static analysis
lint:
    golangci-lint run

# Run all checks (fmt, vet, lint, test)
check: fmt vet lint test cover

# Clean build artifacts
clean:
    rm -rf bin/

# Build and run
dev: build
    ./bin/bored

# Install the application to GOPATH/bin
install:
    go install .
