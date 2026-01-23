package settings

import (
	"strings"
	"sync"
)

// Registry holds setting schemas.
type Registry struct {
	mu      sync.RWMutex
	schemas map[string]Schema
}

// NewRegistry creates a new registry.
func NewRegistry() *Registry {
	return &Registry{schemas: make(map[string]Schema)}
}

// Register adds a schema to the registry.
func (r *Registry) Register(s Schema) {
	r.mu.Lock()
	r.schemas[s.Key] = s
	r.mu.Unlock()
}

// Get retrieves a schema by key.
func (r *Registry) Get(key string) (Schema, bool) {
	r.mu.RLock()
	s, ok := r.schemas[key]
	r.mu.RUnlock()
	return s, ok
}

// All returns all registered schemas.
func (r *Registry) All() []Schema {
	r.mu.RLock()
	defer r.mu.RUnlock()

	out := make([]Schema, 0, len(r.schemas))
	for _, s := range r.schemas {
		out = append(out, s)
	}
	return out
}

// ByPrefix returns schemas whose keys start with the given prefix.
func (r *Registry) ByPrefix(prefix string) []Schema {
	r.mu.RLock()
	defer r.mu.RUnlock()

	var out []Schema
	for _, s := range r.schemas {
		if strings.HasPrefix(s.Key, prefix) {
			out = append(out, s)
		}
	}
	return out
}
