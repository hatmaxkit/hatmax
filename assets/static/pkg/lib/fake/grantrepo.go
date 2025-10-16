package fake

import (
	"context"
	"sync"
	"time"

	"github.com/google/uuid"
)

// GrantRepo fake implementation with programmable responses and call tracking
type GrantRepo struct {
	mu sync.RWMutex

	// Data storage
	grants map[uuid.UUID]*Grant

	// Programmable responses
	CreateError   error
	GetError      error
	SaveError     error
	DeleteError   error
	ListError     error
	ListByUserIDError error
	ListByScopeError  error
	ListExpiredError  error

	// Return values for gets
	GetResponse *Grant

	// Call tracking
	CreateCalls        []CreateGrantCall
	GetCalls           []GetGrantCall
	SaveCalls          []SaveGrantCall
	DeleteCalls        []DeleteGrantCall
	ListCalls          []ListGrantCall
	ListByUserIDCalls  []ListGrantByUserIDCall
	ListByScopeCalls   []ListGrantByScopeCall
	ListExpiredCalls   []ListGrantExpiredCall
}

// Call structures for tracking
type CreateGrantCall struct {
	Ctx   context.Context
	Grant *Grant
}

type GetGrantCall struct {
	Ctx context.Context
	ID  uuid.UUID
}

type SaveGrantCall struct {
	Ctx   context.Context
	Grant *Grant
}

type DeleteGrantCall struct {
	Ctx context.Context
	ID  uuid.UUID
}

type ListGrantCall struct {
	Ctx context.Context
}

type ListGrantByUserIDCall struct {
	Ctx    context.Context
	UserID uuid.UUID
}

type ListGrantByScopeCall struct {
	Ctx   context.Context
	Scope Scope
}

type ListGrantExpiredCall struct {
	Ctx context.Context
}

// Grant represents a grant entity for testing
type Grant struct {
	ID        uuid.UUID
	UserID    uuid.UUID
	GrantType GrantType
	Value     string
	Scope     Scope
	ExpiresAt *time.Time
	Status    UserStatus
	CreatedAt time.Time
	UpdatedAt time.Time
}

type GrantType string
type UserStatus string
type Scope struct {
	Type string
	ID   string
}

// NewGrantRepo creates a new fake GrantRepo
func NewGrantRepo() *GrantRepo {
	return &GrantRepo{
		grants: make(map[uuid.UUID]*Grant),
	}
}

// Reset clears all data and call history
func (f *GrantRepo) Reset() {
	f.mu.Lock()
	defer f.mu.Unlock()
	
	f.grants = make(map[uuid.UUID]*Grant)
	f.CreateError = nil
	f.GetError = nil
	f.SaveError = nil
	f.DeleteError = nil
	f.ListError = nil
	f.ListByUserIDError = nil
	f.ListByScopeError = nil
	f.ListExpiredError = nil
	f.GetResponse = nil
	f.CreateCalls = nil
	f.GetCalls = nil
	f.SaveCalls = nil
	f.DeleteCalls = nil
	f.ListCalls = nil
	f.ListByUserIDCalls = nil
	f.ListByScopeCalls = nil
	f.ListExpiredCalls = nil
}

// Create implements GrantRepo interface
func (f *GrantRepo) Create(ctx context.Context, grant *Grant) error {
	f.mu.Lock()
	defer f.mu.Unlock()

	// Track call
	f.CreateCalls = append(f.CreateCalls, CreateGrantCall{
		Ctx:   ctx,
		Grant: grant,
	})

	// Return programmed error if set
	if f.CreateError != nil {
		return f.CreateError
	}

	// Simulate real behavior
	if grant.ID == uuid.Nil {
		grant.ID = uuid.New()
	}
	
	// Store grant (make a copy)
	grantCopy := *grant
	f.grants[grant.ID] = &grantCopy

	return nil
}

// Get implements GrantRepo interface
func (f *GrantRepo) Get(ctx context.Context, id uuid.UUID) (*Grant, error) {
	f.mu.RLock()
	defer f.mu.RUnlock()

	// Track call
	f.GetCalls = append(f.GetCalls, GetGrantCall{
		Ctx: ctx,
		ID:  id,
	})

	// Return programmed error if set
	if f.GetError != nil {
		return nil, f.GetError
	}

	// Return programmed response if set (overrides storage)
	if f.GetResponse != nil {
		responseCopy := *f.GetResponse
		return &responseCopy, nil
	}

	// Return from storage
	grant, exists := f.grants[id]
	if !exists {
		return nil, nil
	}

	// Return copy to avoid race conditions
	grantCopy := *grant
	return &grantCopy, nil
}

// Save implements GrantRepo interface
func (f *GrantRepo) Save(ctx context.Context, grant *Grant) error {
	f.mu.Lock()
	defer f.mu.Unlock()

	// Track call
	f.SaveCalls = append(f.SaveCalls, SaveGrantCall{
		Ctx:   ctx,
		Grant: grant,
	})

	// Return programmed error if set
	if f.SaveError != nil {
		return f.SaveError
	}

	// Store grant (make a copy)
	grantCopy := *grant
	f.grants[grant.ID] = &grantCopy

	return nil
}

