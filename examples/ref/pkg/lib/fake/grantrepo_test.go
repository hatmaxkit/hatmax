package fake

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
)

func TestGrantRepoCreate(t *testing.T) {
	repo := NewGrantRepo()
	ctx := context.Background()

	grant := &Grant{
		UserID:    uuid.New(),
		GrantType: "role",
		Value:     "admin",
		Scope:     Scope{Type: "global", ID: ""},
		Status:    "active",
	}

	err := repo.Create(ctx, grant)
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}

	// Check ID was generated
	if grant.ID == uuid.Nil {
		t.Error("Expected ID to be generated")
	}

	// Check call was tracked
	if repo.CallCount("Create") != 1 {
		t.Errorf("Expected 1 Create call, got %d", repo.CallCount("Create"))
	}

	// Check grant was stored
	stored := repo.GetGrantByID(grant.ID)
	if stored == nil {
		t.Error("Expected grant to be stored")
	}
	if stored.UserID != grant.UserID {
		t.Errorf("Expected UserID %v, got %v", grant.UserID, stored.UserID)
	}
}

func TestGrantRepoCreateError(t *testing.T) {
	repo := NewGrantRepo()
	ctx := context.Background()
	expectedError := errors.New("create failed")

	// Program error response
	repo.CreateError = expectedError

	grant := &Grant{UserID: uuid.New()}
	err := repo.Create(ctx, grant)

	if err != expectedError {
		t.Errorf("Expected error %v, got %v", expectedError, err)
	}

	// Check call was still tracked
	if repo.CallCount("Create") != 1 {
		t.Errorf("Expected 1 Create call, got %d", repo.CallCount("Create"))
	}
}

func TestGrantRepoGet(t *testing.T) {
	repo := NewGrantRepo()
	ctx := context.Background()

	// Store a grant directly
	id := uuid.New()
	grant := &Grant{
		ID:     id,
		UserID: uuid.New(),
		Status: "active",
	}
	repo.SetGrant(grant)

	// Test getting existing grant
	retrieved, err := repo.Get(ctx, id)
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}
	if retrieved == nil {
		t.Error("Expected grant to be retrieved")
	}
	if retrieved.ID != id {
		t.Errorf("Expected ID %v, got %v", id, retrieved.ID)
	}

	// Test getting non-existent grant
	nonExistentID := uuid.New()
	retrieved, err = repo.Get(ctx, nonExistentID)
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}
	if retrieved != nil {
		t.Error("Expected nil for non-existent grant")
	}

	// Check calls were tracked
	if repo.CallCount("Get") != 2 {
		t.Errorf("Expected 2 Get calls, got %d", repo.CallCount("Get"))
	}
}

func TestGrantRepoGetProgrammedResponse(t *testing.T) {
	repo := NewGrantRepo()
	ctx := context.Background()

	// Program a response that overrides storage
	programmedGrant := &Grant{
		ID:     uuid.New(),
		UserID: uuid.New(),
		Status: "programmed",
	}
	repo.GetResponse = programmedGrant

	// Try to get any ID - should return programmed response
	retrieved, err := repo.Get(ctx, uuid.New())
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}
	if retrieved == nil {
		t.Error("Expected programmed grant to be returned")
	}
	if retrieved.Status != "programmed" {
		t.Errorf("Expected programmed status, got %v", retrieved.Status)
	}
}

func TestGrantRepoListByUserID(t *testing.T) {
	repo := NewGrantRepo()
	ctx := context.Background()

	userID1 := uuid.New()
	userID2 := uuid.New()

	// Store grants for different users
	grants := []*Grant{
		{ID: uuid.New(), UserID: userID1, Status: "active"},
		{ID: uuid.New(), UserID: userID1, Status: "active"},
		{ID: uuid.New(), UserID: userID2, Status: "active"},
	}

	for _, grant := range grants {
		repo.SetGrant(grant)
	}

	// Get grants for user1
	userGrants, err := repo.ListByUserID(ctx, userID1)
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}
	if len(userGrants) != 2 {
		t.Errorf("Expected 2 grants for user1, got %d", len(userGrants))
	}

	// Get grants for user2
	userGrants, err = repo.ListByUserID(ctx, userID2)
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}
	if len(userGrants) != 1 {
		t.Errorf("Expected 1 grant for user2, got %d", len(userGrants))
	}

	// Check calls were tracked
	if repo.CallCount("ListByUserID") != 2 {
		t.Errorf("Expected 2 ListByUserID calls, got %d", repo.CallCount("ListByUserID"))
	}
}

