# Declarative Handlers DSL

**Status:** Draft | **Updated:** 2025-10-06 | **Version:** 0.1

## Overview

Declarative handler definitions using intention-based declarations instead of implementation-specific references. Handlers are defined by their purpose (`source`, `model`, `op`) rather than internal method names.

## Current Implementation

Declarative handlers are fully implemented and working. The DSL uses `source`, `model`, and `op` fields to generate appropriate handler code with automatic name inference.

## Planned Enhancements

Override mechanisms for exceptional cases, custom operations beyond CRUD, and improved validation messages.

## Current Format Examples

### Basic CRUD Handlers
```yaml
version: 0.1
name: "todo-app"
package: "github.com/company/todo-app"
services:
  todo:
    kind: atom
    repo_impl: [sqlite, mongo]
    models:
      Item:
        fields:
          text: {type: text, validations: [{name: required}]}
          done: {type: bool, default: false}

    api:
      base_path: /todo
      handlers:
        - id: todo_items_list
          route: "GET /items"
          source: repo        # repo | service | usecase | aggregate
          model: Item
          op: list           # list|get|create|update|delete|custom

      - id: todo_items_create
        route: "POST /items"
        source: repo
        model: Item
        op: create

      - id: todo_items_get
        route: "GET /items/{id}"
        source: repo
        model: Item
        op: get

      - id: todo_items_update
        route: "PATCH /items/{id}"
        source: repo
        model: Item
        op: update

        - id: todo_items_delete
          route: "DELETE /items/{id}"
          source: repo
          model: Item
          op: delete
```

## Planned Enhancements

### Embedded Declaration in Model
```yaml
version: 0.1
name: "todo-app"
package: "github.com/company/todo-app"
services:
  todo:
    kind: atom
    models:
      Item:
        fields:
          text: {type: text, validations: [{name: required}]}
          done: {type: bool, default: false}

        api:
          base_path: /items
          endpoints:
            list:   {method: GET,    path: "/"}
            create: {method: POST,   path: "/"}
            get:    {method: GET,    path: "/{id}"}
            update: {method: PATCH,  path: "/{id}"}
            delete: {method: DELETE, path: "/{id}"}
```

### Override Mechanisms (Planned)
```yaml
handlers:
  - id: todo_items_create_special
    route: "POST /items"
    source: repo
    model: Item
    op: create
    overrides:
      repo_name: ItemsStorage      # override inferred repo name
      method_name: CreateNewItem   # override inferred method name
      handler_name: CreateItemSpecial  # override handler function name

  - id: todo_items_search
    route: "GET /items/search"
    source: service
    model: Item
    op: custom
    custom_operation: search
    overrides:
      service_method: SearchItems
```

### Custom Operations (Planned)
```yaml
handlers:
  - id: todo_items_toggle
    route: "POST /items/{id}/toggle"
    source: service
    model: Item
    op: custom
    custom_operation: toggle
    # Generator would create: ItemService.Toggle(ctx, id) method

  - id: todo_items_bulk_update
    route: "PATCH /items/bulk"
    source: usecase
    model: Item
    op: custom
    custom_operation: bulk_update
    # Generator would create: BulkUpdateItemsUsecase
```

## Current Generation Rules

### Naming Conventions

**For `atom` services:**
- Repository: `{Model}Repo` (e.g., `ItemRepo`)
- Service: `{Model}Service` (e.g., `ItemService`)
- Handler struct: `{Model}Handler` (e.g., `ItemHandler`)
- CRUD methods: `List()`, `Get(id)`, `Create(req)`, `Update(id, req)`, `Delete(id)`

### Source Resolution (Current)

- `source: repo` → Direct repository call (implemented)
- `source: service` → Business logic layer call (planned)
- `source: usecase` → Use case pattern (planned)
- `source: aggregate` → Domain aggregate method (planned)

### Operation Mapping

**Standard CRUD:**
- `list` → `List(ctx) ([]Model, error)`
- `get` → `Get(ctx, id) (Model, error)`
- `create` → `Create(ctx, req) (Model, error)`
- `update` → `Update(ctx, id, req) (Model, error)`
- `delete` → `Delete(ctx, id) error`

**Custom operations:**
- Require `custom_operation` field
- Generate methods based on operation name
- Can specify additional parameters via schema

## Validation Rules

### Required Fields
- `id` - Stable identifier for handler tracking
- `route` - HTTP method and path
- `source` - Where the operation is implemented
- `model` - Target model for the operation
- `op` - Operation type

### Validation Logic
1. **Route uniqueness** - No duplicate routes within service
2. **ID uniqueness** - Handler IDs must be unique within service
3. **Model existence** - Referenced model must be defined
4. **Source compatibility** - Source must be valid for service kind
5. **Custom operation validation** - Custom ops require `custom_operation`

### Error Messages
```
Error: Handler 'todo_items_create' references undefined model 'Item'
  → Check models section or fix model name

Error: Route 'GET /items' is defined multiple times
  → Handler IDs: todo_items_list, todo_items_duplicate

Error: Custom operation 'toggle' missing custom_operation field
  → Add custom_operation: toggle to handler definition
```

## Current Schema

```go
type Handler struct {
    ID        string        `yaml:"id"`
    Route     string        `yaml:"route"`
    Source    HandlerSource `yaml:"source"`    // repo|service|usecase|aggregate
    Model     string        `yaml:"model"`
    Operation StandardOp    `yaml:"op"`        // list|get|create|update|delete|custom
    // CustomOperation and Overrides planned for future releases
}
```

## Benefits

- **Intention over implementation**: Express what you want, not how it's built
- **Easier editing**: No need to know internal naming conventions
- **Flexible naming**: Can change internal conventions without breaking YAMLs
- **Clear separation**: DSL concerns separate from implementation details

## Next Steps

1. **Custom operations**: Support for non-CRUD operations
2. **Override mechanisms**: Fine-grained control for exceptional cases
3. **Enhanced validation**: Better error messages and conflict detection
4. **Multiple sources**: Service, usecase, and aggregate handlers

---

**Summary**: Declarative handlers using intention-based declarations (`source: repo, model: Item, op: create`) instead of implementation-specific references. Currently implemented for basic CRUD operations with repository sources.