// Delete implements GrantRepo interface
func (f *GrantRepo) Delete(ctx context.Context, id uuid.UUID) error {
	f.mu.Lock()
	defer f.mu.Unlock()

	// Track call
	f.DeleteCalls = append(f.DeleteCalls, DeleteGrantCall{
		Ctx: ctx,
		ID:  id,
	})

	// Return programmed error if set
	if f.DeleteError != nil {
		return f.DeleteError
	}

	// Delete from storage
	delete(f.grants, id)

	return nil
}

// List implements GrantRepo interface
func (f *GrantRepo) List(ctx context.Context) ([]*Grant, error) {
	f.mu.RLock()
	defer f.mu.RUnlock()

	// Track call
	f.ListCalls = append(f.ListCalls, ListGrantCall{Ctx: ctx})

	// Return programmed error if set
	if f.ListError != nil {
		return nil, f.ListError
	}

	// Return all grants as copies
	var grants []*Grant
	for _, grant := range f.grants {
		grantCopy := *grant
		grants = append(grants, &grantCopy)
	}

	return grants, nil
}

// ListByUserID implements GrantRepo interface
func (f *GrantRepo) ListByUserID(ctx context.Context, userID uuid.UUID) ([]*Grant, error) {
	f.mu.RLock()
	defer f.mu.RUnlock()

	// Track call
	f.ListByUserIDCalls = append(f.ListByUserIDCalls, ListGrantByUserIDCall{
		Ctx:    ctx,
		UserID: userID,
	})

	// Return programmed error if set
	if f.ListByUserIDError != nil {
		return nil, f.ListByUserIDError
	}

	// Filter grants by user ID
	var grants []*Grant
	for _, grant := range f.grants {
		if grant.UserID == userID {
			grantCopy := *grant
			grants = append(grants, &grantCopy)
		}
	}

	return grants, nil
}

// ListByScope implements GrantRepo interface
func (f *GrantRepo) ListByScope(ctx context.Context, scope Scope) ([]*Grant, error) {
	f.mu.RLock()
	defer f.mu.RUnlock()

	// Track call
	f.ListByScopeCalls = append(f.ListByScopeCalls, ListGrantByScopeCall{
		Ctx:   ctx,
		Scope: scope,
	})

	// Return programmed error if set
	if f.ListByScopeError != nil {
		return nil, f.ListByScopeError
	}

	// Filter grants by scope (simple match for now)
	var grants []*Grant
	for _, grant := range f.grants {
		if grant.Scope.Type == scope.Type && grant.Scope.ID == scope.ID {
			grantCopy := *grant
			grants = append(grants, &grantCopy)
		}
	}

	return grants, nil
}

// ListExpired implements GrantRepo interface
func (f *GrantRepo) ListExpired(ctx context.Context) ([]*Grant, error) {
	f.mu.RLock()
	defer f.mu.RUnlock()

	// Track call
	f.ListExpiredCalls = append(f.ListExpiredCalls, ListGrantExpiredCall{Ctx: ctx})

	// Return programmed error if set
	if f.ListExpiredError != nil {
		return nil, f.ListExpiredError
	}

	// Filter expired grants
	var grants []*Grant
	now := time.Now()
	for _, grant := range f.grants {
		if grant.ExpiresAt != nil && grant.ExpiresAt.Before(now) {
			grantCopy := *grant
			grants = append(grants, &grantCopy)
		}
	}

	return grants, nil
}

// Helper methods for testing

// SetGrant stores a grant for testing (thread-safe)
func (f *GrantRepo) SetGrant(grant *Grant) {
	f.mu.Lock()
	defer f.mu.Unlock()
	grantCopy := *grant
	f.grants[grant.ID] = &grantCopy
}

// GetGrant retrieves a grant for testing (thread-safe)
func (f *GrantRepo) GetGrantByID(id uuid.UUID) *Grant {
	f.mu.RLock()
	defer f.mu.RUnlock()
	grant, exists := f.grants[id]
	if !exists {
		return nil
	}
	grantCopy := *grant
	return &grantCopy
}

// CallCount returns the number of calls to a specific method
func (f *GrantRepo) CallCount(method string) int {
	f.mu.RLock()
	defer f.mu.RUnlock()
	
	switch method {
	case "Create":
		return len(f.CreateCalls)
	case "Get":
		return len(f.GetCalls)
	case "Save":
		return len(f.SaveCalls)
	case "Delete":
		return len(f.DeleteCalls)
	case "List":
		return len(f.ListCalls)
	case "ListByUserID":
		return len(f.ListByUserIDCalls)
	case "ListByScope":
		return len(f.ListByScopeCalls)
	case "ListExpired":
		return len(f.ListExpiredCalls)
	default:
		return 0
	}
}