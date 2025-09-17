# CloudRecon Makefile

# Variables
BINARY_NAME=cloudrecon
VERSION=$(shell git describe --tags --always --dirty)
COMMIT=$(shell git rev-parse HEAD)
DATE=$(shell date -u +%Y-%m-%dT%H:%M:%SZ)
LDFLAGS=-ldflags "-X main.version=$(VERSION) -X main.commit=$(COMMIT) -X main.date=$(DATE)"

# Go parameters
GOCMD=go
GOBUILD=$(GOCMD) build
GOCLEAN=$(GOCMD) clean
GOTEST=$(GOCMD) test
GOGET=$(GOCMD) get
GOMOD=$(GOCMD) mod

# Build flags
BUILD_FLAGS=-v
TEST_FLAGS=-v -race -coverprofile=coverage.out
LINT_FLAGS=--timeout=5m

# Directories
CMD_DIR=./cmd/cloudrecon
DIST_DIR=./dist
COVERAGE_DIR=./coverage

.PHONY: all build clean test test-unit test-integration test-e2e test-performance test-loadtest lint fmt vet security docker help

# Default target
all: clean fmt vet lint test build

# Build the binary
build:
	@echo "Building $(BINARY_NAME)..."
	$(GOBUILD) $(BUILD_FLAGS) $(LDFLAGS) -o $(BINARY_NAME) $(CMD_DIR)

# Build for multiple platforms
build-all:
	@echo "Building for multiple platforms..."
	@mkdir -p $(DIST_DIR)
	
	# Linux AMD64
	GOOS=linux GOARCH=amd64 $(GOBUILD) $(BUILD_FLAGS) $(LDFLAGS) -o $(DIST_DIR)/$(BINARY_NAME)-linux-amd64 $(CMD_DIR)
	
	# Linux ARM64
	GOOS=linux GOARCH=arm64 $(GOBUILD) $(BUILD_FLAGS) $(LDFLAGS) -o $(DIST_DIR)/$(BINARY_NAME)-linux-arm64 $(CMD_DIR)
	
	# Windows AMD64
	GOOS=windows GOARCH=amd64 $(GOBUILD) $(BUILD_FLAGS) $(LDFLAGS) -o $(DIST_DIR)/$(BINARY_NAME)-windows-amd64.exe $(CMD_DIR)
	
	# macOS AMD64
	GOOS=darwin GOARCH=amd64 $(GOBUILD) $(BUILD_FLAGS) $(LDFLAGS) -o $(DIST_DIR)/$(BINARY_NAME)-darwin-amd64 $(CMD_DIR)
	
	# macOS ARM64
	GOOS=darwin GOARCH=arm64 $(GOBUILD) $(BUILD_FLAGS) $(LDFLAGS) -o $(DIST_DIR)/$(BINARY_NAME)-darwin-arm64 $(CMD_DIR)
	
	@echo "Build complete. Binaries are in $(DIST_DIR)/"

# Clean build artifacts
clean:
	@echo "Cleaning..."
	$(GOCLEAN)
	rm -f $(BINARY_NAME)
	rm -rf $(DIST_DIR)
	rm -rf $(COVERAGE_DIR)
	rm -f coverage.out coverage.html

# Format code
fmt:
	@echo "Formatting code..."
	$(GOCMD) fmt ./...

# Run go vet
vet:
	@echo "Running go vet..."
	$(GOCMD) vet ./...

# Run linter
lint:
	@echo "Running linter..."
	golangci-lint run $(LINT_FLAGS)

# Run security scan
security:
	@echo "Running security scan..."
	gosec ./...

# Run all tests
test: test-unit test-integration test-e2e test-performance test-loadtest

# Run unit tests
test-unit:
	@echo "Running unit tests..."
	$(GOTEST) $(TEST_FLAGS) ./internal/...

# Run integration tests
test-integration:
	@echo "Running integration tests..."
	$(GOTEST) $(TEST_FLAGS) -tags=integration ./tests/integration/...

# Run E2E tests
test-e2e:
	@echo "Running E2E tests..."
	$(GOTEST) $(TEST_FLAGS) -tags=e2e ./tests/e2e/...

