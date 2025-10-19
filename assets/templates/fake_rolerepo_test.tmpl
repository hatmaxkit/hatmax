package fake

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
)

func TestRoleRepoCreate(t *testing.T) {
	repo := NewRoleRepo()
	ctx := context.Background()

	role := &Role{
		Name:        "admin",
		Permissions: []string{"users:read", "users:write"},
		Status:      "active",
	}

	err := repo.Create(ctx, role)
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}

	// Check ID was generated
	if role.ID == uuid.Nil {
		t.Error("Expected ID to be generated")
	}

	// Check call was tracked
	if repo.CallCount("Create") != 1 {
		t.Errorf("Expected 1 Create call, got %d", repo.CallCount("Create"))
	}

	// Check role was stored
	stored := repo.GetRoleByID(role.ID)
	if stored == nil {
		t.Error("Expected role to be stored")
	}
	if stored.Name != role.Name {
		t.Errorf("Expected name %s, got %s", role.Name, stored.Name)
	}
}

func TestRoleRepoCreateError(t *testing.T) {
	repo := NewRoleRepo()
	ctx := context.Background()
	expectedError := errors.New("create failed")

	// Program error response
	repo.CreateError = expectedError

	role := &Role{Name: "admin"}
	err := repo.Create(ctx, role)

	if err != expectedError {
		t.Errorf("Expected error %v, got %v", expectedError, err)
	}

	// Check call was still tracked
	if repo.CallCount("Create") != 1 {
		t.Errorf("Expected 1 Create call, got %d", repo.CallCount("Create"))
	}
}

func TestRoleRepoGet(t *testing.T) {
	repo := NewRoleRepo()
	ctx := context.Background()

	// Store a role directly
	id := uuid.New()
	role := &Role{
		ID:     id,
		Name:   "admin",
		Status: "active",
	}
	repo.SetRole(role)

	// Test getting existing role
	retrieved, err := repo.Get(ctx, id)
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}
	if retrieved == nil {
		t.Error("Expected role to be retrieved")
	}
	if retrieved.ID != id {
		t.Errorf("Expected ID %v, got %v", id, retrieved.ID)
	}

	// Test getting non-existent role
	nonExistentID := uuid.New()
	retrieved, err = repo.Get(ctx, nonExistentID)
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}
	if retrieved != nil {
		t.Error("Expected nil for non-existent role")
	}

	// Check calls were tracked
	if repo.CallCount("Get") != 2 {
		t.Errorf("Expected 2 Get calls, got %d", repo.CallCount("Get"))
	}
}

func TestRoleRepoGetByName(t *testing.T) {
	repo := NewRoleRepo()
	ctx := context.Background()

	// Store roles directly
	roles := []*Role{
		{ID: uuid.New(), Name: "admin", Status: "active"},
		{ID: uuid.New(), Name: "user", Status: "active"},
		{ID: uuid.New(), Name: "guest", Status: "suspended"},
	}

	for _, role := range roles {
		repo.SetRole(role)
	}

	// Test getting existing role by name
	retrieved, err := repo.GetByName(ctx, "admin")
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}
	if retrieved == nil {
		t.Error("Expected role to be retrieved")
	}
	if retrieved.Name != "admin" {
		t.Errorf("Expected name 'admin', got %s", retrieved.Name)
	}

	// Test getting non-existent role
	retrieved, err = repo.GetByName(ctx, "nonexistent")
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}
	if retrieved != nil {
		t.Error("Expected nil for non-existent role")
	}

	// Check calls were tracked
	if repo.CallCount("GetByName") != 2 {
		t.Errorf("Expected 2 GetByName calls, got %d", repo.CallCount("GetByName"))
	}
}

func TestRoleRepoGetByNameProgrammedResponse(t *testing.T) {
	repo := NewRoleRepo()
	ctx := context.Background()

	// Program a response that overrides storage
	programmedRole := &Role{
		ID:     uuid.New(),
		Name:   "programmed",
		Status: "programmed",
	}
	repo.GetByNameResponse = programmedRole

	// Try to get any name - should return programmed response
	retrieved, err := repo.GetByName(ctx, "any-name")
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}
	if retrieved == nil {
		t.Error("Expected programmed role to be returned")
	}
	if retrieved.Status != "programmed" {
		t.Errorf("Expected programmed status, got %v", retrieved.Status)
	}
}

