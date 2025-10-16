package auth

import (
	"context"
	"fmt"
	"testing"
	"time"
)

// Mock AuthzClient for testing
type mockAuthzClient struct {
	callCount   int
	permissions map[string]bool
	shouldError bool
}

func (m *mockAuthzClient) CheckPermission(ctx context.Context, userID, permission, resource string) (bool, error) {
	m.callCount++
	if m.shouldError {
		return false, fmt.Errorf("mock error")
	}

	key := fmt.Sprintf("%s:%s:%s", userID, permission, resource)
	allowed, exists := m.permissions[key]
	if !exists {
		return false, nil
	}
	return allowed, nil
}

func TestAuthzHelper_CheckPermission(t *testing.T) {
	tests := []struct {
		name       string
		userID     string
		permission string
		resource   string
		allowed    bool
		wantError  bool
	}{
		{
			name:       "user has permission",
			userID:     "user1",
			permission: "read",
			resource:   "/api/todos",
			allowed:    true,
			wantError:  false,
		},
		{
			name:       "user does not have permission",
			userID:     "user1",
			permission: "write",
			resource:   "/api/todos",
			allowed:    false,
			wantError:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockClient := &mockAuthzClient{
				permissions: map[string]bool{
					"user1:read:/api/todos": true,
				},
			}

			helper := NewAuthzHelper(mockClient, 5*time.Minute)
			ctx := context.Background()

			got, err := helper.CheckPermission(ctx, tt.userID, tt.permission, tt.resource)
			if (err != nil) != tt.wantError {
				t.Errorf("CheckPermission() error = %v, wantError %v", err, tt.wantError)
				return
			}
			if got != tt.allowed {
				t.Errorf("CheckPermission() = %v, want %v", got, tt.allowed)
			}
		})
	}
}

func TestAuthzHelper_Caching(t *testing.T) {
	mockClient := &mockAuthzClient{
		permissions: map[string]bool{
			"user1:read:/api/todos": true,
		},
	}

	helper := NewAuthzHelper(mockClient, 5*time.Minute)
	ctx := context.Background()

	// First call should hit the service
	_, err := helper.CheckPermission(ctx, "user1", "read", "/api/todos")
	if err != nil {
		t.Errorf("CheckPermission() error = %v", err)
	}
	if mockClient.callCount != 1 {
		t.Errorf("Expected 1 service call, got %d", mockClient.callCount)
	}

	// Second call should use cache
	_, err = helper.CheckPermission(ctx, "user1", "read", "/api/todos")
	if err != nil {
		t.Errorf("CheckPermission() error = %v", err)
	}
	if mockClient.callCount != 1 {
		t.Errorf("Expected 1 service call (cached), got %d", mockClient.callCount)
	}
}

func TestAuthzHelper_CacheExpiration(t *testing.T) {
	mockClient := &mockAuthzClient{
		permissions: map[string]bool{
			"user1:read:/api/todos": true,
		},
	}

	helper := NewAuthzHelper(mockClient, 1*time.Millisecond) // Very short TTL
	ctx := context.Background()

	// First call
	_, err := helper.CheckPermission(ctx, "user1", "read", "/api/todos")
	if err != nil {
		t.Errorf("CheckPermission() error = %v", err)
	}
	if mockClient.callCount != 1 {
		t.Errorf("Expected 1 service call, got %d", mockClient.callCount)
	}

	// Wait for cache to expire
	time.Sleep(2 * time.Millisecond)

	// Second call should hit service again
	_, err = helper.CheckPermission(ctx, "user1", "read", "/api/todos")
	if err != nil {
		t.Errorf("CheckPermission() error = %v", err)
	}
	if mockClient.callCount != 2 {
		t.Errorf("Expected 2 service calls (cache expired), got %d", mockClient.callCount)
	}
}

func TestAuthzHelper_CheckMultiplePermissions(t *testing.T) {
	mockClient := &mockAuthzClient{
		permissions: map[string]bool{
			"user1:read:/api/todos":   true,
			"user1:write:/api/todos":  false,
			"user1:delete:/api/todos": true,
		},
	}

	helper := NewAuthzHelper(mockClient, 5*time.Minute)
	ctx := context.Background()

	checks := []PermissionCheck{
		{Permission: "read", Resource: "/api/todos"},
		{Permission: "write", Resource: "/api/todos"},
		{Permission: "delete", Resource: "/api/todos"},
	}

	results, err := helper.CheckMultiplePermissions(ctx, "user1", checks)
	if err != nil {
		t.Errorf("CheckMultiplePermissions() error = %v", err)
		return
	}

	expected := map[string]bool{
		"read:/api/todos":   true,
		"write:/api/todos":  false,
		"delete:/api/todos": true,
	}

	for key, expectedValue := range expected {
		if got, exists := results[key]; !exists || got != expectedValue {
			t.Errorf("CheckMultiplePermissions() result[%s] = %v, want %v", key, got, expectedValue)
		}
	}
}

func TestAuthzHelper_ClearUserCache(t *testing.T) {
	mockClient := &mockAuthzClient{
		permissions: map[string]bool{
			"user1:read:/api/todos": true,
			"user2:read:/api/todos": true,
		},
	}

	helper := NewAuthzHelper(mockClient, 5*time.Minute)
	ctx := context.Background()

	// Cache permissions for both users
	_, _ = helper.CheckPermission(ctx, "user1", "read", "/api/todos")
	_, _ = helper.CheckPermission(ctx, "user2", "read", "/api/todos")

	if mockClient.callCount != 2 {
		t.Errorf("Expected 2 service calls, got %d", mockClient.callCount)
	}

	// Clear cache for user1
	helper.ClearUserCache("user1")

	// user1 should hit service again, user2 should use cache
	_, _ = helper.CheckPermission(ctx, "user1", "read", "/api/todos")
	_, _ = helper.CheckPermission(ctx, "user2", "read", "/api/todos")

	if mockClient.callCount != 3 {
		t.Errorf("Expected 3 service calls (user1 cache cleared), got %d", mockClient.callCount)
	}
}

func TestHasAnyPermission(t *testing.T) {
	mockClient := &mockAuthzClient{
		permissions: map[string]bool{
			"user1:read:/api/todos":  true,
			"user1:write:/api/todos": false,
		},
	}

	helper := NewAuthzHelper(mockClient, 5*time.Minute)
	ctx := context.Background()

	// Should return true because user has "read" permission
	permissions := []string{"write", "read"}
	allowed, err := HasAnyPermission(ctx, helper, "user1", permissions, "/api/todos")
	if err != nil {
		t.Errorf("HasAnyPermission() error = %v", err)
	}
	if !allowed {
		t.Errorf("HasAnyPermission() = %v, want true", allowed)
	}
}

func TestHasAllPermissions(t *testing.T) {
	mockClient := &mockAuthzClient{
		permissions: map[string]bool{
			"user1:read:/api/todos":  true,
			"user1:write:/api/todos": false,
		},
	}

	helper := NewAuthzHelper(mockClient, 5*time.Minute)
	ctx := context.Background()

	// Should return false because user doesn't have "write" permission
	permissions := []string{"read", "write"}
	allowed, err := HasAllPermissions(ctx, helper, "user1", permissions, "/api/todos")
	if err != nil {
		t.Errorf("HasAllPermissions() error = %v", err)
	}
	if allowed {
		t.Errorf("HasAllPermissions() = %v, want false", allowed)
	}
}
