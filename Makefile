# Makefile for interop project
# Go parameters
GOCMD=go
GOBUILD=$(GOCMD) build
GOCLEAN=$(GOCMD) clean
GOTEST=$(GOCMD) test
GOGET=$(GOCMD) get
GOMOD=$(GOCMD) mod
GOFMT=$(GOCMD) fmt
GOVET=$(GOCMD) vet

# Binary names
BINARY_NAME=interop
BINARY_UNIX=$(BINARY_NAME)_unix
BINARY_WINDOWS=$(BINARY_NAME).exe
BINARY_DARWIN=$(BINARY_NAME)_darwin

# Directories
CMD_DIR=./cmd/cli
BUILD_DIR=./bin
DIST_DIR=./dist

# Version information
VERSION ?= $(shell git describe --tags --always --dirty)
COMMIT ?= $(shell git rev-parse --short HEAD)
DATE ?= $(shell date +%Y-%m-%d)

# Build flags
LDFLAGS=-ldflags "-X main.version=$(VERSION) -X main.commit=$(COMMIT) -X main.date=$(DATE)"

.PHONY: all build clean test coverage deps fmt vet lint install uninstall run help
.PHONY: build-linux build-darwin build-windows build-all
.PHONY: release docker-build docker-run

# Default target
all: test build

## Build targets

# Build the binary for current platform
build:
	@echo "Building $(BINARY_NAME)..."
	@mkdir -p $(BUILD_DIR)
	$(GOBUILD) $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME) -v $(CMD_DIR)

# Build for Linux
build-linux:
	@echo "Building for Linux..."
	@mkdir -p $(BUILD_DIR)
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 $(GOBUILD) $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_UNIX) -v $(CMD_DIR)

# Build for macOS
build-darwin:
	@echo "Building for macOS..."
	@mkdir -p $(BUILD_DIR)
	CGO_ENABLED=0 GOOS=darwin GOARCH=amd64 $(GOBUILD) $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_DARWIN) -v $(CMD_DIR)

# Build for Windows
build-windows:
	@echo "Building for Windows..."
	@mkdir -p $(BUILD_DIR)
	CGO_ENABLED=0 GOOS=windows GOARCH=amd64 $(GOBUILD) $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_WINDOWS) -v $(CMD_DIR)

# Build for all platforms
build-all: build-linux build-darwin build-windows

## Test targets

# Run tests
test:
	@echo "Running tests..."
	$(GOTEST) -v ./...

# Run tests with coverage
coverage:
	@echo "Running tests with coverage..."
	$(GOTEST) -v -coverprofile=coverage.out ./...
	$(GOCMD) tool cover -html=coverage.out -o coverage.html
	@echo "Coverage report generated: coverage.html"

# Run tests with race detection
test-race:
	@echo "Running tests with race detection..."
	$(GOTEST) -v -race ./...

# Run benchmarks
bench:
	@echo "Running benchmarks..."
	$(GOTEST) -bench=. -benchmem ./...

## Code quality targets

# Format code
fmt:
	@echo "Formatting code..."
	$(GOFMT) ./...

# Run go vet
vet:
	@echo "Running go vet..."
	$(GOVET) ./...

# Run golangci-lint (requires golangci-lint to be installed)
lint:
	@echo "Running linter..."
	@if command -v golangci-lint >/dev/null 2>&1; then \
		golangci-lint run; \
	else \
		echo "golangci-lint not installed. Install with: go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest"; \
	fi

# Check code quality (fmt, vet, lint)
check: fmt vet lint

## Dependency management

# Download dependencies
deps:
	@echo "Downloading dependencies..."
	$(GOMOD) download

# Tidy dependencies
tidy:
	@echo "Tidying dependencies..."
	$(GOMOD) tidy

# Verify dependencies
verify:
	@echo "Verifying dependencies..."
	$(GOMOD) verify

# Update dependencies
update:
	@echo "Updating dependencies..."
	$(GOGET) -u ./...
	$(GOMOD) tidy

## Installation targets