# Run performance tests
test-performance:
	@echo "Running performance tests..."
	$(GOTEST) $(TEST_FLAGS) -tags=performance -bench=. -benchmem ./tests/performance/...

# Run load tests
test-loadtest:
	@echo "Running load tests..."
	$(GOTEST) $(TEST_FLAGS) -tags=loadtest ./tests/loadtest/...

# Generate coverage report
coverage:
	@echo "Generating coverage report..."
	@mkdir -p $(COVERAGE_DIR)
	$(GOTEST) $(TEST_FLAGS) ./internal/...
	$(GOCMD) tool cover -html=coverage.out -o $(COVERAGE_DIR)/coverage.html
	@echo "Coverage report generated in $(COVERAGE_DIR)/coverage.html"

# Run benchmarks
bench:
	@echo "Running benchmarks..."
	$(GOTEST) -bench=. -benchmem ./...

# Download dependencies
deps:
	@echo "Downloading dependencies..."
	$(GOMOD) download
	$(GOMOD) tidy

# Update dependencies
update-deps:
	@echo "Updating dependencies..."
	$(GOCMD) get -u ./...
	$(GOMOD) tidy

# Build Docker image
docker:
	@echo "Building Docker image..."
	docker build -t $(BINARY_NAME):$(VERSION) .
	docker build -t $(BINARY_NAME):latest .

# Run Docker container
docker-run:
	@echo "Running Docker container..."
	docker run --rm -it $(BINARY_NAME):latest

# Run with docker-compose
compose-up:
	@echo "Starting services with docker-compose..."
	docker-compose up -d

# Stop docker-compose services
compose-down:
	@echo "Stopping docker-compose services..."
	docker-compose down

# Install the binary
install: build
	@echo "Installing $(BINARY_NAME)..."
	cp $(BINARY_NAME) /usr/local/bin/

# Uninstall the binary
uninstall:
	@echo "Uninstalling $(BINARY_NAME)..."
	rm -f /usr/local/bin/$(BINARY_NAME)

# Create release
release: clean build-all
	@echo "Creating release..."
	@mkdir -p $(DIST_DIR)
	cd $(DIST_DIR) && sha256sum * > checksums.txt
	@echo "Release artifacts created in $(DIST_DIR)/"

# Run the application
run: build
	@echo "Running $(BINARY_NAME)..."
	./$(BINARY_NAME)

# Run with specific command
run-discover: build
	@echo "Running discovery..."
	./$(BINARY_NAME) discover --help

# Run with query
run-query: build
	@echo "Running query..."
	./$(BINARY_NAME) query "SELECT * FROM resources LIMIT 1"

# Run with export
run-export: build
	@echo "Running export..."
	./$(BINARY_NAME) export --format json --output /tmp/test-export.json

# Show help
help:
	@echo "Available targets:"
	@echo "  all              - Clean, format, vet, lint, test, and build"
	@echo "  build            - Build the binary"
	@echo "  build-all        - Build for multiple platforms"
	@echo "  clean            - Clean build artifacts"
	@echo "  fmt              - Format code"
	@echo "  vet              - Run go vet"
	@echo "  lint             - Run linter"
	@echo "  security         - Run security scan"
	@echo "  test             - Run all tests"
	@echo "  test-unit        - Run unit tests"
	@echo "  test-integration - Run integration tests"
	@echo "  test-e2e         - Run E2E tests"
	@echo "  test-performance - Run performance tests"
	@echo "  test-loadtest    - Run load tests"
	@echo "  coverage         - Generate coverage report"
	@echo "  bench            - Run benchmarks"
	@echo "  deps             - Download dependencies"
	@echo "  update-deps      - Update dependencies"
	@echo "  docker           - Build Docker image"
	@echo "  docker-run       - Run Docker container"
	@echo "  compose-up       - Start services with docker-compose"
	@echo "  compose-down     - Stop docker-compose services"
	@echo "  install          - Install the binary"
	@echo "  uninstall        - Uninstall the binary"
	@echo "  release          - Create release"
	@echo "  run              - Run the application"
	@echo "  run-discover     - Run discovery command"
	@echo "  run-query        - Run query command"
	@echo "  run-export       - Run export command"
	@echo "  help             - Show this help"