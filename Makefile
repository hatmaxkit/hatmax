# Variables
LIB_NAME = hatmax
MODULE_NAME = github.com/hatmaxkit/hatmax
LINT_CACHE_DIR = $(CURDIR)/.tmp/lint
LINT_GOCACHE = $(LINT_CACHE_DIR)/gocache

# Default target
all: test

# Help target
help:
	@echo "Available targets:"
	@echo ""
	@echo "Testing:"
	@echo "  test                  - Run all tests"
	@echo "  test-v                - Run tests with verbose output"
	@echo "  test-short            - Run tests in short mode"
	@echo "  test-coverage         - Run tests with coverage report"
	@echo "  test-coverage-profile - Generate coverage profile"
	@echo "  test-coverage-html    - Generate HTML coverage report"
	@echo "  test-coverage-func    - Show function-level coverage"
	@echo "  test-coverage-check   - Check coverage meets 85% threshold"
	@echo "  test-coverage-100     - Check coverage is 100%"
	@echo "  test-coverage-summary - Display coverage table by package"
	@echo ""
	@echo "Quality Checks:"
	@echo "  lint                  - Alias of lint-check"
	@echo "  lint-check            - Check formatting + lint rules"
	@echo "  lint-strict           - Run strict lint checks"
	@echo "  lint-fix              - Apply format + auto-fixable lint rules"
	@echo "  format                - Format code"
	@echo "  vet                   - Run go vet"
	@echo "  check                 - Run all quality checks (fmt, vet, test, test-coverage-check, lint-strict)"
	@echo "  ci                    - Run CI pipeline (strict, 100% coverage)"
	@echo ""
	@echo "Utilities:"
	@echo "  clean                 - Clean coverage files and test cache"
	@echo "  tidy                  - Run go mod tidy"
	@echo "  download              - Download dependencies"
	@echo "  install-hooks         - Configure git to use .githooks/"

# Lint alias (soft/default)
lint: lint-check

lint-strict:
	@mkdir -p $(LINT_GOCACHE)
	@echo "Checking gofmt on tracked Go files..."
	@UNFORMATTED=$$(gofmt -l $$(git ls-files '*.go')); \
	if [ -n "$$UNFORMATTED" ]; then \
		echo "❌ Unformatted files:"; \
		echo "$$UNFORMATTED"; \
		exit 1; \
	fi
	@echo "Running lint rules (nlreturn, noinlineerr, wsl_v5)..."
	@GOCACHE=$(LINT_GOCACHE) golangci-lint run --default=none --enable=nlreturn --enable=noinlineerr --enable=wsl_v5

lint-check: lint-strict

lint-fix:
	@mkdir -p $(LINT_GOCACHE)
	@echo "Applying gofmt to tracked Go files..."
	@gofmt -w $$(git ls-files '*.go')
	@echo "Applying auto-fixable lint rules (nlreturn, wsl_v5)..."
	@GOCACHE=$(LINT_GOCACHE) golangci-lint run --fix --default=none --enable=nlreturn --enable=wsl_v5
	@echo "Reporting non-auto-fixable lint (noinlineerr)..."
	@GOCACHE=$(LINT_GOCACHE) golangci-lint run --default=none --enable=noinlineerr --issues-exit-code=0

# Format code
format:
	@echo "Formatting code..."
	@gofmt -w .

# Run tests
test:
	@go test ./...

# Run tests with verbose output
test-v:
	@go test -v ./...

# Run tests in short mode
test-short:
	@go test -short ./...

# Run tests with coverage
test-coverage:
	@go test -cover ./...

# Generate coverage profile and show percentage
test-coverage-profile:
	@go test -coverprofile=coverage.out ./...
	@go tool cover -func=coverage.out | tail -1

# Generate HTML coverage report
test-coverage-html: test-coverage-profile
	@go tool cover -html=coverage.out -o coverage.html
	@echo "Coverage report generated: coverage.html"

