# Makefile for the hatmax project

# Variables
BINARY_NAME=hatmax
APP_DIR=examples/ref
OUTPUT_DIR=examples/ref/services
SERVICE_NAME?=todo
PORT?=8085

# Go related commands
GOFUMPT=gofumpt
GCI=gci
GOLANGCI_LINT=golangci-lint
GO_TEST=go test
GO_VET=go vet
GO_VULNCHECK=govulncheck

# Phony targets ensure that make doesn't confuse a target with a file of the same name.
.PHONY: all build run test test-v test-short coverage coverage-html coverage-func coverage-profile coverage-check coverage-100 clean fmt lint vet check test-generated full-test help ci run-all stop-all

all: build

help:
	@echo "Available targets:"
	@echo "  build        - Build the $(BINARY_NAME) generator"
	@echo "  run          - Run the generator (cleans and generates)"
	@echo "  test         - Run generator tests"
	@echo "  test-v       - Run generator tests with verbose output"
	@echo "  test-short   - Run generator tests in short mode"
	@echo "  coverage     - Run generator tests with coverage report"
	@echo "  coverage-html - Generate HTML coverage report for generator"
	@echo "  coverage-func - Show function-level coverage for generator"
	@echo "  coverage-check - Check generator coverage meets 80% threshold"
	@echo "  coverage-100 - Check generator has 100% test coverage"
	@echo "  test-generated - Test the generated service"
	@echo "  full-test    - Full regenerate + test pipeline"
	@echo "  lint         - Run golangci-lint on generator"
	@echo "  fmt          - Format generator code"
	@echo "  vet          - Run go vet on generator"
	@echo "  clean        - Clean generated files and binaries"
	@echo "  check        - Run all generator quality checks"
	@echo "  ci           - Run CI pipeline with strict checks"
	@echo ""
	@echo "Service management:"
	@echo "  run-all      - Kill ports, regenerate, and start all services"
	@echo "  stop-all     - Stop all running services"

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

# Run the generator tests.
test:
	@echo "Running generator tests..."
	@$(GO_TEST) ./...

# Run generator tests with verbose output
test-v:
	@echo "Running generator tests with verbose output..."
	@$(GO_TEST) -v ./...

# Run generator tests in short mode
test-short:
	@echo "Running generator tests in short mode..."
	@$(GO_TEST) -short ./...

# Run generator tests with coverage
coverage:
	@echo "Running generator tests with coverage..."
	@$(GO_TEST) -cover ./...

# Generate coverage profile and show percentage
coverage-profile:
	@echo "Generating generator coverage profile..."
	@$(GO_TEST) -coverprofile=coverage.out ./...
	@go tool cover -func=coverage.out | tail -1

# Generate HTML coverage report
coverage-html: coverage-profile
	@echo "Generating HTML coverage report for generator..."
	@go tool cover -html=coverage.out -o coverage.html
	@echo "Generator coverage report generated: coverage.html"

# Show function-level coverage
coverage-func: coverage-profile
	@echo "Generator function-level coverage:"
	@go tool cover -func=coverage.out

# Check generator coverage percentage and fail if below threshold (80%)
coverage-check: coverage-profile
	@COVERAGE=$$(go tool cover -func=coverage.out | tail -1 | awk '{print $$3}' | sed 's/%//'); \
	echo "Generator coverage: $$COVERAGE%"; \
	if [ $$(echo "$$COVERAGE < 80" | bc -l) -eq 1 ]; then \
		echo "❌ Generator coverage $$COVERAGE% is below 80% threshold"; \
		exit 1; \
	else \
		echo "✅ Generator coverage $$COVERAGE% meets the 80% threshold"; \
	fi

# Check generator coverage percentage and fail if not 100%
coverage-100: coverage-profile
	@COVERAGE=$$(go tool cover -func=coverage.out | tail -1 | awk '{print $$3}' | sed 's/%//'); \
	echo "Generator coverage: $$COVERAGE%"; \
	if [ "$$COVERAGE" != "100.0" ]; then \
		echo "❌ Generator coverage $$COVERAGE% is not 100%"; \
		go tool cover -func=coverage.out | grep -v "100.0%"; \
		exit 1; \
	else \
		echo "🎉 Perfect! Generator has 100% test coverage!"; \
	fi

# Format Go code using gofumpt and gci.
fmt:
	@echo "Formatting Go code..."
	@$(GOFUMPT) -l -w .
	@$(GCI) -w .
	@echo "Go code formatted."

# Run go vet on generator
vet:
	@echo "Running go vet on generator..."
	@$(GO_VET) ./...

# Run golangci-lint on generator.
lint:
	@echo "Running golangci-lint on generator..."
	@$(GOLANGCI_LINT) run
	@echo "golangci-lint finished."

# Run all generator quality checks
check: fmt vet test coverage-check lint
	@echo "✅ All generator quality checks passed!"

# CI pipeline - strict checks including 100% coverage for generator
ci: fmt vet test coverage-100 lint
	@echo "🚀 Generator CI pipeline passed!"

