# Auth Service: Pure Logic Implementation Considerations

**Status:** Draft | **Updated:** 2025-10-14 | **Version:** 0.1

## Overview

This document explores implementation strategies for the authentication and authorization service (draft 020) with a focus on separating pure logic from side effects. The goal is to achieve referential transparency where it matters most while maintaining pragmatic Go idioms and consistency with HatMax's architectural principles.

## 1. The Problem

Authentication and authorization logic naturally falls into two categories:

**Pure Logic** (referentially transparent):
- Email normalization and validation
- Password hashing and verification
- Permission evaluation algorithms  
- Policy matching and resolution
- Token claim validation
- Cryptographic operations

**Side Effects** (I/O bound, non-deterministic):
- Database queries and updates
- Key management system calls
- Session storage operations
- Logging and metrics
- HTTP requests and responses
- Cache operations

The question is: should we implement this as a traditional microservice (mixing pure and impure code) or separate the pure logic into a reusable library?

## 2. Implementation Approaches

### 2.1 Traditional Microservice (Status Quo)

Following the established HatMax pattern where all logic lives within the service:

```go
// services/auth/internal/auth/service.go
type AuthService struct {
    repo Repository
    kms  KMSProvider
    log  Logger
}

func (s *AuthService) Authenticate(ctx context.Context, req AuthRequest) (*AuthResponse, error) {
    // Mixed pure + impure logic
    normalizedEmail := strings.ToLower(strings.TrimSpace(req.Email))
    lookupHash := s.computeLookup(normalizedEmail) // impure: calls KMS
    
    user, err := s.repo.FindByEmailLookup(ctx, lookupHash) // impure: DB
    if err != nil {
        return nil, err
    }
    
    if !s.verifyPassword(req.Password, user.Hash, user.Salt) { // impure: calls KMS
        return nil, ErrInvalidCredentials
    }
    
    return s.createSession(ctx, user) // impure: DB + cache
}
```

**Pros:**
- Consistent with other HatMax services
- Simple to understand and maintain
- No additional complexity

**Cons:**
- Testing requires mocks for all dependencies
- Logic is not reusable across services
- Harder to reason about correctness
- Mixed concerns in business logic

### 2.2 Pure Core + Microservice Wrapper (Recommended)

Separate pure logic into a standalone package while maintaining the microservice structure:

```go
// pkg/lib/auth/crypto.go (Pure functions)
package auth

func NormalizeEmail(email string) string {
    return strings.ToLower(strings.TrimSpace(email))
}

func ComputeLookupHash(email string, key []byte) []byte {
    h := hmac.New(sha256.New, key)
    h.Write([]byte(email))
    return h.Sum(nil)
}

func VerifyPasswordHash(password, hash, salt []byte) bool {
    derived := argon2.IDKey(password, salt, 1, 64*1024, 4, 32)
    return subtle.ConstantTimeCompare(derived, hash) == 1
}
```

```go
// pkg/lib/auth/permissions.go (Pure functions)
func EvaluatePermissions(grants []Grant, permission string, scope Scope) bool {
    for _, grant := range grants {
        if grant.ExpiresAt != nil && grant.ExpiresAt.Before(time.Now()) {
            continue
        }
        
        if !scopeMatches(grant.Scope, scope) {
            continue
        }
        
        if grant.GrantType == "permission" && grant.Value == permission {
            return true
        }
        
        if grant.GrantType == "role" {
            // Would need role expansion, but pure if roles passed as parameter
        }
    }
    return false
}
```

```go
// services/auth/internal/auth/service.go (Orchestration + Side Effects)
package auth

import (
    "context"
    "fmt"
    
    "github.com/company/project/pkg/lib/auth" // Import from monorepo root
)

type AuthService struct {
    repo Repository
    kms  KMSProvider  
    log  Logger
}

func (s *AuthService) Authenticate(ctx context.Context, req AuthRequest) (*AuthResponse, error) {
    // Pure logic
    normalizedEmail := auth.NormalizeEmail(req.Email)
    
    // Side effect: get key from KMS
    lookupKey, err := s.kms.GetLookupKey(ctx)
    if err != nil {
        return nil, fmt.Errorf("kms lookup key: %w", err)
    }
    
    // Pure logic
    lookupHash := auth.ComputeLookupHash(normalizedEmail, lookupKey)
    
    // Side effect: database query
    user, err := s.repo.FindByEmailLookup(ctx, lookupHash)
    if err != nil {
        return nil, fmt.Errorf("find user: %w", err)
    }
    
    // Pure logic
    if !auth.VerifyPasswordHash([]byte(req.Password), user.PasswordHash, user.PasswordSalt) {
        s.log.Warn("invalid password attempt", "user_id", user.ID)
        return nil, ErrInvalidCredentials
    }
    
    // Side effect: create session
    return s.createSession(ctx, user)
}
```

