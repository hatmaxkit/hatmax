# ADR-004: Validation Strategy in Handlers

**Status:** Accepted
**Date:** 2025-10-03
**Deciders:** Adrian

## Context

We need a consistent approach to validate input data in handlers without relying on tags or external libraries. The validation system must be easily testable, maintain a clear error contract for clients, and support both structural and business rule validation.

## Decision

### Validation Architecture

**Pure functions per resource type** - no reflection or magic:
```go
func ValidateCreateUser(ctx context.Context, req CreateUserRequest) []ValidationError
func ValidateUpdateUser(ctx context.Context, id string, req UpdateUserRequest) []ValidationError
func ValidateDeleteUser(ctx context.Context, id string) []ValidationError
```

**Two validation levels:**

1. **Shape validation** - Structural payload validation:
   - Required fields present
   - Basic type constraints
   - Format validation (email, UUID, etc.)
   - No external dependencies

2. **Business validation** - Domain rule validation:
   - Uniqueness constraints
   - Cross-field dependencies
   - State-dependent rules
   - May query repositories

### Validation Composition

**Error accumulation** - never fail fast:
```go
func ValidateCreateUser(ctx context.Context, req CreateUserRequest) []ValidationError {
    var errors []ValidationError

    // Shape validation
    errors = append(errors, validateRequiredFields(req)...)
    errors = append(errors, validateEmailFormat(req.Email)...)

    // Business validation
    errors = append(errors, validateEmailUniqueness(ctx, req.Email)...)

    return errors
}
```

**Composable validation functions:**
```go
func validateRequiredFields(req CreateUserRequest) []ValidationError
func validateEmailFormat(email string) []ValidationError
func validateEmailUniqueness(ctx context.Context, email string) []ValidationError
```

### Function-Based Validation

**Production default** - handlers use generated validation functions directly:
```go
func NewUserHandler(deps UserHandlerDeps) *UserHandler {
    return &UserHandler{
        repo: deps.UserRepo,
        // No validator dependency needed
    }
}
```

**Testing with inline functions** - simple and expressive:
```go
func TestUserHandler_Create_ValidationError(t *testing.T) {
    // Override validation function for this test
    originalValidate := ValidateCreateUser
    ValidateCreateUser = func(ctx context.Context, req CreateUserRequest) []ValidationError {
        return []ValidationError{
            {Field: "email", Code: "required", Message: "Email is required"},
        }
    }
    defer func() { ValidateCreateUser = originalValidate }()

    handler := NewUserHandler(UserHandlerDeps{
        UserRepo: mockRepo,
    })

    // Test validation error handling
}
```

**Alternative testing approach** - function variables for easier mocking:
```go
// In handler
type UserHandler struct {
    repo           UserRepo
    validateCreate func(context.Context, CreateUserRequest) []ValidationError
}

func NewUserHandler(deps UserHandlerDeps) *UserHandler {
    return &UserHandler{
        repo:           deps.UserRepo,
        validateCreate: ValidateCreateUser, // Default to generated function
    }
}

// In tests
func TestUserHandler_Create_ValidationError(t *testing.T) {
    handler := NewUserHandler(UserHandlerDeps{
        UserRepo: mockRepo,
    })

    // Simple function assignment
    handler.validateCreate = func(ctx context.Context, req CreateUserRequest) []ValidationError {
        return []ValidationError{{Field: "email", Code: "required"}}
    }

    // Test validation error handling
}
```

### Error Contract

**ValidationError structure:**
```go
type ValidationError struct {
    Field   string `json:"field"`
    Code    string `json:"code"`
    Message string `json:"message"`
}
```

**HTTP 400 response format:**
```json
{
  "error": {
    "code": "validation",
    "message": "Request validation failed",
    "details": [
      { "field": "email", "code": "required", "message": "Email is required" },
      { "field": "age", "code": "min", "message": "Must be >= 0" }
    ]
  }
}
```

**Success responses** maintain standard format:
```json
{
  "data": { "id": "u-1234", "email": "user@example.com" },
  "meta": { "created_at": "2025-10-03T10:30:00Z" }
}
```

## Implementation Patterns

### Validation Functions
```go
func ValidateCreateUser(ctx context.Context, req CreateUserRequest) []ValidationError
func ValidateUpdateUser(ctx context.Context, id string, req UpdateUserRequest) []ValidationError
func ValidateDeleteUser(ctx context.Context, id string) []ValidationError
```