func TestGrantRepoListExpired(t *testing.T) {
	repo := NewGrantRepo()
	ctx := context.Background()

	now := time.Now()
	expiredTime := now.Add(-time.Hour)
	futureTime := now.Add(time.Hour)

	// Store grants with different expiration states
	grants := []*Grant{
		{ID: uuid.New(), UserID: uuid.New(), ExpiresAt: nil},          // No expiration
		{ID: uuid.New(), UserID: uuid.New(), ExpiresAt: &expiredTime}, // Expired
		{ID: uuid.New(), UserID: uuid.New(), ExpiresAt: &futureTime},  // Future
		{ID: uuid.New(), UserID: uuid.New(), ExpiresAt: &expiredTime}, // Another expired
	}

	for _, grant := range grants {
		repo.SetGrant(grant)
	}

	// Get expired grants
	expiredGrants, err := repo.ListExpired(ctx)
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}
	if len(expiredGrants) != 2 {
		t.Errorf("Expected 2 expired grants, got %d", len(expiredGrants))
	}
}

func TestGrantRepoReset(t *testing.T) {
	repo := NewGrantRepo()
	ctx := context.Background()

	// Add some data and errors
	grant := &Grant{ID: uuid.New(), UserID: uuid.New()}
	repo.SetGrant(grant)
	repo.CreateError = errors.New("test error")

	// Make some calls
	repo.Create(ctx, &Grant{UserID: uuid.New()})
	repo.Get(ctx, uuid.New())

	// Verify data exists
	if len(repo.grants) == 0 {
		t.Error("Expected grants to exist before reset")
	}
	if repo.CallCount("Create") == 0 {
		t.Error("Expected calls to exist before reset")
	}

	// Reset
	repo.Reset()

	// Verify everything is cleared
	if len(repo.grants) != 0 {
		t.Error("Expected grants to be cleared after reset")
	}
	if repo.CreateError != nil {
		t.Error("Expected CreateError to be cleared after reset")
	}
	if repo.CallCount("Create") != 0 {
		t.Error("Expected call history to be cleared after reset")
	}
}

func TestGrantRepoConcurrentAccess(t *testing.T) {
	repo := NewGrantRepo()
	ctx := context.Background()

	const numGoroutines = 10
	const numOperationsPerGoroutine = 100

	var wg sync.WaitGroup

	// Test concurrent creates
	wg.Add(numGoroutines)
	for i := 0; i < numGoroutines; i++ {
		go func() {
			defer wg.Done()
			for j := 0; j < numOperationsPerGoroutine; j++ {
				grant := &Grant{
					UserID: uuid.New(),
					Status: "active",
				}
				repo.Create(ctx, grant)
			}
		}()
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
				repo.ListByUserID(ctx, uuid.New())
			}
		}()
	}

	wg.Wait()
}

func TestGrantRepoCallTracking(t *testing.T) {
	repo := NewGrantRepo()
	ctx := context.Background()

	// Make various calls
	grant := &Grant{ID: uuid.New(), UserID: uuid.New()}

	repo.Create(ctx, grant)
	repo.Create(ctx, grant)
	repo.Get(ctx, grant.ID)
	repo.Save(ctx, grant)
	repo.Delete(ctx, grant.ID)
	repo.List(ctx)
	repo.ListByUserID(ctx, grant.UserID)
	repo.ListByScope(ctx, Scope{Type: "test"})
	repo.ListExpired(ctx)

	// Verify call counts
	tests := []struct {
		method   string
		expected int
	}{
		{"Create", 2},
		{"Get", 1},
		{"Save", 1},
		{"Delete", 1},
		{"List", 1},
		{"ListByUserID", 1},
		{"ListByScope", 1},
		{"ListExpired", 1},
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
