# Makefile for the hatmax project

# Variables
BINARY_NAME=hatmax
APP_DIR=example/ref

# Go related commands
GOFUMPT=gofumpt
GCI=gci
GOLANGCI_LINT=golangci-lint
GO_TEST=go test
GO_VET=go vet
GO_VULNCHECK=govulncheck

# Phony targets ensure that make doesn't confuse a target with a file of the same name.
.PHONY: all build run test clean fmt lint check

all: build

# Build the generator binary.
build:
	@echo "Building $(BINARY_NAME)..."
	@go build -o $(BINARY_NAME) main.go
	@echo "$(BINARY_NAME) built successfully."

# Run the generator. This will first clean the generated app directory.
run: clean
	@echo "Running generator to scaffold the application..."
	@go run main.go generate --dev
	@echo "Generator run complete."

# Run the tests.
test:
	@echo "Running tests..."
	@$(GO_TEST) -v ./...

# Format Go code using gofumpt and gci.
fmt:
	@echo "Formatting Go code..."
	@$(GOFUMPT) -l -w .
	@$(GCI) -w .
	@echo "Go code formatted."

# Run golangci-lint.
lint:
	@echo "Running golangci-lint..."
	@$(GOLANGCI_LINT) run
	@echo "golangci-lint finished."

# Run all checks (format, lint, test).
check: fmt lint test
	@echo "All checks passed."

# Clean the generated directory and the binary.
clean:
	@echo "Cleaning up generated files and binary..."
	@rm -rf $(APP_DIR)
	@rm -f $(BINARY_NAME)
	@echo "Cleanup complete."
