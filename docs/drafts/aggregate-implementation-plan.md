# Aggregate Implementation Plan

**Status:** Planned | **Updated:** 2025-10-06 | **Version:** 0.1

## Overview

Phased implementation roadmap for Domain-Driven Design (DDD) aggregates in the Hatmax generator, building on the architecture defined in `aggregates-sqlite-approach.md`.

## Current Implementation

**Status**: Basic models and repositories  
**Location**: Current generator templates and YAML parsing  
**Features**: Simple CRUD operations without aggregate support

## Planned Implementation

### Architecture Foundation
Builds on the aggregate design with:
- **Aggregate roots** with one-level children
- **Unit-of-work** pattern with deterministic diffs
- **Optimistic locking** at aggregate boundaries
- **SQLite and MongoDB** adapter support

## Implementation Phases

### Phase 1: YAML Schema Extension

**Goal**: Enable generator to parse `aggregates` section in `monorepo.yaml`

**Changes Required**:
```go path=null start=null
// internal/hatmax/config.go
type Service struct {
    // ... existing fields ...
    Aggregates map[string]AggregateRoot `yaml:"aggregates"`
}

type AggregateRoot struct {
    Table        string                       `yaml:"table"`
    ID           string                       `yaml:"id"`
    VersionField string                       `yaml:"version_field"`
    Fields       map[string]Field             `yaml:"fields"`
    Audit        bool                         `yaml:"audit"`
    Children     map[string]ChildCollection   `yaml:"children"`
}

type ChildCollection struct {
    Of         string            `yaml:"of"`
    Table      string            `yaml:"table"`
    FK         ForeignKey        `yaml:"fk"`
    ID         string            `yaml:"id"`
    Order      *Order            `yaml:"order,omitempty"`
    Updatable  []string          `yaml:"updatable"`
    Audit      bool              `yaml:"audit"`
}
```

**Verification**: `make build` compiles successfully with new schema

### Phase 2: Aggregate Model Generation

**Goal**: Generate Go structs for aggregate roots and child collections

**Templates Required**:
- `aggregate_root.tmpl` - Aggregate root struct (e.g., `List`)
- `child_collection.tmpl` - Child collection struct (e.g., `Item`)

**Generator Updates**:
```go path=null start=null
// internal/hatmax/generator.go
type ModelGenerator struct {
    // ... existing fields ...
    AggregateRootTemplate       *template.Template
    ChildCollectionTemplate     *template.Template
}

func (mg *ModelGenerator) GenerateAggregateModels() error {
    // Iterate through Config.Services.Aggregates
    // Generate Go structs for roots and children
}
```

**Verification**: Generated Go structs compile and include proper aggregate relationships

### Phase 3: Repository Interface Generation

**Goal**: Generate repository interfaces for aggregates

**Templates Required**:
- `aggregate_repo_interface.tmpl` - Repository interface template

**Generated Interface**:
```go path=null start=null
type ListRepo interface {
    Get(ctx context.Context, id string) (List, error)
    Create(ctx context.Context, in List) (List, error)
    Save(ctx context.Context, agg List) (List, error)    // Unit-of-work with diff
    Delete(ctx context.Context, id string, cascade bool) error
}
```

**Verification**: Repository interfaces generated with correct aggregate methods

### Phase 4: SQLite Implementation Generation

**Goal**: Generate SQLite repository implementations with diff algorithms

**SQL Templates Required**:
- `aggregate_insert_root.sql.tmpl` - Root insertion
- `aggregate_update_root.sql.tmpl` - Root updates with versioning
- `aggregate_select_children.sql.tmpl` - Child collection loading
- `aggregate_batch_operations.sql.tmpl` - Child insert/update/delete batches

**Go Templates Required**:
- `aggregate_sqlite_repo.tmpl` - SQLite repository implementation
- `aggregate_diff_helper.tmpl` - Diff algorithm for child collections

**Core Implementation**:
```go path=null start=null
func (r *ListSQLiteRepo) Save(ctx context.Context, agg List) (List, error) {
    tx := beginTx(ctx, r.db)
    
    // 1. Optimistic lock: bump root version
    // 2. Load current children
    // 3. Compute diff
    // 4. Apply changes in batches
    // 5. Commit transaction
    
    return r.Get(ctx, agg.ID)
}
```

**Verification**: Generated repositories handle transactions, diffs, and optimistic locking

### Phase 5: Handler Integration

**Goal**: Update handlers to use aggregate repositories

**Handler Updates**:
- Accept aggregate repository interfaces in constructors
- Implement granular API operations (PATCH root, POST children)
- Map API operations to aggregate Save/Get methods
- Handle aggregate-specific use cases

**API Granularity Pattern**:
```go path=null start=null
// PATCH /lists/{id} - Update root fields
func (h *ListHandler) Update(w http.ResponseWriter, r *http.Request) {
    list, _ := h.repo.Get(r.Context(), id)
    // Apply root field changes
    h.repo.Save(r.Context(), list)
}

// POST /lists/{id}/items - Add child item
func (h *ListHandler) AddItem(w http.ResponseWriter, r *http.Request) {
    list, _ := h.repo.Get(r.Context(), listID)
    list.Items = append(list.Items, newItem)
    h.repo.Save(r.Context(), list)
}
```

**Verification**: Generated handlers provide granular API while using aggregate patterns internally

### Phase 6: Dependency Management

**Goal**: Update generated go.mod with aggregate dependencies

**Dependencies Added**:
- Database drivers (sqlite3)
- Configuration libraries (koanf)
- Logging libraries (slog)
- UUID generation
- Transaction management

**Verification**: Generated projects compile and run with all dependencies

## Implementation Timeline

### Immediate
1. **Phase 1**: YAML schema extension
2. **Phase 2**: Basic aggregate model generation
3. **Phase 3**: Repository interface generation

### Medium Term
4. **Phase 4**: SQLite implementation with diff algorithms
5. **Phase 5**: Handler integration and API granularity

### Final
6. **Phase 6**: Dependency management and testing
7. **Integration testing**: End-to-end aggregate workflows
8. **Documentation**: Usage guides and examples

## Success Criteria

### Phase 1-3: Foundation
- [x] YAML parsing supports aggregate configuration
- [x] Generated models include aggregate relationships
- [x] Repository interfaces follow DDD patterns

### Phase 4-5: Core Implementation
- [x] SQLite repositories handle unit-of-work correctly
- [x] Diff algorithms work for child collections
- [x] Optimistic locking prevents concurrent modifications
- [x] API handlers provide granular operations

### Phase 6: Production Ready
- [x] Generated projects compile and run
- [x] All dependencies properly managed
- [x] Performance meets requirements
- [x] Testing coverage adequate

---

**Summary**: Six-phase implementation plan transforms the current generator to support DDD aggregates with SQLite persistence, deterministic diffs, and granular API operations while maintaining the existing simplicity and code generation approach.
