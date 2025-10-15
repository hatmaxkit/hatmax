# RBAC Service Design

**Status:** Planned | **Updated:** 2025-10-06 | **Version:** 0.1

## Overview

Role-Based Access Control (RBAC) service design with contextual permissions and simplified interface patterns. Based on proven Hermes implementation analysis, focusing on pragmatic patterns while avoiding over-engineering.

## Current Implementation

**Status**: Basic authentication middleware exists  
**Location**: Generated auth middleware templates  
**Features**: JWT validation, basic permission checking

## Planned Implementation

### Key Principles
- **Contextual roles** with org/team hierarchy support
- **Direct permissions** to avoid micro-role explosion  
- **UNION queries** for efficient permission resolution
- **Simplified interfaces** (focused, essential methods only)
- **Clean data models** without unnecessary complexity

### Core Entities

```yaml path=null start=null
# Core entities (simplified approach)
user:
  id: uuid
  email: string
  name: string
  active: boolean

role:
  id: uuid
  code: string           # admin, team_lead, developer, viewer
  name: string

permission:
  id: uuid
  action: string         # read, write, delete, admin
  resource: string       # todos, billing, users, teams

organization:
  id: uuid
  name: string
  active: boolean

team:
  id: uuid
  org_id: uuid          # FK to organization
  name: string
  active: boolean

# Relations (contextual patterns)
user_roles:
  user_id: uuid
  role_id: uuid
  context_type: string?  # null=global, 'org', 'team'
  context_id: uuid?      # org_id or team_id

user_permissions:       # Direct permissions
  user_id: uuid
  permission_id: uuid

role_permissions:
  role_id: uuid
  permission_id: uuid
```

### Service Interface

```go path=null start=null
// Focused interface with essential methods only
type RBACService interface {
    // Core authorization
    Authorize(ctx context.Context, req AuthzRequest) (*AuthzResponse, error)
    GetUserPermissions(ctx context.Context, userID uuid.UUID, contextType, contextID string) ([]Permission, error)

    // User management
    CreateUser(ctx context.Context, user *User) error
    GetUser(ctx context.Context, id uuid.UUID) (*User, error)
    UpdateUser(ctx context.Context, user *User) error

    // Role assignment (contextual)
    AssignRole(ctx context.Context, userID, roleID uuid.UUID, contextType, contextID string) error
    RevokeRole(ctx context.Context, userID, roleID uuid.UUID, contextType, contextID string) error

    // Direct permissions
    GrantPermission(ctx context.Context, userID, permissionID uuid.UUID) error
    RevokePermission(ctx context.Context, userID, permissionID uuid.UUID) error
}
```

### Authorization Logic

```go path=null start=null
func (s *RBACService) Authorize(ctx context.Context, req AuthzRequest) (*AuthzResponse, error) {
    // UNION pattern for effective permissions
    query := `
    SELECT DISTINCT permission_id
    FROM (
        -- Direct permissions
        SELECT permission_id FROM user_permissions WHERE user_id = ?
        UNION
        -- Role-based permissions with context fallback
        SELECT rp.permission_id
        FROM user_roles ur
        JOIN role_permissions rp ON ur.role_id = rp.role_id
        WHERE ur.user_id = ?
        AND (
            (ur.context_type = ? AND ur.context_id = ?) OR  -- Exact context
            (ur.context_type IS NULL)                        -- Global fallback
        )
    )`

    // Check if action+resource exists in effective permissions
    // Return with TTL for caching
}
```

### **Authorization Logic (from Hermes)**

```go
func (s *RBACService) Authorize(ctx context.Context, req AuthzRequest) (*AuthzResponse, error) {
    // Use Hermes UNION pattern for effective permissions
    query := `
    SELECT DISTINCT permission_id
    FROM (
        -- Direct permissions
        SELECT permission_id FROM user_permissions WHERE user_id = ?
        UNION
        -- Role-based permissions with context fallback
        SELECT rp.permission_id
        FROM user_roles ur
        JOIN role_permissions rp ON ur.role_id = rp.role_id
        WHERE ur.user_id = ?
        AND (
            (ur.context_type = ? AND ur.context_id = ?) OR  -- Exact context
            (ur.context_type IS NULL)                        -- Global fallback
        )
    )`

    // Check if action+resource exists in effective permissions
    // Return with TTL for caching
}
```

### Data Models

