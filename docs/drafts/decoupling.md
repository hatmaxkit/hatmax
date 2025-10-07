# Decoupling Generator and Generated Code

**Status:** Implemented | **Updated:** 2025-10-06 | **Version:** 0.1

## Overview

Architectural separation between generator internals and generated code dependencies through intentional code duplication and public interface stabilization.

## Current Implementation

**Status**: Active separation maintained  
**Location**: `internal/hm/` (public interfaces), `internal/core/` (generator internals)  
**Strategy**: Intentional duplication for architectural decoupling

### Core Principle

The primitives (interfaces, helpers) used by generated code are intentionally duplicated from the generator's internal `core` package into a public `hm` package. This introduces code duplication but serves a critical architectural purpose: **decoupling the generated application domain from the generator domain**.

### Current Architecture

```go path=null start=null
// Generator domain (internal/core/)
package core

type Logger interface {
    Info(msg string, args ...interface{})
    Error(msg string, args ...interface{})
}

func Respond(w http.ResponseWriter, code int, data interface{}, meta interface{}) {
    // Internal implementation for generator use
}
```

```go path=null start=null
// Generated code domain (internal/hm/)
package hm

type Logger interface {
    Info(msg string, args ...interface{})
    Error(msg string, args ...interface{})
}

func Respond(w http.ResponseWriter, code int, data interface{}, meta interface{}) {
    // Stable public API for generated code
}
```

### Benefits

1. **Version Stability**: `hm` package can be versioned and stabilized independently
2. **Generator Freedom**: `core` package can evolve without breaking generated code
3. **Clear Dependencies**: Generated code depends only on public, stable interfaces
4. **Breaking Change Protection**: Internal generator changes don't affect generated applications
5. **Dependency Clarity**: Clear separation between generator and generated code concerns

## Implementation Details

### Package Structure

```text path=null start=null
internal/
├── core/           # Generator-internal primitives
│   ├── logger.go   # Internal logging interfaces
│   ├── response.go # Internal response helpers
│   └── ...         # Other internal utilities
│
├── hm/             # Public API for generated code
│   ├── logger.go   # Stable logging interfaces
│   ├── response.go # Stable response helpers
│   └── ...         # Other public utilities
│
└── hatmax/         # Generator implementation
    ├── generator.go
    └── ...
```

### Interface Duplication Strategy

**Internal Core** (for generator use):
- Can change freely based on generator needs
- May include experimental features
- Optimized for code generation tasks
- No backward compatibility guarantees

**Public HM** (for generated code):
- Stable, versioned interface
- Backward compatibility maintained
- Focused on generated code needs
- Minimal, essential functionality only

### Generated Code Dependencies

```go path=null start=null
// Generated handler always imports from hm package
package handlers

import (
    "context"
    "net/http"
    
    "github.com/adrianpk/hatmax/internal/hm"  // Stable dependency
    // Never imports from internal/core/       // Generator internals
)

func (h *ItemHandler) List(w http.ResponseWriter, r *http.Request) {
    // Uses stable hm.Respond interface
    hm.Respond(w, 200, items, meta)
}
```

## Planned Enhancements

### Public HM Package
- **Status**: Planned
- **Description**: Move `hm` package to public location for external consumption
- **Location**: `pkg/hm/` or separate repository
- **Benefit**: Generated code can import from stable, public package

### Semantic Versioning
- **Status**: Planned
- **Description**: Proper semantic versioning for `hm` package
- **Implementation**: Independent release cycle from generator
- **Benefit**: Generated code can pin to specific stable versions

### Interface Evolution
- **Status**: Planned
- **Description**: Structured approach to interface changes
- **Strategy**: Deprecation warnings, migration guides, backward compatibility
- **Benefit**: Smooth upgrades for generated applications

## Benefits of This Approach

### For Generated Applications
1. **Stability**: Code doesn't break from generator updates
2. **Predictability**: Clear, stable API surface
3. **Independence**: Can evolve separately from generator
4. **Testing**: Easier to test against stable interfaces

### For Generator Development
1. **Freedom**: Internal changes don't affect users
2. **Experimentation**: Can try new approaches internally
3. **Optimization**: Can optimize internals without API constraints
4. **Maintenance**: Clear separation of concerns

### For the Ecosystem
1. **Trust**: Users know generated code won't break unexpectedly
2. **Adoption**: Lower risk for production use
3. **Extension**: Others can build on stable `hm` interfaces
4. **Documentation**: Clear API surface to document

## Trade-offs

### Costs
- **Code Duplication**: Interfaces exist in both packages
- **Synchronization**: Changes need coordination between packages
- **Maintenance**: More code to maintain and test
- **Complexity**: Additional architectural layer

### Benefits
- **Decoupling**: Clean separation of concerns
- **Stability**: Generated code remains stable
- **Evolution**: Generator can evolve independently
- **Trust**: Predictable API for users

## Next Steps

### Immediate
1. **Document current interfaces** in `hm` package
2. **Establish versioning strategy** for public API
3. **Create migration guide** for interface changes

### Medium Term
4. **Move hm to public package** location
5. **Implement semantic versioning** for hm releases
6. **Create compatibility test suite** between versions

---

**Summary**: Intentional architectural decoupling through code duplication provides stability for generated applications while enabling generator evolution. The trade-off of code duplication is justified by the significant benefits in stability, trust, and development freedom.
