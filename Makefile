# Git Generator Makefile

# Variables
BINARY_NAME=git-generator
MAIN_PATH=cmd/git-generator/main.go
BUILD_DIR=build
VERSION?=1.0.0
LDFLAGS=-ldflags "-X main.version=$(VERSION)"

# Go parameters
GOCMD=go
GOBUILD=$(GOCMD) build
GOCLEAN=$(GOCMD) clean
GOTEST=$(GOCMD) test
GOGET=$(GOCMD) get
GOMOD=$(GOCMD) mod

# Build targets
.PHONY: all build clean test coverage deps help install uninstall

all: clean deps test build

# Build the binary
build:
	@echo "Building $(BINARY_NAME)..."
	@mkdir -p $(BUILD_DIR)
	$(GOBUILD) $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME) $(MAIN_PATH)
	@echo "Build complete: $(BUILD_DIR)/$(BINARY_NAME)"

# Build for multiple platforms
build-all: clean deps test
	@echo "Building for multiple platforms..."
	@mkdir -p $(BUILD_DIR)
	
	# Linux
	GOOS=linux GOARCH=amd64 $(GOBUILD) $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME)-linux-amd64 $(MAIN_PATH)
	GOOS=linux GOARCH=arm64 $(GOBUILD) $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME)-linux-arm64 $(MAIN_PATH)
	
	# macOS
	GOOS=darwin GOARCH=amd64 $(GOBUILD) $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME)-darwin-amd64 $(MAIN_PATH)
	GOOS=darwin GOARCH=arm64 $(GOBUILD) $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME)-darwin-arm64 $(MAIN_PATH)
	
	# Windows
	GOOS=windows GOARCH=amd64 $(GOBUILD) $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME)-windows-amd64.exe $(MAIN_PATH)
	
	@echo "Multi-platform build complete"

# Install the binary to GOPATH/bin
install: build
	@echo "Installing $(BINARY_NAME)..."
	cp $(BUILD_DIR)/$(BINARY_NAME) $(GOPATH)/bin/$(BINARY_NAME)
	@echo "Installation complete"

# Uninstall the binary
uninstall:
	@echo "Uninstalling $(BINARY_NAME)..."
	rm -f $(GOPATH)/bin/$(BINARY_NAME)
	@echo "Uninstall complete"

# Clean build artifacts
clean:
	@echo "Cleaning..."
	$(GOCLEAN)
	rm -rf $(BUILD_DIR)
	@echo "Clean complete"

# Download dependencies
deps:
	@echo "Downloading dependencies..."
	$(GOMOD) download
	$(GOMOD) tidy
	@echo "Dependencies updated"

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

# Lint the code
lint:
	@echo "Running linter..."
	golangci-lint run

# Format the code
fmt:
	@echo "Formatting code..."
	$(GOCMD) fmt ./...

# Vet the code
vet:
	@echo "Vetting code..."
	$(GOCMD) vet ./...

# Run all checks
check: fmt vet lint test

# Development build (faster, no optimizations)
dev:
	@echo "Building development version..."
	@mkdir -p $(BUILD_DIR)
	$(GOBUILD) -o $(BUILD_DIR)/$(BINARY_NAME) $(MAIN_PATH)

# Run the application
run: dev
	@echo "Running $(BINARY_NAME)..."
	./$(BUILD_DIR)/$(BINARY_NAME)

# Create a release
release: clean deps test build-all
	@echo "Creating release $(VERSION)..."
	@mkdir -p $(BUILD_DIR)/release
	
	# Create archives
	cd $(BUILD_DIR) && tar -czf release/$(BINARY_NAME)-$(VERSION)-linux-amd64.tar.gz $(BINARY_NAME)-linux-amd64
	cd $(BUILD_DIR) && tar -czf release/$(BINARY_NAME)-$(VERSION)-linux-arm64.tar.gz $(BINARY_NAME)-linux-arm64
	cd $(BUILD_DIR) && tar -czf release/$(BINARY_NAME)-$(VERSION)-darwin-amd64.tar.gz $(BINARY_NAME)-darwin-amd64
	cd $(BUILD_DIR) && tar -czf release/$(BINARY_NAME)-$(VERSION)-darwin-arm64.tar.gz $(BINARY_NAME)-darwin-arm64
	cd $(BUILD_DIR) && zip release/$(BINARY_NAME)-$(VERSION)-windows-amd64.zip $(BINARY_NAME)-windows-amd64.exe
	
	@echo "Release $(VERSION) created in $(BUILD_DIR)/release/"

# Generate documentation
docs:
	@echo "Generating documentation..."
	$(GOCMD) doc -all ./... > docs/api.md
	@echo "Documentation generated"

# Initialize development environment
init:
	@echo "Initializing development environment..."
	$(GOMOD) init github.com/nguyendkn/git-generator || true
	$(GOGET) github.com/spf13/cobra@latest
	$(GOGET) github.com/spf13/viper@latest
	$(GOGET) github.com/google/generative-ai-go@latest
	$(GOGET) github.com/stretchr/testify@latest
	@echo "Development environment initialized"

# Docker build
docker-build:
	@echo "Building Docker image..."
	docker build -t $(BINARY_NAME):$(VERSION) .
	docker build -t $(BINARY_NAME):latest .

# Docker run
docker-run:
	@echo "Running Docker container..."
	docker run --rm -it $(BINARY_NAME):latest

# Help
help:
	@echo "Git Generator Makefile"
	@echo ""
	@echo "Usage:"
	@echo "  make [target]"
	@echo ""
	@echo "Targets:"
	@echo "  all          Build everything (clean, deps, test, build)"
	@echo "  build        Build the binary"
	@echo "  build-all    Build for multiple platforms"
	@echo "  install      Install binary to GOPATH/bin"
	@echo "  uninstall    Remove binary from GOPATH/bin"
	@echo "  clean        Clean build artifacts"
	@echo "  deps         Download and update dependencies"
	@echo "  test         Run tests"
	@echo "  coverage     Run tests with coverage report"
	@echo "  test-race    Run tests with race detection"
	@echo "  bench        Run benchmarks"
	@echo "  lint         Run linter"
	@echo "  fmt          Format code"
	@echo "  vet          Vet code"
	@echo "  check        Run all checks (fmt, vet, lint, test)"
	@echo "  dev          Quick development build"
	@echo "  run          Build and run the application"
	@echo "  release      Create release archives"
	@echo "  docs         Generate documentation"
	@echo "  init         Initialize development environment"
	@echo "  docker-build Build Docker image"
	@echo "  docker-run   Run Docker container"
	@echo "  help         Show this help message"
	@echo ""
	@echo "Variables:"
	@echo "  VERSION      Version to build (default: $(VERSION))"
	@echo ""
	@echo "Examples:"
	@echo "  make build VERSION=1.2.0"
	@echo "  make release VERSION=1.2.0"
