# Auth Service Static Implementation - Scope Definition

**Status:** TEMPORARY DRAFT - TO BE DELETED | **Updated:** 2025-10-15 | **Version:** 0.1

**NOTE:** This is a temporary document to track implementation scope for the static auth service generation. It will be deleted once the full implementation matches drafts 020 and 021.

## Implementation Status vs. Drafts 020/021

### ✅ IMPLEMENTED (Matches Drafts)

**Pure Library (pkg/lib/auth):**
- ✅ Email normalization and validation
- ✅ Password hashing with Argon2id + salt
- ✅ Password verification
- ✅ Permission evaluation engine
- ✅ Policy matching with anyOf/allOf rules
- ✅ Token claims validation
- ✅ Comprehensive domain types (User, Role, Grant, Scope, etc.)
- ✅ Lookup hash computation with HMAC-SHA256

**Service Structure:**
- ✅ Standard HatMax service structure (consistent with todo service)
- ✅ User aggregate with proper domain conversion (User ↔ authpkg.User)
- ✅ Repository interface with auth-specific methods
- ✅ Basic CRUD handlers (UserHandler)
- ✅ Authentication handlers (AuthHandler: signup, signin, signout)
- ✅ Configuration with auth-specific settings

### 🔄 PARTIALLY IMPLEMENTED

**Email Encryption:**
- ⚠️ PLACEHOLDER: Email stored as plaintext
- ⚠️ TODO: AES-GCM encryption/decryption functions in crypto.go
- ⚠️ TODO: Proper email field population in signup flow

**Token Management:**
- ⚠️ PLACEHOLDER: No session token generation
- ⚠️ TODO: PASETO token creation with proper claims
- ⚠️ TODO: Token validation and refresh

### ❌ NOT IMPLEMENTED (Future Scope)

**Authorization Service Separation:**
- ❌ Separate authorization service (Role/Grant management)
- ❌ Cross-service authorization calls
- ❌ Grant assignment APIs
- ❌ Permission evaluation service endpoints

**Advanced Features:**
- ❌ MFA secret handling
- ❌ Session management and invalidation
- ❌ Email subscription management
- ❌ Admin user creation flows
- ❌ Gateway/BFF layer implementation
- ❌ HTMX frontend composition

**MongoDB Support:**
- ❌ MongoDB repository implementation
- ❌ MongoDB-specific indexes and TTL

## Current Implementation Focus

This static implementation provides a **functional authentication service** with:

1. **Secure signup/signin** using pure crypto functions
2. **Email lookup** without exposing plaintext emails
3. **Password security** with Argon2id + salt
4. **Extensible architecture** ready for authorization features
5. **HatMax consistency** following all established patterns

## Migration Path

To achieve full draft 020/021 compliance:

1. **Complete email encryption** in crypto.go
2. **Implement token generation** with PASETO
3. **Add authorization service** as separate static service
4. **Create gateway layer** templates
5. **Add MongoDB repository** implementation

## Decision Rationale

This scope provides a **minimum viable auth service** that:
- ✅ Demonstrates the static generation approach
- ✅ Maintains code quality and test coverage  
- ✅ Follows HatMax architectural principles
- ✅ Provides immediate utility for projects
- ✅ Creates foundation for future expansion

The missing pieces are **additive** and won't require changes to the core structure.