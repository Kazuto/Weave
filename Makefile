BINARY_NAME=weave
VERSION?=$(shell git describe --tags --always --dirty)
BUILD_TIME=$(shell date -u '+%Y-%m-%d_%H:%M:%S')
LDFLAGS=-ldflags "-X github.com/Kazuto/Weave/pkg/version.Version=$(VERSION) -X github.com/Kazuto/Weave/pkg/version.BuildTime=$(BUILD_TIME)"

# Default target
.PHONY: all
all: clean test build

# Build the binary for current platform
.PHONY: build
build:
	@echo "Building $(BINARY_NAME) $(VERSION)..."
	go build $(LDFLAGS) -o $(BINARY_NAME)

# Build for all supported platforms
.PHONY: build-all
build-all: clean
	@echo "Building for all platforms..."
	@mkdir -p dist
	
	# Linux
	GOOS=linux GOARCH=amd64 go build $(LDFLAGS) -o dist/$(BINARY_NAME)-linux-amd64
	GOOS=linux GOARCH=arm64 go build $(LDFLAGS) -o dist/$(BINARY_NAME)-linux-arm64
	
	# macOS
	GOOS=darwin GOARCH=amd64 go build $(LDFLAGS) -o dist/$(BINARY_NAME)-darwin-amd64
	GOOS=darwin GOARCH=arm64 go build $(LDFLAGS) -o dist/$(BINARY_NAME)-darwin-arm64
	
	# Windows
	GOOS=windows GOARCH=amd64 go build $(LDFLAGS) -o dist/$(BINARY_NAME)-windows-amd64.exe
	
	@echo "Built binaries in dist/ directory"

# Run tests
.PHONY: test
test:
	@echo "Running tests..."
	go test -v ./...

# Run tests with coverage
.PHONY: test-coverage
test-coverage:
	@echo "Running tests with coverage..."
	go test -cover ./...
	go test -coverprofile=coverage.out ./...
	go tool cover -html=coverage.out -o coverage.html
	@echo "Coverage report generated: coverage.html"

# Run tests with race detection
.PHONY: test-race
test-race:
	@echo "Running tests with race detection..."
	go test -race ./...

# Clean build artifacts
.PHONY: clean
clean:
	@echo "Cleaning..."
	rm -f $(BINARY_NAME)
	rm -rf dist/
	rm -f coverage.out coverage.html

# Install dependencies
.PHONY: deps
deps:
	@echo "Installing dependencies..."
	go mod download
	go mod tidy

# Run development version
.PHONY: run
run:
	@echo "Running development version..."
	go run main.go

# Run with example flags
.PHONY: run-example
run-example:
	@echo "Running branch example..."
	go run . branch --type feature --title "Example feature" PROJ-123

# Install locally
.PHONY: install
install: build
	@echo "Installing $(BINARY_NAME) to /usr/local/bin/..."
	sudo mv $(BINARY_NAME) /usr/local/bin/

# Uninstall
.PHONY: uninstall
uninstall:
	@echo "Removing $(BINARY_NAME) from /usr/local/bin/..."
	sudo rm -f /usr/local/bin/$(BINARY_NAME)

# Development setup
.PHONY: dev-setup
dev-setup: deps
	@echo "Setting up development environment..."
	@if ! command -v golangci-lint >/dev/null 2>&1; then \
		echo "Installing golangci-lint..."; \
		go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest; \
	fi
	@echo "Development setup complete!"

# Lint code
.PHONY: lint
lint:
	@echo "Running linter..."
	golangci-lint run

# Format code
.PHONY: fmt
fmt:
	@echo "Formatting code..."
	go fmt ./...

# Check for security issues
.PHONY: security
security:
	@echo "Running security checks..."
	@if command -v gosec >/dev/null 2>&1; then \
		gosec ./...; \
	else \
		echo "gosec not installed. Install with: go install github.com/securecodewarrior/gosec/v2/cmd/gosec@latest"; \
	fi

# Generate release archives
.PHONY: release
release: build-all
	@echo "Creating release archives..."
	@mkdir -p dist/archives
	
	# Create tar.gz for Unix systems
	cd dist && tar -czf archives/$(BINARY_NAME)-$(VERSION)-linux-amd64.tar.gz $(BINARY_NAME)-linux-amd64
	cd dist && tar -czf archives/$(BINARY_NAME)-$(VERSION)-linux-arm64.tar.gz $(BINARY_NAME)-linux-arm64
	cd dist && tar -czf archives/$(BINARY_NAME)-$(VERSION)-darwin-amd64.tar.gz $(BINARY_NAME)-darwin-amd64
	cd dist && tar -czf archives/$(BINARY_NAME)-$(VERSION)-darwin-arm64.tar.gz $(BINARY_NAME)-darwin-arm64
	
	# Create zip for Windows
	cd dist && zip archives/$(BINARY_NAME)-$(VERSION)-windows-amd64.zip $(BINARY_NAME)-windows-amd64.exe
	
	@echo "Release archives created in dist/archives/"

# Show help
.PHONY: help
help:
	@echo "Weave Build Commands:"
	@echo ""
	@echo "  build         Build binary for current platform"
	@echo "  build-all     Build binaries for all supported platforms"
	@echo "  test          Run tests"
	@echo "  test-coverage Run tests with coverage report"
	@echo "  test-race     Run tests with race detection"
	@echo "  clean         Clean build artifacts"
	@echo "  deps          Install/update dependencies"
	@echo "  run           Run development version"
	@echo "  run-example   Run with example flags (dry-run)"
	@echo "  install       Install binary to /usr/local/bin/"
	@echo "  uninstall     Remove binary from /usr/local/bin/"
	@echo "  dev-setup     Set up development environment"
	@echo "  lint          Run code linter"
	@echo "  fmt           Format code"
	@echo "  security      Run security checks"
	@echo "  release       Create release archives"
	@echo "  help          Show this help message"
	@echo ""
	@echo "Examples:"
	@echo "  make build              # Build for current platform"
	@echo "  make test               # Run all tests"
	@echo "  make build-all          # Build for all platforms"
	@echo "  make install            # Build and install locally"