## 3. Project Structure

### 3.1 Pure Core Library Structure (Monorepo Root)

```
{monorepo-root}/
├── pkg/lib/auth/          # Pure auth logic at monorepo level
│   ├── crypto.go          # Pure cryptographic operations
│   ├── permissions.go     # Pure permission evaluation logic
│   ├── policies.go        # Pure policy matching algorithms
│   ├── validation.go      # Pure input validation functions
│   ├── tokens.go          # Pure token claim validation
│   ├── types.go          # Domain types (User, Grant, Scope, etc.)
│   └── errors.go         # Domain errors
└── services/
    └── auth/              # Auth microservice imports from pkg/lib/auth
```

### 3.2 Microservice Structure

```
services/auth/
├── internal/
│   ├── auth/
│   │   ├── service.go      # Orchestrates pure functions + effects  
│   │   ├── handlers.go     # HTTP handlers
│   │   ├── repository.go   # Database operations
│   │   └── kms.go         # Key management operations
│   ├── config/
│   └── middleware/
├── cmd/auth/
│   └── main.go
├── go.mod                  # Imports ../../../pkg/lib/auth
└── Dockerfile
```

## 4. Testing Strategy

### 4.1 Pure Logic Testing

Pure functions can be tested without any mocks or setup:

```go
// pkg/lib/auth/crypto_test.go
func TestNormalizeEmail(t *testing.T) {
    tests := []struct {
        input    string
        expected string
    }{
        {"User@Example.COM  ", "user@example.com"},
        {"  test@test.org", "test@test.org"},
    }
    
    for _, tt := range tests {
        result := NormalizeEmail(tt.input)
        assert.Equal(t, tt.expected, result)
    }
}

func TestComputeLookupHash(t *testing.T) {
    key := []byte("test-key")
    email := "user@example.com"
    
    hash1 := ComputeLookupHash(email, key)
    hash2 := ComputeLookupHash(email, key)
    
    // Same inputs always produce same outputs (referential transparency)
    assert.Equal(t, hash1, hash2)
    assert.Len(t, hash1, 32) // SHA256 output length
}
```

### 4.2 Service Layer Testing

Integration tests focus on orchestration and side effects:

```go
// services/auth/internal/auth/service_test.go
func TestAuthService_Authenticate(t *testing.T) {
    // Setup mocks only for side effects
    mockRepo := &MockRepository{}
    mockKMS := &MockKMSProvider{}
    
    service := NewAuthService(mockRepo, mockKMS, slog.Default())
    
    // Test orchestration and error handling
    // Pure logic correctness is already tested in authcore package
}
```

## 5. HatMax Integration

### 5.1 YAML Configuration

The HatMax service definition would specify the pure core approach:

```yaml
version: 0.1
name: "auth-system"
package: "github.com/company/auth-system"

    services:
  auth:
    kind: auth
    implementation:
      pure_core: true          # Generate pkg/lib/auth + service wrapper
      core_package: "auth"     # Name of the pure logic package in pkg/lib/
    
    components:
      - authentication
      - authorization  
      - gateway
    
    storage:
      impl: [mongo]
      encryption:
        provider: kms
        fields: [email, mfa_secret]
```

### 5.2 Generated Code Structure

HatMax would generate:

**Pure Core Package (Monorepo Root):**
- `{monorepo-root}/pkg/lib/auth/*.go` - All pure business logic functions
- `{monorepo-root}/pkg/lib/auth/*_test.go` - Unit tests with no mocks

**Service Package:**  
- `services/auth/internal/auth/service.go` - Orchestration layer calling pure functions
- `services/auth/internal/auth/handlers.go` - HTTP handlers  
- `services/auth/internal/auth/repository.go` - Database operations
- `services/auth/internal/auth/*_test.go` - Integration tests with mocks for side effects

### 5.3 Code Generation Templates

The generator would use different templates based on the `pure_core` flag:

