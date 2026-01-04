package model

import "slices"

// HasRole checks if the given roles slice contains the specified role.
// If "superadmin" is in roles, always returns true (bypass).
func HasRole(roles []string, role string) bool {
	if slices.Contains(roles, "superadmin") {
		return true
	}
	return slices.Contains(roles, role)
}
