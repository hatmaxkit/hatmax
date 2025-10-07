# Module Flexibility Solution

**Status:** Planned | **Updated:** 2025-10-06 | **Version:** 0.1

## Overview

Dynamic module path and output directory configuration for the generator, supporting both development and production use cases with flexible CLI options.

## Current Implementation

**Status**: Hardcoded paths  
**Location**: `internal/hatmax/generate.go`, `cmd/hatmax/app.go`  
**Issues**: Fixed output directory (`app`), hardcoded module paths, no CLI flexibility

## Planned Implementation

### Problem Statement
- **Hardcoded output**: Fixed `app` directory
- **Fixed module paths**: No customization for different environments
- **Limited CLI**: No configuration flags for generation modes
- **Inflexible structure**: Cannot generate multiple examples or standalone projects

### CLI Flags Enhancement

```bash path=null start=null
hatmax generate --help

# Output directory for generated code (default: "app")
--output, -o string

# Go module path for generated code (auto-inferred if not specified)
--module-path, -m string

# Enable dev mode (generates in example/ref structure)
--dev-mode, -d bool
```

### Usage Patterns

**Dev Mode** (for development/testing):
```bash path=null start=null
# Generate in example/ref with proper module path
hatmax generate --dev-mode

# Equivalent to:
hatmax generate --output example/ref --module-path github.com/adrianpk/hatmax/example/ref
```

**Standalone Mode** (for real projects):
```bash path=null start=null
# Generate in custom directory
hatmax generate --output my-project

# With explicit module path
hatmax generate --output my-project --module-path github.com/user/my-project
```

**Multiple Examples**:
```bash path=null start=null
# Generate different examples
hatmax generate --output example/auth --module-path github.com/adrianpk/hatmax/example/auth
hatmax generate --output example/billing --module-path github.com/adrianpk/hatmax/example/billing
```

### Implementation Details

**CLI Changes**:
```go path=null start=null
// internal/hatmax/app.go
Flags: []cli.Flag{
    &cli.StringFlag{
        Name:    "output",
        Aliases: []string{"o"},
        Usage:   "Output directory for generated code",
        Value:   "app",
    },
    &cli.StringFlag{
        Name:    "module-path",
        Aliases: []string{"m"},
        Usage:   "Go module path for generated code (auto-inferred if not specified)",
    },
    &cli.BoolFlag{
        Name:    "dev-mode",
        Aliases: []string{"d"},
        Usage:   "Enable dev mode (generates in example/ref structure)",
    },
}
```

**Generate Action Changes**:
```go path=null start=null
// internal/hatmax/generate.go
func GenerateAction(c *cli.Context, tmplFS fs.FS) error {
    outputDir := c.String("output")
    devMode := c.Bool("dev-mode")
    
    // Override output directory for dev mode
    if devMode {
        outputDir = "example/ref"
    }

    // Determine module path
    modulePath := c.String("module-path")
    if modulePath == "" {
        if devMode {
            modulePath = "github.com/adrianpk/hatmax/example/ref"
        } else {
            modulePath = inferModulePath(outputDir)
        }
    }

    config.ModulePath = modulePath
    // ... rest of generation ...
}
```

**Module Path Inference**:
```go path=null start=null
func inferModulePath(outputDir string) string {
    if outputDir == "app" {
        return "generatedapp"
    }
    
    // Clean the directory name and use it as module name
    cleanDir := strings.ReplaceAll(outputDir, "/", "-")
    cleanDir = strings.ReplaceAll(cleanDir, "_", "-")
    return cleanDir
}
```

**Dynamic go.mod Generation**:
```go path=null start=null
func (mg *ModelGenerator) GenerateGoMod() error {
    goModPath := filepath.Join(mg.OutputDir, "go.mod")
    
    // Determine if we need a replace directive
    var replaceDirective string
    if strings.Contains(mg.Config.ModulePath, "github.com/adrianpk/hatmax/example/") {
        // Dev mode - use replace directive
        replaceDirective = "\nreplace github.com/adrianpk/hatmax => ../../\n"
    }
    
    goModContent := fmt.Sprintf(`module %s\n\ngo 1.22.7%s\n\nrequire (...)`, 
        mg.Config.ModulePath, replaceDirective)

    return os.WriteFile(goModPath, []byte(goModContent), 0o644)
}
```

### Generated go.mod Examples

**Dev Mode** (`example/ref/go.mod`):
```go path=null start=null
module github.com/adrianpk/hatmax/example/ref

go 1.22.7

replace github.com/adrianpk/hatmax => ../../

require (
    github.com/adrianpk/hatmax v0.0.0-00010101000000-000000000000
    github.com/go-chi/chi/v5 v5.1.0
    github.com/google/uuid v1.6.0
)
```

**Standalone Mode** (`my-project/go.mod`):
```go path=null start=null
module my-project

go 1.22.7

require (
    github.com/adrianpk/hatmax v0.0.0-00010101000000-000000000000
    github.com/go-chi/chi/v5 v5.1.0
    github.com/google/uuid v1.6.0
)
```

## Benefits

1. **Flexibility**: Support both dev and production use cases
2. **Consistency**: Proper module paths for all scenarios
3. **Developer Experience**: Simple flags for common patterns
4. **Extensibility**: Easy to add new example directories
5. **Backward Compatibility**: Default behavior unchanged

## Implementation Roadmap

### Phase 1: Core CLI Enhancement
1. **Add CLI flags** - output, module-path, dev-mode
2. **Update GenerateAction** - flag processing logic
3. **Implement inferModulePath** - automatic path inference
4. **Dynamic go.mod generation** - conditional replace directives

### Phase 2: Testing & Validation
5. **Dev mode testing** - example/ref structure
6. **Standalone testing** - custom output directories
7. **Module path validation** - proper Go module naming
8. **Backward compatibility** - existing workflow preservation

### Phase 3: Documentation & Examples
9. **CLI documentation** - flag usage and examples
10. **Directory structure** - example organization
11. **Migration guide** - existing project updates
12. **Best practices** - recommended usage patterns

## Next Steps

### Immediate
1. **Implement CLI flags** in `cmd/hatmax/app.go`
2. **Update generate action** with flag processing
3. **Test basic functionality** with current YAML configs

### Medium Term
4. **Add module path inference** logic
5. **Implement dynamic go.mod** generation
6. **Create comprehensive tests** for all generation modes

---

**Summary**: Flexible CLI-based module generation supporting development and production use cases with automatic path inference, dev mode shortcuts, and backward compatibility.