# Show function-level coverage
test-coverage-func: test-coverage-profile
	@go tool cover -func=coverage.out

# Check coverage percentage and fail if below threshold (85%)
test-coverage-check: test-coverage-profile
	@COVERAGE=$$(go tool cover -func=coverage.out | tail -1 | awk '{print $$3}' | sed 's/%//'); \
	echo "Current coverage: $$COVERAGE%"; \
	if [ $$(awk -v cov="$$COVERAGE" 'BEGIN {print (cov < 85)}') -eq 1 ]; then \
		echo "❌ Coverage $$COVERAGE% is below 85% threshold"; \
		exit 1; \
	else \
		echo "✅ Coverage $$COVERAGE% meets the 85% threshold"; \
	fi

# Check coverage percentage and fail if not 100%
test-coverage-100: test-coverage-profile
	@COVERAGE=$$(go tool cover -func=coverage.out | tail -1 | awk '{print $$3}' | sed 's/%//'); \
	echo "Current coverage: $$COVERAGE%"; \
	if [ "$$COVERAGE" != "100.0" ]; then \
		echo "❌ Coverage $$COVERAGE% is not 100%"; \
		go tool cover -func=coverage.out | grep -v "100.0%"; \
		exit 1; \
	else \
		echo "🎉 Perfect! 100% test coverage achieved!"; \
	fi

# Display coverage summary table by package
test-coverage-summary:
	@echo "🧪 Running coverage tests by package..."
	@echo ""
	@echo "Coverage by package:"
	@echo "┌────────────────────────────────────────────────────────┬──────────┐"
	@echo "│ Package                                                │ Coverage │"
	@echo "├────────────────────────────────────────────────────────┼──────────┤"
	@for pkg in $$(go list ./... | grep -v -e "/tmp/" -e "/build/"); do \
		pkgname=$$(echo $$pkg | sed 's|$(MODULE_NAME)||' | sed 's|^/||'); \
		if [ -z "$$pkgname" ]; then pkgname="."; fi; \
		result=$$(go test -cover $$pkg 2>&1); \
		cov=$$(echo "$$result" | grep -oE '[0-9]+\.[0-9]+% of statements' | grep -v '^0\.0%' | tail -1 | grep -oE '[0-9]+\.[0-9]+%'); \
		if [ -z "$$cov" ]; then \
			if echo "$$result" | grep -qE '\[no test files\]|no test files'; then \
				cov="no tests"; \
			elif echo "$$result" | grep -q "FAIL"; then \
				cov="FAIL"; \
			else \
				cov="0.0%"; \
			fi; \
		fi; \
		printf "│ %-54s │ %8s │\n" "$$pkgname" "$$cov"; \
	done
	@echo "└────────────────────────────────────────────────────────┴──────────┘"

# Run go vet
vet:
	@go vet ./...

# Run all quality checks
check: format vet test test-coverage-check lint-strict
	@echo "✅ All quality checks passed!"

# CI pipeline - strict checks including 100% coverage
ci: format vet test test-coverage-100 lint-strict
	@echo "🚀 CI pipeline passed!"

# Clean coverage files and test cache
clean:
	@echo "Cleaning up..."
	@go clean -testcache
	@rm -f coverage.out coverage.html
	@echo "Clean complete."

# Run go mod tidy
tidy:
	@echo "Running go mod tidy..."
	@go mod tidy

# Download dependencies
download:
	@echo "Downloading dependencies..."
	@go mod download

install-hooks:
	@git config core.hooksPath .githooks
	@chmod +x .githooks/pre-commit
	@echo "✅ Git hooks installed (core.hooksPath=.githooks)"

# Phony targets
.PHONY: all test test-v test-short test-coverage test-coverage-profile test-coverage-html test-coverage-func test-coverage-check test-coverage-100 test-coverage-summary vet check ci lint lint-check lint-strict lint-fix format help clean tidy download install-hooks