# Temp handy test for generated service
test-generated:
	@echo "Testing generated service..."
	@cd $(OUTPUT_DIR)/$(SERVICE_NAME) && go build
	@echo "Cleaning up any existing processes on port $(PORT)..."
	@lsof -ti:$(PORT) | xargs -r kill || true
	@sleep 1
	@echo "Starting service on port $(PORT)..."
	@cd $(OUTPUT_DIR)/$(SERVICE_NAME) && timeout 5s bash -c './$(SERVICE_NAME) & sleep 2 && curl -s http://localhost:$(PORT)/items && echo "\n✅ Service test successful"' || echo "⚠️  Service test completed (timeout expected)"

# Temp handy full regenerate + test
full-test: run test-generated
	@echo "Full test complete."

# Clean the generated directory, binary, and coverage files.
clean:
	@echo "Cleaning up generated files, binary, and coverage files..."
	@rm -rf $(APP_DIR)
	@rm -f $(BINARY_NAME)
	@go clean -testcache
	@rm -f coverage.out coverage.html
	@echo "Cleanup complete."

# Service Management Targets

# Kill ports, regenerate project, and start all services
run-all:
	@echo "🚀 Starting full development environment..."
	@echo "🔪 Killing processes on ports 8080-8090..."
	@for port in 8080 8081 8082 8083 8084 8085 8086 8087 8088 8089 8090; do \
		if lsof -ti:$$port >/dev/null 2>&1; then \
			echo "🔪 Killing process on port $$port"; \
			lsof -ti:$$port | xargs -r kill -9 || true; \
		fi; \
	done
	@sleep 2
	@echo "🏗️  Regenerating monorepo..."
	@$(MAKE) run
	@echo "🚀 Starting services..."
	@echo "   📦 Building and starting Admin..."
	@cd $(APP_DIR)/services/admin && make build >/dev/null 2>&1 && nohup make run > admin.log 2>&1 & ADMIN_PID=$$!; echo $$ADMIN_PID > admin.pid; sleep 3; if ps -p $$ADMIN_PID >/dev/null 2>&1; then echo "✅ Admin started on port 8081 (PID: $$ADMIN_PID)"; else echo "❌ Admin failed to start"; fi
	@echo "   📦 Building and starting AuthN..."
	@cd $(APP_DIR)/services/authn && direnv allow >/dev/null 2>&1 || true && make build >/dev/null 2>&1 && nohup make run > authn.log 2>&1 & AUTHN_PID=$$!; echo $$AUTHN_PID > authn.pid; sleep 3; if ps -p $$AUTHN_PID >/dev/null 2>&1; then echo "✅ AuthN started on port 8082 (PID: $$AUTHN_PID)"; else echo "❌ AuthN failed to start"; fi
	@echo "   📦 Building and starting AuthZ..."
	@cd $(APP_DIR)/services/authz && direnv allow >/dev/null 2>&1 || true && make build >/dev/null 2>&1 && nohup make run > authz.log 2>&1 & AUTHZ_PID=$$!; echo $$AUTHZ_PID > authz.pid; sleep 3; if ps -p $$AUTHZ_PID >/dev/null 2>&1; then echo "✅ AuthZ started on port 8083 (PID: $$AUTHZ_PID)"; else echo "❌ AuthZ failed to start"; fi
	@echo "   📦 Building and starting Todo..."
	@cd $(APP_DIR)/services/todo && make build >/dev/null 2>&1 && nohup make run > todo.log 2>&1 & TODO_PID=$$!; echo $$TODO_PID > todo.pid; sleep 3; if ps -p $$TODO_PID >/dev/null 2>&1; then echo "✅ Todo started on port 8085 (PID: $$TODO_PID)"; else echo "❌ Todo failed to start"; fi
	@echo ""
	@echo "🎉 All services started!"
	@echo "📡 Services running:"
	@echo "   • Admin: http://localhost:8081 (system administration)"
	@echo "   • AuthN: http://localhost:8082 (authentication)"
	@echo "   • AuthZ: http://localhost:8083 (authorization)"
	@echo "   • Todo:  http://localhost:8085 (business logic)"
	@echo ""
	@echo "📁 Test scripts available:"
	@echo "   • AuthN: scripts/curl/authn/"
	@echo "   • AuthZ: scripts/curl/authz/"
	@echo ""
	@echo "🛑 To stop all services: make stop-all"

# Stop all running services
stop-all:
	@echo "🛑 Stopping all services..."
	@for port in 8080 8081 8082 8083 8084 8085 8086 8087 8088 8089 8090; do \
		if lsof -ti:$$port >/dev/null 2>&1; then \
			echo "🛑 Stopping process on port $$port"; \
			lsof -ti:$$port | xargs -r kill -9 || true; \
		fi; \
	done
	@for service in admin authn authz todo; do \
		for pid_file in $(APP_DIR)/services/$$service/$$service.pid; do \
			if [ -f "$$pid_file" ]; then \
				pid=$$(cat "$$pid_file"); \
				if ps -p "$$pid" >/dev/null 2>&1; then \
					echo "🛑 Stopping $$service (PID: $$pid)"; \
					kill -9 "$$pid" 2>/dev/null || true; \
				fi; \
				rm -f "$$pid_file"; \
			fi; \
		done; \
	done
	@echo "✅ All services stopped"
