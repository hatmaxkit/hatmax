package fake

import (
	"context"
	"sync"
	"time"

	"github.com/google/uuid"
)

// RoleRepo fake implementation with programmable responses and call tracking
type RoleRepo struct {
	mu sync.RWMutex

	// Data storage
	roles map[uuid.UUID]*Role

	// Programmable responses
	CreateError      error
	GetError         error
	GetByNameError   error
	SaveError        error
	DeleteError      error
	ListError        error
	ListByStatusError error

	// Return values for gets
	GetResponse       *Role
	GetByNameResponse *Role

	// Call tracking
	CreateCalls       []CreateRoleCall
	GetCalls          []GetRoleCall
	GetByNameCalls    []GetRoleByNameCall
	SaveCalls         []SaveRoleCall
	DeleteCalls       []DeleteRoleCall
	ListCalls         []ListRoleCall
	ListByStatusCalls []ListRoleByStatusCall
}

// Call structures for tracking
type CreateRoleCall struct {
	Ctx  context.Context
	Role *Role
}

type GetRoleCall struct {
	Ctx context.Context
	ID  uuid.UUID
}

type GetRoleByNameCall struct {
	Ctx  context.Context
	Name string
}

type SaveRoleCall struct {
	Ctx  context.Context
	Role *Role
}

type DeleteRoleCall struct {
	Ctx context.Context
	ID  uuid.UUID
}

type ListRoleCall struct {
	Ctx context.Context
}

type ListRoleByStatusCall struct {
	Ctx    context.Context
	Status string
}

// Role represents a role entity for testing
type Role struct {
	ID          uuid.UUID
	Name        string
	Permissions []string
	Status      UserStatus
	CreatedAt   time.Time
	UpdatedAt   time.Time
	CreatedBy   string
	UpdatedBy   string
}

// NewRoleRepo creates a new fake RoleRepo
func NewRoleRepo() *RoleRepo {
	return &RoleRepo{
		roles: make(map[uuid.UUID]*Role),
	}
}

// Reset clears all data and call history
func (f *RoleRepo) Reset() {
	f.mu.Lock()
	defer f.mu.Unlock()
	
	f.roles = make(map[uuid.UUID]*Role)
	f.CreateError = nil
	f.GetError = nil
	f.GetByNameError = nil
	f.SaveError = nil
	f.DeleteError = nil
	f.ListError = nil
	f.ListByStatusError = nil
	f.GetResponse = nil
	f.GetByNameResponse = nil
	f.CreateCalls = nil
	f.GetCalls = nil
	f.GetByNameCalls = nil
	f.SaveCalls = nil
	f.DeleteCalls = nil
	f.ListCalls = nil
	f.ListByStatusCalls = nil
}

// Create implements RoleRepo interface
func (f *RoleRepo) Create(ctx context.Context, role *Role) error {
	f.mu.Lock()
	defer f.mu.Unlock()

	// Track call
	f.CreateCalls = append(f.CreateCalls, CreateRoleCall{
		Ctx:  ctx,
		Role: role,
	})

	// Return programmed error if set
	if f.CreateError != nil {
		return f.CreateError
	}

	// Simulate real behavior
	if role.ID == uuid.Nil {
		role.ID = uuid.New()
	}
	
	// Store role (make a copy)
	roleCopy := *role
	f.roles[role.ID] = &roleCopy

	return nil
}

