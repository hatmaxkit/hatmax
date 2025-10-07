# Aggregates & SQLite Repository Implementation

**Status:** Planned | **Updated:** 2025-10-06 | **Version:** 0.1

## Overview

Implementation plan for Domain-Driven Design (DDD) aggregates using SQLite repositories with deterministic diff algorithms. Enables predictable, portable repositories without ORM complexity.

## Current Implementation

**Status**: Basic repository interfaces exist  
**Location**: `internal/hatmax/generator.go`, repository templates  
**Features**: Simple CRUD operations without aggregate support

## Planned Implementation

### Core Principles
- **Aggregate roots** with one-level children only
- **Unit-of-work** pattern for all child mutations
- **Optimistic locking** with version fields
- **Deterministic diffs** for child collections
- **Multi-adapter support** (SQLite, MongoDB)

### YAML Configuration

```yaml path=null start=null
version: 0.2
services:
  todos:
    kind: domain
    repo_impl: [sqlite, mongo]
    
    aggregates:
      List:
        table: lists
        id: id
        version_field: version
        fields:
          name: {type: string, validations: [required, {max_len: 120}]}
        audit: true
        children:
          items:
            of: Item
            table: items
            fk: {name: list_id, ref: lists.id, on_delete: cascade}
            id: id
            order: {field: pos, unique_scope: [list_id]}
            updatable: [text, done]
            audit: true
            
    models:
      Item:
        fields:
          id: {type: uuid}
          text: {type: text, validations: [required, {max_len: 500}]}
          done: {type: bool, default: false}
```

### Generated Database Schema

**Root Table**:
```sql path=null start=null
CREATE TABLE lists (
  id           TEXT PRIMARY KEY,
  name         TEXT NOT NULL,
  version      INTEGER NOT NULL DEFAULT 0,
  created_at   TEXT NOT NULL,
  created_by   TEXT,
  updated_at   TEXT NOT NULL,
  updated_by   TEXT
);
CREATE INDEX idx_lists_updated_at ON lists(updated_at);
```

**Child Table**:
```sql path=null start=null
CREATE TABLE items (
  id         TEXT PRIMARY KEY,
  list_id    TEXT NOT NULL REFERENCES lists(id) ON DELETE CASCADE,
  pos        INTEGER,
  text       TEXT NOT NULL,
  done       INTEGER NOT NULL DEFAULT 0,
  created_at TEXT NOT NULL,
  created_by TEXT,
  updated_at TEXT NOT NULL,
  updated_by TEXT
);

CREATE INDEX idx_items_list ON items(list_id);
CREATE INDEX idx_items_done ON items(list_id, done);
CREATE UNIQUE INDEX idx_items_list_pos ON items(list_id, pos);
```

### Repository Interface

```go path=null start=null
type ListRepo interface {
    Get(ctx context.Context, id string) (List, error)
    Create(ctx context.Context, in List) (List, error)
    Save(ctx context.Context, agg List) (List, error)   // unit-of-work with diff
    Delete(ctx context.Context, id string, cascade bool) error
}
```

### Key Operations
- **Create**: Inserts root with version=0, optional initial children
- **Save**: Updates root + reconciles children via deterministic diff
- **Delete**: Removes root, CASCADE removes children automatically
- **Get**: Loads aggregate root with all children

### Diff Algorithm Implementation

```go path=null start=null
func (r *ListSQLiteRepo) Save(ctx context.Context, agg List) (List, error) {
    tx := beginTx(ctx, r.db)

    // Optimistic lock: bump root version
    res := exec(tx, updateListSQL, agg.Name, now(), actor(ctx), agg.ID, agg.Version)
    if res.RowsAffected() == 0 { 
        rollback(tx)
        return ErrConcurrent 
    }

    // Load current children
    curr := selectItems(tx, agg.ID)

    // Compute diff
    want := toItemRows(agg.Items)
    ins, upd, del, reord := diffItems(curr, want, updatable, orderField)

    // Apply changes
    if len(ins) > 0 { batchInsertItems(tx, agg.ID, ins, now(), actor(ctx)) }
    if len(upd) > 0 { batchUpdateItems(tx, upd, now(), actor(ctx)) }
    if len(del) > 0 { batchDeleteItems(tx, agg.ID, del) }
    if len(reord) > 0 { batchReorderItems(tx, reord) }

    commit(tx)
    return r.Get(ctx, agg.ID)
}
```

## Planned Features

### Optimistic Locking
- **Status**: Planned
- **Description**: Version-based concurrency control at aggregate root
- **Implementation**: `version` field incremented on each Save operation
- **Error handling**: `ErrConcurrent` on version conflicts

### Multi-Adapter Support
- **Status**: Planned
- **Description**: Same repository interface, multiple storage adapters
- **Adapters**: SQLite (normalized tables), MongoDB (embedded documents)
- **Benefit**: Portable aggregate implementations

### Audit Trail
- **Status**: Planned
- **Description**: Automatic audit stamping and change logging
- **Implementation**: `created_*`/`updated_*` fields, optional audit events table
- **Format**: JSON-based `before/after` snapshots

### Query Projections
- **Status**: Planned
- **Description**: Read-only query helpers for cross-aggregate views
- **Examples**: `ItemsByList()`, `ListsWithCounts()`
- **Benefit**: Flexible reads without breaking aggregate boundaries

### Performance Optimizations
- **Status**: Planned
- **Description**: SQLite-specific optimizations
- **Features**: WAL mode, prepared statements, batch chunking
- **Limits**: Parameter limit handling (999 variables per query)

## Implementation Roadmap

### Phase 1: Core Infrastructure
1. **Aggregate DSL parsing** - Extend YAML configuration
2. **Diff algorithm** - Deterministic child collection diffing
3. **SQLite adapter** - Transaction-based unit-of-work
4. **Migration generation** - Schema creation from DSL

### Phase 2: Advanced Features
5. **Optimistic locking** - Version-based concurrency control
6. **Audit trail** - Change tracking and logging
7. **Query projections** - Read-only helpers
8. **MongoDB adapter** - Alternative storage implementation

### Phase 3: Developer Experience
9. **Use case generation** - Higher-level domain operations
10. **Testing utilities** - Golden files and test helpers
11. **Performance tooling** - Query analysis and optimization
12. **Documentation** - Usage guides and examples

## Next Steps

### Immediate
1. **Design DSL extensions** for aggregate configuration
2. **Prototype diff algorithm** for child collections
3. **Implement basic SQLite adapter** with transactions

### Medium Term
4. **Add optimistic locking** with version fields
5. **Generate migration files** from aggregate definitions
6. **Implement audit trail** with change tracking

---

**Summary**: Comprehensive DDD aggregate implementation with deterministic diff algorithms, optimistic locking, and multi-adapter support. Focuses on predictable, testable repositories without ORM complexity while maintaining portability across storage systems.
