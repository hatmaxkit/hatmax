# ADR-001: Linting and Code Quality Enforcement

**Status:** Accepted
**Date:** 2025-10-03
**Deciders:** Adrian

## Context

Code quality and consistency are fundamental for maintainability, especially in a code generator project where the output must follow strict patterns. We need automated enforcement of coding standards, formatting rules, and architectural constraints to prevent cognitive fatigue and subtle bugs.

## Decision

### Core Linting Stack

Use **golangci-lint** as the umbrella tool with the following configuration:

**Enabled linters:**
- `staticcheck` - Go static analysis
- `revive` - Fast, configurable, extensible, flexible linter
- `gofumpt` - Stricter gofmt
- `gci` - Import organization
- `errcheck` - Check for unchecked errors
- `ineffassign` - Detect ineffectual assignments
- `unparam` - Detect unused function parameters
- `gosimple` - Simplify code suggestions
- `gocritic` - Additional checks
- `forbidigo` - Forbid specific identifiers
- `depguard` - Dependency restrictions
- `govet` - Go vet
- `misspell` - Spelling mistakes
- `lll` - Line length limit (140 characters)
- `nestif` - Nested if statements
- `prealloc` - Slice preallocation
- `rowserrcheck` - SQL rows.Err check
- `wrapcheck` - Error wrapping

### Formatting and Import Organization

- **gofumpt**: Stricter formatting than standard gofmt
- **gci**: Import grouping in order: standard → third-party → local project modules
- **Line length**: 140 characters maximum

### Revive Rules

Specific revive rules to enforce:
- `var-naming` - Variable naming conventions
- `exported` - Exported identifiers documentation
- `unused-parameter` - Detect unused parameters
- `indent-error-flow` - Proper error handling flow
- `time-naming` - Time variable naming
- `context-as-argument` - Context as first argument

### Forbidden Patterns (forbidigo)

Prohibit direct printing to enforce structured logging:
- `fmt.Print*` functions
- `log.Print*` functions
- Inconsistent logging field names (enforced via regex)

### Dependency Management (depguard)

Layer-based dependency restrictions:
- `domain/**` cannot import `net/http`, `log`, database drivers, or HTTP clients
- `web/**` cannot import `internal/repo`
- Generated `app/` directory is excluded from linting

### Custom Analyzer

Implement a custom `loglint` analyzer using `go/analysis` to enforce project-specific policies not covered by generic linters:

- Verify infrastructure types have `log Logger` field and `Log() Logger` method
- Prohibit logging-related fields/methods in domain packages
- Ensure consistent naming conventions

Integration via golangci-lint's plugin system.

### Make Targets

Standard targets for development workflow:
```makefile
fmt:     # gofumpt + gci formatting
lint:    # golangci-lint run
check:   # fmt + lint + test sequence
```

### Pre-commit and CI

**Pre-commit hooks** (lefthook/pre-commit):
- `gofumpt -l -w .`
- `gci -w .`
- `golangci-lint run`

**CI Pipeline** (GitHub Actions):
- `lint` job with caching
- `vet` job (`go vet` + `govulncheck`)
- `test` job with race detection

## Consequences

### Positive
- Consistent code style across the project
- Early detection of potential issues
- Reduced cognitive load during code review
- Automated enforcement reduces human error
- Clear architectural boundaries

### Negative
- Initial setup complexity with custom analyzer
- Potential friction during development
- Build time increase due to comprehensive linting

### Mitigation
- Comprehensive documentation of rules and rationale
- IDE integration for immediate feedback
- Caching in CI to minimize performance impact

## Examples

### golangci-lint configuration (.golangci.yml)
```yaml
run:
  timeout: 5m
  skip-dirs: [app]

linters:
  disable-all: true
  enable: [staticcheck, revive, gofumpt, gci, errcheck, ...]

linters-settings:
  gci:
    sections: [standard, default, "prefix(hatmax.com/hatmax)"]
  lll:
    line-length: 140
  depguard:
    list-type: strict
    packages:
      - "net/http": "^hatmax.com/hatmax/domain/.*"
```

### Custom loglint analyzer structure
```go
var Analyzer = &analysis.Analyzer{
  Name: "loglint",
  Doc:  "enforce logging conventions",
  Run: func(pass *analysis.Pass) (any, error) {
    // Validate logging patterns in infrastructure
    // Prohibit logging in domain packages
    return nil, nil
  },
}
```

---

**TL;DR**: Comprehensive linting with golangci-lint + custom analyzer, strict formatting with gofumpt/gci, and CI enforcement to maintain code quality and architectural boundaries.