```go
// Template for pure functions
func {{.FunctionName}}({{range .Params}}{{.Name}} {{.Type}}, {{end}}) {{.ReturnType}} {
    // Generated pure logic
    {{.Body}}
}

// Template for service orchestration  
func (s *{{.ServiceName}}) {{.MethodName}}(ctx context.Context, req {{.RequestType}}) (*{{.ResponseType}}, error) {
    // Pure logic calls
    {{range .PureCalls}}
    {{.Variable}} := {{.Package}}.{{.Function}}({{.Args}})
    {{end}}
    
    // Side effect calls  
    {{range .SideEffects}}
    {{.Variable}}, err := s.{{.Dependency}}.{{.Method}}(ctx, {{.Args}})
    if err != nil {
        return nil, fmt.Errorf("{{.ErrorContext}}: %w", err)
    }
    {{end}}
    
    return {{.ReturnValue}}, nil
}
```

## 6. Benefits and Trade-offs

### 6.1 Benefits

**Testability:**
- Pure functions need no mocks, dependencies, or setup
- Tests run fast and are deterministic
- Easy to achieve high coverage on business logic

**Reusability:**
- `pkg/lib/auth` can be imported by other services
- Gateway service can use permission evaluation directly
- Shared validation logic across components

**Reasoning:**
- Pure functions are easier to understand and debug
- Referential transparency makes behavior predictable
- Clear separation between "what" (pure logic) and "how" (side effects)

**Performance:**
- Pure functions can be memoized/cached safely
- No I/O in critical path operations like permission evaluation
- Easier to optimize and profile

### 6.2 Trade-offs

**Complexity:**
- Additional package to maintain
- Need to carefully design pure function interfaces
- More files and structure to understand

**Learning Curve:**
- Team needs to understand pure vs impure separation
- Requires discipline to maintain boundaries
- New pattern for HatMax services

**Dependencies:**
- Pure functions can't directly access external state
- Need to pass all required data as parameters
- May require interface design changes

## 7. Decision Framework

### 7.1 When to Use Pure Core

Use the pure core approach when:

- **Complex business logic** that benefits from isolated testing
- **Reusable algorithms** across multiple services  
- **Performance-critical** operations that need optimization
- **Regulatory/compliance** requirements for logic auditability

### 7.2 When to Use Traditional Approach

Use traditional microservice when:

- **Simple CRUD** operations with minimal logic
- **Service-specific** logic unlikely to be reused
- **Team preference** for consistency over purity
- **Time constraints** don't allow additional complexity

### 7.3 For Auth Service Specifically

Auth service **should use pure core** because:

- **Security-critical** logic benefits from isolated testing
- **Permission evaluation** is reused across gateway and services
- **Cryptographic operations** are inherently pure and testable
- **Compliance** often requires logic auditability
- **Performance** matters for permission checks

## 8. Migration Strategy

### 8.1 Gradual Adoption

1. **Start traditional** - Generate standard microservice first
2. **Extract pure functions** - Move testable logic to `{monorepo-root}/pkg/lib/auth`
3. **Update service layer** - Change to orchestration pattern
4. **Add pure tests** - Create fast test suite for core logic
5. **Optimize** - Cache and optimize pure functions

### 8.2 Team Training

- **Workshop on purity** - Explain concepts and benefits
- **Code review guidelines** - What belongs in pure vs impure layers
- **Testing practices** - How to test both layers effectively

## 9. Implementation Guidelines

### 9.1 Pure Function Design Rules

1. **No I/O operations** - No database, network, file system access
2. **No global state** - No reading from global variables or singletons
3. **No time dependencies** - Pass time.Time as parameter if needed
4. **No logging** - Return errors instead, let caller log
5. **Deterministic** - Same inputs always produce same outputs
6. **No mutation** - Don't modify input parameters

### 9.2 Service Layer Responsibilities  

1. **Dependency injection** - Manage all external dependencies
2. **Error handling** - Convert domain errors to appropriate responses  
3. **Logging and metrics** - All observability concerns
4. **Transaction management** - Database and cache consistency
5. **Input/output mapping** - Convert between HTTP and domain types

## 10. Conclusion

The pure core approach offers significant benefits for the auth service specifically, while maintaining consistency with HatMax's generation philosophy. The key insight is that auth logic naturally separates into pure algorithms (crypto, permissions, policies) and side effects (storage, key management, sessions).

This approach provides:
- **Better testing** through pure function isolation
- **Higher confidence** in security-critical logic  
- **Improved reusability** across the microservice ecosystem
- **Performance optimization** opportunities
- **Regulatory compliance** through auditable logic

The trade-off in complexity is justified for auth services given their critical nature and cross-cutting concerns. Other services can continue using the traditional approach unless they have similar characteristics requiring pure logic separation.

The implementation should be gradual, starting with traditional generation and evolving toward pure core as the pattern proves successful and the team develops familiarity with the concepts.