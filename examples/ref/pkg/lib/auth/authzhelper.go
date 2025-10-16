package auth

import (
	"context"
	"fmt"
	"sync"
	"time"
)

// AuthzClient interface for authorization service calls
type AuthzClient interface {
	CheckPermission(ctx context.Context, userID, permission, resource string) (bool, error)
}

// PermissionCache manages cached permission results
type PermissionCache struct {
	permissions map[string]CachedPermission
	mutex       sync.RWMutex
	defaultTTL  time.Duration
}

// CachedPermission represents a cached permission check result
type CachedPermission struct {
	Allowed   bool
	ExpiresAt time.Time
}

// NewPermissionCache creates a new permission cache with default TTL
func NewPermissionCache(defaultTTL time.Duration) *PermissionCache {
	return &PermissionCache{
		permissions: make(map[string]CachedPermission),
		defaultTTL:  defaultTTL,
	}
}

// AuthzHelper provides transparent caching for authorization checks
type AuthzHelper struct {
	client AuthzClient
	cache  *PermissionCache
}

// NewAuthzHelper creates a new authorization helper with caching
func NewAuthzHelper(client AuthzClient, cacheTTL time.Duration) *AuthzHelper {
	return &AuthzHelper{
		client: client,
		cache:  NewPermissionCache(cacheTTL),
	}
}

// CheckPermission checks if user has permission with transparent caching
// This is the main function used throughout the application
func (h *AuthzHelper) CheckPermission(ctx context.Context, userID, permission, resource string) (bool, error) {
	// Try cache first
	if allowed, found := h.getCachedPermission(userID, permission, resource); found {
		return allowed, nil
	}

	// Cache miss - call AuthZ service
	allowed, err := h.client.CheckPermission(ctx, userID, permission, resource)
	if err != nil {
		return false, err
	}

	// Cache the result
	h.setCachedPermission(userID, permission, resource, allowed)

	return allowed, nil
}

// CheckMultiplePermissions checks multiple permissions efficiently
// Returns map of permission results - useful for UI rendering
func (h *AuthzHelper) CheckMultiplePermissions(ctx context.Context, userID string, checks []PermissionCheck) (map[string]bool, error) {
	results := make(map[string]bool)

	for _, check := range checks {
		allowed, err := h.CheckPermission(ctx, userID, check.Permission, check.Resource)
		if err != nil {
			return nil, fmt.Errorf("error check %s:%s: %w", check.Permission, check.Resource, err)
		}

		key := fmt.Sprintf("%s:%s", check.Permission, check.Resource)
		results[key] = allowed
	}

	return results, nil
}

// PermissionCheck represents a permission to check
type PermissionCheck struct {
	Permission string
	Resource   string
}

// getCachedPermission retrieves cached permission if not expired
func (h *AuthzHelper) getCachedPermission(userID, permission, resource string) (bool, bool) {
	key := h.cacheKey(userID, permission, resource)

	h.cache.mutex.RLock()
	defer h.cache.mutex.RUnlock()

	cached, exists := h.cache.permissions[key]
	if !exists {
		return false, false
	}

	// Check if expired
	if time.Now().After(cached.ExpiresAt) {
		return false, false
	}

	return cached.Allowed, true
}

// setCachedPermission caches a permission result
func (h *AuthzHelper) setCachedPermission(userID, permission, resource string, allowed bool) {
	key := h.cacheKey(userID, permission, resource)

	h.cache.mutex.Lock()
	defer h.cache.mutex.Unlock()

	h.cache.permissions[key] = CachedPermission{
		Allowed:   allowed,
		ExpiresAt: time.Now().Add(h.cache.defaultTTL),
	}
}

// cacheKey generates a unique cache key for a permission check
func (h *AuthzHelper) cacheKey(userID, permission, resource string) string {
	return fmt.Sprintf("%s:%s:%s", userID, permission, resource)
}

// ClearUserCache removes all cached permissions for a specific user
// Useful when user permissions change
func (h *AuthzHelper) ClearUserCache(userID string) {
	h.cache.mutex.Lock()
	defer h.cache.mutex.Unlock()

	prefix := userID + ":"
	for key := range h.cache.permissions {
		if len(key) > len(prefix) && key[:len(prefix)] == prefix {
			delete(h.cache.permissions, key)
		}
	}
}

// ClearExpiredCache removes expired entries from cache
// Should be called periodically to prevent memory leaks
func (h *AuthzHelper) ClearExpiredCache() {
	h.cache.mutex.Lock()
	defer h.cache.mutex.Unlock()

	now := time.Now()
	for key, cached := range h.cache.permissions {
		if now.After(cached.ExpiresAt) {
			delete(h.cache.permissions, key)
		}
	}
}

// Pure helper functions for common permission patterns

// HasAnyPermission checks if user has any of the specified permissions (OR logic)
func HasAnyPermission(ctx context.Context, helper *AuthzHelper, userID string, permissions []string, resource string) (bool, error) {
	for _, permission := range permissions {
		allowed, err := helper.CheckPermission(ctx, userID, permission, resource)
		if err != nil {
			return false, err
		}
		if allowed {
			return true, nil
		}
	}
	return false, nil
}

// HasAllPermissions checks if user has all specified permissions (AND logic)
func HasAllPermissions(ctx context.Context, helper *AuthzHelper, userID string, permissions []string, resource string) (bool, error) {
	for _, permission := range permissions {
		allowed, err := helper.CheckPermission(ctx, userID, permission, resource)
		if err != nil {
			return false, err
		}
		if !allowed {
			return false, nil
		}
	}
	return true, nil
}

// IsResourceOwner checks if user owns/created the resource
func IsResourceOwner(ctx context.Context, helper *AuthzHelper, userID, resourceID string) (bool, error) {
	return helper.CheckPermission(ctx, userID, "own", resourceID)
}