func TestRoleRepoSave(t *testing.T) {
	repo := NewRoleRepo()
	ctx := context.Background()

	role := &Role{
		ID:          uuid.New(),
		Name:        "admin",
		Permissions: []string{"users:read"},
		Status:      "active",
	}

	err := repo.Save(ctx, role)
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}

	// Check role was stored
	stored := repo.GetRoleByID(role.ID)
	if stored == nil {
		t.Error("Expected role to be stored")
	}
	if len(stored.Permissions) != 1 || stored.Permissions[0] != "users:read" {
		t.Errorf("Expected permissions [users:read], got %v", stored.Permissions)
	}

	// Check call was tracked
	if repo.CallCount("Save") != 1 {
		t.Errorf("Expected 1 Save call, got %d", repo.CallCount("Save"))
	}
}

func TestRoleRepoDelete(t *testing.T) {
	repo := NewRoleRepo()
	ctx := context.Background()

	// Store a role first
	id := uuid.New()
	role := &Role{ID: id, Name: "admin"}
	repo.SetRole(role)

	// Verify it exists
	if repo.GetRoleByID(id) == nil {
		t.Error("Role should exist before deletion")
	}

	// Delete it
	err := repo.Delete(ctx, id)
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}

	// Verify it's gone
	if repo.GetRoleByID(id) != nil {
		t.Error("Role should not exist after deletion")
	}

	// Check call was tracked
	if repo.CallCount("Delete") != 1 {
		t.Errorf("Expected 1 Delete call, got %d", repo.CallCount("Delete"))
	}
}

func TestRoleRepoList(t *testing.T) {
	repo := NewRoleRepo()
	ctx := context.Background()

	// Store some roles
	roles := []*Role{
		{ID: uuid.New(), Name: "admin", Status: "active"},
		{ID: uuid.New(), Name: "user", Status: "active"},
		{ID: uuid.New(), Name: "guest", Status: "suspended"},
	}

	for _, role := range roles {
		repo.SetRole(role)
	}

	// List all roles
	allRoles, err := repo.List(ctx)
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}
	if len(allRoles) != 3 {
		t.Errorf("Expected 3 roles, got %d", len(allRoles))
	}

	// Check call was tracked
	if repo.CallCount("List") != 1 {
		t.Errorf("Expected 1 List call, got %d", repo.CallCount("List"))
	}
}

func TestRoleRepoListByStatus(t *testing.T) {
	repo := NewRoleRepo()
	ctx := context.Background()

	// Store roles with different statuses
	roles := []*Role{
		{ID: uuid.New(), Name: "admin1", Status: "active"},
		{ID: uuid.New(), Name: "admin2", Status: "active"},
		{ID: uuid.New(), Name: "guest1", Status: "suspended"},
		{ID: uuid.New(), Name: "guest2", Status: "suspended"},
		{ID: uuid.New(), Name: "deleted1", Status: "deleted"},
	}

	for _, role := range roles {
		repo.SetRole(role)
	}

	// List active roles
	activeRoles, err := repo.ListByStatus(ctx, "active")
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}
	if len(activeRoles) != 2 {
		t.Errorf("Expected 2 active roles, got %d", len(activeRoles))
	}

	// List suspended roles
	suspendedRoles, err := repo.ListByStatus(ctx, "suspended")
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}
	if len(suspendedRoles) != 2 {
		t.Errorf("Expected 2 suspended roles, got %d", len(suspendedRoles))
	}

	// List deleted roles
	deletedRoles, err := repo.ListByStatus(ctx, "deleted")
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}
	if len(deletedRoles) != 1 {
		t.Errorf("Expected 1 deleted role, got %d", len(deletedRoles))
	}

	// List non-existent status
	nonExistentRoles, err := repo.ListByStatus(ctx, "nonexistent")
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}
	if len(nonExistentRoles) != 0 {
		t.Errorf("Expected 0 roles for non-existent status, got %d", len(nonExistentRoles))
	}

	// Check calls were tracked
	if repo.CallCount("ListByStatus") != 4 {
		t.Errorf("Expected 4 ListByStatus calls, got %d", repo.CallCount("ListByStatus"))
	}
}

