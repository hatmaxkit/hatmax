package seed

import "fmt"

// Ref represents a symbolic reference to an entity before its UUID is known.
type Ref string

// RefMap maps symbolic references to actual UUIDs after entity creation.
type RefMap map[Ref]string

// NewRefMap creates an empty RefMap.
func NewRefMap() RefMap {
	return make(RefMap)
}

// Register associates a symbolic reference with a UUID.
func (m RefMap) Register(ref Ref, uuid string) {
	m[ref] = uuid
}

// Resolve returns the UUID for a given reference.
// Returns an error if the reference was not previously registered.
func (m RefMap) Resolve(ref Ref) (string, error) {
	if uuid, ok := m[ref]; ok {
		return uuid, nil
	}

	return "", fmt.Errorf("unresolved reference: %s", ref)
}

// MustResolve returns the UUID for a given reference.
// Panics if the reference was not previously registered.
func (m RefMap) MustResolve(ref Ref) string {
	uuid, err := m.Resolve(ref)
	if err != nil {
		panic(err)
	}

	return uuid
}

// Has checks if a reference is registered.
func (m RefMap) Has(ref Ref) bool {
	_, ok := m[ref]

	return ok
}
