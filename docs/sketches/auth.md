# Dev Auth Probe

**Status**: Archived  
**Type**: Development experiment  
**Purpose**: Historical reference for authentication middleware testing

## Context

A previous experiment added a `test_auth.go` file at the repository root as a manual executable to validate the authentication middleware generated in the `todo` feature. When regenerating the app (previously `app/`, now `example/ref/`), the import `github.com/adrianpk/hatmax/app/services/todo/internal/todo` became invalid and the snippet was lost.

This document archives the original experiment for reference when it's time to template authentication tests.

> **Note**: The current fake implementation lives in `internal/hm`; this fragment preserves the original experiment for historical context.

## Original Test Implementation

```go
package main

import (
    "bytes"
    "encoding/json"
    "fmt"
    "net/http"
    "net/http/httptest"

    "github.com/adrianpk/hatmax/app/services/todo/internal/todo"
    "github.com/adrianpk/hatmax/pkg/lib/hm"
)

// Mock repository for testing
type MockItemRepo struct{}

func (m *MockItemRepo) Create(item *todo.Item) error {
    item.ID = "test-123"
    return nil
}

func (m *MockItemRepo) Get(id string) (*todo.Item, error) {
    return &todo.Item{ID: id, Text: "Test Item", Done: false}, nil
}

func (m *MockItemRepo) List() ([]*todo.Item, error) {
    return []*todo.Item{
        {ID: "1", Text: "Item 1", Done: false},
        {ID: "2", Text: "Item 2", Done: true},
    }
}

func (m *MockItemRepo) Update(item *todo.Item) error {
    return nil
}

func (m *MockItemRepo) Delete(id string) error {
    return nil
}

func main() {
    // Create dependencies
    repo := &MockItemRepo{}
    log := hm.NewNoopLogger()
    authService := todo.NewFakeAuthService()

    // Create handler
    handler := todo.NewItemHandler(repo, log, authService)
    router := handler.Routes()

    fmt.Println("🧪 Testing Auth Integration...")

    // Test 1: Request without token (should fail)
    fmt.Println("\n1. Testing request without auth token...")
    req := httptest.NewRequest("GET", "/", nil)
    w := httptest.NewRecorder()
    router.ServeHTTP(w, req)
    fmt.Printf("   Status: %d (expected: 401)\n", w.Code)
    if w.Code == 401 {
        fmt.Println("   ✅ Correctly rejected unauthorized request")
    } else {
        fmt.Println("   ❌ Should have rejected unauthorized request")
    }

    // Test 2: Request with invalid token (should fail)
    fmt.Println("\n2. Testing request with invalid token...")
    req = httptest.NewRequest("GET", "/", nil)
    req.Header.Set("Authorization", "Bearer invalid-token")
    w = httptest.NewRecorder()
    router.ServeHTTP(w, req)
    fmt.Printf("   Status: %d (expected: 401)\n", w.Code)
    if w.Code == 401 {
        fmt.Println("   ✅ Correctly rejected invalid token")
    } else {
        fmt.Println("   ❌ Should have rejected invalid token")
    }

    // Test 3: Request with valid dev token (should succeed)
    fmt.Println("\n3. Testing request with valid dev token...")
    req = httptest.NewRequest("GET", "/", nil)
    req.Header.Set("Authorization", "Bearer dev-admin")
    w = httptest.NewRecorder()
    router.ServeHTTP(w, req)
    fmt.Printf("   Status: %d (expected: 200)\n", w.Code)
    if w.Code == 200 {
        fmt.Println("   ✅ Successfully authenticated with dev token")
        fmt.Printf("   Response: %s\n", w.Body.String()[:100]+"...")
    } else {
        fmt.Println("   ❌ Should have authenticated with valid dev token")
    }

    // Test 4: POST request with valid token
    fmt.Println("\n4. Testing POST request with valid token...")
    body := map[string]interface{}{
        "text": "New todo item",
        "done": false,
    }
    bodyBytes, _ := json.Marshal(body)
    req = httptest.NewRequest("POST", "/", bytes.NewBuffer(bodyBytes))
    req.Header.Set("Authorization", "Bearer dev-user")
    req.Header.Set("Content-Type", "application/json")
    w = httptest.NewRecorder()
    router.ServeHTTP(w, req)
    fmt.Printf("   Status: %d (expected: 201)\n", w.Code)
    if w.Code == 201 {
        fmt.Println("   ✅ Successfully created item with auth")
    } else {
        fmt.Println("   ❌ Should have created item with valid auth")
    }

    fmt.Println("\n🎉 Auth integration test complete!")
    fmt.Println("\n💡 To test manually:")
    fmt.Println("   curl -H 'Authorization: Bearer dev-admin' http://localhost:8080/todo/items")
    fmt.Println("   curl -H 'Authorization: Bearer invalid' http://localhost:8080/todo/items")
}
```

## Key Test Scenarios

1. **No Authorization Header** - Should return 401 Unauthorized
2. **Invalid Bearer Token** - Should return 401 Unauthorized  
3. **Valid Dev Token** - Should return 200 with data
4. **Authenticated POST Request** - Should create resource successfully

## Integration Points

- **Mock Repository**: `MockItemRepo` for testing without database
- **Fake Auth Service**: `todo.NewFakeAuthService()` for development
- **Generated Handler**: `todo.NewItemHandler()` with auth middleware
- **Dev Tokens**: `dev-admin`, `dev-user` for testing different permission levels

## Future Template Considerations

When creating authentication test templates:

- **Generate test files** alongside handlers
- **Include mock implementations** for all dependencies
- **Provide dev token scenarios** for different permission levels
- **Test both success and failure paths** comprehensively
- **Include manual testing examples** with curl commands

---

**Summary**: Historical authentication middleware test experiment archived for reference when implementing generated authentication testing templates.