```go path=null start=null
type User struct {
    ID       uuid.UUID `json:"id" db:"id"`
    Email    string    `json:"email" db:"email"`
    Name     string    `json:"name" db:"name"`
    Active   bool      `json:"active" db:"active"`
    CreatedAt time.Time `json:"created_at" db:"created_at"`
    UpdatedAt time.Time `json:"updated_at" db:"updated_at"`
    
    // Loaded relations (not stored)
    Roles       []Role       `json:"roles,omitempty"`
    Permissions []Permission `json:"permissions,omitempty"`
}

type AuthzRequest struct {
    Subject     string `json:"subject"`      // user_id
    Action      string `json:"action"`       // read, write, delete
    Resource    string `json:"resource"`     // todos, billing
    ContextType string `json:"context_type"` // org, team, null
    ContextID   string `json:"context_id"`   // org_id or team_id
}

type AuthzResponse struct {
    Allowed bool `json:"allowed"`
    TTL     int  `json:"ttl"`     // Cache duration in seconds
}
```

## Generator Integration

### YAML Configuration

```yaml path=null start=null
services:
  rbac:
    kind: domain
    repo_impl: [sqlite]
    
    models:
      User:
        fields:
          email: {type: email, validations: [required, unique]}
          name: {type: string, validations: [required]}
          active: {type: bool, default: true}
      
      Role:
        fields:
          code: {type: string, validations: [required, unique]}
          name: {type: string, validations: [required]}
      
      Permission:
        fields:
          action: {type: string, validations: [required]}
          resource: {type: string, validations: [required]}
    
    relations:
      - {from: User, to: Role, kind: many_to_many, via: user_roles, context: {type: string?, id: uuid?}}
      - {from: User, to: Permission, kind: many_to_many, via: user_permissions}
      - {from: Role, to: Permission, kind: many_to_many, via: role_permissions}
    
    api:
      base_path: /rbac
      handlers:
        - {route: "POST /authorize", source: usecase, usecase: Authorize}
        - {route: "GET /users", source: repo, model: User, op: list}
        - {route: "POST /users/{user_id}/roles", source: usecase, usecase: AssignRole}
```

### Authentication Middleware

```yaml path=null start=null
# Any service can enable auth
services:
  billing:
    kind: domain
    auth:
      enabled: true
      mode: jwt_local
      rbac_service: "http://rbac:8080"
      required_scopes: ["read:billing"]
      cache_ttl: 300
```

**Generated Middleware**:
```go path=null start=null
func (s *BillingService) AuthMiddleware(next http.Handler) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        // 1. Validate JWT
        claims, err := s.validateJWT(extractToken(r))
        if err != nil {
            hm.Error(w, 401, "unauthorized", "Invalid token")
            return
        }
        
        // 2. RBAC check
        allowed, err := s.rbacClient.Authorize(claims.Subject, r.Method, "billing", extractContext(r))
        if err != nil || !allowed {
            hm.Error(w, 403, "forbidden", "Insufficient permissions")
            return
        }
        
        // 3. Set user context
        ctx := context.WithValue(r.Context(), "user_id", claims.Subject)
        next.ServeHTTP(w, r.WithContext(ctx))
    })
}
```

## Planned Features

### Contextual Relations
- **Status**: Planned
- **Description**: Support for context-aware role assignments (org/team scope)
- **Implementation**: Extended relation syntax with context fields

### Custom SQL in Use Cases
- **Status**: Planned
- **Description**: Allow raw SQL queries in use case steps
- **Benefit**: Complex authorization queries with optimal performance

### Pluggable Auth Modes
- **Status**: Planned
- **Description**: Multiple authentication strategies per service
- **Modes**: `jwt_local`, `jwt_remote`, `rbac_full`

### Permission Caching
- **Status**: Planned
- **Description**: TTL-based permission caching for performance
- **Implementation**: Redis or in-memory cache with configurable TTL

## Implementation Roadmap

### Phase 1: Core RBAC Service
1. **Basic RBAC entities** - User, Role, Permission models
2. **Contextual relations** - Extend generator for context support
3. **Authorization logic** - UNION-based permission resolution
4. **Basic API endpoints** - CRUD operations and authorization

### Phase 2: Generator Integration  
5. **Auth middleware generation** - JWT validation and RBAC checks
6. **Custom SQL support** - Raw queries in use case steps
7. **Multi-mode auth** - Different auth strategies per service
8. **Permission caching** - TTL-based performance optimization

### Phase 3: Advanced Features
9. **Audit trail** - Permission change logging
10. **Hierarchical contexts** - Nested org/team structures
11. **Dynamic permissions** - Runtime permission evaluation
12. **Integration testing** - End-to-end auth flow testing

## Next Steps

### Immediate
1. **Design contextual relations** syntax for YAML
2. **Implement basic RBAC models** with current generator
3. **Prototype authorization endpoint** with UNION queries

### Medium Term
4. **Extend generator for auth middleware** generation
5. **Add custom SQL support** in use case steps
6. **Implement permission caching** with configurable TTL

---

**Summary**: Pragmatic RBAC service design with contextual permissions, simplified interfaces, and seamless generator integration. Focuses on essential authorization patterns while maintaining flexibility for complex permission scenarios.