# Install binary to GOPATH/bin
install: build
	@echo "Installing $(BINARY_NAME)..."
	$(GOBUILD) $(LDFLAGS) -o $(GOPATH)/bin/$(BINARY_NAME) $(CMD_DIR)

# Uninstall binary from GOPATH/bin
uninstall:
	@echo "Uninstalling $(BINARY_NAME)..."
	@rm -f $(GOPATH)/bin/$(BINARY_NAME)

## Development targets

# Run the application
run: build
	@echo "Running $(BINARY_NAME)..."
	$(BUILD_DIR)/$(BINARY_NAME)

# Run with arguments (usage: make run-with ARGS="commands")
run-with: build
	@echo "Running $(BINARY_NAME) with args: $(ARGS)"
	$(BUILD_DIR)/$(BINARY_NAME) $(ARGS)

# Clean build artifacts
clean:
	@echo "Cleaning..."
	$(GOCLEAN)
	@rm -rf $(BUILD_DIR)
	@rm -rf $(DIST_DIR)
	@rm -f coverage.out coverage.html

## Release targets

# Create release using goreleaser (requires goreleaser to be installed)
release:
	@echo "Creating release..."
	@if command -v goreleaser >/dev/null 2>&1; then \
		goreleaser release --rm-dist; \
	else \
		echo "goreleaser not installed. Install with: go install github.com/goreleaser/goreleaser@latest"; \
	fi

# Create snapshot release (no git tags required)
release-snapshot:
	@echo "Creating snapshot release..."
	@if command -v goreleaser >/dev/null 2>&1; then \
		goreleaser release --snapshot --rm-dist; \
	else \
		echo "goreleaser not installed. Install with: go install github.com/goreleaser/goreleaser@latest"; \
	fi

## Docker targets (if needed)

# Build docker image
docker-build:
	@echo "Building Docker image..."
	docker build -t $(BINARY_NAME):$(VERSION) .

# Run docker container
docker-run: docker-build
	@echo "Running Docker container..."
	docker run --rm -it $(BINARY_NAME):$(VERSION)

## Utility targets

# Show build information
info:
	@echo "Build Information:"
	@echo "  Version: $(VERSION)"
	@echo "  Commit:  $(COMMIT)"
	@echo "  Date:    $(DATE)"
	@echo "  Binary:  $(BINARY_NAME)"

# Show help
help:
	@echo "Available targets:"
	@echo ""
	@echo "Build targets:"
	@echo "  build          Build binary for current platform"
	@echo "  build-linux    Build binary for Linux"
	@echo "  build-darwin   Build binary for macOS"
	@echo "  build-windows  Build binary for Windows"
	@echo "  build-all      Build binaries for all platforms"
	@echo ""
	@echo "Test targets:"
	@echo "  test           Run tests"
	@echo "  coverage       Run tests with coverage report"
	@echo "  test-race      Run tests with race detection"
	@echo "  bench          Run benchmarks"
	@echo ""
	@echo "Code quality targets:"
	@echo "  fmt            Format code"
	@echo "  vet            Run go vet"
	@echo "  lint           Run golangci-lint"
	@echo "  check          Run fmt, vet, and lint"
	@echo ""
	@echo "Dependency targets:"
	@echo "  deps           Download dependencies"
	@echo "  tidy           Tidy dependencies"
	@echo "  verify         Verify dependencies"
	@echo "  update         Update dependencies"
	@echo ""
	@echo "Installation targets:"
	@echo "  install        Install binary to GOPATH/bin"
	@echo "  uninstall      Remove binary from GOPATH/bin"
	@echo ""
	@echo "Development targets:"
	@echo "  run            Build and run the application"
	@echo "  run-with       Run with arguments (usage: make run-with ARGS='commands')"
	@echo "  clean          Clean build artifacts"
	@echo ""
	@echo "Release targets:"
	@echo "  release        Create release using goreleaser"
	@echo "  release-snapshot Create snapshot release"
	@echo ""
	@echo "Docker targets:"
	@echo "  docker-build   Build Docker image"
	@echo "  docker-run     Build and run Docker container"
	@echo ""
	@echo "Utility targets:"
	@echo "  info           Show build information"
	@echo "  help           Show this help message" 