### Function Implementation
```go
func ValidateCreateUser(ctx context.Context, req CreateUserRequest) []ValidationError {
    var errors []ValidationError

    // Shape validation
    if req.Email == "" {
        errors = append(errors, ValidationError{
            Field:   "email",
            Code:    "required",
            Message: "Email is required",
        })
    }

    if req.Email != "" && !isValidEmail(req.Email) {
        errors = append(errors, ValidationError{
            Field:   "email",
            Code:    "format",
            Message: "Email format is invalid",
        })
    }

    if req.Age < 0 {
        errors = append(errors, ValidationError{
            Field:   "age",
            Code:    "min",
            Message: "Age must be >= 0",
        })
    }

    // Business validation
    if req.Email != "" {
        errors = append(errors, validateEmailUniqueness(ctx, req.Email)...)
    }

    return errors
}

// Helper function for business validation
func validateEmailUniqueness(ctx context.Context, email string) []ValidationError {
    // In real implementation, this would access a repository
    // For generated code, this could be injected as a dependency
    return nil // Placeholder
}
```

### Handler Integration
```go
func (h *UserHandler) CreateUser(w http.ResponseWriter, r *http.Request) {
    var req CreateUserRequest
    if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
        WriteError(w, 400, "invalid_json", "Invalid JSON format", nil)
        return
    }

    // Validate request
    if validationErrors := ValidateCreateUser(r.Context(), req); len(validationErrors) > 0 {
        WriteValidationError(w, validationErrors)
        return
    }

    // Process valid request
    user, err := h.userService.CreateUser(r.Context(), req)
    if err != nil {
        // Handle business errors
        WriteError(w, 500, "creation_failed", "Failed to create user", nil)
        return
    }

    WriteSuccess(w, user, nil)
}
```

### Common Validation Helpers
```go
func isValidEmail(email string) bool {
    // Email format validation
}

func isValidUUID(id string) bool {
    // UUID format validation
}

func validateStringLength(value string, min, max int) *ValidationError {
    if len(value) < min {
        return &ValidationError{
            Code:    "min_length",
            Message: fmt.Sprintf("Must be at least %d characters", min),
        }
    }
    if len(value) > max {
        return &ValidationError{
            Code:    "max_length",
            Message: fmt.Sprintf("Must be at most %d characters", max),
        }
    }
    return nil
}
```

### Test Patterns
```go
// Testing with function variables - simple and direct
func TestUserHandler_Create_ValidationError(t *testing.T) {
    // Create test-specific validation function
    testValidateCreate := func(ctx context.Context, req CreateUserRequest) []ValidationError {
        return []ValidationError{
            {Field: "email", Code: "required", Message: "Email is required"},
        }
    }

    // Test the handler behavior with validation errors
    // (Implementation would depend on how functions are injected)
}

// Testing with function composition
func TestValidationComposition(t *testing.T) {
    alwaysValid := func(ctx context.Context, req CreateUserRequest) []ValidationError {
        return nil
    }

    alwaysInvalid := func(ctx context.Context, req CreateUserRequest) []ValidationError {
        return []ValidationError{{Field: "test", Code: "invalid"}}
    }

    conditionalValidator := func(forbiddenEmail string) func(context.Context, CreateUserRequest) []ValidationError {
        return func(ctx context.Context, req CreateUserRequest) []ValidationError {
            if req.Email == forbiddenEmail {
                return []ValidationError{{Field: "email", Code: "forbidden"}}
            }
            return nil
        }
    }

    // Test composition
    composed := composeValidators(alwaysValid, conditionalValidator("test@evil.com"))
    errors := composed(context.Background(), CreateUserRequest{Email: "test@evil.com"})

    if len(errors) == 0 {
        t.Error("Expected validation errors")
    }
}

// Helper for composing validators
func composeValidators(validators ...func(context.Context, CreateUserRequest) []ValidationError) func(context.Context, CreateUserRequest) []ValidationError {
    return func(ctx context.Context, req CreateUserRequest) []ValidationError {
        var allErrors []ValidationError
        for _, validate := range validators {
            errors := validate(ctx, req)
            allErrors = append(allErrors, errors...)
        }
        return allErrors
    }
}
```

## Consequences

### Positive
- **Zero ceremony**: No interfaces, constructors, or dependency injection complexity
- **Extremely testable**: Functions can be replaced inline or assigned as variables
- **Natural composition**: Easy to combine, chain, and conditionally apply validators
- **No external dependencies**: Pure Go functions with no reflection or magic
- **Clear separation**: Shape vs business validation with explicit error accumulation
- **Expressive testing**: `alwaysValid`, `alwaysInvalid`, `conditionalValidator` patterns
- **Simple mocking**: Function assignment instead of complex mock frameworks

### Negative
- **Manual implementation**: No tag-based shortcuts for common patterns
- **Function management**: Need to organize validation functions appropriately
- **Testing coordination**: Global function replacement requires careful cleanup

### Mitigation
- **Code generation**: Automatic generation of validation functions from model definitions
- **Helper libraries**: Shared validation primitives in `hm` package
- **Clear patterns**: Documented examples for composition and testing
- **Function organization**: Generated functions grouped by resource type

---

**TL;DR**: Pure function validators per resource, two-level validation (shape + business), error accumulation, zero-ceremony testing with function replacement, and uniform HTTP 400 error contract with structured details.
