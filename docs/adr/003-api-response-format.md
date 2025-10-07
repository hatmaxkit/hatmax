# ADR-003: API Response Format (JSON)

**Status:** Accepted
**Date:** 2025-10-03
**Deciders:** Adrian

## Context

We need a simple, consistent, and stable JSON contract for HTTP responses that avoids exposing internal structs and minimizes breaking changes. The API should be predictable for consumers while providing flexibility for evolution.

## Decision

### Envelope Structure

**Unified envelope pattern** with two distinct response types:

**Success responses:**
```json
{
  "data": <payload>,
  "meta": <metadata>?  // optional
}
```

**Error responses:**
```json
{
  "error": {
    "code": "error_type",
    "message": "Human readable message",
    "details": [...]?  // optional, for validation errors
  }
}
```

### Response Patterns

**Implicit typing by endpoint** - no redundant wrapping:
- `/users/:id` returns user object directly in `data`
- No additional `{ "user": { ... } }` wrapper

**Pointwise composition** for multi-resource responses:
```json
{
  "data": {
    "user": { "id": "u-1234", ... },
    "roles": [ { "id": "r-1", ... } ]
  }
}
```

**Collections** with pagination metadata:
```json
{
  "data": [ { "id": "u-1" }, { "id": "u-2" } ],
  "meta": {
    "page": 1,
    "per_page": 20,
    "total": 42
  }
}
```

### Data Formatting

**Date format:** ISO-8601 UTC (e.g., `2025-10-02T18:30:00Z`)

**Naming convention:** snake_case for JSON fields (consistent across all responses)

**Null handling:** Omit null fields from response rather than including `"field": null`

### Versioning Strategy

**Path-based versioning:** `/api/v1/...`

**Compatibility rules:**
- Adding fields is backward compatible
- Renaming or removing fields requires new API version
- Field type changes require new API version

### Error Handling

**No `data` field in error responses** - clear separation of success/error states

**HTTP status code mapping:**
- `400` - Validation errors
- `401` - Authentication required
- `403` - Authorization failed
- `404` - Resource not found
- `409` - Conflict (e.g., duplicate resource)
- `422` - Unprocessable entity (optional, detailed validation)
- `500` - Internal server error

**Structured error codes** for programmatic handling - no reliance on HTTP status alone

## Response Examples

### Single Resource Success
```json
{
  "data": {
    "id": "u-1234",
    "email": "user@example.com",
    "created_at": "2025-10-02T18:30:00Z"
  }
}
```

### Collection Success
```json
{
  "data": [
    { "id": "u-1", "email": "user1@example.com" },
    { "id": "u-2", "email": "user2@example.com" }
  ],
  "meta": {
    "page": 1,
    "per_page": 20,
    "total": 42
  }
}
```

### Composite Resource Success
```json
{
  "data": {
    "user": {
      "id": "u-1234",
      "email": "user@example.com"
    },
    "roles": [
      { "id": "r-1", "name": "admin" },
      { "id": "r-2", "name": "user" }
    ]
  }
}
```

### Validation Error
```json
{
  "error": {
    "code": "validation",
    "message": "Request validation failed",
    "details": [
      {
        "field": "email",
        "code": "required",
        "message": "Email is required"
      },
      {
        "field": "password",
        "code": "min_length",
        "message": "Password must be at least 8 characters"
      }
    ]
  }
}
```

### Business Logic Error
```json
{
  "error": {
    "code": "insufficient_funds",
    "message": "Account balance insufficient for this transaction"
  }
}
```

### Not Found Error
```json
{
  "error": {
    "code": "not_found",
    "message": "User not found"
  }
}
```

## Implementation Guidelines

### Handler Response Pattern
```go
type APIResponse struct {
    Data interface{} `json:"data,omitempty"`
    Meta interface{} `json:"meta,omitempty"`
}

type APIError struct {
    Code    string      `json:"code"`
    Message string      `json:"message"`
    Details interface{} `json:"details,omitempty"`
}

type ErrorResponse struct {
    Error APIError `json:"error"`
}
```

### Pagination Metadata
```go
type PaginationMeta struct {
    Page    int `json:"page"`
    PerPage int `json:"per_page"`
    Total   int `json:"total"`
}
```

### Response Helpers
```go
func WriteSuccess(w http.ResponseWriter, data interface{}, meta interface{}) error
func WriteError(w http.ResponseWriter, statusCode int, code, message string, details interface{}) error
func WriteValidationError(w http.ResponseWriter, errors []ValidationError) error
```

## Consequences

### Positive
- Predictable contract easy for clients to consume
- Low coupling with internal models
- Flexibility for composition without endpoint inflation
- Controlled evolution via versioning
- Programmatic error handling via structured codes
- Clear separation of success/error states

### Negative
- Additional mapping layer between internal models and API responses
- Potential duplication of field definitions
- Need for careful version management

### Mitigation
- Code generation for response mapping
- Shared type definitions for common patterns
- Clear documentation and examples
- Automated compatibility testing between versions

---

**TL;DR**: Unified envelope pattern (`data`/`meta` for success, `error` for failures), implicit typing by endpoint, snake_case naming, ISO-8601 dates, path-based versioning, and structured error codes for programmatic handling.
