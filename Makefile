# Makefile for multistage_pipeline_fanout
# A comprehensive Makefile with best practices for Go projects

# Variables
BINARY_NAME=multistage_pipeline_fanout
MAIN_PACKAGE=./cmd
GO=go
GOTEST=$(GO) test
GOVET=$(GO) vet
GOBUILD=$(GO) build
GOCLEAN=$(GO) clean
GOGET=$(GO) get
GOFMT=$(GO) fmt
GOLINT=golangci-lint
GOSEC=gosec
GODOC=godoc
GOBENCHMARK=$(GO) test -bench=. -benchmem
INPUT_FILE=test.txt
OUTPUT_FILE=output.bin

# Build settings
BUILD_DIR=build
VERSION=$(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
BUILD_TIME=$(shell date -u '+%Y-%m-%d_%H:%M:%S')
COMMIT=$(shell git rev-parse --short HEAD 2>/dev/null || echo "unknown")
LDFLAGS=-ldflags "-X main.version=$(VERSION) -X main.buildTime=$(BUILD_TIME) -X main.commit=$(COMMIT)"

# Test settings
COVERAGE_FILE=coverage.out
COVERAGE_HTML=coverage.html
TEST_TIMEOUT=5m

# Colors for terminal output
YELLOW=\033[0;33m
GREEN=\033[0;32m
RED=\033[0;31m
NC=\033[0m # No Color

.PHONY: all build clean test test-all test-ci test-race test-integration coverage lint fmt vet run help deps update-deps sec doc install verify tidy bench outdated mocks

# Default target
all: clean lint test build

# Verify go.mod is tidy
tidy:
	@echo "${YELLOW}Verifying go.mod is tidy...${NC}"
	@$(GO) mod tidy
	@echo "${GREEN}go.mod is tidy!${NC}"

# Verify dependencies
verify:
	@echo "${YELLOW}Verifying dependencies...${NC}"
	@$(GO) mod verify
	@echo "${GREEN}Dependencies verified!${NC}"

# Build the application
build: tidy verify
	@echo "${YELLOW}Building $(BINARY_NAME)...${NC}"
	@mkdir -p $(BUILD_DIR)
	$(GOBUILD) $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME) $(MAIN_PACKAGE)
	@echo "${GREEN}Build successful!${NC}"

# Clean build artifacts
clean:
	@echo "${YELLOW}Cleaning...${NC}"
	$(GOCLEAN)
	@rm -rf $(BUILD_DIR)
	@rm -f $(BINARY_NAME)
	@rm -f $(COVERAGE_FILE)
	@rm -f $(COVERAGE_HTML)
	@echo "${GREEN}Clean complete!${NC}"

# Run tests
test:
	@echo "${YELLOW}Running tests...${NC}"
	$(GOTEST) -v -timeout=$(TEST_TIMEOUT) ./...
	@echo "${GREEN}Tests passed!${NC}"

# Run tests with race detection
test-race:
	@echo "${YELLOW}Running tests with race detection...${NC}"
	$(GOTEST) -race -v -timeout=$(TEST_TIMEOUT) ./...
	@echo "${GREEN}Race detection tests passed!${NC}"

# Run tests for CI environments (with structured output)
test-ci:
	@echo "${YELLOW}Running tests for CI...${NC}"
	$(GOTEST) -v -timeout=$(TEST_TIMEOUT) ./... -json > test-results.json
	@echo "${GREEN}CI tests complete!${NC}"

# Run integration tests
test-integration:
	@echo "${YELLOW}Running integration tests...${NC}"
	$(GOTEST) -v -timeout=$(TEST_TIMEOUT) -run "^TestIntegration_" ./
	@echo "${GREEN}Integration tests passed!${NC}"

# Run unit tests only (excluding integration tests)
test-unit:
	@echo "${YELLOW}Running unit tests only...${NC}"
	$(GOTEST) -v -timeout=$(TEST_TIMEOUT) -run "^Test[^Integration]" ./...
	@echo "${GREEN}Unit tests passed!${NC}"

# Run tests for a specific package
test-package:
	@echo "${YELLOW}Running tests for package: ${PKG}${NC}"
	@if [ -z "$(PKG)" ]; then \
		echo "${RED}Error: PKG variable is required. Usage: make test-package PKG=./pkg/compression${NC}"; \
		exit 1; \
	fi
	$(GOTEST) -v -timeout=$(TEST_TIMEOUT) $(PKG)
	@echo "${GREEN}Package tests passed!${NC}"

# Run tests with a specific tag
test-tag:
	@echo "${YELLOW}Running tests with tag: ${TAG}${NC}"
	@if [ -z "$(TAG)" ]; then \
		echo "${RED}Error: TAG variable is required. Usage: make test-tag TAG=unit${NC}"; \
		exit 1; \
	fi
	$(GOTEST) -v -timeout=$(TEST_TIMEOUT) -tags=$(TAG) ./...
	@echo "${GREEN}Tagged tests passed!${NC}"

# Run tests with coverage
coverage:
	@echo "${YELLOW}Running tests with coverage...${NC}"
	$(GOTEST) -coverprofile=$(COVERAGE_FILE) -covermode=atomic -timeout=$(TEST_TIMEOUT) ./...
	$(GO) tool cover -html=$(COVERAGE_FILE) -o $(COVERAGE_HTML)
	@echo "${GREEN}Coverage: ${NC}"
	@$(GO) tool cover -func=$(COVERAGE_FILE) | grep total | awk '{print $$3}'
	@echo "${GREEN}Coverage analysis complete! See $(COVERAGE_HTML) for details.${NC}"

# Run short tests (faster)
test-short:
	@echo "${YELLOW}Running short tests...${NC}"
	$(GOTEST) -v -short -timeout=$(TEST_TIMEOUT) ./...
	@echo "${GREEN}Short tests passed!${NC}"

# Run all tests (comprehensive testing)
test-all:
	@echo "${YELLOW}Running all tests comprehensively...${NC}"
	@echo "${YELLOW}1. Running standard tests...${NC}"
	$(GOTEST) -v -timeout=$(TEST_TIMEOUT) ./...
	@echo "${YELLOW}2. Running tests with race detection...${NC}"
	$(GOTEST) -race -v -timeout=$(TEST_TIMEOUT) ./...
	@echo "${YELLOW}3. Running integration tests...${NC}"
	$(GOTEST) -v -timeout=$(TEST_TIMEOUT) -run "^TestIntegration_" ./
	@echo "${GREEN}All tests passed successfully!${NC}"

# Run benchmarks
bench:
	@echo "${YELLOW}Running benchmarks...${NC}"
	$(GOBENCHMARK) ./...
	@echo "${GREEN}Benchmarks complete!${NC}"

# Run linter
lint:
	@echo "${YELLOW}Running linter...${NC}"
	@if command -v $(GOLINT) > /dev/null; then \
		$(GOLINT) run ./...; \
	else \
		echo "${RED}golangci-lint not installed. Run 'make deps' to install.${NC}"; \
		exit 1; \
	fi
	@echo "${GREEN}Lint complete!${NC}"

# Format code
fmt:
	@echo "${YELLOW}Formatting code...${NC}"
	$(GOFMT) ./...
	@echo "${GREEN}Formatting complete!${NC}"

# Run go vet
vet:
	@echo "${YELLOW}Running go vet...${NC}"
	$(GOVET) ./...
	@echo "${GREEN}Vet complete!${NC}"

# Run the application
run: build
	@echo "${YELLOW}Running $(BINARY_NAME)...${NC}"
	./$(BUILD_DIR)/$(BINARY_NAME) $(INPUT_FILE) $(OUTPUT_FILE)
	@echo "${GREEN}Execution complete!${NC}"

# Install dependencies
deps:
	@echo "${YELLOW}Installing dependencies...${NC}"
	$(GOGET) -v ./...
	@if ! command -v $(GOLINT) > /dev/null; then \
		echo "Installing golangci-lint..."; \
		curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh | sh -s -- -b $$(go env GOPATH)/bin; \
	fi
	@if ! command -v $(GOSEC) > /dev/null; then \
		echo "Installing gosec..."; \
		curl -sfL https://raw.githubusercontent.com/securego/gosec/master/install.sh | sh -s -- -b $$(go env GOPATH)/bin; \
	fi
	@if ! command -v go-mod-outdated > /dev/null; then \
		echo "Installing go-mod-outdated..."; \
		$(GO) install github.com/psampaz/go-mod-outdated@latest; \
	fi
	@if ! command -v mockgen > /dev/null; then \
		echo "Installing mockgen..."; \
		$(GO) install github.com/golang/mock/mockgen@latest; \
	fi
	@if ! command -v $(GODOC) > /dev/null; then \
		echo "Installing godoc..."; \
		$(GO) install golang.org/x/tools/cmd/godoc@latest; \
	fi
	@echo "${GREEN}Dependencies installed!${NC}"

# Update dependencies
update-deps:
	@echo "${YELLOW}Updating dependencies...${NC}"
	$(GO) get -u ./...
	$(GO) mod tidy
	@echo "${GREEN}Dependencies updated!${NC}"

# Check for outdated dependencies
outdated:
	@echo "${YELLOW}Checking for outdated dependencies...${NC}"
	@if command -v go-mod-outdated > /dev/null; then \
		$(GO) list -u -m -json all | go-mod-outdated -update -direct; \
	else \
		echo "${RED}go-mod-outdated not installed. Run 'go install github.com/psampaz/go-mod-outdated@latest' to install.${NC}"; \
		$(GO) list -u -m all | grep -v 'indirect'; \
	fi
	@echo "${GREEN}Outdated dependencies check complete!${NC}"

# Generate mocks for testing
mocks:
	@echo "${YELLOW}Generating mocks...${NC}"
	@if command -v mockgen > /dev/null; then \
		echo "Finding interfaces to mock..."; \
		find . -name "*.go" | xargs grep -l "type.*interface" | xargs -I{} dirname {} | sort -u | \
		while read dir; do \
			echo "Processing $$dir"; \
			interfaces=$$(grep -l "type.*interface" $$dir/*.go | xargs grep "type" | grep "interface" | awk '{print $$2}' | grep -v "{" || echo ""); \
			for interface in $$interfaces; do \
				echo "Generating mock for $$interface in $$dir"; \
				mockgen -source=$$dir/$$(basename $$dir).go -destination=$$dir/mock_$$(basename $$dir).go -package=$$(basename $$dir) $$interface; \
			done; \
		done; \
	else \
		echo "${RED}mockgen not installed. Run 'go install github.com/golang/mock/mockgen@latest' to install.${NC}"; \
		exit 1; \
	fi
	@echo "${GREEN}Mocks generated!${NC}"

# Security check
sec:
	@echo "${YELLOW}Running security check...${NC}"
	@if command -v $(GOSEC) > /dev/null; then \
		$(GOSEC) ./...; \
	else \
		echo "${RED}gosec not installed. Run 'make deps' to install.${NC}"; \
		exit 1; \
	fi
	@echo "${GREEN}Security check complete!${NC}"

# Generate documentation
doc:
	@echo "${YELLOW}Starting documentation server...${NC}"
	@if command -v $(GODOC) > /dev/null; then \
		$(GODOC) -http=:6060; \
	else \
		echo "${RED}godoc not installed. Run 'go install golang.org/x/tools/cmd/godoc@latest' to install.${NC}"; \
		exit 1; \
	fi

# Install the binary
install: build
	@echo "${YELLOW}Installing $(BINARY_NAME)...${NC}"
	cp $(BUILD_DIR)/$(BINARY_NAME) $$(go env GOPATH)/bin/
	@echo "${GREEN}Installation complete! $(BINARY_NAME) is now available in your PATH.${NC}"

# Help target
help:
	@echo "Available targets:"
	@echo "  all          - Clean, lint, test, and build"
	@echo "  build        - Build the application"
	@echo "  clean        - Remove build artifacts"
	@echo "  test         - Run all tests"
	@echo "  test-all     - Run comprehensive testing (standard tests, race detection, and integration tests)"
	@echo "  test-short   - Run short tests (faster)"
	@echo "  test-race    - Run tests with race detection"
	@echo "  test-ci      - Run tests with JSON output for CI"
	@echo "  test-integration - Run integration tests"
	@echo "  test-unit    - Run unit tests only (excluding integration tests)"
	@echo "  test-package - Run tests for a specific package (usage: make test-package PKG=./pkg/compression)"
	@echo "  test-tag     - Run tests with a specific tag (usage: make test-tag TAG=unit)"
	@echo "  coverage     - Run tests with coverage"
	@echo "  bench        - Run benchmarks"
	@echo "  lint         - Run linter"
	@echo "  fmt          - Format code"
	@echo "  vet          - Run go vet"
	@echo "  run          - Run the application"
	@echo "  deps         - Install dependencies"
	@echo "  update-deps  - Update dependencies"
	@echo "  outdated     - Check for outdated dependencies"
	@echo "  sec          - Run security check"
	@echo "  doc          - Generate and serve documentation"
	@echo "  install      - Install the binary to GOPATH/bin"
	@echo "  tidy         - Run go mod tidy"
	@echo "  verify       - Verify dependencies"
	@echo "  mocks        - Generate mocks for testing"
	@echo "  help         - Show this help message"