func TestRoleRepoReset(t *testing.T) {
	repo := NewRoleRepo()
	ctx := context.Background()

	// Add some data and errors
	role := &Role{ID: uuid.New(), Name: "admin"}
	repo.SetRole(role)
	repo.CreateError = errors.New("test error")
	
	// Make some calls
	repo.Create(ctx, &Role{Name: "test"})
	repo.Get(ctx, uuid.New())

	// Verify data exists
	if len(repo.roles) == 0 {
		t.Error("Expected roles to exist before reset")
	}
	if repo.CallCount("Create") == 0 {
		t.Error("Expected calls to exist before reset")
	}

	// Reset
	repo.Reset()

	// Verify everything is cleared
	if len(repo.roles) != 0 {
		t.Error("Expected roles to be cleared after reset")
	}
	if repo.CreateError != nil {
		t.Error("Expected CreateError to be cleared after reset")
	}
	if repo.CallCount("Create") != 0 {
		t.Error("Expected call history to be cleared after reset")
	}
}

func TestRoleRepoConcurrentAccess(t *testing.T) {
	repo := NewRoleRepo()
	ctx := context.Background()

	const numGoroutines = 10
	const numOperationsPerGoroutine = 100

	var wg sync.WaitGroup

	// Test concurrent creates
	wg.Add(numGoroutines)
	for i := 0; i < numGoroutines; i++ {
		go func(goroutineID int) {
			defer wg.Done()
			for j := 0; j < numOperationsPerGoroutine; j++ {
				role := &Role{
					Name:   fmt.Sprintf("role-%d-%d", goroutineID, j),
					Status: "active",
				}
				repo.Create(ctx, role)
			}
		}(i)
	}

	wg.Wait()

	// Verify all creates were tracked
	expectedCalls := numGoroutines * numOperationsPerGoroutine
	if repo.CallCount("Create") != expectedCalls {
		t.Errorf("Expected %d Create calls, got %d", expectedCalls, repo.CallCount("Create"))
	}

	// Test concurrent reads don't panic
	wg.Add(numGoroutines)
	for i := 0; i < numGoroutines; i++ {
		go func() {
			defer wg.Done()
			for j := 0; j < numOperationsPerGoroutine; j++ {
				repo.List(ctx)
				repo.ListByStatus(ctx, "active")
				repo.GetByName(ctx, "admin")
			}
		}()
	}

	wg.Wait()
}

func TestRoleRepoCallTracking(t *testing.T) {
	repo := NewRoleRepo()
	ctx := context.Background()

	// Make various calls
	role := &Role{ID: uuid.New(), Name: "admin"}
	
	repo.Create(ctx, role)
	repo.Create(ctx, role)
	repo.Get(ctx, role.ID)
	repo.GetByName(ctx, role.Name)
	repo.Save(ctx, role)
	repo.Delete(ctx, role.ID)
	repo.List(ctx)
	repo.ListByStatus(ctx, "active")

	// Verify call counts
	tests := []struct {
		method   string
		expected int
	}{
		{"Create", 2},
		{"Get", 1},
		{"GetByName", 1},
		{"Save", 1},
		{"Delete", 1},
		{"List", 1},
		{"ListByStatus", 1},
	}

	for _, tt := range tests {
		if count := repo.CallCount(tt.method); count != tt.expected {
			t.Errorf("Expected %d %s calls, got %d", tt.expected, tt.method, count)
		}
	}

	// Test unknown method
	if count := repo.CallCount("UnknownMethod"); count != 0 {
		t.Errorf("Expected 0 for unknown method, got %d", count)
	}
}

func TestRoleRepoHelperMethods(t *testing.T) {
	repo := NewRoleRepo()

	role := &Role{
		ID:     uuid.New(),
		Name:   "admin",
		Status: "active",
	}

	// Test SetRole and GetRoleByID
	repo.SetRole(role)
	retrieved := repo.GetRoleByID(role.ID)
	if retrieved == nil {
		t.Error("Expected role to be retrieved")
	}
	if retrieved.Name != "admin" {
		t.Errorf("Expected name 'admin', got %s", retrieved.Name)
	}

	// Test GetRoleByNameDirect
	byName := repo.GetRoleByNameDirect("admin")
	if byName == nil {
		t.Error("Expected role to be retrieved by name")
	}
	if byName.ID != role.ID {
		t.Errorf("Expected ID %v, got %v", role.ID, byName.ID)
	}

	// Test non-existent
	nonExistent := repo.GetRoleByID(uuid.New())
	if nonExistent != nil {
		t.Error("Expected nil for non-existent role")
	}

	nonExistentByName := repo.GetRoleByNameDirect("nonexistent")
	if nonExistentByName != nil {
		t.Error("Expected nil for non-existent role by name")
	}
}