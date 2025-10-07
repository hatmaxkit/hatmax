# API Design and Generation

**Status:** Draft | **Updated:** 2025-10-06 | **Version:** 0.1

## Overview

Defines API design standards and code generation patterns for consistent REST endpoints across all generated services.

## Current Implementation

API standards are fully implemented and working:

### Response Envelopes ✅
- **Success**: `{"data": <payload>, "meta": <metadata>}`
- **Error**: `{"error": {"code": "string", "message": "string", "details": [...]}}`

### JSON Format ✅
- Field names in `snake_case` (e.g., `created_at`, `updated_at`)
- Consistent JSON tags across all generated models

### Response Helpers ✅
- `hm.Respond(w, code, data, meta)` for success responses
- `hm.Error(w, code, errorCode, message, validationErrors...)` for errors

### Validation ✅
- Generated validators per model (e.g., `ValidateCreateItem`, `ValidateUpdateItem`)
- Structured validation errors with field-level details
- Integration with response envelopes

### Logging ✅
- Structured logging with `hm.Logger`
- Request-scoped logging with request ID, method, path
- Consistent logging patterns across handlers

## Generated Code Examples

### Handler Pattern
```go
func (h *ItemHandler) CreateItem(w http.ResponseWriter, r *http.Request) {
    log := h.logForRequest(r)
    ctx := r.Context()

    model, ok := h.decodeItemPayload(w, r, log)
    if !ok {
        return
    }

    model.EnsureID()
    model.BeforeCreate()

    validationErrors := ValidateCreateItem(ctx, model)
    if len(validationErrors) > 0 {
        log.Debug("validation failed", "errors", validationErrors)
        hm.Error(w, http.StatusBadRequest, "validation_failed", "Validation failed", validationErrors...)
        return
    }

    if err := h.svc.Create(ctx, &model); err != nil {
        log.Error("failed to create item", "error", err)
        hm.Error(w, http.StatusInternalServerError, "internal_error", "Could not create item")
        return
    }

    hm.Respond(w, http.StatusCreated, model, nil)
}
```

### Model Generation
```go
type Item struct {
    id        uuid.UUID `json:"id"`
    Text      string    `json:"text"`
    Done      bool      `json:"done"`
    CreatedAt time.Time `json:"created_at"`
    UpdatedAt time.Time `json:"updated_at"`
    CreatedBy uuid.UUID `json:"created_by"`
    UpdatedBy uuid.UUID `json:"updated_by"`
}
```

### Validation Functions
```go
func ValidateCreateItem(ctx context.Context, model Item) []ValidationError {
    // Generated validation logic based on YAML field definitions
}
```

## Planned Enhancements

### Pagination
- **Status**: Planned
- **Description**: Implement pagination metadata in `meta` object
- **Example**: `{"page": 1, "per_page": 20, "total": 100}`

### Route Versioning
- **Status**: Planned  
- **Description**: Add `/internal/v1` prefix support
- **Implementation**: Router-level prefix, not per-handler

### Advanced Validation Rules
- **Status**: Planned
- **Description**: Extended validation types (email, min/max length, custom rules)
- **Current**: Basic required validation implemented

## Next Steps

### Immediate
1. **Pagination support** - Add pagination to List operations
2. **Route versioning** - Implement `/internal/v1` prefix pattern
3. **Enhanced validations** - Extend validation rule types

### Medium Term
4. **OpenAPI generation** - Auto-generate API documentation
5. **Content negotiation** - Support multiple response formats
6. **Rate limiting** - Built-in rate limiting middleware

---

**Summary**: API design standards are fully implemented with response envelopes, snake_case JSON fields, structured validation, and consistent logging. Generated handlers follow established patterns with proper error handling and request tracing.