// Get implements RoleRepo interface
func (f *RoleRepo) Get(ctx context.Context, id uuid.UUID) (*Role, error) {
	f.mu.RLock()
	defer f.mu.RUnlock()

	// Track call
	f.GetCalls = append(f.GetCalls, GetRoleCall{
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
	role, exists := f.roles[id]
	if !exists {
		return nil, nil
	}

	// Return copy to avoid race conditions
	roleCopy := *role
	return &roleCopy, nil
}

// GetByName implements RoleRepo interface
func (f *RoleRepo) GetByName(ctx context.Context, name string) (*Role, error) {
	f.mu.RLock()
	defer f.mu.RUnlock()

	// Track call
	f.GetByNameCalls = append(f.GetByNameCalls, GetRoleByNameCall{
		Ctx:  ctx,
		Name: name,
	})

	// Return programmed error if set
	if f.GetByNameError != nil {
		return nil, f.GetByNameError
	}

	// Return programmed response if set (overrides storage)
	if f.GetByNameResponse != nil {
		responseCopy := *f.GetByNameResponse
		return &responseCopy, nil
	}

	// Search by name in storage
	for _, role := range f.roles {
		if role.Name == name {
			roleCopy := *role
			return &roleCopy, nil
		}
	}

	return nil, nil
}

// Save implements RoleRepo interface
func (f *RoleRepo) Save(ctx context.Context, role *Role) error {
	f.mu.Lock()
	defer f.mu.Unlock()

	// Track call
	f.SaveCalls = append(f.SaveCalls, SaveRoleCall{
		Ctx:  ctx,
		Role: role,
	})

	// Return programmed error if set
	if f.SaveError != nil {
		return f.SaveError
	}

	// Store role (make a copy)
	roleCopy := *role
	f.roles[role.ID] = &roleCopy

	return nil
}

// Delete implements RoleRepo interface
func (f *RoleRepo) Delete(ctx context.Context, id uuid.UUID) error {
	f.mu.Lock()
	defer f.mu.Unlock()

	// Track call
	f.DeleteCalls = append(f.DeleteCalls, DeleteRoleCall{
		Ctx: ctx,
		ID:  id,
	})

	// Return programmed error if set
	if f.DeleteError != nil {
		return f.DeleteError
	}

	// Delete from storage
	delete(f.roles, id)

	return nil
}

// List implements RoleRepo interface
func (f *RoleRepo) List(ctx context.Context) ([]*Role, error) {
	f.mu.RLock()
	defer f.mu.RUnlock()

	// Track call
	f.ListCalls = append(f.ListCalls, ListRoleCall{Ctx: ctx})

	// Return programmed error if set
	if f.ListError != nil {
		return nil, f.ListError
	}

	// Return all roles as copies
	var roles []*Role
	for _, role := range f.roles {
		roleCopy := *role
		roles = append(roles, &roleCopy)
	}

	return roles, nil
}

// ListByStatus implements RoleRepo interface
func (f *RoleRepo) ListByStatus(ctx context.Context, status string) ([]*Role, error) {
	f.mu.RLock()
	defer f.mu.RUnlock()

	// Track call
	f.ListByStatusCalls = append(f.ListByStatusCalls, ListRoleByStatusCall{
		Ctx:    ctx,
		Status: status,
	})

	// Return programmed error if set
	if f.ListByStatusError != nil {
		return nil, f.ListByStatusError
	}

	// Filter roles by status
	var roles []*Role
	for _, role := range f.roles {
		if string(role.Status) == status {
			roleCopy := *role
			roles = append(roles, &roleCopy)
		}
	}

	return roles, nil
}

// Helper methods for testing

// SetRole stores a role for testing (thread-safe)
func (f *RoleRepo) SetRole(role *Role) {
	f.mu.Lock()
	defer f.mu.Unlock()
	roleCopy := *role
	f.roles[role.ID] = &roleCopy
}

// GetRoleByID retrieves a role for testing (thread-safe)
func (f *RoleRepo) GetRoleByID(id uuid.UUID) *Role {
	f.mu.RLock()
	defer f.mu.RUnlock()
	role, exists := f.roles[id]
	if !exists {
		return nil
	}
	roleCopy := *role
	return &roleCopy
}

// GetRoleByNameDirect retrieves a role by name for testing (thread-safe)
func (f *RoleRepo) GetRoleByNameDirect(name string) *Role {
	f.mu.RLock()
	defer f.mu.RUnlock()
	for _, role := range f.roles {
		if role.Name == name {
			roleCopy := *role
			return &roleCopy
		}
	}
	return nil
}

// CallCount returns the number of calls to a specific method
func (f *RoleRepo) CallCount(method string) int {
	f.mu.RLock()
	defer f.mu.RUnlock()
	
	switch method {
	case "Create":
		return len(f.CreateCalls)
	case "Get":
		return len(f.GetCalls)
	case "GetByName":
		return len(f.GetByNameCalls)
	case "Save":
		return len(f.SaveCalls)
	case "Delete":
		return len(f.DeleteCalls)
	case "List":
		return len(f.ListCalls)
	case "ListByStatus":
		return len(f.ListByStatusCalls)
	default:
		return 0
	}
}