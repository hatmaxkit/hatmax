package settings

// NamespaceSchema defines metadata for a settings namespace.
// This is optional - apps can use it for UI organization.
type NamespaceSchema struct {
	Key         string
	Label       string
	Description string
}

// DisplayLabel returns the label or falls back to the key.
func (n NamespaceSchema) DisplayLabel() string {
	if n.Label != "" {
		return n.Label
	}

	return n.Key
}
